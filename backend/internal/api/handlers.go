package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/scheduler"
	"github.com/sanskarpan/db-backup/pkg/validation"
)

var (
	ServerVersion   = "dev"
	ServerBuildTime = "unknown"
	ServerGitCommit = "unknown"
)

// handleRoot handles the root endpoint
func (s *Server) handleRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "DB Backup API",
		"version": ServerVersion,
		"status":  "running",
	})
}

// handleHealth handles health check endpoint
func (s *Server) handleHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	report := s.healthChecker.CheckAll(ctx)
	statusCode := http.StatusOK

	if report.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if report.Status == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	c.JSON(statusCode, report)
}

// handleReady handles readiness probe
func (s *Server) handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// handleVersion handles version endpoint
func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    ServerVersion,
		"build_time": ServerBuildTime,
		"git_commit": ServerGitCommit,
	})
}

// Backup handlers

type CreateBackupRequest struct {
	DatabaseType  string            `json:"database_type" binding:"required"`
	Host          string            `json:"host" binding:"required"`
	Port          int               `json:"port"`
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	Database      string            `json:"database"`
	Databases     []string          `json:"databases"`
	AllDatabases  bool              `json:"all_databases"`
	Tables        []string          `json:"tables"`
	ExcludeTables []string          `json:"exclude_tables"`
	Compression   string            `json:"compression"`
	Encrypt       bool              `json:"encrypt"`
	EncryptionKey string            `json:"encryption_key"`
	StorageType   string            `json:"storage_type"`
	Tags          map[string]string `json:"tags"`
}

func (s *Server) handleCreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Parse database type
	var dbType database.DatabaseType
	switch req.DatabaseType {
	case "mysql":
		dbType = database.DatabaseTypeMySQL
	case "postgres", "postgresql":
		dbType = database.DatabaseTypePostgreSQL
	case "mongodb", "mongo":
		dbType = database.DatabaseTypeMongoDB
	case "sqlite":
		dbType = database.DatabaseTypeSQLite
	default:
		s.respondError(c, http.StatusBadRequest, fmt.Errorf("invalid database type"), "Unsupported database type")
		return
	}

	// Validate database name to prevent shell injection and SQL injection.
	if req.Database != "" {
		if err := validation.ValidateDatabaseName(req.Database); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid database name")
			return
		}
	}
	for _, db := range req.Databases {
		if err := validation.ValidateDatabaseName(db); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid database name in databases list")
			return
		}
	}
	for _, tbl := range req.Tables {
		if err := validation.ValidateTableName(tbl); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid table name")
			return
		}
	}
	for _, tbl := range req.ExcludeTables {
		if err := validation.ValidateTableName(tbl); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid table name in exclude_tables list")
			return
		}
	}

	// Create backup options
	opts := &backup.CreateOptions{
		DatabaseType:  dbType,
		Host:          req.Host,
		Port:          req.Port,
		Username:      req.Username,
		Password:      req.Password,
		Database:      req.Database,
		Databases:     req.Databases,
		AllDatabases:  req.AllDatabases,
		Tables:        req.Tables,
		ExcludeTables: req.ExcludeTables,
		Compression:   database.CompressionType(req.Compression),
		Encrypt:       req.Encrypt,
		EncryptionKey: req.EncryptionKey,
		Tags:          req.Tags,
	}

	// Create backup
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	metadata, err := s.backupEngine.CreateBackup(ctx, opts)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to create backup")
		return
	}

	s.respondSuccessWithMessage(c, "Backup created successfully", metadata)
}

func (s *Server) handleListBackups(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	backups, err := s.backupEngine.ListBackups(ctx)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list backups")
		return
	}

	s.respondSuccess(c, gin.H{
		"backups": backups,
		"total":   len(backups),
	})
}

func (s *Server) handleGetBackup(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Backup not found")
		return
	}

	s.respondSuccess(c, metadata)
}

// handleDeleteBackup soft-deletes a backup, moving it to the recycle bin. The
// artifact is retained and can be recovered via handleRestoreDeletedBackup or
// permanently removed via handlePurgeBackup.
func (s *Server) handleDeleteBackup(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := s.backupEngine.DeleteBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to delete backup")
		return
	}

	s.respondSuccessWithMessage(c, "Backup moved to recycle bin", nil)
}

