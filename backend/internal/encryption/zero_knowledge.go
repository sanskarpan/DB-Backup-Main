// Package encryption provides zero-knowledge encryption for database backups
// ensuring the server never sees plaintext data
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// ZeroKnowledgeEncryptor implements client-side encryption where the server
// never has access to plaintext data or encryption keys
type ZeroKnowledgeEncryptor struct {
	algorithm Algorithm
}

// Algorithm represents the encryption algorithm to use
type Algorithm string

const (
	AlgorithmAES256GCM     Algorithm = "aes-256-gcm"
	AlgorithmChaCha20Poly1305 Algorithm = "chacha20-poly1305"
)

// KeyDerivationParams holds parameters for key derivation
type KeyDerivationParams struct {
	Time    uint32 // Argon2 time parameter
	Memory  uint32 // Argon2 memory parameter (in KB)
	Threads uint8  // Argon2 parallelism parameter
	KeyLen  uint32 // Derived key length
}

// DefaultKeyDerivationParams returns secure default parameters
func DefaultKeyDerivationParams() *KeyDerivationParams {
	return &KeyDerivationParams{
		Time:    3,
		Memory:  64 * 1024, // 64 MB
		Threads: 4,
		KeyLen:  32, // 256 bits
	}
}

// NewZeroKnowledgeEncryptor creates a new zero-knowledge encryptor
func NewZeroKnowledgeEncryptor(alg Algorithm) (*ZeroKnowledgeEncryptor, error) {
	if alg != AlgorithmAES256GCM && alg != AlgorithmChaCha20Poly1305 {
		return nil, fmt.Errorf("unsupported algorithm: %s", alg)
	}

	return &ZeroKnowledgeEncryptor{
		algorithm: alg,
	}, nil
}

// DeriveKey derives an encryption key from a password using Argon2id
// This ensures the server never sees the actual encryption key
func (e *ZeroKnowledgeEncryptor) DeriveKey(password string, salt []byte, params *KeyDerivationParams) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("password cannot be empty")
	}
	if len(salt) < 16 {
		return nil, errors.New("salt must be at least 16 bytes")
	}
	if params == nil {
		params = DefaultKeyDerivationParams()
	}

	// Use Argon2id for key derivation (recommended by OWASP)
	key := argon2.IDKey(
		[]byte(password),
		salt,
		params.Time,
		params.Memory,
		params.Threads,
		params.KeyLen,
	)

	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt
func (e *ZeroKnowledgeEncryptor) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32) // 256 bits
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts data using client-side encryption
// The server never sees the plaintext or encryption key
func (e *ZeroKnowledgeEncryptor) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext cannot be empty")
	}
	if len(key) == 0 {
		return nil, errors.New("encryption key cannot be empty")
	}

	switch e.algorithm {
	case AlgorithmAES256GCM:
		return e.encryptAESGCM(plaintext, key)
	case AlgorithmChaCha20Poly1305:
		return e.encryptChaCha20Poly1305(plaintext, key)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", e.algorithm)
	}
}

// Decrypt decrypts data that was encrypted with zero-knowledge encryption
func (e *ZeroKnowledgeEncryptor) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext cannot be empty")
	}
	if len(key) == 0 {
		return nil, errors.New("decryption key cannot be empty")
	}

	switch e.algorithm {
	case AlgorithmAES256GCM:
		return e.decryptAESGCM(ciphertext, key)
	case AlgorithmChaCha20Poly1305:
		return e.decryptChaCha20Poly1305(ciphertext, key)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", e.algorithm)
	}
}

// encryptAESGCM encrypts using AES-256-GCM
func (e *ZeroKnowledgeEncryptor) encryptAESGCM(plaintext []byte, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) != 32 {
		key = e.deriveAESKey(key)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAESGCM decrypts using AES-256-GCM
func (e *ZeroKnowledgeEncryptor) decryptAESGCM(ciphertext []byte, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) != 32 {
		key = e.deriveAESKey(key)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// encryptChaCha20Poly1305 encrypts using ChaCha20-Poly1305
func (e *ZeroKnowledgeEncryptor) encryptChaCha20Poly1305(plaintext []byte, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes
	if len(key) != 32 {
		key = e.deriveAESKey(key)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptChaCha20Poly1305 decrypts using ChaCha20-Poly1305
func (e *ZeroKnowledgeEncryptor) decryptChaCha20Poly1305(ciphertext []byte, key []byte) ([]byte, error) {
	// Ensure key is 32 bytes
	if len(key) != 32 {
		key = e.deriveAESKey(key)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// deriveAESKey derives a 32-byte key using SHA-256
func (e *ZeroKnowledgeEncryptor) deriveAESKey(key []byte) []byte {
	hash := sha256.Sum256(key)
	return hash[:]
}

// EncryptStream encrypts a stream of data
func (e *ZeroKnowledgeEncryptor) EncryptStream(reader io.Reader, writer io.Writer, key []byte) error {
	// Read all data (for simplicity, in production use chunked encryption)
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read plaintext: %w", err)
	}

	ciphertext, err := e.Encrypt(plaintext, key)
	if err != nil {
		return err
	}

	if _, err := writer.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write ciphertext: %w", err)
	}

	return nil
}

// DecryptStream decrypts a stream of data
func (e *ZeroKnowledgeEncryptor) DecryptStream(reader io.Reader, writer io.Writer, key []byte) error {
	// Read all data (for simplicity, in production use chunked decryption)
	ciphertext, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read ciphertext: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertext, key)
	if err != nil {
		return err
	}

	if _, err := writer.Write(plaintext); err != nil {
		return fmt.Errorf("failed to write plaintext: %w", err)
	}

	return nil
}

// EncryptBackup encrypts a backup file using zero-knowledge encryption
func (e *ZeroKnowledgeEncryptor) EncryptBackup(backupData []byte, password string) (*EncryptedBackup, error) {
	// Generate salt
	salt, err := e.GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Derive key from password
	key, err := e.DeriveKey(password, salt, nil)
	if err != nil {
		return nil, err
	}

	// Encrypt backup
	ciphertext, err := e.Encrypt(backupData, key)
	if err != nil {
		return nil, err
	}

	return &EncryptedBackup{
		Ciphertext: ciphertext,
		Salt:       salt,
		Algorithm:  string(e.algorithm),
		Version:    1,
	}, nil
}

// DecryptBackup decrypts a backup file using zero-knowledge encryption
func (e *ZeroKnowledgeEncryptor) DecryptBackup(encBackup *EncryptedBackup, password string) ([]byte, error) {
	// Derive key from password
	key, err := e.DeriveKey(password, encBackup.Salt, nil)
	if err != nil {
		return nil, err
	}

	// Decrypt backup
	plaintext, err := e.Decrypt(encBackup.Ciphertext, key)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// EncryptedBackup represents an encrypted backup
type EncryptedBackup struct {
	Ciphertext []byte `json:"ciphertext"`
	Salt       []byte `json:"salt"`
	Algorithm  string `json:"algorithm"`
	Version    int    `json:"version"`
}

// ToBase64 encodes the encrypted backup to base64
func (eb *EncryptedBackup) ToBase64() string {
	return base64.StdEncoding.EncodeToString(eb.Ciphertext)
}

// VerifyIntegrity verifies the integrity of encrypted data
func (e *ZeroKnowledgeEncryptor) VerifyIntegrity(ciphertext []byte, key []byte) (bool, error) {
	// Try to decrypt - if it fails, integrity check fails
	_, err := e.Decrypt(ciphertext, key)
	return err == nil, err
}
