package models

import (
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
)

// BackupMetadata contains metadata about a backup.
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

	// Immutable indicates the backup was stored with object-lock (WORM)
	// protection, so it cannot be deleted or altered before its retention
	// expires.
	Immutable bool `json:"immutable,omitempty"`
	// ImmutableUntil is the retention expiry: the backup is protected from
	// deletion until this time. It is a pointer so it is omitted for
	// non-immutable backups.
	ImmutableUntil *time.Time `json:"immutable_until,omitempty"`
	// LockMode is the object-lock retention mode (GOVERNANCE or COMPLIANCE)
	// applied to an immutable backup.
	LockMode string `json:"lock_mode,omitempty"`
	// LegalHold indicates an indefinite legal hold is applied, preventing
	// deletion regardless of the retention expiry.
	LegalHold bool `json:"legal_hold,omitempty"`
}

// BackupStatus represents the status of a backup operation.
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusCancelled  BackupStatus = "canceled"
)
