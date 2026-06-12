// Package encryption_test provides comprehensive tests for encryption and key rotation
package encryption

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_EncryptDecrypt(t *testing.T) {
	// Use file-based key store for testing
	keyStore := NewFileKeyStore(t.TempDir(), "test-key")

	// Initialize with a key
	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "test-key")
	require.NoError(t, err)

	// Create manager
	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:     keyStore,
		CurrentKeyID: "test-key",
	})
	require.NoError(t, err)
	defer manager.Close()

	// Test encryption and decryption
	plaintext := []byte("This is sensitive data that needs encryption")

	encrypted, err := manager.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := manager.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestManager_KeyRotation(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "master")

	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "master")
	require.NoError(t, err)

	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:     keyStore,
		CurrentKeyID: "master",
	})
	require.NoError(t, err)
	defer manager.Close()

	// Encrypt data with original key
	plaintext := []byte("Data encrypted with original key")
	encrypted1, err := manager.Encrypt(plaintext)
	require.NoError(t, err)

	// Get original key ID
	originalKeyID := manager.GetCurrentKeyID()

	// Rotate key
	err = manager.RotateKey(ctx)
	require.NoError(t, err)

	// Verify new key ID is different
	newKeyID := manager.GetCurrentKeyID()
	assert.NotEqual(t, originalKeyID, newKeyID)

	// Encrypt with new key
	encrypted2, err := manager.Encrypt(plaintext)
	require.NoError(t, err)

	// Both encrypted versions should decrypt to same plaintext
	decrypted1, err := manager.Decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := manager.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)

	// But encrypted data should be different (different keys)
	assert.NotEqual(t, encrypted1, encrypted2)
}

func TestManager_ReencryptWithNewKey(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "master")

	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "master")
	require.NoError(t, err)

	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:     keyStore,
		CurrentKeyID: "master",
	})
	require.NoError(t, err)
	defer manager.Close()

	// Encrypt with old key
	plaintext := []byte("Data to be re-encrypted")
	oldEncrypted, err := manager.Encrypt(plaintext)
	require.NoError(t, err)

	// Rotate key
	err = manager.RotateKey(ctx)
	require.NoError(t, err)

	// Re-encrypt with new key
	newEncrypted, err := manager.ReencryptWithNewKey(oldEncrypted)
	require.NoError(t, err)

	// Verify re-encrypted data decrypts correctly
	decrypted, err := manager.Decrypt(newEncrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestManager_EncryptFile(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "file-key")

	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "file-key")
	require.NoError(t, err)

	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:     keyStore,
		CurrentKeyID: "file-key",
	})
	require.NoError(t, err)
	defer manager.Close()

	// Create test file
	tmpDir := t.TempDir()
	inputPath := tmpDir + "/input.txt"
	outputPath := tmpDir + "/encrypted.enc"
	decryptedPath := tmpDir + "/decrypted.txt"

	testContent := []byte("This is the content of the test file")
	err = os.WriteFile(inputPath, testContent, 0600)
	require.NoError(t, err)

	// Encrypt file
	err = manager.EncryptFile(inputPath, outputPath)
	require.NoError(t, err)

	// Decrypt file
	err = manager.DecryptFile(outputPath, decryptedPath)
	require.NoError(t, err)

	// Verify decrypted content matches original
	decryptedContent, err := os.ReadFile(decryptedPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, decryptedContent)
}

func TestManager_AutoRotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping auto-rotation test in short mode")
	}

	keyStore := NewFileKeyStore(t.TempDir(), "auto-key")

	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "auto-key")
	require.NoError(t, err)

	// Create manager with very short rotation interval for testing
	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:          keyStore,
		CurrentKeyID:      "auto-key",
		AutoRotate:        true,
		RotationInterval:  2 * time.Second,
		ReencryptOnRotate: false,
	})
	require.NoError(t, err)
	defer manager.Close()

	originalKeyID := manager.GetCurrentKeyID()

	// Wait for auto-rotation
	time.Sleep(3 * time.Second)

	newKeyID := manager.GetCurrentKeyID()
	assert.NotEqual(t, originalKeyID, newKeyID, "Key should have been rotated automatically")
}

