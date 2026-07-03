package secrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// KeyRotationConfig configures the key rotation system.
type KeyRotationConfig struct {
	// RotationInterval is how often keys should be rotated
	RotationInterval time.Duration `mapstructure:"rotation_interval"`

	// GracePeriod is how long old keys remain valid after rotation
	GracePeriod time.Duration `mapstructure:"grace_period"`

	// KeyLength is the length of generated keys in bytes
	KeyLength int `mapstructure:"key_length"`

	// AutoRotate enables automatic key rotation
	AutoRotate bool `mapstructure:"auto_rotate"`

	// NotifyOnRotation callback for rotation notifications
	NotifyOnRotation func(keyID string, oldKey, newKey string) error
}

// KeyRotationManager manages automatic key rotation.
type KeyRotationManager struct {
	config       *KeyRotationConfig
	vault        *VaultClient
	keys         map[string]*RotatableKey
	mu           sync.RWMutex
	stopChan     chan struct{}
	stopOnce     sync.Once
	rotationChan chan string
}

// RotatableKey represents a key that can be rotated.
type RotatableKey struct {
	ID              string
	CurrentKey      string
	CurrentVersion  int
	PreviousKey     string
	PreviousVersion int
	LastRotation    time.Time
	NextRotation    time.Time
	Path            string
	mu              sync.RWMutex
}

// NewKeyRotationManager creates a new key rotation manager.
func NewKeyRotationManager(config *KeyRotationConfig, vault *VaultClient) *KeyRotationManager {
	if config.KeyLength == 0 {
		config.KeyLength = 32 // Default to 256 bits
	}

	if config.RotationInterval == 0 {
		config.RotationInterval = 30 * 24 * time.Hour // Default to 30 days
	}

	if config.GracePeriod == 0 {
		config.GracePeriod = 24 * time.Hour // Default to 1 day
	}

	return &KeyRotationManager{
		config:       config,
		vault:        vault,
		keys:         make(map[string]*RotatableKey),
		stopChan:     make(chan struct{}),
		rotationChan: make(chan string, 100),
	}
}

// RegisterKey registers a key for rotation.
func (m *KeyRotationManager) RegisterKey(keyID, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if key already exists
	if _, exists := m.keys[keyID]; exists {
		return fmt.Errorf("key %s is already registered", keyID)
	}

	ctx := context.Background()

	// Try to get existing key from Vault
	existingKey, err := m.vault.GetSecret(ctx, path)
	if err != nil {
		// Key doesn't exist, generate new one
		currentKey, err := m.generateKey()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		// Store in Vault
		keyData := map[string]interface{}{
			"current_key":     currentKey,
			"current_version": 1,
			"last_rotation":   time.Now().Unix(),
		}

		if err := m.vault.PutSecret(ctx, path, keyData); err != nil {
			return fmt.Errorf("failed to store key in vault: %w", err)
		}

		m.keys[keyID] = &RotatableKey{
			ID:             keyID,
			CurrentKey:     currentKey,
			CurrentVersion: 1,
			LastRotation:   time.Now(),
			NextRotation:   time.Now().Add(m.config.RotationInterval),
			Path:           path,
		}
	} else {
		// Load existing key
		currentKey := stringField(existingKey, "current_key")
		previousKey := stringField(existingKey, "previous_key")
		currentVersion := floatField(existingKey, "current_version")
		previousVersion := floatField(existingKey, "previous_version")
		lastRotation := floatField(existingKey, "last_rotation")

		m.keys[keyID] = &RotatableKey{
			ID:              keyID,
			CurrentKey:      currentKey,
			CurrentVersion:  int(currentVersion),
			PreviousKey:     previousKey,
			PreviousVersion: int(previousVersion),
			LastRotation:    time.Unix(int64(lastRotation), 0),
			NextRotation:    time.Unix(int64(lastRotation), 0).Add(m.config.RotationInterval),
			Path:            path,
		}
	}

	return nil
}

