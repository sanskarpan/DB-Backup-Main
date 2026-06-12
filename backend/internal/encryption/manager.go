// Package encryption provides encryption management with key rotation
package encryption

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// Manager manages encryption with key rotation support
type Manager struct {
	keyStore      KeyStore
	currentKeyID  string
	encryptors    map[string]*AESEncryptor
	mu            sync.RWMutex
	rotationTimer *time.Ticker
	ctx           context.Context
	cancel        context.CancelFunc
}

// ManagerConfig holds configuration for the encryption manager
type ManagerConfig struct {
	KeyStore          KeyStore
	CurrentKeyID      string
	RotationInterval  time.Duration
	AutoRotate        bool
	ReencryptOnRotate bool
}

// NewManager creates a new encryption manager
func NewManager(ctx context.Context, config *ManagerConfig) (*Manager, error) {
	if config.KeyStore == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "key store is required")
	}

	mgrCtx, cancel := context.WithCancel(ctx)

	manager := &Manager{
		keyStore:     config.KeyStore,
		currentKeyID: config.CurrentKeyID,
		encryptors:   make(map[string]*AESEncryptor),
		ctx:          mgrCtx,
		cancel:       cancel,
	}

	// Load current key
	if err := manager.loadCurrentKey(); err != nil {
		cancel()
		return nil, err
	}

	// Setup automatic rotation if enabled
	if config.AutoRotate && config.RotationInterval > 0 {
		manager.rotationTimer = time.NewTicker(config.RotationInterval)
		go manager.autoRotate()
	}

	return manager, nil
}

// Encrypt encrypts data with the current key
func (m *Manager) Encrypt(plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	encryptor, exists := m.encryptors[m.currentKeyID]
	m.mu.RUnlock()

	if !exists {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, "no active encryption key")
	}

	// Encrypt the data
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	// Create envelope with metadata
	envelope := &EncryptionEnvelope{
		KeyID:      m.currentKeyID,
		Algorithm:  "AES-256-GCM",
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Timestamp:  time.Now(),
	}

	// Serialize envelope
	envelopeData, err := json.Marshal(envelope)
	if err != nil {
		return nil, pkgErrors.ErrEncryptionFailed(err)
	}

	return envelopeData, nil
}

// Decrypt decrypts data, automatically using the correct key version
func (m *Manager) Decrypt(envelopeData []byte) ([]byte, error) {
	// Parse envelope
	var envelope EncryptionEnvelope
	if err := json.Unmarshal(envelopeData, &envelope); err != nil {
		return nil, pkgErrors.ErrDecryptionFailed(err)
	}

	// Get or load the encryptor for this key version
	encryptor, err := m.getEncryptor(envelope.KeyID)
	if err != nil {
		return nil, err
	}

	// Decode ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, pkgErrors.ErrDecryptionFailed(err)
	}

	// Decrypt
	plaintext, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptFile encrypts a file
func (m *Manager) EncryptFile(inputPath, outputPath string) error {
	// Read input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}
	defer inputFile.Close()

	plaintext, err := io.ReadAll(inputFile)
	if err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Encrypt
	envelopeData, err := m.Encrypt(plaintext)
	if err != nil {
		return err
	}

	// Write encrypted data
	if err := os.WriteFile(outputPath, envelopeData, 0600); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	return nil
}

// DecryptFile decrypts a file
func (m *Manager) DecryptFile(inputPath, outputPath string) error {
	// Read encrypted file
	envelopeData, err := os.ReadFile(inputPath)
	if err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	// Decrypt
	plaintext, err := m.Decrypt(envelopeData)
	if err != nil {
		return err
	}

	// Write decrypted data
	if err := os.WriteFile(outputPath, plaintext, 0600); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	return nil
}

// RotateKey rotates the encryption key
func (m *Manager) RotateKey(ctx context.Context) error {
	// Perform key rotation
	newKeyID, newKey, err := m.keyStore.RotateKey(ctx, m.currentKeyID)
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("key rotation failed: %v", err))
	}

	// Create encryptor for new key
	encryptor, err := NewAESEncryptor(newKey)
	if err != nil {
		return err
	}

	// Update current key
	m.mu.Lock()
	m.encryptors[newKeyID] = encryptor
	m.currentKeyID = newKeyID
	m.mu.Unlock()

	return nil
}

// ReencryptWithNewKey re-encrypts data with the current key
func (m *Manager) ReencryptWithNewKey(oldEnvelopeData []byte) ([]byte, error) {
	// Decrypt with old key
	plaintext, err := m.Decrypt(oldEnvelopeData)
	if err != nil {
		return nil, err
	}

	// Encrypt with current key
	return m.Encrypt(plaintext)
}

