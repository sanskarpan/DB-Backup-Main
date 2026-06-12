// Package encryption provides Vault integration for key management
package encryption

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path"
	"time"

	vault "github.com/hashicorp/vault/api"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// VaultKeyStore implements KeyStore using HashiCorp Vault
type VaultKeyStore struct {
	client     *vault.Client
	mountPath  string
	keyPrefix  string
	currentKey string
}

// VaultConfig holds Vault configuration
type VaultConfig struct {
	Address    string
	Token      string
	MountPath  string // KV secrets engine mount path (default: "secret")
	KeyPrefix  string // Prefix for keys in Vault (default: "db-backup/encryption")
	Namespace  string // Vault namespace (enterprise feature)
	CurrentKey string // Current key ID to use
}

// NewVaultKeyStore creates a new Vault-backed key store
func NewVaultKeyStore(ctx context.Context, config *VaultConfig) (*VaultKeyStore, error) {
	if config.Address == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Vault address is required")
	}
	if config.Token == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Vault token is required")
	}

	// Create Vault client
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to create Vault client: %v", err))
	}

	// Set token
	client.SetToken(config.Token)

	// Set namespace if provided (Vault Enterprise feature)
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	// Set defaults
	mountPath := config.MountPath
	if mountPath == "" {
		mountPath = "secret"
	}

	keyPrefix := config.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "db-backup/encryption"
	}

	currentKey := config.CurrentKey
	if currentKey == "" {
		currentKey = "master"
	}

	store := &VaultKeyStore{
		client:     client,
		mountPath:  mountPath,
		keyPrefix:  keyPrefix,
		currentKey: currentKey,
	}

	// Test connection
	if _, err := client.Sys().Health(); err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNetwork, fmt.Sprintf("failed to connect to Vault: %v", err))
	}

	return store, nil
}

// GetKey retrieves an encryption key by ID from Vault
func (v *VaultKeyStore) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	secretPath := v.buildPath(keyID)

	// Read from KV v2 (default in newer Vault versions)
	secret, err := v.client.KVv2(v.mountPath).Get(ctx, secretPath)
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to read key from Vault: %v", err))
	}

	if secret == nil || secret.Data == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("key %s not found in Vault", keyID))
	}

	// Extract key data
	keyData, ok := secret.Data["key"].(string)
	if !ok {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, "invalid key format in Vault")
	}

	return []byte(keyData), nil
}

// StoreKey stores an encryption key in Vault
func (v *VaultKeyStore) StoreKey(ctx context.Context, keyID string, key []byte) error {
	secretPath := v.buildPath(keyID)

	// Prepare secret data
	data := map[string]interface{}{
		"key":        string(key),
		"created_at": time.Now().Format(time.RFC3339),
		"algorithm":  "AES-256-GCM",
	}

	// Write to KV v2
	_, err := v.client.KVv2(v.mountPath).Put(ctx, secretPath, data)
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to store key in Vault: %v", err))
	}

	return nil
}

// RotateKey creates a new key version
func (v *VaultKeyStore) RotateKey(ctx context.Context, keyID string) (newKeyID string, newKey []byte, err error) {
	// Get current versions
	versions, err := v.ListKeyVersions(ctx, keyID)
	if err != nil && !isNotFoundError(err) {
		return "", nil, err
	}

	// Calculate next version
	nextVersion := 1
	if len(versions) > 0 {
		for _, v := range versions {
			if v.Version >= nextVersion {
				nextVersion = v.Version + 1
			}
		}
	}

	// Generate new key
	newKey = make([]byte, 32) // 256 bits for AES-256
	if _, err := rand.Read(newKey); err != nil {
		return "", nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to generate new key: %v", err))
	}

	// Create new key ID with version
	newKeyID = fmt.Sprintf("%s-v%d", keyID, nextVersion)

	// Store the new key
	if err := v.StoreKey(ctx, newKeyID, newKey); err != nil {
		return "", nil, err
	}

	// Mark the new key as current
	if err := v.setCurrentKey(ctx, newKeyID); err != nil {
		return "", nil, err
	}

	// Optionally deprecate old versions (but don't delete for re-encryption)
	if len(versions) > 0 {
		for _, oldVersion := range versions {
			if err := v.deprecateKey(ctx, oldVersion.ID); err != nil {
				// Log but don't fail
				continue
			}
		}
	}

	return newKeyID, newKey, nil
}

// ListKeyVersions returns all versions of a key
func (v *VaultKeyStore) ListKeyVersions(ctx context.Context, keyID string) ([]KeyVersion, error) {
	// List secrets under the key prefix
	basePath := path.Join(v.keyPrefix, keyID)

	// Use KV v2 list
	secret, err := v.client.Logical().List(fmt.Sprintf("%s/metadata/%s", v.mountPath, basePath))
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to list key versions: %v", err))
	}

	if secret == nil || secret.Data == nil {
		return []KeyVersion{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []KeyVersion{}, nil
	}

	var versions []KeyVersion
	for _, k := range keys {
		keyName, ok := k.(string)
		if !ok {
			continue
		}

		// Parse version from key name (format: keyID-vN)
		var version int
		var versionID string
		if _, err := fmt.Sscanf(keyName, "%s-v%d", &versionID, &version); err == nil {
			versions = append(versions, KeyVersion{
				ID:      path.Join(basePath, keyName),
				Version: version,
			})
		}
	}

	return versions, nil
}

// DeleteKey deletes a key from Vault
func (v *VaultKeyStore) DeleteKey(ctx context.Context, keyID string) error {
	secretPath := v.buildPath(keyID)

	// Delete from KV v2 (this creates a soft delete, can be recovered)
	err := v.client.KVv2(v.mountPath).Delete(ctx, secretPath)
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to delete key from Vault: %v", err))
	}

	return nil
}

