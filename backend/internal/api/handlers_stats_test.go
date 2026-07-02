package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/scheduler"
	"github.com/sanskarpan/db-backup/internal/storageregistry"
)

func newStatsTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	log := logger.New(logger.Config{Level: "error", Format: "json"})

	// A backup engine rooted at an empty temp dir reports zero backups (a real,
	// honest count), which is enough to exercise the aggregation logic.
	engine := backup.NewEngine(&backup.Config{
		TempDirectory: filepath.Join(t.TempDir(), "backups"),
	})

	sched, err := scheduler.NewScheduler(engine, log, nil)
	if err != nil {
		t.Fatalf("NewScheduler: %v", err)
	}

	dbStore, err := dbregistry.NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("dbregistry NewStore: %v", err)
	}
	if _, err = dbStore.Create(&dbregistry.CreateRequest{
		Name: "pg", Type: "postgres", Host: "localhost", Port: 5432, Username: "admin",
	}); err != nil {
		t.Fatalf("seed database: %v", err)
	}

	storageStore, err := storageregistry.NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("storageregistry NewStore: %v", err)
	}
	if _, err := storageStore.Create(&storageregistry.CreateRequest{
		Name: "local", Type: "local", Config: map[string]interface{}{"path": t.TempDir()},
	}); err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	s := &Server{
		backupEngine: engine,
		scheduler:    sched,
		dbStore:      dbStore,
		storageStore: storageStore,
		config:       &Config{},
		logger:       log,
	}

	r := gin.New()
	r.GET("/api/v1/stats", s.handleGetStats)
	r.GET("/api/v1/stats/storage", s.handleGetStorageStats)
	return r
}

func TestHandleGetStats_RealValues(t *testing.T) {
	r := newStatsTestServer(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("stats: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"total_backups", "successful_backups", "failed_backups", "total_size", "databases", "schedules"} {
		if _, ok := data[key]; !ok {
			t.Fatalf("stats missing key %q: %s", key, w.Body.String())
		}
	}
	if data["total_backups"] != float64(0) {
		t.Fatalf("expected 0 backups, got %v", data["total_backups"])
	}
	if data["databases"] != float64(1) {
		t.Fatalf("expected 1 database, got %v", data["databases"])
	}
	if data["schedules"] != float64(0) {
		t.Fatalf("expected 0 schedules, got %v", data["schedules"])
	}
}

func TestHandleGetStorageStats_RealValues(t *testing.T) {
	r := newStatsTestServer(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/stats/storage", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("storage stats: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"total_backups", "total_size", "total_compressed_size", "configured_providers", "by_location"} {
		if _, ok := data[key]; !ok {
			t.Fatalf("storage stats missing key %q: %s", key, w.Body.String())
		}
	}
	if data["configured_providers"] != float64(1) {
		t.Fatalf("expected 1 configured provider, got %v", data["configured_providers"])
	}
}
