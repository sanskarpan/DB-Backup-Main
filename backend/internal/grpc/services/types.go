//go:build grpc

package services

import (
	"context"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
)

// Placeholder interfaces for services that may not be fully implemented yet
// These will be replaced with actual implementations

// BackupServiceInterface defines the backup service interface
type BackupServiceInterface interface {
	CreateBackup(ctx context.Context, backupID string, config *BackupConfig) (*database.BackupResult, error)
}

// RestoreServiceInterface defines the restore service interface
type RestoreServiceInterface interface {
	Restore(ctx context.Context, restoreID string, config *RestoreConfig) (*RestoreResult, error)
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	DatabaseID        string
	Incremental       bool
	Compression       bool
	CompressionType   string
	Encryption        bool
	StorageProvider   string
	Tags              map[string]string
	RetentionDays     int
	Tables            []string
	ExcludeTables     []string
}

// RestoreConfig holds restore configuration
type RestoreConfig struct {
	BackupID         string
	TargetDatabaseID string
	PointInTime      *time.Time
	Tables           []string
	ExcludeTables    []string
	Options          map[string]string
}

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	RestoreID       string
	BackupID        string
	Status          RestoreStatus
	TargetDatabase  string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	ProgressPercent float64
	BytesRestored   int64
	Error           error
}

// RestoreStatus represents the status of a restore operation
type RestoreStatus int

const (
	RestoreStatusPending RestoreStatus = iota
	RestoreStatusInProgress
	RestoreStatusCompleted
	RestoreStatusFailed
	RestoreStatusCancelled
)

// BackupProgress represents backup progress updates
type BackupProgress struct {
	BackupID        string
	Stage           string
	ProgressPercent float64
	BytesProcessed  int64
	TotalBytes      int64
}

// RestoreProgress represents restore progress updates
type RestoreProgress struct {
	RestoreID       string
	Stage           string
	ProgressPercent float64
	BytesRestored   int64
	TotalBytes      int64
}

// HealthChecker defines health checking interface
type HealthChecker interface {
	Check(ctx context.Context) HealthCheckResult
}

// HealthCheckResult represents health check result
type HealthCheckResult struct {
	Status     HealthStatus
	Components map[string]HealthStatus
}

// HealthStatus represents health status
type HealthStatus int

const (
	HealthStatusUnknown HealthStatus = iota
	HealthStatusHealthy
	HealthStatusDegraded
	HealthStatusUnhealthy
)

// MetricsCollector defines metrics collection interface
type MetricsCollector interface {
	RecordMetric(name string, value float64, labels map[string]string)
}