// GetCurrentKeyID returns the ID of the current active key
func (v *VaultKeyStore) GetCurrentKeyID(ctx context.Context) (string, error) {
	metadataPath := path.Join(v.keyPrefix, "_metadata", "current")

	secret, err := v.client.KVv2(v.mountPath).Get(ctx, metadataPath)
	if err != nil {
		// If metadata doesn't exist, return default
		return v.currentKey, nil
	}

	if secret == nil || secret.Data == nil {
		return v.currentKey, nil
	}

	currentKeyID, ok := secret.Data["current_key_id"].(string)
	if !ok {
		return v.currentKey, nil
	}

	return currentKeyID, nil
}

// Close closes the Vault client connection
func (v *VaultKeyStore) Close() error {
	// Vault client doesn't require explicit closing
	return nil
}

// Helper methods

func (v *VaultKeyStore) buildPath(keyID string) string {
	return path.Join(v.keyPrefix, keyID)
}

func (v *VaultKeyStore) setCurrentKey(ctx context.Context, keyID string) error {
	metadataPath := path.Join(v.keyPrefix, "_metadata", "current")

	data := map[string]interface{}{
		"current_key_id": keyID,
		"updated_at":     time.Now().Format(time.RFC3339),
	}

	_, err := v.client.KVv2(v.mountPath).Put(ctx, metadataPath, data)
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeInternal, fmt.Sprintf("failed to update current key metadata: %v", err))
	}

	v.currentKey = keyID
	return nil
}

func (v *VaultKeyStore) deprecateKey(ctx context.Context, keyID string) error {
	secretPath := v.buildPath(keyID)

	// Read current data
	secret, err := v.client.KVv2(v.mountPath).Get(ctx, secretPath)
	if err != nil {
		return err
	}

	if secret == nil || secret.Data == nil {
		return nil
	}

	// Update with deprecated flag
	data := secret.Data
	data["deprecated"] = true
	data["deprecated_at"] = time.Now().Format(time.RFC3339)

	_, err = v.client.KVv2(v.mountPath).Put(ctx, secretPath, data)
	return err
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var backupErr *pkgErrors.BackupError
	if errors.As(err, &backupErr) {
		return backupErr.Type == pkgErrors.ErrorTypeNotFound
	}
	return false
}

// FileKeyStore implements a simple file-based key store for testing/development
type FileKeyStore struct {
	basePath    string
	currentKey  string
	keys        map[string][]byte
	keyVersions map[string][]KeyVersion
}

// NewFileKeyStore creates a file-based key store
func NewFileKeyStore(basePath string, currentKey string) *FileKeyStore {
	if currentKey == "" {
		currentKey = "master"
	}

	return &FileKeyStore{
		basePath:    basePath,
		currentKey:  currentKey,
		keys:        make(map[string][]byte),
		keyVersions: make(map[string][]KeyVersion),
	}
}

// GetKey retrieves a key from memory/file
func (f *FileKeyStore) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	key, exists := f.keys[keyID]
	if !exists {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("key %s not found", keyID))
	}
	return key, nil
}

// StoreKey stores a key in memory/file
func (f *FileKeyStore) StoreKey(ctx context.Context, keyID string, key []byte) error {
	f.keys[keyID] = key

	// Extract base key name and version from keyID
	baseKey := keyID
	version := 1

	// Try to parse version from key ID (e.g., "master-v2")
	// Look for "-v" followed by a number
	if idx := len(keyID) - 1; idx >= 0 {
		for i := len(keyID) - 1; i >= 2; i-- {
			if keyID[i-2:i] == "-v" {
				// Found version suffix
				var v int
				if n, err := fmt.Sscanf(keyID[i:], "%d", &v); n == 1 && err == nil {
					baseKey = keyID[:i-2]
					version = v
					break
				}
			}
		}
	}

	// Store version info
	if _, exists := f.keyVersions[baseKey]; !exists {
		f.keyVersions[baseKey] = []KeyVersion{}
	}

	// Check if this version already exists
	for _, v := range f.keyVersions[baseKey] {
		if v.ID == keyID {
			// Already stored, don't add duplicate
			return nil
		}
	}

	f.keyVersions[baseKey] = append(f.keyVersions[baseKey], KeyVersion{
		ID:        keyID,
		Version:   version,
		CreatedAt: time.Now(),
		IsActive:  true,
	})

	return nil
}

// RotateKey creates a new key version
func (f *FileKeyStore) RotateKey(ctx context.Context, keyID string) (newKeyID string, newKey []byte, err error) {
	versions := f.keyVersions[keyID]
	nextVersion := len(versions) + 1

	// Generate new key
	newKey = make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return "", nil, err
	}

	newKeyID = fmt.Sprintf("%s-v%d", keyID, nextVersion)
	f.StoreKey(ctx, newKeyID, newKey)
	f.currentKey = newKeyID

	return newKeyID, newKey, nil
}

// ListKeyVersions returns all versions
func (f *FileKeyStore) ListKeyVersions(ctx context.Context, keyID string) ([]KeyVersion, error) {
	versions, exists := f.keyVersions[keyID]
	if !exists {
		return []KeyVersion{}, nil
	}
	return versions, nil
}

// DeleteKey removes a key
func (f *FileKeyStore) DeleteKey(ctx context.Context, keyID string) error {
	delete(f.keys, keyID)
	return nil
}

// GetCurrentKeyID returns current key ID
func (f *FileKeyStore) GetCurrentKeyID(ctx context.Context) (string, error) {
	return f.currentKey, nil
}

// Close closes the store
func (f *FileKeyStore) Close() error {
	return nil
}
