package encryption

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32) // 256-bit key
	_, err := rand.Read(key)
	require.NoError(t, err)

	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)
	testData := []byte("This is sensitive backup data that needs to be encrypted!")

	// Test Encrypt
	encrypted, err := encryptor.Encrypt(testData)
	require.NoError(t, err)
	assert.Greater(t, len(encrypted), 0)
	assert.NotEqual(t, testData, encrypted)

	// Test Decrypt
	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestAESEncryptor_DifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	encryptor1, err := NewAESEncryptor(key1)
	require.NoError(t, err)
	encryptor2, err := NewAESEncryptor(key2)
	require.NoError(t, err)

	testData := []byte("Secret data")

	// Encrypt with key1
	encrypted, err := encryptor1.Encrypt(testData)
	require.NoError(t, err)

	// Try to decrypt with key2 (should fail)
	_, err = encryptor2.Decrypt(encrypted)
	assert.Error(t, err) // Should fail with wrong key
}

func TestAESEncryptor_EmptyData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)
	testData := []byte("")

	encrypted, err := encryptor.Encrypt(testData)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestAESEncryptor_LargeData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// Create 1MB of test data
	testData := make([]byte, 1024*1024)
	rand.Read(testData)

	encrypted, err := encryptor.Encrypt(testData)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testData, decrypted)
}

func TestAESEncryptor_KeyHashing(t *testing.T) {
	// Test with different key sizes (should be hashed to 256 bits)
	keys := [][]byte{
		[]byte("short"),
		[]byte("this is a longer key"),
		make([]byte, 16), // 128 bits
		make([]byte, 32), // 256 bits
		make([]byte, 64), // 512 bits
	}

	testData := []byte("Test data")

	for _, key := range keys {
		t.Run("", func(t *testing.T) {
			encryptor, err := NewAESEncryptor(key)
			require.NoError(t, err)

			encrypted, err := encryptor.Encrypt(testData)
			require.NoError(t, err)

			decrypted, err := encryptor.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, testData, decrypted)
		})
	}
}

func TestAESEncryptor_CorruptedData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)
	testData := []byte("Original data")

	// Encrypt
	encrypted, err := encryptor.Encrypt(testData)
	require.NoError(t, err)

	// Corrupt the encrypted data
	corruptedData := make([]byte, len(encrypted))
	copy(corruptedData, encrypted)
	if len(corruptedData) > 10 {
		corruptedData[10] ^= 0xFF // Flip bits
	}

	// Try to decrypt corrupted data
	_, err = encryptor.Decrypt(corruptedData)
	assert.Error(t, err) // Should fail
}

// Benchmark tests.
func BenchmarkAESEncrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewAESEncryptor(key)

	data := make([]byte, 1024*1024) // 1MB
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(data)
	}
}

func BenchmarkAESDecrypt(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewAESEncryptor(key)

	data := make([]byte, 1024*1024)
	rand.Read(data)

	encryptedData, _ := encryptor.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(encryptedData)
	}
}
