package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/security/siem"
	"github.com/sanskarpan/db-backup/internal/storage"
	storageLocal "github.com/sanskarpan/db-backup/internal/storage/local"
	"github.com/sanskarpan/db-backup/internal/storage/replication"
	"github.com/sanskarpan/db-backup/internal/storageregistry"

	// Register the sqlite driver so a skipped-scan restore reaches driver creation.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
)

// writeBackupMeta persists a backup metadata JSON where backup.Engine expects it.
func writeBackupMeta(t *testing.T, tempDir string, m *models.BackupMetadata) {
	t.Helper()
	metaDir := filepath.Join(tempDir, "metadata")
	if err := os.MkdirAll(metaDir, 0o700); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, m.ID+".json"), data, 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

// fakeScanner is an ArtifactScanner test double reporting a fixed threat level.
type fakeScanner struct {
	level ransomware.ThreatLevel
}

func (f fakeScanner) ScanFile(_ context.Context, path string) (*ransomware.ThreatReport, error) {
	return &ransomware.ThreatReport{
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		ThreatLevel: f.level,
		FilePath:    path,
		Description: "synthetic detection for test",
	}, nil
}

func TestHandleGetBackupImmutability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := t.TempDir()
	until := time.Now().Add(48 * time.Hour).UTC()
	writeBackupMeta(t, tempDir, &models.BackupMetadata{
		ID:             "bkp-1",
		Immutable:      true,
		ImmutableUntil: &until,
		LockMode:       storage.LockModeCompliance,
	})

	s := &Server{
		config:       &Config{},
		backupEngine: backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
	}
	r := gin.New()
	r.GET("/backups/:id/immutability", s.handleGetBackupImmutability)

	w := doJSON(t, r, http.MethodGet, "/backups/bkp-1/immutability", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data ImmutabilityResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Data.Immutable || resp.Data.LockMode != storage.LockModeCompliance {
		t.Fatalf("unexpected immutability: %+v", resp.Data)
	}
	if resp.Data.RetentionUntil == nil {
		t.Fatal("expected retention_until to be set")
	}
}

func TestHandleApplyLegalHold_NoObjectLockProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir := t.TempDir()
	writeBackupMeta(t, tempDir, &models.BackupMetadata{ID: "bkp-2"})

	s := &Server{
		config:       &Config{},
		backupEngine: backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
	}
	r := gin.New()
	r.POST("/backups/:id/legal-hold", s.handleApplyLegalHold)

	// With no object-lock-capable provider, enabling a hold must fail honestly.
	w := doJSON(t, r, http.MethodPost, "/backups/bkp-2/legal-hold", map[string]interface{}{"on": true})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleReplicateBackup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	// Source provider holding the backup artifact.
	srcDir := t.TempDir()
	src, err := storageLocal.NewLocalProvider(&storage.LocalConfig{Path: srcDir})
	if err != nil {
		t.Fatalf("source provider: %v", err)
	}
	const remotePath = "backups/test.gz"
	if uerr := src.UploadStream(ctx, strings.NewReader("backup-bytes"), remotePath, nil); uerr != nil {
		t.Fatalf("seed source object: %v", uerr)
	}

	// Registered destination provider.
	store, err := storageregistry.NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	destDir := t.TempDir()
	dest, err := store.Create(&storageregistry.CreateRequest{
		Name: "dest-local", Type: "local", Enabled: true,
		Config: map[string]interface{}{"path": destDir},
	})
	if err != nil {
		t.Fatalf("create dest: %v", err)
	}

	tempDir := t.TempDir()
	writeBackupMeta(t, tempDir, &models.BackupMetadata{
		ID:              "bkp-3",
		StorageLocation: "local://" + remotePath,
	})

	s := newReplicationTestServer(store, src, tempDir)
	r := gin.New()
	r.POST("/backups/:id/replicate", s.handleReplicateBackup)

	w := doJSON(t, r, http.MethodPost, "/backups/bkp-3/replicate", map[string]interface{}{
		"target_provider_ids": []string{dest.ID},
		"verify":              true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Destinations []ReplicationDestResult `json:"destinations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || len(resp.Data.Destinations) != 1 {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
	if resp.Data.Destinations[0].Status != string(replication.StatusCopied) {
		t.Fatalf("expected copied status, got %q", resp.Data.Destinations[0].Status)
	}

	// The object must actually exist at the destination.
	dp, _ := storageLocal.NewLocalProvider(&storage.LocalConfig{Path: destDir})
	exists, _ := dp.Exists(ctx, remotePath)
	if !exists {
		t.Fatal("expected replicated object at destination")
	}
}

func newReplicationTestServer(store *storageregistry.Store, src storage.Provider, tempDir string) *Server {
	return &Server{
		config:          &Config{},
		storageStore:    store,
		storageProvider: src,
		backupEngine:    backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
		// replicator is normally set in NewServer; construct it directly here.
		replicator: replication.NewReplicator(),
	}
}

func TestHandleSIEMTest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var received int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		received++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := &Server{
		config:       &Config{},
		siemExporter: siem.NewExporter(siem.Config{Endpoint: ts.URL, Format: siem.FormatWebhook}),
	}
	r := gin.New()
	r.POST("/security/siem/test", s.handleSIEMTest)

	w := doJSON(t, r, http.MethodPost, "/security/siem/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if received != 1 {
		t.Fatalf("expected exactly one SIEM POST, got %d", received)
	}
}

func TestHandleSIEMTest_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{config: &Config{}} // no exporter configured
	r := gin.New()
	r.POST("/security/siem/test", s.handleSIEMTest)

	w := doJSON(t, r, http.MethodPost, "/security/siem/test", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRestoreBackup_MalwareAbortAndSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Local backup artifact (no remote provider needed).
	artifact := filepath.Join(t.TempDir(), "backup.sql")
	if err := os.WriteFile(artifact, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	tempDir := t.TempDir()
	writeBackupMeta(t, tempDir, &models.BackupMetadata{
		ID:           "bkp-4",
		DatabaseType: "sqlite",
		BackupPath:   artifact,
	})

	s := newRestoreTestServer(t, tempDir)
	r := gin.New()
	r.POST("/backups/:id/restore", s.handleRestoreBackup)

	// Selective restore (tables) without skip: the fake scanner reports HIGH,
	// so the restore must abort with 409 and surface the threat report.
	w := doJSON(t, r, http.MethodPost, "/backups/bkp-4/restore", map[string]interface{}{
		"tables": []string{"users"},
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 malware abort, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "threat_report") {
		t.Fatalf("expected threat_report in body: %s", w.Body.String())
	}

	// With skip_malware_scan the scan is bypassed, so the restore proceeds past
	// the scan (and fails later on the target DB) — crucially it is NOT a 409.
	w = doJSON(t, r, http.MethodPost, "/backups/bkp-4/restore", map[string]interface{}{
		"tables":            []string{"users"},
		"skip_malware_scan": true,
	})
	if w.Code == http.StatusConflict {
		t.Fatalf("skip_malware_scan should bypass the scan, but got 409: %s", w.Body.String())
	}
}

func newRestoreTestServer(t *testing.T, tempDir string) *Server {
	t.Helper()
	eng := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(tempDir, "restore-tmp"),
		ValidateFirst:   false,
		ArtifactScanner: fakeScanner{level: ransomware.ThreatLevelHigh},
	})
	return &Server{
		config:        &Config{},
		backupEngine:  backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
		restoreEngine: eng,
	}
}