// ReencryptDirectory re-encrypts all encrypted files in a directory
func (m *Manager) ReencryptDirectory(ctx context.Context, dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file appears to be encrypted (has .enc extension or is JSON envelope)
		if filepath.Ext(path) != ".enc" {
			return nil
		}

		// Read encrypted file
		oldData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Try to parse as envelope
		var envelope EncryptionEnvelope
		if err := json.Unmarshal(oldData, &envelope); err != nil {
			// Not an encrypted envelope, skip
			return nil
		}

		// Check if already using current key
		if envelope.KeyID == m.currentKeyID {
			return nil // Already using current key
		}

		// Re-encrypt
		newData, err := m.ReencryptWithNewKey(oldData)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt %s: %w", path, err)
		}

		// Write back
		if err := os.WriteFile(path, newData, info.Mode()); err != nil {
			return fmt.Errorf("failed to write re-encrypted file %s: %w", path, err)
		}

		return nil
	})
}

// GetCurrentKeyID returns the current active key ID
func (m *Manager) GetCurrentKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKeyID
}

// Close stops the manager and closes resources
func (m *Manager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	if m.rotationTimer != nil {
		m.rotationTimer.Stop()
	}
	return m.keyStore.Close()
}

// Helper methods

func (m *Manager) loadCurrentKey() error {
	// Get current key ID from store
	keyID, err := m.keyStore.GetCurrentKeyID(context.Background())
	if err != nil {
		return err
	}

	// Load the key
	key, err := m.keyStore.GetKey(context.Background(), keyID)
	if err != nil {
		return err
	}

	// Create encryptor
	encryptor, err := NewAESEncryptor(key)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.currentKeyID = keyID
	m.encryptors[keyID] = encryptor
	m.mu.Unlock()

	return nil
}

func (m *Manager) getEncryptor(keyID string) (*AESEncryptor, error) {
	m.mu.RLock()
	encryptor, exists := m.encryptors[keyID]
	m.mu.RUnlock()

	if exists {
		return encryptor, nil
	}

	// Load the key from store
	key, err := m.keyStore.GetKey(context.Background(), keyID)
	if err != nil {
		return nil, err
	}

	// Create encryptor
	encryptor, err = NewAESEncryptor(key)
	if err != nil {
		return nil, err
	}

	// Cache it
	m.mu.Lock()
	m.encryptors[keyID] = encryptor
	m.mu.Unlock()

	return encryptor, nil
}

func (m *Manager) autoRotate() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.rotationTimer.C:
			if err := m.RotateKey(m.ctx); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}

// EncryptionEnvelope wraps encrypted data with metadata
type EncryptionEnvelope struct {
	KeyID      string    `json:"key_id"`
	Algorithm  string    `json:"algorithm"`
	Ciphertext string    `json:"ciphertext"`
	Timestamp  time.Time `json:"timestamp"`
}

// EncryptStream encrypts a stream using the current key
func (m *Manager) EncryptStream(input io.Reader, output io.Writer) error {
	m.mu.RLock()
	encryptor, exists := m.encryptors[m.currentKeyID]
	keyID := m.currentKeyID
	m.mu.RUnlock()

	if !exists {
		return pkgErrors.New(pkgErrors.ErrorTypeInternal, "no active encryption key")
	}

	// Write metadata header
	metadata := map[string]string{
		"key_id":    keyID,
		"algorithm": "AES-256-GCM-Stream",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Write metadata length (4 bytes) + metadata
	metadataLen := len(metadataJSON)
	header := make([]byte, 4)
	header[0] = byte(metadataLen >> 24)
	header[1] = byte(metadataLen >> 16)
	header[2] = byte(metadataLen >> 8)
	header[3] = byte(metadataLen)

	if _, err := output.Write(header); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}
	if _, err := output.Write(metadataJSON); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Encrypt stream
	return encryptor.EncryptStream(input, output)
}

// DecryptStream decrypts a stream
func (m *Manager) DecryptStream(input io.Reader, output io.Writer) error {
	// Read metadata header
	header := make([]byte, 4)
	if _, err := io.ReadFull(input, header); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	metadataLen := int(header[0])<<24 | int(header[1])<<16 | int(header[2])<<8 | int(header[3])
	metadataJSON := make([]byte, metadataLen)
	if _, err := io.ReadFull(input, metadataJSON); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	// Parse metadata
	var metadata map[string]string
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	keyID := metadata["key_id"]

	// Get encryptor for this key
	encryptor, err := m.getEncryptor(keyID)
	if err != nil {
		return err
	}

	// Decrypt stream
	return encryptor.DecryptStream(input, output)
}

// GenerateKey generates a new random encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits for AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to generate key: %v", err))
	}
	return key, nil
}

// InitializeKeyStore initializes a key store with an initial key
func InitializeKeyStore(ctx context.Context, keyStore KeyStore, keyID string) error {
	// Generate initial key
	key, err := GenerateKey()
	if err != nil {
		return err
	}

	// Store it
	if err := keyStore.StoreKey(ctx, keyID, key); err != nil {
		return err
	}

	return nil
}