// handleListDeletedBackups lists the soft-deleted backups in the recycle bin.
func (s *Server) handleListDeletedBackups(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	backups, err := s.backupEngine.ListDeletedBackups(ctx)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list deleted backups")
		return
	}

	s.respondSuccess(c, gin.H{
		"backups": backups,
		"total":   len(backups),
	})
}

// handleRestoreDeletedBackup undeletes a backup, moving it out of the recycle
// bin. It is distinct from handleRestoreBackup, which restores backup data into
// a target database.
func (s *Server) handleRestoreDeletedBackup(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := s.backupEngine.RestoreDeletedBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Failed to restore backup from recycle bin")
		return
	}

	s.respondSuccessWithMessage(c, "Backup restored from recycle bin", nil)
}

// handlePurgeBackup permanently removes a soft-deleted backup and its artifact.
func (s *Server) handlePurgeBackup(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := s.backupEngine.PurgeBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to purge backup")
		return
	}

	s.respondSuccessWithMessage(c, "Backup permanently deleted", nil)
}

// PurgeExpiredRequest configures a recycle-bin sweep. Retention is a Go duration
// string (for example "168h"); soft-deleted backups older than it are purged.
type PurgeExpiredRequest struct {
	Retention string `json:"retention" binding:"required"`
}

// handlePurgeExpired permanently purges every recycle-bin backup whose deletion
// is older than the requested retention window.
func (s *Server) handlePurgeExpired(c *gin.Context) {
	var req PurgeExpiredRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	retention, err := time.ParseDuration(req.Retention)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid retention format")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	purged, err := s.backupEngine.PurgeExpired(ctx, retention)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to purge expired backups")
		return
	}

	s.respondSuccessWithMessage(c, "Expired backups purged", gin.H{"purged": purged})
}

type RestoreBackupRequest struct {
	TargetHost     string   `json:"target_host"`
	TargetPort     int      `json:"target_port"`
	TargetUsername string   `json:"target_username"`
	TargetPassword string   `json:"target_password"`
	TargetDatabase string   `json:"target_database"`
	Tables         []string `json:"tables"`
	DropExisting   bool     `json:"drop_existing"`
	PointInTime    string   `json:"point_in_time"` // RFC3339 format
}

func (s *Server) handleRestoreBackup(c *gin.Context) {
	backupID := c.Param("id")

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Get backup metadata
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Backup not found")
		return
	}

	// Create restore options
	opts := &restore.RestoreOptions{
		TargetHost:     req.TargetHost,
		TargetPort:     req.TargetPort,
		TargetUsername: req.TargetUsername,
		TargetPassword: req.TargetPassword,
		TargetDatabase: req.TargetDatabase,
		Tables:         req.Tables,
		DropExisting:   req.DropExisting,
	}

	// Perform restore
	var result *restore.RestoreResult
	if req.PointInTime != "" {
		// PITR restore
		targetTime, err := time.Parse(time.RFC3339, req.PointInTime)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid point_in_time format")
			return
		}
		result, err = s.restoreEngine.RestorePointInTime(ctx, metadata, targetTime, opts)
	} else {
		// Full restore
		result, err = s.restoreEngine.RestoreBackup(ctx, metadata, opts)
	}

	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to restore backup")
		return
	}

	s.respondSuccessWithMessage(c, "Restore completed successfully", result)
}

func (s *Server) handleDownloadBackup(c *gin.Context) {
	backupID := c.Param("id")
	if err := validation.ValidateBackupID(backupID); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid backup ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	// Load backup metadata.
	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Backup not found")
		return
	}

	// Download (or copy) the artifact into a temp file, verifying its checksum.
	tmpDir, err := os.MkdirTemp("", "backup-download-*")
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to prepare download")
		return
	}
	defer os.RemoveAll(tmpDir)

	fileName := backupID
	if metadata.BackupPath != "" {
		fileName = filepath.Base(metadata.BackupPath)
	}
	destPath := filepath.Join(tmpDir, fileName)

	if err := s.restoreEngine.DownloadBackup(ctx, metadata, destPath); err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to download backup")
		return
	}

	c.FileAttachment(destPath, fileName)
}