// stringField returns the string value for key, or "" if absent or of a
// different type.
func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// floatField returns the float64 value for key, or 0 if absent or of a
// different type.
func floatField(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// GetKey retrieves the current key.
func (m *KeyRotationManager) GetKey(keyID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[keyID]
	if !exists {
		return "", fmt.Errorf("key %s not found", keyID)
	}

	key.mu.RLock()
	defer key.mu.RUnlock()

	return key.CurrentKey, nil
}

// GetKeyWithVersion retrieves a specific key version.
func (m *KeyRotationManager) GetKeyWithVersion(keyID string, version int) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[keyID]
	if !exists {
		return "", fmt.Errorf("key %s not found", keyID)
	}

	key.mu.RLock()
	defer key.mu.RUnlock()

	if version == key.CurrentVersion {
		return key.CurrentKey, nil
	}

	if version == key.PreviousVersion && key.PreviousKey != "" {
		// Check if still in grace period
		if time.Since(key.LastRotation) <= m.config.GracePeriod {
			return key.PreviousKey, nil
		}
		return "", fmt.Errorf("key version %d is expired", version)
	}

	return "", fmt.Errorf("key version %d not found", version)
}

// RotateKey manually rotates a key.
func (m *KeyRotationManager) RotateKey(keyID string) error {
	m.mu.RLock()
	key, exists := m.keys[keyID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("key %s not found", keyID)
	}

	key.mu.Lock()
	defer key.mu.Unlock()

	// Generate new key
	newKey, err := m.generateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Store old key
	oldKey := key.CurrentKey
	oldVersion := key.CurrentVersion

	// Update key
	key.PreviousKey = key.CurrentKey
	key.PreviousVersion = key.CurrentVersion
	key.CurrentKey = newKey
	key.CurrentVersion++
	key.LastRotation = time.Now()
	key.NextRotation = time.Now().Add(m.config.RotationInterval)

	// Save to Vault
	ctx := context.Background()
	keyData := map[string]interface{}{
		"current_key":      key.CurrentKey,
		"current_version":  key.CurrentVersion,
		"previous_key":     key.PreviousKey,
		"previous_version": key.PreviousVersion,
		"last_rotation":    key.LastRotation.Unix(),
	}

	if err := m.vault.PutSecret(ctx, key.Path, keyData); err != nil {
		// Rollback on failure
		key.CurrentKey = oldKey
		key.CurrentVersion = oldVersion
		return fmt.Errorf("failed to save rotated key to vault: %w", err)
	}

	// Notify if callback is configured
	if m.config.NotifyOnRotation != nil {
		if err := m.config.NotifyOnRotation(keyID, oldKey, newKey); err != nil {
			// Log error but don't fail rotation
			fmt.Printf("rotation notification failed: %v\n", err)
		}
	}

	return nil
}

// Start starts the automatic key rotation scheduler.
func (m *KeyRotationManager) Start(ctx context.Context) error {
	if !m.config.AutoRotate {
		return nil
	}

	// Check for keys that need rotation every hour
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.closeStopChan()
				return

			case <-ticker.C:
				m.checkAndRotateKeys()

			case keyID := <-m.rotationChan:
				// Manual rotation request
				if err := m.RotateKey(keyID); err != nil {
					fmt.Printf("failed to rotate key %s: %v\n", keyID, err)
				}
			}
		}
	}()

	return nil
}

// checkAndRotateKeys checks all keys and rotates those that need it.
func (m *KeyRotationManager) checkAndRotateKeys() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()

	for keyID, key := range m.keys {
		key.mu.RLock()
		needsRotation := now.After(key.NextRotation)
		key.mu.RUnlock()

		if needsRotation {
			// Queue for rotation
			select {
			case m.rotationChan <- keyID:
			default:
				fmt.Printf("rotation queue full, skipping key %s\n", keyID)
			}
		}
	}
}

// ForceRotation forces immediate rotation of a key.
func (m *KeyRotationManager) ForceRotation(keyID string) error {
	select {
	case m.rotationChan <- keyID:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("rotation queue timeout")
	}
}

