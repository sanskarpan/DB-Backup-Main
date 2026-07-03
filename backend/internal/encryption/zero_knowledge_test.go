package encryption

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZeroKnowledgeEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		algorithm Algorithm
		wantError bool
	}{
		{"AES-256-GCM", AlgorithmAES256GCM, false},
		{"ChaCha20-Poly1305", AlgorithmChaCha20Poly1305, false},
		{"Invalid algorithm", Algorithm("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewZeroKnowledgeEncryptor(tt.algorithm)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
				assert.Equal(t, tt.algorithm, encryptor.algorithm)
			}
		})
	}
}

func TestGenerateSalt(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	salt1, err := encryptor.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, 32)

	salt2, err := encryptor.GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt2, 32)

	// Salts should be different
	assert.NotEqual(t, salt1, salt2)
}

func TestDeriveKey(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	salt, _ := encryptor.GenerateSalt()
	password := "strong-password-123"

	key1, err := encryptor.DeriveKey(password, salt, nil)
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	// Same password and salt should produce same key
	key2, err := encryptor.DeriveKey(password, salt, nil)
	require.NoError(t, err)
	assert.Equal(t, key1, key2)

	// Different salt should produce different key
	salt2, _ := encryptor.GenerateSalt()
	key3, err := encryptor.DeriveKey(password, salt2, nil)
	require.NoError(t, err)
	assert.NotEqual(t, key1, key3)
}

func TestDeriveKey_EmptyPassword(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)
	salt, _ := encryptor.GenerateSalt()

	_, err := encryptor.DeriveKey("", salt, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password cannot be empty")
}

