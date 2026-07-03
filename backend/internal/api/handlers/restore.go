package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/repository"
	"github.com/sanskarpan/db-backup/internal/restore"
)

type RestoreHandler struct {
	engine *restore.Engine
	repo   repository.Repository
}

func NewRestoreHandler(engine *restore.Engine, repo repository.Repository) *RestoreHandler {
	return &RestoreHandler{
		engine: engine,
		repo:   repo,
	}
}

type RestoreBackupRequest struct {
	TargetHost     string   `json:"target_host"`
	TargetPort     int      `json:"target_port"`
	TargetUsername string   `json:"target_username"`
	TargetPassword string   `json:"target_password"`
	TargetDatabase string   `json:"target_database"`
	PointInTime    string   `json:"point_in_time"`
	Tables         []string `json:"tables"`
	ExcludeTables  []string `json:"exclude_tables"`
	Decrypt        bool     `json:"decrypt"`
	DecryptionKey  string   `json:"decryption_key"`
	SkipValidation bool     `json:"skip_validation"`
	Force          bool     `json:"force"`
}

// HandleRestoreBackup restores a backup.
func (h *RestoreHandler) HandleRestoreBackup(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get backup metadata
	metadata, err := h.repo.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}

	// Parse point-in-time if provided
	var pitTime *time.Time
	if req.PointInTime != "" {
		t, parseErr := time.Parse(time.RFC3339, req.PointInTime)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid point-in-time format"})
			return
		}
		pitTime = &t
	}

	// Create restore options
	opts := &restore.RestoreOptions{
		BackupID:       id,
		TargetHost:     req.TargetHost,
		TargetPort:     req.TargetPort,
		TargetUsername: req.TargetUsername,
		TargetPassword: req.TargetPassword,
		TargetDatabase: req.TargetDatabase,
		PointInTime:    pitTime,
		Tables:         req.Tables,
		ExcludeTables:  req.ExcludeTables,
		Decrypt:        req.Decrypt,
		DecryptionKey:  req.DecryptionKey,
		SkipValidation: req.SkipValidation,
		Force:          req.Force,
	}

	// Perform restore
	result, err := h.engine.RestoreBackup(ctx, metadata, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Restore failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