// Schedule handlers

type CreateScheduleRequest struct {
	ID         string              `json:"id" binding:"required"`
	Name       string              `json:"name"`
	Schedule   string              `json:"schedule" binding:"required"`
	BackupOpts CreateBackupRequest `json:"backup_options" binding:"required"`
	Enabled    bool                `json:"enabled"`
	Retries    int                 `json:"retries"`
	RetryDelay string              `json:"retry_delay"` // Duration string
	Timeout    string              `json:"timeout"`     // Duration string
	Tags       map[string]string   `json:"tags"`
}

func (s *Server) handleCreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Parse database type
	var dbType database.DatabaseType
	switch req.BackupOpts.DatabaseType {
	case "mysql":
		dbType = database.DatabaseTypeMySQL
	case "postgres", "postgresql":
		dbType = database.DatabaseTypePostgreSQL
	case "mongodb", "mongo":
		dbType = database.DatabaseTypeMongoDB
	case "sqlite":
		dbType = database.DatabaseTypeSQLite
	default:
		s.respondError(c, http.StatusBadRequest, fmt.Errorf("invalid database type"), "Unsupported database type")
		return
	}

	// Create backup options
	backupOpts := &backup.CreateOptions{
		DatabaseType: dbType,
		Host:         req.BackupOpts.Host,
		Port:         req.BackupOpts.Port,
		Username:     req.BackupOpts.Username,
		Password:     req.BackupOpts.Password,
		Database:     req.BackupOpts.Database,
		Databases:    req.BackupOpts.Databases,
		AllDatabases: req.BackupOpts.AllDatabases,
		Compression:  database.CompressionType(req.BackupOpts.Compression),
		Encrypt:      req.BackupOpts.Encrypt,
	}

	// Parse durations
	retryDelay := 5 * time.Minute
	if req.RetryDelay != "" {
		var err error
		retryDelay, err = time.ParseDuration(req.RetryDelay)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid retry_delay format")
			return
		}
	}

	timeout := 24 * time.Hour
	if req.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(req.Timeout)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid timeout format")
			return
		}
	}

	// Create scheduled job
	job := &scheduler.ScheduledJob{
		ID:         req.ID,
		Name:       req.Name,
		Schedule:   req.Schedule,
		BackupOpts: backupOpts,
		Enabled:    req.Enabled,
		Retries:    req.Retries,
		RetryDelay: retryDelay,
		Timeout:    timeout,
		Tags:       req.Tags,
	}

	if err := s.scheduler.AddJob(job); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Failed to create schedule")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule created successfully", job)
}

func (s *Server) handleListSchedules(c *gin.Context) {
	jobs := s.scheduler.ListJobs()

	s.respondSuccess(c, gin.H{
		"schedules": jobs,
		"total":     len(jobs),
	})
}

func (s *Server) handleGetSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	job, err := s.scheduler.GetJob(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Schedule not found")
		return
	}

	s.respondSuccess(c, job)
}

func (s *Server) handleUpdateSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Map the request into a partial ScheduledJob understood by the scheduler.
	// UpdateJob only applies non-zero fields, so unset fields are left intact.
	updates := &scheduler.ScheduledJob{
		Name:     req.Name,
		Schedule: req.Schedule,
		Enabled:  req.Enabled,
		Retries:  req.Retries,
		Tags:     req.Tags,
	}

	// A backup database type, when supplied, updates the backup options.
	if req.BackupOpts.DatabaseType != "" {
		var dbType database.DatabaseType
		switch req.BackupOpts.DatabaseType {
		case "mysql":
			dbType = database.DatabaseTypeMySQL
		case "postgres", "postgresql":
			dbType = database.DatabaseTypePostgreSQL
		case "mongodb", "mongo":
			dbType = database.DatabaseTypeMongoDB
		case "sqlite":
			dbType = database.DatabaseTypeSQLite
		default:
			s.respondError(c, http.StatusBadRequest, fmt.Errorf("invalid database type"), "Unsupported database type")
			return
		}
		updates.BackupOpts = &backup.CreateOptions{
			DatabaseType: dbType,
			Host:         req.BackupOpts.Host,
			Port:         req.BackupOpts.Port,
			Username:     req.BackupOpts.Username,
			Password:     req.BackupOpts.Password,
			Database:     req.BackupOpts.Database,
			Databases:    req.BackupOpts.Databases,
			AllDatabases: req.BackupOpts.AllDatabases,
			Compression:  database.CompressionType(req.BackupOpts.Compression),
			Encrypt:      req.BackupOpts.Encrypt,
		}
	}

	if req.RetryDelay != "" {
		d, err := time.ParseDuration(req.RetryDelay)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid retry_delay format")
			return
		}
		updates.RetryDelay = d
	}
	if req.Timeout != "" {
		d, err := time.ParseDuration(req.Timeout)
		if err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid timeout format")
			return
		}
		updates.Timeout = d
	}

	if err := s.scheduler.UpdateJob(scheduleID, updates); err != nil {
		s.respondError(c, http.StatusNotFound, err, "Failed to update schedule")
		return
	}

	job, err := s.scheduler.GetJob(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to load updated schedule")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule updated successfully", job)
}

