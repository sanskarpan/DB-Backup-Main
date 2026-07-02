package models

import (
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
)

// BackupMetadata contains metadata about a backup
type BackupMetadata struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	DatabaseType    database.DatabaseType    `json:"database_type"`
	DatabaseVersion string                   `json:"database_version"`
	Database        string                   `json:"database"`
	Databases       []string                 `json:"databases,omitempty"`
	Tables          []database.TableInfo     `json:"tables"`
	Size            int64                    `json:"size"`
	CompressedSize  int64                    `json:"compressed_size"`
	Compression     database.CompressionType `json:"compression"`
	Encrypted       bool                     `json:"encrypted"`
	Checksum        string                   `json:"checksum"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	// DeletedAt marks a backup as soft-deleted (in the recycle bin). A nil value
	// means the backup is live; a non-nil value is the time it was moved to the
	// recycle bin. It is a pointer so it is omitted from serialized metadata for
	// live backups.
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
	Duration        time.Duration          `json:"duration"`
	Status          database.BackupStatus  `json:"status"`
	Tags            map[string]string      `json:"tags,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	StorageLocation string                 `json:"storage_location"`
	BackupPath      string                 `json:"backup_path"`
}

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusCancelled  BackupStatus = "cancelled"
)
