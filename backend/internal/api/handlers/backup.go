package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/repository"
)

type BackupHandler struct {
	engine *backup.Engine
	repo   repository.Repository
}

func NewBackupHandler(engine *backup.Engine, repo repository.Repository) *BackupHandler {
	return &BackupHandler{
		engine: engine,
		repo:   repo,
	}
}

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

// HandleCreateBackup creates a new backup
func (h *BackupHandler) HandleCreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported database type"})
		return
	}

	// Parse compression
	compression := database.CompressionNone
	switch req.Compression {
	case "gzip":
		compression = database.CompressionGzip
	case "zstd":
		compression = database.CompressionZstd
	case "lz4":
		compression = database.CompressionLZ4
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
		Compression:   compression,
		Encrypt:       req.Encrypt,
		EncryptionKey: req.EncryptionKey,
		Tags:          req.Tags,
	}

	// Create backup
	ctx := c.Request.Context()
	metadata, err := h.engine.CreateBackup(ctx, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Backup failed", "details": err.Error()})
		return
	}

	// Save metadata
	if err := h.repo.Save(ctx, metadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save metadata", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metadata)
}

// HandleListBackups lists all backups
// validateQueryParams validates and sanitizes query parameters
func validateQueryParams(c *gin.Context) error {
	// Validate sort_by parameter
	sortBy := c.DefaultQuery("sort_by", "date")
	validSortFields := map[string]bool{
		"date": true, "name": true, "size": true, "status": true,
	}
	if !validSortFields[sortBy] {
		return fmt.Errorf("invalid sort_by parameter: %s", sortBy)
	}
	
	// Validate sort_order parameter
	sortOrder := c.DefaultQuery("sort_order", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		return fmt.Errorf("invalid sort_order parameter: %s", sortOrder)
	}
	
	// Validate limit parameter
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		return fmt.Errorf("invalid limit parameter: must be between 1 and 1000")
	}
	
	return nil
}

func (h *BackupHandler) HandleListBackups(c *gin.Context) {
	ctx := c.Request.Context()

	// Build filter from query params
	filter := &repository.ListFilter{
		Database:     c.Query("database"),
		DatabaseType: c.Query("type"),
		SortBy:       c.DefaultQuery("sort_by", "date"),
		SortOrder:    c.DefaultQuery("sort_order", "desc"),
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
			filter.Limit = limit
		}
	}

	backups, err := h.repo.List(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"total":   len(backups),
	})
}

// HandleGetBackup retrieves a single backup by ID
func (h *BackupHandler) HandleGetBackup(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	metadata, err := h.repo.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// HandleDeleteBackup deletes a backup
func (h *BackupHandler) HandleDeleteBackup(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	// Get metadata first to find backup file
	metadata, err := h.repo.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}

	// Delete backup file if exists
	if metadata.BackupPath != "" {
		os.Remove(metadata.BackupPath)
	}

	// Delete metadata
	if err := h.repo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Backup deleted successfully"})
}

// HandleDownloadBackup downloads a backup file
func (h *BackupHandler) HandleDownloadBackup(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	metadata, err := h.repo.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}

	// Check if backup file exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup file not found"})
		return
	}

	c.File(metadata.BackupPath)
}
// Bulk Operations

type BulkOperationRequest struct {
	BackupIDs []string `json:"backup_ids" binding:"required,min=1"`
	Operation string   `json:"operation" binding:"required"`
}