func TestDeriveKey_ShortSalt(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	_, err := encryptor.DeriveKey("password", []byte("short"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "salt must be at least 16 bytes")
}

func TestEncryptDecrypt_AES256GCM(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("This is sensitive backup data that must be encrypted")
	password := "strong-password-123"
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey(password, salt, nil)

	// Encrypt
	ciphertext, err := encryptor.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	assert.Greater(t, len(ciphertext), len(plaintext))

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_ChaCha20Poly1305(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmChaCha20Poly1305)

	plaintext := []byte("This is sensitive backup data encrypted with ChaCha20")
	password := "strong-password-456"
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey(password, salt, nil)

	// Encrypt
	ciphertext, err := encryptor.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	_, err := encryptor.Encrypt([]byte{}, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plaintext cannot be empty")
}

func TestEncrypt_EmptyKey(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)
	plaintext := []byte("data")

	_, err := encryptor.Encrypt(plaintext, []byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryption key cannot be empty")
}

func TestDecrypt_WrongKey(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("secret data")
	salt, _ := encryptor.GenerateSalt()
	key1, _ := encryptor.DeriveKey("password1", salt, nil)
	key2, _ := encryptor.DeriveKey("password2", salt, nil)

	ciphertext, _ := encryptor.Encrypt(plaintext, key1)

	// Decrypting with wrong key should fail
	_, err := encryptor.Decrypt(ciphertext, key2)
	assert.Error(t, err)
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("secret data")
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	ciphertext, _ := encryptor.Encrypt(plaintext, key)

	// Corrupt the ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	// Decryption should fail
	_, err := encryptor.Decrypt(ciphertext, key)
	assert.Error(t, err)
}

func TestEncryptBackup(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	backupData := []byte("database backup content with sensitive information")
	password := "backup-password-123"

	encBackup, err := encryptor.EncryptBackup(backupData, password)
	require.NoError(t, err)
	assert.NotNil(t, encBackup)
	assert.NotEmpty(t, encBackup.Ciphertext)
	assert.NotEmpty(t, encBackup.Salt)
	assert.Equal(t, string(AlgorithmAES256GCM), encBackup.Algorithm)
	assert.Equal(t, 1, encBackup.Version)
}

func TestDecryptBackup(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	backupData := []byte("original backup data")
	password := "secure-password"

	// Encrypt
	encBackup, err := encryptor.EncryptBackup(backupData, password)
	require.NoError(t, err)

	// Decrypt
	decrypted, err := encryptor.DecryptBackup(encBackup, password)
	require.NoError(t, err)
	assert.Equal(t, backupData, decrypted)
}

func TestDecryptBackup_WrongPassword(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	backupData := []byte("secret backup")
	password1 := "correct-password"
	password2 := "wrong-password"

	encBackup, _ := encryptor.EncryptBackup(backupData, password1)

	// Decrypt with wrong password should fail
	_, err := encryptor.DecryptBackup(encBackup, password2)
	assert.Error(t, err)
}

func TestEncryptStream(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("stream data to encrypt")
	reader := bytes.NewReader(plaintext)
	writer := &bytes.Buffer{}

	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	err := encryptor.EncryptStream(reader, writer, key)
	require.NoError(t, err)
	assert.Greater(t, writer.Len(), len(plaintext))
}

func TestDecryptStream(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("stream data to encrypt and decrypt")
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	// Encrypt
	encryptedBuf := &bytes.Buffer{}
	reader := bytes.NewReader(plaintext)
	err := encryptor.EncryptStream(reader, encryptedBuf, key)
	require.NoError(t, err)

	// Decrypt
	decryptedBuf := &bytes.Buffer{}
	err = encryptor.DecryptStream(encryptedBuf, decryptedBuf, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decryptedBuf.Bytes())
}

func TestVerifyIntegrity(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	plaintext := []byte("data to verify")
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	ciphertext, _ := encryptor.Encrypt(plaintext, key)

	// Valid ciphertext
	valid, err := encryptor.VerifyIntegrity(ciphertext, key)
	assert.True(t, valid)
	assert.NoError(t, err)

	// Corrupted ciphertext
	ciphertext[0] ^= 0xFF
	valid, err = encryptor.VerifyIntegrity(ciphertext, key)
	assert.False(t, valid)
	assert.Error(t, err)
}

func TestEncryptedBackup_ToBase64(t *testing.T) {
	encBackup := &EncryptedBackup{
		Ciphertext: []byte("encrypted data"),
		Salt:       []byte("salt value"),
		Algorithm:  "aes-256-gcm",
		Version:    1,
	}

	base64Str := encBackup.ToBase64()
	assert.NotEmpty(t, base64Str)
	assert.Contains(t, base64Str, "ZW5jcnlwdGVkIGRhdGE=") // base64 of "encrypted data"
}

func TestLargeDataEncryption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data test in short mode")
	}

	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmChaCha20Poly1305)

	// Create 10MB of data
	largeData := make([]byte, 10*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	// Encrypt
	ciphertext, err := encryptor.Encrypt(largeData, key)
	require.NoError(t, err)

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, largeData, decrypted)
}

func TestCustomKeyDerivationParams(t *testing.T) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)

	customParams := &KeyDerivationParams{
		Time:    1,
		Memory:  32 * 1024,
		Threads: 2,
		KeyLen:  32,
	}

	salt, _ := encryptor.GenerateSalt()
	key, err := encryptor.DeriveKey("password", salt, customParams)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

// Benchmark tests.
func BenchmarkEncryptAES256GCM(b *testing.B) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)
	plaintext := make([]byte, 1024*1024) // 1MB
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Encrypt(plaintext, key)
	}
}

func BenchmarkEncryptChaCha20(b *testing.B) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmChaCha20Poly1305)
	plaintext := make([]byte, 1024*1024) // 1MB
	salt, _ := encryptor.GenerateSalt()
	key, _ := encryptor.DeriveKey("password", salt, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.Encrypt(plaintext, key)
	}
}

func BenchmarkKeyDerivation(b *testing.B) {
	encryptor, _ := NewZeroKnowledgeEncryptor(AlgorithmAES256GCM)
	salt, _ := encryptor.GenerateSalt()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encryptor.DeriveKey("password", salt, nil)
	}
}
