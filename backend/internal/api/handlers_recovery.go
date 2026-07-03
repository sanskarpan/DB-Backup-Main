package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/backup/queryable"
	"github.com/sanskarpan/db-backup/internal/recovery/cleanroom"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// QueryBackupRequest runs a single read-only SQL query against a mounted
// backup without performing a full production restore.
type QueryBackupRequest struct {
	// Query is the read-only statement to run (must be SELECT/PRAGMA/WITH).
	Query string `json:"query" binding:"required"`
	// Args are optional positional arguments bound to the query placeholders.
	Args []any `json:"args,omitempty"`
}

// CleanRoomRecoveryRequest configures an isolated clean-room recovery of a
// stored backup.
type CleanRoomRecoveryRequest struct {
	// ExpectedTables, when non-empty, requires at least one of the named tables
	// to exist in the recovered database for the integrity check to pass.
	ExpectedTables []string `json:"expected_tables,omitempty"`
}

// handleQueryBackup handles POST /backups/:id/query. It mounts the backup
// read-only and runs the supplied read-only query against it, returning the
// resulting columns and rows. The backup artifact is never mutated.
func (s *Server) handleQueryBackup(c *gin.Context) {
	backupID := c.Param("id")

	var req QueryBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if s.restoreEngine == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("restore engine is not configured"), "Backup query is unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, msgBackupNotFound)
		return
	}

	mounter := queryable.NewMounter(s.restoreEngine)
	mount, err := mounter.Mount(ctx, metadata, &queryable.MountOptions{})
	if err != nil {
		s.respondError(c, http.StatusUnprocessableEntity, err, "Failed to mount backup")
		return
	}
	defer func() {
		// Best-effort unmount; the response already reflects the query outcome.
		if uerr := mount.Unmount(); uerr != nil {
			_ = uerr
		}
	}()

	result, err := mount.Query(ctx, req.Query, req.Args...)
	if err != nil {
		if errors.Is(err, queryable.ErrNotReadOnly) {
			s.respondError(c, http.StatusBadRequest, err, "Only read-only queries are permitted")
			return
		}
		s.respondError(c, http.StatusUnprocessableEntity, err, "Query failed")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Query executed against mounted backup",
		Data: gin.H{
			"backup_id": backupID,
			"columns":   result.Columns,
			"rows":      result.Rows,
			"row_count": result.RowCount,
		},
	})
}

// handleCleanRoomRecovery handles POST /backups/:id/clean-room-recovery. It
// restores the backup into an isolated environment, scans it for malware, and
// runs an integrity check, returning a verdict of whether the backup is
// promotable — without ever touching a production target.
func (s *Server) handleCleanRoomRecovery(c *gin.Context) {
	backupID := c.Param("id")

	var req CleanRoomRecoveryRequest
	// A missing/empty body is acceptable because every field is optional; only
	// reject a body that is present but malformed.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if s.restoreEngine == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("restore engine is not configured"), "Clean-room recovery is unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, msgBackupNotFound)
		return
	}

	scanner := s.detector
	if scanner == nil {
		scanner = ransomware.NewDetector(nil)
	}
	orch := cleanroom.NewOrchestrator(s.restoreEngine, scanner)
	report, recErr := orch.Recover(ctx, metadata, &cleanroom.Options{
		ExpectedTables: req.ExpectedTables,
		Cleanup:        true,
	})

	// A quarantined or failed verdict is a determinate, successful evaluation of
	// a suspect backup, not an API error; surface it in the payload.
	c.JSON(http.StatusOK, SuccessResponse{
		Success: report.Verdict == cleanroom.VerdictPromotable,
		Message: cleanRoomMessage(report.Verdict, recErr),
		Data: gin.H{
			"backup_id":    backupID,
			"verdict":      string(report.Verdict),
			"isolated":     report.Isolated,
			"threat_level": string(report.ThreatLevel),
			"steps":        cleanRoomSteps(report.Steps),
			"elapsed_ms":   report.Elapsed.Milliseconds(),
		},
	})
}

// cleanRoomStep is the API projection of a clean-room recovery step.
type cleanRoomStep struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// cleanRoomSteps projects orchestrator steps into their API representation.
func cleanRoomSteps(steps []cleanroom.Step) []cleanRoomStep {
	out := make([]cleanRoomStep, 0, len(steps))
	for _, st := range steps {
		out = append(out, cleanRoomStep{Name: st.Name, OK: st.OK, Detail: st.Detail})
	}
	return out
}

// cleanRoomMessage renders a human-readable summary for a recovery verdict.
func cleanRoomMessage(verdict cleanroom.Verdict, err error) string {
	switch verdict {
	case cleanroom.VerdictPromotable:
		return "Backup passed clean-room recovery and is promotable"
	case cleanroom.VerdictQuarantined:
		return "Backup quarantined: a threat was detected during clean-room recovery"
	default:
		if err != nil {
			return "Clean-room recovery failed: " + err.Error()
		}
		return "Clean-room recovery failed"
	}
}
