// Package metrics provides Prometheus metrics collection
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Backup metrics
	backupsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_backups_total",
			Help: "Total number of backup operations",
		},
		[]string{"database_type", "status"},
	)

	backupDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_duration_seconds",
			Help:    "Duration of backup operations in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to ~1h
		},
		[]string{"database_type"},
	)

	backupSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_size_bytes",
			Help:    "Size of backups in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to ~1TB
		},
		[]string{"database_type"},
	)

	backupCompressedSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_compressed_size_bytes",
			Help:    "Size of compressed backups in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 20),
		},
		[]string{"database_type", "compression_type"},
	)

	backupCompressionRatio = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_compression_ratio",
			Help:    "Compression ratio of backups",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"database_type", "compression_type"},
	)

	// Restore metrics
	restoresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_restores_total",
			Help: "Total number of restore operations",
		},
		[]string{"database_type", "status"},
	)

	restoreDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_restore_duration_seconds",
			Help:    "Duration of restore operations in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		},
		[]string{"database_type"},
	)

	// Storage metrics
	storageUploads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_storage_uploads_total",
			Help: "Total number of storage uploads",
		},
		[]string{"provider", "status"},
	)

	storageDownloads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_storage_downloads_total",
			Help: "Total number of storage downloads",
		},
		[]string{"provider", "status"},
	)

	storageUploadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_storage_upload_duration_seconds",
			Help:    "Duration of storage uploads in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12),
		},
		[]string{"provider"},
	)

	// Scheduler metrics
	scheduledJobsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_backup_scheduled_jobs_total",
			Help: "Total number of scheduled jobs",
		},
		[]string{"status"}, // enabled, disabled
	)

	scheduledJobExecutions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_scheduled_job_executions_total",
			Help: "Total number of scheduled job executions",
		},
		[]string{"job_id", "status"},
	)

	// Database connection metrics
	databaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_backup_database_connections",
			Help: "Number of active database connections",
		},
		[]string{"database_type"},
	)

	databaseConnectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_database_connection_errors_total",
			Help: "Total number of database connection errors",
		},
		[]string{"database_type"},
	)

	// System metrics
	systemErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_system_errors_total",
			Help: "Total number of system errors",
		},
		[]string{"type", "operation"},
	)
)

// Collector provides methods to record metrics
type Collector struct {
	mu sync.Mutex
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{}
}

// RecordBackup records a backup operation
func (c *Collector) RecordBackup(dbType string, duration time.Duration, originalSize, compressedSize int64, compressionType, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	backupsTotal.WithLabelValues(dbType, status).Inc()
	backupDuration.WithLabelValues(dbType).Observe(duration.Seconds())

	if originalSize > 0 {
		backupSize.WithLabelValues(dbType).Observe(float64(originalSize))
	}

	if compressedSize > 0 {
		backupCompressedSize.WithLabelValues(dbType, compressionType).Observe(float64(compressedSize))

		if originalSize > 0 {
			ratio := float64(compressedSize) / float64(originalSize)
			backupCompressionRatio.WithLabelValues(dbType, compressionType).Observe(ratio)
		}
	}
}

// RecordRestore records a restore operation
func (c *Collector) RecordRestore(dbType string, duration time.Duration, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	restoresTotal.WithLabelValues(dbType, status).Inc()
	restoreDuration.WithLabelValues(dbType).Observe(duration.Seconds())
}

// RecordStorageUpload records a storage upload
func (c *Collector) RecordStorageUpload(provider string, duration time.Duration, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	storageUploads.WithLabelValues(provider, status).Inc()
	storageUploadDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordStorageDownload records a storage download
func (c *Collector) RecordStorageDownload(provider string, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	storageDownloads.WithLabelValues(provider, status).Inc()
}

// UpdateScheduledJobs updates the scheduled jobs gauge
func (c *Collector) UpdateScheduledJobs(enabled, disabled int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	scheduledJobsTotal.WithLabelValues("enabled").Set(float64(enabled))
	scheduledJobsTotal.WithLabelValues("disabled").Set(float64(disabled))
}

// RecordScheduledJobExecution records a scheduled job execution
func (c *Collector) RecordScheduledJobExecution(jobID, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	scheduledJobExecutions.WithLabelValues(jobID, status).Inc()
}

// UpdateDatabaseConnections updates the database connections gauge
func (c *Collector) UpdateDatabaseConnections(dbType string, count int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	databaseConnections.WithLabelValues(dbType).Set(float64(count))
}

// RecordDatabaseConnectionError records a database connection error
func (c *Collector) RecordDatabaseConnectionError(dbType string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	databaseConnectionErrors.WithLabelValues(dbType).Inc()
}

// RecordError records a system error
func (c *Collector) RecordError(errorType, operation string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	systemErrors.WithLabelValues(errorType, operation).Inc()
}

// Global metrics collector instance
var globalCollector = NewCollector()

// GetCollector returns the global metrics collector
func GetCollector() *Collector {
	return globalCollector
}

// RecordBackup is a convenience function to record a backup using the global collector
func RecordBackup(dbType string, duration time.Duration, originalSize, compressedSize int64, compressionType, status string) {
	globalCollector.RecordBackup(dbType, duration, originalSize, compressedSize, compressionType, status)
}

// RecordRestore is a convenience function to record a restore using the global collector
func RecordRestore(dbType string, duration time.Duration, status string) {
	globalCollector.RecordRestore(dbType, duration, status)
}

// RecordStorageUpload is a convenience function to record a storage upload using the global collector
func RecordStorageUpload(provider string, duration time.Duration, status string) {
	globalCollector.RecordStorageUpload(provider, duration, status)
}

// RecordStorageDownload is a convenience function to record a storage download using the global collector
func RecordStorageDownload(provider string, status string) {
	globalCollector.RecordStorageDownload(provider, status)
}

// UpdateScheduledJobs is a convenience function to update scheduled jobs using the global collector
func UpdateScheduledJobs(enabled, disabled int) {
	globalCollector.UpdateScheduledJobs(enabled, disabled)
}

// RecordScheduledJobExecution is a convenience function to record a scheduled job execution using the global collector
func RecordScheduledJobExecution(jobID, status string) {
	globalCollector.RecordScheduledJobExecution(jobID, status)
}

// UpdateDatabaseConnections is a convenience function to update database connections using the global collector
func UpdateDatabaseConnections(dbType string, count int) {
	globalCollector.UpdateDatabaseConnections(dbType, count)
}

// RecordDatabaseConnectionError is a convenience function to record a database connection error using the global collector
func RecordDatabaseConnectionError(dbType string) {
	globalCollector.RecordDatabaseConnectionError(dbType)
}

// RecordError is a convenience function to record an error using the global collector
func RecordError(errorType, operation string) {
	globalCollector.RecordError(errorType, operation)
}
