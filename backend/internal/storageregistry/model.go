// Package storageregistry provides persistent storage for the set of storage
// providers (local, s3, gcs, azure, minio, wasabi, b2) that the platform can
// use to hold backups. It powers the /api/v1/security/storage/providers REST
// endpoints consumed by the web, mobile and browser-extension clients.
//
// Secret configuration values (access keys, secret keys, account keys,
// application keys) are persisted encrypted at rest and are NEVER serialized
// into an API response.
package storageregistry

import (
	"fmt"
	"time"
)

// SupportedTypes is the set of storage provider types accepted by the registry.
// It mirrors the type union expected by the API clients.
var SupportedTypes = map[string]bool{
	"local":  true,
	"s3":     true,
	"gcs":    true,
	"azure":  true,
	"minio":  true,
	"wasabi": true,
	"b2":     true,
}

// secretConfigKeys lists the configuration keys whose values are secrets. Their
// values are stored encrypted at rest and are stripped from every API response.
var secretConfigKeys = map[string]bool{
	"access_key":      true,
	"secret_key":      true,
	"account_key":     true,
	"application_key": true,
}

// StorageProvider is the API-facing representation of a registered storage
// provider. The Config map contains only NON-secret configuration; secret keys
// are removed before a StorageProvider is ever constructed, so they can never
// be serialized into a response.
type StorageProvider struct {
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    map[string]interface{} `json:"config"`
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
}

// CreateRequest is the body accepted by POST /security/storage/providers.
type CreateRequest struct {
	Config  map[string]interface{} `json:"config"`
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Enabled bool                   `json:"enabled"`
}

// UpdateRequest is the body accepted by PUT /security/storage/providers/:id.
// Secret config values are optional on update: if a secret key is absent or
// empty in the request, the existing stored secret is preserved.
type UpdateRequest struct {
	Config  map[string]interface{} `json:"config"`
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Enabled bool                   `json:"enabled"`
}

// ResolvedProvider carries the full configuration (public values merged with
// decrypted secrets) for internal use, e.g. a real connection test. It is never
// serialized into an API response.
type ResolvedProvider struct {
	Config map[string]interface{}
	Type   string
	ID     string
}

// ConnectionTestResponse is returned by POST
// /security/storage/providers/:id/test.
type ConnectionTestResponse struct {
	Message string  `json:"message"`
	Latency float64 `json:"latency,omitempty"` // milliseconds
	Success bool    `json:"success"`
}

// ValidationError describes an invalid field in a create/update request.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ErrNotFound is returned when a storage provider record does not exist.
var ErrNotFound = fmt.Errorf("storage provider not found")

// validate checks the common fields shared by create and update requests.
func validate(name, providerType string) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if providerType == "" {
		return &ValidationError{Field: "type", Message: "type is required"}
	}
	if !SupportedTypes[providerType] {
		return &ValidationError{Field: "type", Message: fmt.Sprintf("unsupported storage provider type: %s", providerType)}
	}
	return nil
}
