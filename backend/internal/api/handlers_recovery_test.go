package api

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/storage"
	storageLocal "github.com/sanskarpan/db-backup/internal/storage/local"

	// Register the platform sqlite driver so the backup/restore engines can
	// dump and restore, and the go-sqlite3 SQL driver used to seed the source.
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
)

// newRecoveryTestServer seeds a real sqlite database, backs it up through the
// real backup engine into a real local storage provider, and returns a Server
// wired with matching backup + restore engines plus the created backup's ID.
func newRecoveryTestServer(t *testing.T) (server *Server, backupID string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	dbPath := filepath.Join(root, "source.db")
	seedRecoveryDB(t, dbPath)

	provider, err := storageLocal.NewLocalProvider(&storage.LocalConfig{Path: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}

	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:   filepath.Join(root, "backup-temp"),
		StorageProvider: provider,
	})
	meta, err := backupEngine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "recovery-api",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	s := &Server{
		config:        &Config{},
		backupEngine:  backupEngine,
		restoreEngine: restoreEngine,
		detector:      ransomware.NewDetector(nil),
	}
	return s, meta.ID
}

// seedRecoveryDB creates a users table with two rows at path.
func seedRecoveryDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (1,'alice'),(2,'bob');`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestHandleQueryBackup_ReadOnly(t *testing.T) {
	s, id := newRecoveryTestServer(t)
	r := gin.New()
	r.POST("/backups/:id/query", s.handleQueryBackup)

	// A read-only SELECT returns the seeded rows.
	w := doJSON(t, r, http.MethodPost, "/backups/"+id+"/query", map[string]interface{}{
		"query": "SELECT name FROM users ORDER BY id;",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "alice") || !strings.Contains(body, "bob") {
		t.Errorf("expected seeded rows in body: %s", body)
	}

	// A mutating statement is rejected so the mounted backup stays immutable.
	w = doJSON(t, r, http.MethodPost, "/backups/"+id+"/query", map[string]interface{}{
		"query": "DELETE FROM users;",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a write statement, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCleanRoomRecovery_Promotable(t *testing.T) {
	s, id := newRecoveryTestServer(t)
	r := gin.New()
	r.POST("/backups/:id/clean-room-recovery", s.handleCleanRoomRecovery)

	w := doJSON(t, r, http.MethodPost, "/backups/"+id+"/clean-room-recovery", map[string]interface{}{
		"expected_tables": []string{"users"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "PROMOTABLE") {
		t.Errorf("expected PROMOTABLE verdict in body: %s", w.Body.String())
	}
}

func TestHandleQueryBackup_NotFound(t *testing.T) {
	s, _ := newRecoveryTestServer(t)
	r := gin.New()
	r.POST("/backups/:id/query", s.handleQueryBackup)

	w := doJSON(t, r, http.MethodPost, "/backups/does-not-exist/query", map[string]interface{}{
		"query": "SELECT 1;",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
