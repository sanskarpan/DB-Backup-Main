// Package dbregistry provides persistent storage for the set of databases
// that the platform is configured to back up. It powers the /api/v1/databases
// REST endpoints consumed by the web, mobile and browser-extension clients.
package dbregistry

import (
	"fmt"
	"time"
)

// SupportedTypes is the set of database types accepted by the registry.
// It mirrors the type union expected by the API clients.
var SupportedTypes = map[string]bool{
	"postgres":      true,
	"mysql":         true,
	"mongodb":       true,
	"redis":         true,
	"cassandra":     true,
	"scylladb":      true,
	"elasticsearch": true,
	"opensearch":    true,
	"dynamodb":      true,
	"influxdb":      true,
	"timescaledb":   true,
	"sqlite":        true,
}

// Database is a registered database connection.
//
// The credential fields (Password, Database, SSLMode, Options) are used only
// to establish connections for backups and connection tests. They are tagged
// json:"-" so they are NEVER serialized into an API response. Persistence to
// disk is handled separately by the Store, which serializes credentials into
// its own on-disk record format.
type Database struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Host      string    `json:"host"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Write-only credentials — never returned to clients.
	Password string            `json:"-"`
	Database string            `json:"-"`
	SSLMode  string            `json:"-"`
	Options  map[string]string `json:"-"`

	Port int `json:"port"`
}

// CreateRequest is the body accepted by POST /api/v1/databases.
type CreateRequest struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	SSLMode  string            `json:"ssl_mode"`
	Options  map[string]string `json:"options"`
	Port     int               `json:"port"`
}

// UpdateRequest is the body accepted by PUT /api/v1/databases/:id.
// Password is optional on update: an empty password leaves the stored
// credential unchanged.
type UpdateRequest struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	SSLMode  string            `json:"ssl_mode"`
	Options  map[string]string `json:"options"`
	Port     int               `json:"port"`
}

// ConnectionTestResponse is returned by POST /api/v1/databases/:id/test.
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

// ErrNotFound is returned when a database record does not exist.
var ErrNotFound = fmt.Errorf("database not found")

// validate checks the common fields shared by create and update requests.
func validate(name, dbType, host string, port int) error {
	if name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if dbType == "" {
		return &ValidationError{Field: "type", Message: "type is required"}
	}
	if !SupportedTypes[dbType] {
		return &ValidationError{Field: "type", Message: fmt.Sprintf("unsupported database type: %s", dbType)}
	}
	// SQLite is file-based and does not require host/port.
	if dbType != "sqlite" {
		if host == "" {
			return &ValidationError{Field: "host", Message: "host is required"}
		}
		if port < 1 || port > 65535 {
			return &ValidationError{Field: "port", Message: "port must be between 1 and 65535"}
		}
	}
	return nil
}
