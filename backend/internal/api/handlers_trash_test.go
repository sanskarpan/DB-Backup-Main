package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/models"
)

func newTrashServer(t *testing.T) (srv *Server, dir string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tempDir := filepath.Join(t.TempDir(), "backups")
	return &Server{
		backupEngine: backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
		config:       &Config{},
		logger:       logger.New(logger.Config{Level: "error", Format: "json"}),
	}, tempDir
}

// registerBackupRoutes mirrors the /backups group from SetupRoutes with a
// passthrough (no CSRF/auth) so the recycle-bin flow can be driven directly.
func registerBackupRoutes(s *Server) *gin.Engine {
	r := gin.New()
	grp := r.Group("/api/v1/backups")
	grp.GET("", s.handleListBackups)
	grp.GET("/trash", s.handleListDeletedBackups)
	grp.POST("/trash/purge-expired", s.handlePurgeExpired)
	grp.GET("/:id", s.handleGetBackup)
	grp.DELETE("/:id", s.handleDeleteBackup)
	grp.POST("/:id/restore-from-trash", s.handleRestoreDeletedBackup)
	grp.DELETE("/:id/purge", s.handlePurgeBackup)
	return r
}

// TestSetupRoutes_TrashRoutesRegisterWithoutPanic proves the full route table —
// including the static /backups/trash segment alongside the /backups/:id param —
// registers without gin panicking on a route conflict.
func TestSetupRoutes_TrashRoutesRegisterWithoutPanic(t *testing.T) {
	s, _ := newTrashServer(t)

	r := gin.New()
	s.SetupRoutes(r) // panics on route conflict; test fails if it does.

	want := map[string]bool{
		"GET /api/v1/backups/trash":                   false,
		"POST /api/v1/backups/trash/purge-expired":    false,
		"POST /api/v1/backups/:id/restore-from-trash": false,
		"DELETE /api/v1/backups/:id/purge":            false,
	}
	for _, rt := range r.Routes() {
		key := rt.Method + " " + rt.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("expected route not registered: %s", key)
		}
	}
}

func seedBackupMeta(t *testing.T, tempDir string, meta *models.BackupMetadata) {
	t.Helper()
	metaDir := filepath.Join(tempDir, "metadata")
	if err := os.MkdirAll(metaDir, 0o700); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("marshal meta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, meta.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("write meta: %v", err)
	}
}

func decodeBackupIDs(t *testing.T, body []byte) map[string]bool {
	t.Helper()
	var resp struct {
		Data struct {
			Backups []struct {
				ID string `json:"id"`
			} `json:"backups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	set := make(map[string]bool)
	for _, b := range resp.Data.Backups {
		set[b.ID] = true
	}
	return set
}

// TestTrashHandlers_Flow drives soft-delete -> restore -> soft-delete -> purge
// through the HTTP handlers.
func TestTrashHandlers_Flow(t *testing.T) {
	s, tempDir := newTrashServer(t)
	r := registerBackupRoutes(s)

	artifact := filepath.Join(tempDir, "b1.sql")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("dump"), 0o600); err != nil {
		t.Fatal(err)
	}
	seedBackupMeta(t, tempDir, &models.BackupMetadata{
		ID: "b1", Name: "b1", BackupPath: artifact, StorageLocation: artifact,
	})

	// Soft delete: DELETE /backups/:id moves the backup to the recycle bin.
	if w := doJSON(t, r, http.MethodDelete, "/api/v1/backups/b1", nil); w.Code != http.StatusOK {
		t.Fatalf("soft delete: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Hidden from the live list, present in the trash list.
	w := doJSON(t, r, http.MethodGet, "/api/v1/backups", nil)
	if decodeBackupIDs(t, w.Body.Bytes())["b1"] {
		t.Fatalf("b1 should be hidden from live list: %s", w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/api/v1/backups/trash", nil)
	if !decodeBackupIDs(t, w.Body.Bytes())["b1"] {
		t.Fatalf("b1 should appear in trash: %s", w.Body.String())
	}

	// Restore from trash brings it back to the live list.
	if w = doJSON(t, r, http.MethodPost, "/api/v1/backups/b1/restore-from-trash", nil); w.Code != http.StatusOK {
		t.Fatalf("restore-from-trash: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodGet, "/api/v1/backups", nil)
	if !decodeBackupIDs(t, w.Body.Bytes())["b1"] {
		t.Fatalf("b1 should be live after restore: %s", w.Body.String())
	}

	// Purging a live backup is rejected.
	if w = doJSON(t, r, http.MethodDelete, "/api/v1/backups/b1/purge", nil); w.Code == http.StatusOK {
		t.Fatalf("purge of live backup should fail, got 200")
	}

	// Soft delete again, then purge permanently.
	if w = doJSON(t, r, http.MethodDelete, "/api/v1/backups/b1", nil); w.Code != http.StatusOK {
		t.Fatalf("second soft delete: got %d", w.Code)
	}
	if w = doJSON(t, r, http.MethodDelete, "/api/v1/backups/b1/purge", nil); w.Code != http.StatusOK {
		t.Fatalf("purge: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Errorf("artifact should be gone after purge: %v", err)
	}
	w = doJSON(t, r, http.MethodGet, "/api/v1/backups/trash", nil)
	if len(decodeBackupIDs(t, w.Body.Bytes())) != 0 {
		t.Fatalf("trash should be empty after purge: %s", w.Body.String())
	}
}