func TestFileKeyStore_StoreAndRetrieve(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "test")

	ctx := context.Background()
	testKey := []byte("this-is-a-test-encryption-key-32")

	// Store key
	err := keyStore.StoreKey(ctx, "test-key-1", testKey)
	require.NoError(t, err)

	// Retrieve key
	retrievedKey, err := keyStore.GetKey(ctx, "test-key-1")
	require.NoError(t, err)
	assert.Equal(t, testKey, retrievedKey)
}

func TestFileKeyStore_Rotation(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "rotation-test")

	ctx := context.Background()
	initialKey := []byte("initial-key-32-bytes-long!!!!")
	err := keyStore.StoreKey(ctx, "rotation-test", initialKey)
	require.NoError(t, err)

	// Rotate key
	newKeyID, newKey, err := keyStore.RotateKey(ctx, "rotation-test")
	require.NoError(t, err)
	assert.NotEmpty(t, newKeyID)
	assert.NotEmpty(t, newKey)
	assert.NotEqual(t, initialKey, newKey)

	// Verify new key is current
	currentKeyID, err := keyStore.GetCurrentKeyID(ctx)
	require.NoError(t, err)
	assert.Equal(t, newKeyID, currentKeyID)

	// Verify we can retrieve the new key
	retrievedKey, err := keyStore.GetKey(ctx, newKeyID)
	require.NoError(t, err)
	assert.Equal(t, newKey, retrievedKey)
}

func TestFileKeyStore_ListVersions(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "versioned-key")

	ctx := context.Background()

	// Store initial key
	err := keyStore.StoreKey(ctx, "versioned-key", []byte("key-v1"))
	require.NoError(t, err)

	// Rotate to create version 2
	_, _, err = keyStore.RotateKey(ctx, "versioned-key")
	require.NoError(t, err)

	// List versions
	versions, err := keyStore.ListKeyVersions(ctx, "versioned-key")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(versions), 2, "Should have at least 2 versions")
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key, 32, "Key should be 32 bytes (256 bits)")

	// Generate another key and ensure they're different
	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.NotEqual(t, key, key2, "Generated keys should be unique")
}

func TestManager_StreamEncryptDecrypt(t *testing.T) {
	keyStore := NewFileKeyStore(t.TempDir(), "stream-key")

	ctx := context.Background()
	err := InitializeKeyStore(ctx, keyStore, "stream-key")
	require.NoError(t, err)

	manager, err := NewManager(ctx, &ManagerConfig{
		KeyStore:     keyStore,
		CurrentKeyID: "stream-key",
	})
	require.NoError(t, err)
	defer manager.Close()

	// Create test files
	tmpDir := t.TempDir()
	inputPath := tmpDir + "/input.txt"
	encryptedPath := tmpDir + "/encrypted.bin"
	decryptedPath := tmpDir + "/decrypted.txt"

	testData := []byte("This is streaming data to encrypt and decrypt")
	err = os.WriteFile(inputPath, testData, 0600)
	require.NoError(t, err)

	// Open files for streaming
	inputFile, err := os.Open(inputPath)
	require.NoError(t, err)
	defer inputFile.Close()

	encryptedFile, err := os.Create(encryptedPath)
	require.NoError(t, err)
	defer encryptedFile.Close()

	// Encrypt stream
	err = manager.EncryptStream(inputFile, encryptedFile)
	require.NoError(t, err)
	encryptedFile.Close()

	// Decrypt stream
	encryptedFile, err = os.Open(encryptedPath)
	require.NoError(t, err)
	defer encryptedFile.Close()

	decryptedFile, err := os.Create(decryptedPath)
	require.NoError(t, err)
	defer decryptedFile.Close()

	err = manager.DecryptStream(encryptedFile, decryptedFile)
	require.NoError(t, err)
	decryptedFile.Close()

	// Verify decrypted content matches original
	decryptedData, err := os.ReadFile(decryptedPath)
	require.NoError(t, err)
	assert.Equal(t, testData, decryptedData)
}
