package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/scheduler"
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
	DatabaseType string   `json:"database_type" binding:"required"`
	Host         string   `json:"host" binding:"required"`
	Port         int      `json:"port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	Database     string   `json:"database"`
	Databases    []string `json:"databases"`
	AllDatabases bool     `json:"all_databases"`
	Tables       []string `json:"tables"`
	ExcludeTables []string `json:"exclude_tables"`
	Compression  string   `json:"compression"`
	Encrypt      bool     `json:"encrypt"`
	EncryptionKey string  `json:"encryption_key"`
	StorageType  string   `json:"storage_type"`
	Tags         map[string]string `json:"tags"`
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

func (s *Server) handleDeleteBackup(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := s.backupEngine.DeleteBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to delete backup")
		return
	}

	s.respondSuccessWithMessage(c, "Backup deleted successfully", nil)
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

	// This would implement backup file download
	// For now, return not implemented
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Download not implemented yet",
		"backup_id": backupID,
	})
}

// Schedule handlers

type CreateScheduleRequest struct {
	ID           string                  `json:"id" binding:"required"`
	Name         string                  `json:"name"`
	Schedule     string                  `json:"schedule" binding:"required"`
	BackupOpts   CreateBackupRequest     `json:"backup_options" binding:"required"`
	Enabled      bool                    `json:"enabled"`
	Retries      int                     `json:"retries"`
	RetryDelay   string                  `json:"retry_delay"` // Duration string
	Timeout      string                  `json:"timeout"`      // Duration string
	Tags         map[string]string       `json:"tags"`
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
		DatabaseType:  dbType,
		Host:          req.BackupOpts.Host,
		Port:          req.BackupOpts.Port,
		Username:      req.BackupOpts.Username,
		Password:      req.BackupOpts.Password,
		Database:      req.BackupOpts.Database,
		Databases:     req.BackupOpts.Databases,
		AllDatabases:  req.BackupOpts.AllDatabases,
		Compression:   database.CompressionType(req.BackupOpts.Compression),
		Encrypt:       req.BackupOpts.Encrypt,
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
		ID:          req.ID,
		Name:        req.Name,
		Schedule:    req.Schedule,
		BackupOpts:  backupOpts,
		Enabled:     req.Enabled,
		Retries:     req.Retries,
		RetryDelay:  retryDelay,
		Timeout:     timeout,
		Tags:        req.Tags,
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

	// Implementation would update the schedule
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Update not fully implemented",
		"schedule_id": scheduleID,
	})
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
	// Implementation would return statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Stats endpoint - implementation pending",
	})
}

func (s *Server) handleGetStorageStats(c *gin.Context) {
	// Implementation would return storage statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Storage stats endpoint - implementation pending",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	report, err := s.detector.ScanFile(ctx, req.FilePath)
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	reports, err := s.detector.ScanDirectory(ctx, req.DirectoryPath)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to scan directory")
		return
	}

	s.respondSuccess(c, gin.H{
		"reports": reports,
		"total":   len(reports),
	})
}

// handleGetSecurityStats returns security statistics
func (s *Server) handleGetSecurityStats(c *gin.Context) {
	// Mock data - in production, would aggregate from detector history and storage providers
	stats := gin.H{
		"protected_backups": 156,
		"active_scans":      "24/7",
		"threats_blocked":   0,
		"security_score":    98,
		"last_scan_time":    time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
		"files_scanned":     1247,
		"threats_detected":  0,
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

// handleListThreatAlerts returns all threat alerts
func (s *Server) handleListThreatAlerts(c *gin.Context) {
	// Mock data - in production, would fetch from database
	alerts := []ThreatAlert{
		{
			ID:                "1",
			Timestamp:         time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
			Severity:          "INFO",
			Type:              "INFO",
			Title:             "Backup Scan Completed Successfully",
			Description:       "Regular security scan completed with no threats detected",
			AffectedResources: []string{"All backup files"},
			Status:            "resolved",
			DetectionMethod:   "Scheduled Scan",
			RecommendedAction: "No action required",
		},
	}

	severity := c.Query("severity")
	status := c.Query("status")

	// Filter alerts based on query parameters
	filteredAlerts := []ThreatAlert{}
	for _, alert := range alerts {
		if (severity == "" || alert.Severity == severity) &&
			(status == "" || alert.Status == status) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	s.respondSuccess(c, gin.H{
		"alerts": filteredAlerts,
		"total":  len(filteredAlerts),
	})
}

// handleGetThreatAlert returns a specific threat alert
func (s *Server) handleGetThreatAlert(c *gin.Context) {
	alertID := c.Param("id")

	// Mock implementation - in production, fetch from database
	alert := ThreatAlert{
		ID:                alertID,
		Timestamp:         time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
		Severity:          "INFO",
		Type:              "INFO",
		Title:             "Backup Scan Completed Successfully",
		Description:       "Regular security scan completed with no threats detected",
		AffectedResources: []string{"All backup files"},
		Status:            "resolved",
		DetectionMethod:   "Scheduled Scan",
		RecommendedAction: "No action required",
	}

	s.respondSuccess(c, alert)
}

// UpdateThreatAlertRequest represents a request to update an alert
type UpdateThreatAlertRequest struct {
	Status string `json:"status" binding:"required,oneof=active investigating resolved dismissed"`
}

// handleUpdateThreatAlert updates a threat alert
func (s *Server) handleUpdateThreatAlert(c *gin.Context) {
	alertID := c.Param("id")

	var req UpdateThreatAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Mock implementation - in production, update in database
	s.respondSuccessWithMessage(c, "Alert updated successfully", gin.H{
		"alert_id": alertID,
		"status":   req.Status,
	})
}

// StorageProviderConfig represents immutable storage configuration
type StorageProviderConfig struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	Enabled              bool   `json:"enabled"`
	ImmutabilityEnabled  bool   `json:"immutability_enabled"`
	RetentionDays        int    `json:"retention_days"`
	Mode                 string `json:"mode"`
	LegalHold            bool   `json:"legal_hold"`
	ProtectedBackups     int    `json:"protected_backups"`
}

// handleListStorageProviders returns all storage provider configurations
func (s *Server) handleListStorageProviders(c *gin.Context) {
	// Mock data - in production, fetch from config and storage providers
	providers := []StorageProviderConfig{
		{
			ID:                  "1",
			Name:                "AWS S3 Production",
			Type:                "S3",
			Enabled:             true,
			ImmutabilityEnabled: true,
			RetentionDays:       30,
			Mode:                "GOVERNANCE",
			LegalHold:           false,
			ProtectedBackups:    87,
		},
		{
			ID:                  "2",
			Name:                "Azure Blob Storage",
			Type:                "AZURE",
			Enabled:             true,
			ImmutabilityEnabled: true,
			RetentionDays:       90,
			Mode:                "LOCKED",
			LegalHold:           false,
			ProtectedBackups:    42,
		},
		{
			ID:                  "3",
			Name:                "Google Cloud Storage",
			Type:                "GCS",
			Enabled:             true,
			ImmutabilityEnabled: true,
			RetentionDays:       60,
			Mode:                "UNLOCKED",
			LegalHold:           false,
			ProtectedBackups:    27,
		},
	}

	s.respondSuccess(c, gin.H{
		"providers": providers,
		"total":     len(providers),
	})
}

// handleGetStorageProvider returns a specific storage provider configuration
func (s *Server) handleGetStorageProvider(c *gin.Context) {
	providerID := c.Param("id")

	// Mock implementation - in production, fetch from config
	provider := StorageProviderConfig{
		ID:                  providerID,
		Name:                "AWS S3 Production",
		Type:                "S3",
		Enabled:             true,
		ImmutabilityEnabled: true,
		RetentionDays:       30,
		Mode:                "GOVERNANCE",
		LegalHold:           false,
		ProtectedBackups:    87,
	}

	s.respondSuccess(c, provider)
}

// UpdateStorageProviderRequest represents a request to update storage provider config
type UpdateStorageProviderRequest struct {
	ImmutabilityEnabled bool   `json:"immutability_enabled"`
	RetentionDays       int    `json:"retention_days" binding:"min=1,max=3650"`
	Mode                string `json:"mode" binding:"required"`
	LegalHold           bool   `json:"legal_hold"`
}

// handleUpdateStorageProvider updates storage provider configuration
func (s *Server) handleUpdateStorageProvider(c *gin.Context) {
	providerID := c.Param("id")

	var req UpdateStorageProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Validate mode based on provider type
	// In production, would interact with actual storage provider APIs
	s.respondSuccessWithMessage(c, "Storage provider configuration updated successfully", gin.H{
		"provider_id":          providerID,
		"immutability_enabled": req.ImmutabilityEnabled,
		"retention_days":       req.RetentionDays,
		"mode":                 req.Mode,
		"legal_hold":           req.LegalHold,
	})
}