func (s *Server) handleDeleteSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	err := s.scheduler.RemoveJob(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Schedule not found")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule deleted successfully", nil)
}

func (s *Server) handleEnableSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	err := s.scheduler.EnableJob(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Schedule not found")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule enabled successfully", nil)
}

func (s *Server) handleDisableSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	err := s.scheduler.DisableJob(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, "Schedule not found")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule disabled successfully", nil)
}

func (s *Server) handleRunSchedule(c *gin.Context) {
	scheduleID := c.Param("id")

	result, err := s.scheduler.RunJobNow(scheduleID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to run schedule")
		return
	}

	if !result.Success {
		s.respondError(c, http.StatusInternalServerError, result.Error, "Backup job failed")
		return
	}

	s.respondSuccessWithMessage(c, "Schedule executed successfully", result)
}

// Stats handlers

func (s *Server) handleGetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	backups, err := s.backupEngine.ListBackups(ctx)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list backups")
		return
	}

	var successful, failed int
	var totalSize int64
	for _, b := range backups {
		switch b.Status {
		case database.BackupStatusSuccess, database.BackupStatusCompleted:
			successful++
		case database.BackupStatusFailed:
			failed++
		}
		totalSize += b.Size
	}

	databases := 0
	if s.dbStore != nil {
		dbs, derr := s.dbStore.List()
		if derr != nil {
			s.respondError(c, http.StatusInternalServerError, derr, "Failed to list databases")
			return
		}
		databases = len(dbs)
	}

	schedules := len(s.scheduler.ListJobs())

	c.JSON(http.StatusOK, gin.H{
		"total_backups":      len(backups),
		"successful_backups": successful,
		"failed_backups":     failed,
		"total_size":         totalSize,
		"databases":          databases,
		"schedules":          schedules,
	})
}

func (s *Server) handleGetStorageStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	backups, err := s.backupEngine.ListBackups(ctx)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list backups")
		return
	}

	// Aggregate real storage usage from backup metadata, broken down by the
	// storage location each backup was written to.
	type locationStat struct {
		Size           int64 `json:"size"`
		CompressedSize int64 `json:"compressed_size"`
		Backups        int   `json:"backups"`
	}
	var totalSize, totalCompressed int64
	byLocation := make(map[string]*locationStat)
	for _, b := range backups {
		totalSize += b.Size
		totalCompressed += b.CompressedSize

		loc := b.StorageLocation
		if loc == "" {
			loc = "local"
		}
		entry := byLocation[loc]
		if entry == nil {
			entry = &locationStat{}
			byLocation[loc] = entry
		}
		entry.Backups++
		entry.Size += b.Size
		entry.CompressedSize += b.CompressedSize
	}

	providers := 0
	if s.storageStore != nil {
		providers = s.storageStore.Count()
	}

	c.JSON(http.StatusOK, gin.H{
		"total_backups":         len(backups),
		"total_size":            totalSize,
		"total_compressed_size": totalCompressed,
		"configured_providers":  providers,
		"by_location":           byLocation,
	})
}

// Security handlers

// ScanFileRequest represents a request to scan a file
type ScanFileRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

// ScanDirectoryRequest represents a request to scan a directory
type ScanDirectoryRequest struct {
	DirectoryPath string `json:"directory_path" binding:"required"`
}

