// Package encryption provides encryption and decryption functionality
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// AESEncryptor implements AES-256-GCM encryption
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor creates a new AES encryptor
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	// Ensure key is 32 bytes (256 bits)
	if len(key) != 32 {
		// Hash the key to get 32 bytes
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	return &AESEncryptor{
		key: key,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, pkgErrors.ErrEncryptionFailed(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, pkgErrors.ErrEncryptionFailed(err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, pkgErrors.ErrEncryptionFailed(err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, pkgErrors.ErrDecryptionFailed(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, pkgErrors.ErrDecryptionFailed(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, pkgErrors.ErrDecryptionFailed(pkgErrors.New(pkgErrors.ErrorTypeEncryption, "ciphertext too short"))
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, pkgErrors.ErrDecryptionFailed(err)
	}

	return plaintext, nil
}

// EncryptStream encrypts a stream of data
func (e *AESEncryptor) EncryptStream(input io.Reader, output io.Writer) error {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Create initialization vector
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Write IV to output
	if _, err := output.Write(iv); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	// Create stream cipher
	stream := cipher.NewCTR(block, iv)
	writer := &cipher.StreamWriter{S: stream, W: output}

	// Encrypt data
	if _, err := io.Copy(writer, input); err != nil {
		return pkgErrors.ErrEncryptionFailed(err)
	}

	return nil
}

// DecryptStream decrypts a stream of data
func (e *AESEncryptor) DecryptStream(input io.Reader, output io.Writer) error {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	// Read IV from input
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(input, iv); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	// Create stream cipher
	stream := cipher.NewCTR(block, iv)
	reader := &cipher.StreamReader{S: stream, R: input}

	// Decrypt data
	if _, err := io.Copy(output, reader); err != nil {
		return pkgErrors.ErrDecryptionFailed(err)
	}

	return nil
}
