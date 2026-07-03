// Package encryption provides encryption key management
package encryption

import (
	"context"
	"time"
)

// KeyStore defines the interface for external key storage.
type KeyStore interface {
	// GetKey retrieves an encryption key by ID
	GetKey(ctx context.Context, keyID string) ([]byte, error)

	// StoreKey stores an encryption key with the given ID
	StoreKey(ctx context.Context, keyID string, key []byte) error

	// RotateKey creates a new key version and returns the new key ID
	RotateKey(ctx context.Context, keyID string) (newKeyID string, newKey []byte, err error)

	// ListKeyVersions returns all versions of a key
	ListKeyVersions(ctx context.Context, keyID string) ([]KeyVersion, error)

	// DeleteKey deletes a key (should be used with caution)
	DeleteKey(ctx context.Context, keyID string) error

	// GetCurrentKeyID returns the ID of the current active key
	GetCurrentKeyID(ctx context.Context) (string, error)

	// Close closes the connection to the key store
	Close() error
}

// KeyVersion represents a version of an encryption key.
type KeyVersion struct {
	ID         string
	Version    int
	CreatedAt  time.Time
	IsActive   bool
	Deprecated bool
}

// KeyMetadata stores metadata about encrypted data.
type KeyMetadata struct {
	KeyID      string
	KeyVersion int
	Algorithm  string
	Timestamp  time.Time
}