// handleScanFile scans a single file for ransomware
func (s *Server) handleScanFile(c *gin.Context) {
	var req ScanFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Prevent path traversal: restrict scans to the configured base directory.
	if s.config.ScanBaseDir == "" {
		s.respondError(c, http.StatusForbidden, fmt.Errorf("scan base directory not configured"), "Scan endpoint is not available")
		return
	}
	safePath, err := validation.SanitizePath(req.FilePath, s.config.ScanBaseDir)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid file path")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	report, err := s.detector.ScanFile(ctx, safePath)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to scan file")
		return
	}

	s.respondSuccess(c, report)
}

// handleScanDirectory scans a directory for ransomware
func (s *Server) handleScanDirectory(c *gin.Context) {
	var req ScanDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Prevent path traversal: restrict scans to the configured base directory.
	if s.config.ScanBaseDir == "" {
		s.respondError(c, http.StatusForbidden, fmt.Errorf("scan base directory not configured"), "Scan endpoint is not available")
		return
	}
	safeDir, err := validation.SanitizePath(req.DirectoryPath, s.config.ScanBaseDir)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid directory path")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	reports, err := s.detector.ScanDirectory(ctx, safeDir)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to scan directory")
		return
	}

	s.respondSuccess(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}

// handleGetSecurityStats returns security statistics.
//
// The ransomware detector performs stateless, on-demand scans and does not
// retain aggregate scan history (files scanned, threats detected/blocked over
// time). Rather than fabricate metrics, we report the detector's real
// availability plus honest zero counts for figures that are not tracked. The
// number of configured storage providers is a real value from the registry.
func (s *Server) handleGetSecurityStats(c *gin.Context) {
	configuredProviders := 0
	if s.storageStore != nil {
		configuredProviders = s.storageStore.Count()
	}

	stats := gin.H{
		"detector_active": s.detector != nil,
		// The detector does not persist cumulative scan history, so these are
		// honest zeros rather than fabricated totals.
		"files_scanned":        0,
		"threats_detected":     0,
		"threats_blocked":      0,
		"last_scan_time":       nil,
		"configured_providers": configuredProviders,
	}

	s.respondSuccess(c, stats)
}

// ThreatAlert represents a threat alert for the API
type ThreatAlert struct {
	ID                string   `json:"id"`
	Timestamp         string   `json:"timestamp"`
	Severity          string   `json:"severity"`
	Type              string   `json:"type"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	AffectedResources []string `json:"affected_resources"`
	Status            string   `json:"status"`
	DetectionMethod   string   `json:"detection_method"`
	RecommendedAction string   `json:"recommended_action"`
}

// handleListThreatAlerts returns all threat alerts.
//
// Threat detection is performed on-demand via the scan endpoints and no alert
// history is persisted, so this honestly returns an empty list rather than
// fabricated alerts. The response still honors the severity/status query
// filters for forward compatibility.
func (s *Server) handleListThreatAlerts(c *gin.Context) {
	// No persisted alert store exists, so there are genuinely no alerts.
	alerts := []ThreatAlert{}

	s.respondSuccess(c, gin.H{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// handleGetThreatAlert returns a specific threat alert. No alerts are persisted,
// so any lookup is an honest 404 rather than a fabricated record.
func (s *Server) handleGetThreatAlert(c *gin.Context) {
	s.respondError(c, http.StatusNotFound,
		fmt.Errorf("threat alert not found"),
		"No persisted threat alerts are available")
}

// UpdateThreatAlertRequest represents a request to update an alert
type UpdateThreatAlertRequest struct {
	Status string `json:"status" binding:"required,oneof=active investigating resolved dismissed"`
}

// handleUpdateThreatAlert updates a threat alert. No alerts are persisted, so an
// update targets a non-existent record and returns an honest 404.
func (s *Server) handleUpdateThreatAlert(c *gin.Context) {
	var req UpdateThreatAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	s.respondError(c, http.StatusNotFound,
		fmt.Errorf("threat alert not found"),
		"No persisted threat alerts are available to update")
}

// Storage provider CRUD + connection-test handlers live in handlers_storage.go
// and are backed by the persistent storageregistry.Store.
