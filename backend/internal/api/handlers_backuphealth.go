package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/observability/backuphealth"
)

// defaultBackupHealthWindow is the protection-gap window used when the caller
// does not supply a "window" query parameter. A database with no successful
// backup within this window is flagged under-protected.
const defaultBackupHealthWindow = 24 * time.Hour

// handleBackupHealth serves GET /api/v1/stats/backup-health. It gathers the
// current schedules, backups and registered databases, then delegates to the
// pure backuphealth.Analyze to detect stale/missed backups and protection gaps.
//
// The optional "window" query parameter is a Go duration (e.g. "24h", "168h")
// controlling the under-protection window; it defaults to 24h.
func (s *Server) handleBackupHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	window, err := parseRecentWindow(c.Query("window"))
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid window parameter")
		return
	}

	backups, err := s.backupEngine.ListBackups(ctx)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list backups")
		return
	}

	var databases []*dbregistry.Database
	if s.dbStore != nil {
		databases, err = s.dbStore.List()
		if err != nil {
			s.respondError(c, http.StatusInternalServerError, err, "Failed to list databases")
			return
		}
	}

	jobs := s.scheduler.ListJobs()

	report := backuphealth.Analyze(jobs, backups, databases, time.Now(), window)
	s.respondSuccess(c, report)
}

// parseRecentWindow resolves the protection-gap window from a query value,
// falling back to the default when empty and rejecting invalid durations.
func parseRecentWindow(raw string) (window time.Duration, err error) {
	if raw == "" {
		return defaultBackupHealthWindow, nil
	}
	window, err = time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return window, nil
}