// GetKeyInfo returns information about a key.
func (m *KeyRotationManager) GetKeyInfo(keyID string) (*KeyInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key %s not found", keyID)
	}

	key.mu.RLock()
	defer key.mu.RUnlock()

	return &KeyInfo{
		ID:                key.ID,
		CurrentVersion:    key.CurrentVersion,
		PreviousVersion:   key.PreviousVersion,
		LastRotation:      key.LastRotation,
		NextRotation:      key.NextRotation,
		TimeSinceRotation: time.Since(key.LastRotation),
		TimeUntilRotation: time.Until(key.NextRotation),
		HasPreviousKey:    key.PreviousKey != "",
		InGracePeriod:     time.Since(key.LastRotation) <= m.config.GracePeriod,
	}, nil
}

// ListKeys returns all registered keys.
func (m *KeyRotationManager) ListKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.keys))
	for keyID := range m.keys {
		keys = append(keys, keyID)
	}

	return keys
}

// DeleteKey removes a key from rotation management (does not delete from Vault).
func (m *KeyRotationManager) DeleteKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[keyID]; !exists {
		return fmt.Errorf("key %s not found", keyID)
	}

	delete(m.keys, keyID)
	return nil
}

// generateKey generates a cryptographically secure random key.
func (m *KeyRotationManager) generateKey() (string, error) {
	key := make([]byte, m.config.KeyLength)

	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Encode as base64 for storage
	return base64.StdEncoding.EncodeToString(key), nil
}

// Stop stops the key rotation manager.
func (m *KeyRotationManager) Stop() {
	m.closeStopChan()
}

// closeStopChan closes stopChan exactly once, making it safe to call from both
// the Start goroutine (on ctx cancellation) and Stop without risking a
// double-close panic.
func (m *KeyRotationManager) closeStopChan() {
	m.stopOnce.Do(func() {
		close(m.stopChan)
	})
}

// KeyInfo contains information about a key.
type KeyInfo struct {
	ID                string
	CurrentVersion    int
	PreviousVersion   int
	LastRotation      time.Time
	NextRotation      time.Time
	TimeSinceRotation time.Duration
	TimeUntilRotation time.Duration
	HasPreviousKey    bool
	InGracePeriod     bool
}

// EncryptionKeyManager manages encryption keys with rotation.
type EncryptionKeyManager struct {
	rotationManager *KeyRotationManager
	vault           *VaultClient
}

// NewEncryptionKeyManager creates a new encryption key manager.
func NewEncryptionKeyManager(config *KeyRotationConfig, vault *VaultClient) *EncryptionKeyManager {
	return &EncryptionKeyManager{
		rotationManager: NewKeyRotationManager(config, vault),
		vault:           vault,
	}
}

// GetEncryptionKey gets the current encryption key for a purpose.
func (m *EncryptionKeyManager) GetEncryptionKey(purpose string) (key string, version int, err error) {
	keyID := fmt.Sprintf("encryption/%s", purpose)

	// Ensure key is registered
	path := fmt.Sprintf("encryption-keys/%s", purpose)
	if err = m.rotationManager.RegisterKey(keyID, path); err != nil {
		// Key might already be registered, that's okay
		err = nil
	}

	key, err = m.rotationManager.GetKey(keyID)
	if err != nil {
		return "", 0, err
	}

	info, err := m.rotationManager.GetKeyInfo(keyID)
	if err != nil {
		return "", 0, err
	}

	return key, info.CurrentVersion, nil
}

// DecryptWithVersion decrypts data using a specific key version.
func (m *EncryptionKeyManager) DecryptWithVersion(purpose string, version int) (string, error) {
	keyID := fmt.Sprintf("encryption/%s", purpose)

	return m.rotationManager.GetKeyWithVersion(keyID, version)
}

// RotateEncryptionKey rotates an encryption key.
func (m *EncryptionKeyManager) RotateEncryptionKey(purpose string) error {
	keyID := fmt.Sprintf("encryption/%s", purpose)
	return m.rotationManager.RotateKey(keyID)
}

// Start starts the encryption key manager.
func (m *EncryptionKeyManager) Start(ctx context.Context) error {
	return m.rotationManager.Start(ctx)
}

// Stop stops the encryption key manager.
func (m *EncryptionKeyManager) Stop() {
	m.rotationManager.Stop()
}
