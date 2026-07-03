package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// resolveRetentionUntil resolves an immutable-backup retention expiry from an
// optional RFC3339 timestamp and an optional relative day count. The explicit
// timestamp takes precedence; when both are empty the zero time is returned
// (no retention window specified).
func resolveRetentionUntil(retentionUntil string, retentionDays int) (time.Time, error) {
	if retentionUntil != "" {
		return time.Parse(time.RFC3339, retentionUntil)
	}
	if retentionDays > 0 {
		return time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour), nil
	}
	return time.Time{}, nil
}

// ImmutabilityResponse is the JSON body describing a backup's object-lock state.
type ImmutabilityResponse struct {
	BackupID       string     `json:"backup_id"`
	Immutable      bool       `json:"immutable"`
	RetentionUntil *time.Time `json:"retention_until,omitempty"`
	LockMode       string     `json:"lock_mode,omitempty"`
	LegalHold      bool       `json:"legal_hold"`
}

// handleGetBackupImmutability handles GET /backups/:id/immutability, reporting
// the current object-lock (WORM) protection recorded for the backup.
func (s *Server) handleGetBackupImmutability(c *gin.Context) {
	backupID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, msgBackupNotFound)
		return
	}

	until, mode, legalHold, err := s.backupEngine.GetBackupImmutability(ctx, metadata)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to read backup immutability")
		return
	}

	resp := ImmutabilityResponse{
		BackupID:  backupID,
		Immutable: metadata.Immutable,
		LockMode:  mode,
		LegalHold: legalHold,
	}
	if !until.IsZero() {
		resp.RetentionUntil = &until
	}

	s.respondSuccess(c, resp)
}

// LegalHoldRequest toggles a legal hold on a stored backup artifact.
type LegalHoldRequest struct {
	On bool `json:"on"`
}

// handleApplyLegalHold handles POST /backups/:id/legal-hold, turning an
// indefinite legal hold on or off for the backup's stored artifact.
func (s *Server) handleApplyLegalHold(c *gin.Context) {
	backupID := c.Param("id")

	var req LegalHoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, msgBackupNotFound)
		return
	}

	if err := s.backupEngine.ApplyLegalHold(ctx, metadata, req.On); err != nil {
		s.respondError(c, http.StatusUnprocessableEntity, err, "Failed to apply legal hold")
		return
	}

	s.respondSuccessWithMessage(c, "Legal hold updated", ImmutabilityResponse{
		BackupID:  backupID,
		Immutable: metadata.Immutable,
		LegalHold: metadata.LegalHold,
	})
}
