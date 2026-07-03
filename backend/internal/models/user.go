// Package models provides common data models used across the application
package models

import (
	"time"
)

// User represents a user in the system.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose password hash
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
	IsActive     bool      `json:"is_active"`
}

// Schedule represents a backup schedule.
type Schedule struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DatabaseType string            `json:"database_type"`
	DatabaseName string            `json:"database_name"`
	CronExpr     string            `json:"cron_expr"`
	Enabled      bool              `json:"enabled"`
	LastRunAt    time.Time         `json:"last_run_at,omitempty"`
	NextRunAt    time.Time         `json:"next_run_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// DatabaseConnection represents a database connection configuration.
type DatabaseConnection struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // mysql, postgres, mongodb, sqlite
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Database  string            `json:"database"`
	Username  string            `json:"username"`
	Password  string            `json:"-"` // Never expose password
	SSLMode   string            `json:"ssl_mode,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	CreatedAt time.Time              `json:"created_at"`
}

// StorageStats represents storage statistics.
type StorageStats struct {
	TotalBackups     int64            `json:"total_backups"`
	TotalSize        int64            `json:"total_size"` // bytes
	OldestBackup     time.Time        `json:"oldest_backup,omitempty"`
	NewestBackup     time.Time        `json:"newest_backup,omitempty"`
	BackupsByType    map[string]int64 `json:"backups_by_type"`
	BackupsByStatus  map[string]int64 `json:"backups_by_status"`
	AverageSize      int64            `json:"average_size"`
	CompressionRatio float64          `json:"compression_ratio,omitempty"`
}