type BulkOperationResult struct {
	BackupID string `json:"backup_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type BulkOperationResponse struct {
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Results   []BulkOperationResult `json:"results"`
}

// HandleBulkDelete deletes multiple backups
func (h *BackupHandler) HandleBulkDelete(c *gin.Context) {
	var req BulkOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	response := &BulkOperationResponse{
		Total:   len(req.BackupIDs),
		Results: make([]BulkOperationResult, 0, len(req.BackupIDs)),
	}

	for _, backupID := range req.BackupIDs {
		result := BulkOperationResult{BackupID: backupID}

		// Get backup metadata
		metadata, err := h.repo.Get(c.Request.Context(), backupID)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get backup: %v", err)
			response.Failed++
		} else {
			// Delete from storage
			if err := h.repo.Delete(c.Request.Context(), backupID); err != nil {
				result.Error = fmt.Sprintf("Failed to delete backup: %v", err)
				response.Failed++
			} else {
				// Delete physical file
				if metadata.BackupPath != "" {
					os.Remove(metadata.BackupPath)
				}
				result.Success = true
				response.Succeeded++
			}
		}

		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, response)
}

// HandleBulkRetry retries failed backups
func (h *BackupHandler) HandleBulkRetry(c *gin.Context) {
	var req BulkOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	response := &BulkOperationResponse{
		Total:   len(req.BackupIDs),
		Results: make([]BulkOperationResult, 0, len(req.BackupIDs)),
	}

	for _, backupID := range req.BackupIDs {
		result := BulkOperationResult{BackupID: backupID}

		// Get original backup metadata
		metadata, err := h.repo.Get(c.Request.Context(), backupID)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get backup: %v", err)
			response.Failed++
		} else {
			// Only retry failed backups
			if metadata.Status != "failed" {
				result.Error = "Backup is not in failed state"
				response.Failed++
			} else {
				// Create backup options from metadata
				opts := &backup.CreateOptions{
					DatabaseType: metadata.DatabaseType,
					Database:     metadata.Database,
					Tags:         metadata.Tags,
				}

				// Start new backup with same config
				newMetadata, err := h.engine.CreateBackup(c.Request.Context(), opts)
				if err != nil {
					result.Error = fmt.Sprintf("Failed to retry backup: %v", err)
					response.Failed++
				} else {
					result.Success = true
					result.BackupID = newMetadata.ID
					response.Succeeded++
				}
			}
		}

		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, response)
}

// HandleBulkExport exports multiple backups metadata
func (h *BackupHandler) HandleBulkExport(c *gin.Context) {
	var req BulkOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	response := &BulkOperationResponse{
		Total:   len(req.BackupIDs),
		Results: make([]BulkOperationResult, 0, len(req.BackupIDs)),
	}

	exportData := make([]map[string]interface{}, 0)

	for _, backupID := range req.BackupIDs {
		result := BulkOperationResult{BackupID: backupID}

		metadata, err := h.repo.Get(c.Request.Context(), backupID)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get backup: %v", err)
			response.Failed++
		} else {
			exportData = append(exportData, map[string]interface{}{
				"id":            metadata.ID,
				"database_type": metadata.DatabaseType,
				"database":      metadata.Database,
				"status":        metadata.Status,
				"size":          metadata.Size,
				"compression":   metadata.Compression,
				"encrypted":     metadata.Encrypted,
				"start_time":    metadata.StartTime,
				"end_time":      metadata.EndTime,
				"tags":          metadata.Tags,
			})
			result.Success = true
			response.Succeeded++
		}

		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": response,
		"data":    exportData,
	})
}

// HandleBulkUpdateTags updates tags for multiple backups
func (h *BackupHandler) HandleBulkUpdateTags(c *gin.Context) {
	type BulkUpdateTagsRequest struct {
		BackupIDs []string          `json:"backup_ids" binding:"required,min=1"`
		Tags      map[string]string `json:"tags" binding:"required"`
	}

	var req BulkUpdateTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	response := &BulkOperationResponse{
		Total:   len(req.BackupIDs),
		Results: make([]BulkOperationResult, 0, len(req.BackupIDs)),
	}

	for _, backupID := range req.BackupIDs {
		result := BulkOperationResult{BackupID: backupID}

		metadata, err := h.repo.Get(c.Request.Context(), backupID)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to get backup: %v", err)
			response.Failed++
		} else {
			// Update tags
			if metadata.Tags == nil {
				metadata.Tags = make(map[string]string)
			}
			for k, v := range req.Tags {
				metadata.Tags[k] = v
			}

			if err := h.repo.Update(c.Request.Context(), metadata); err != nil {
				result.Error = fmt.Sprintf("Failed to update tags: %v", err)
				response.Failed++
			} else {
				result.Success = true
				response.Succeeded++
			}
		}

		response.Results = append(response.Results, result)
	}

	c.JSON(http.StatusOK, response)
}
