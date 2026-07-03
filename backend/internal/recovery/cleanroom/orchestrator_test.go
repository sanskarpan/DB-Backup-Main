package cleanroom_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/recovery/cleanroom"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	stor "github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"

	// Register the platform SQLite backup/restore driver.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	// Register the go-sqlite3 SQL driver for seeding and verifying databases.
	_ "github.com/mattn/go-sqlite3"
)

// seedSQLiteDB creates a small SQLite database file with one table.
func seedSQLiteDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO t (name) VALUES ('alice'), ('bob');`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func fileChecksum(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// realBackup creates a real SQLite backup stored in a local provider and returns
// its metadata plus the restore engine wired to the same provider.
func realBackup(t *testing.T) (*models.BackupMetadata, *restore.Engine, string) {
	t.Helper()
	root := t.TempDir()
	backupTemp := filepath.Join(root, "backup-temp")
	restoreTemp := filepath.Join(root, "restore-temp")
	storeDir := filepath.Join(root, "store")
	sourceDB := filepath.Join(root, "source.db")

	seedSQLiteDB(t, sourceDB)

	provider, err := local.NewLocalProvider(&stor.LocalConfig{Path: storeDir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}

	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:   backupTemp,
		StorageProvider: provider,
	})

	meta, err := backupEngine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     sourceDB,
		Name:         "cleanroom-src",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if meta.Status != database.BackupStatusSuccess {
		t.Fatalf("backup not successful: %s", meta.Status)
	}

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   restoreTemp,
		StorageProvider: provider,
	})
	return meta, restoreEngine, sourceDB
}

// TestRecover_CleanBackupIsPromotable proves a real, clean SQLite backup
// restored with a real restore.Engine and scanned with a real ransomware
// detector is reported Promotable, with the recovered database living inside the
// isolated working directory and never touching the source.
func TestRecover_CleanBackupIsPromotable(t *testing.T) {
	meta, restoreEngine, sourceDB := realBackup(t)
	sourceSum := fileChecksum(t, sourceDB)

	base := t.TempDir()
	orch := cleanroom.NewOrchestrator(restoreEngine, ransomware.NewDetector(nil))

	report, err := orch.Recover(context.Background(), meta, &cleanroom.Options{
		BaseDir:        base,
		ExpectedTables: []string{"t"},
	})
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if report.Verdict != cleanroom.VerdictPromotable {
		t.Fatalf("expected PROMOTABLE, got %s (steps: %+v)", report.Verdict, report.Steps)
	}
	if !report.Isolated {
		t.Errorf("expected Isolated to be true")
	}
	if report.ThreatLevel != ransomware.ThreatLevelNone {
		t.Errorf("expected no threat, got %s", report.ThreatLevel)
	}
	if report.Elapsed <= 0 {
		t.Errorf("expected positive elapsed time")
	}

	// The recovered database must live inside the isolated base dir.
	if !strings.HasPrefix(report.RecoveredPath, base) {
		t.Errorf("recovered path %q is not inside isolated base %q", report.RecoveredPath, base)
	}
	if report.RecoveredPath == sourceDB {
		t.Errorf("recovered path must not be the production source")
	}

	// The recovered database must be a real, queryable copy with the seeded rows.
	db, err := sql.Open("sqlite3", report.RecoveredPath)
	if err != nil {
		t.Fatalf("open recovered: %v", err)
	}
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM t").Scan(&count); err != nil {
		t.Fatalf("query recovered: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 rows in recovered db, got %d", count)
	}

	// The production source must be byte-for-byte unchanged.
	if got := fileChecksum(t, sourceDB); got != sourceSum {
		t.Errorf("production source was modified during clean-room recovery")
	}

	// Every step must be recorded and successful.
	wantSteps := []string{"isolate", "restore", "scan", "integrity"}
	if len(report.Steps) != len(wantSteps) {
		t.Fatalf("expected %d steps, got %d: %+v", len(wantSteps), len(report.Steps), report.Steps)
	}
	for i, name := range wantSteps {
		if report.Steps[i].Name != name || !report.Steps[i].OK {
			t.Errorf("step %d: expected ok %q, got %+v", i, name, report.Steps[i])
		}
	}
}

// TestRecover_CleanupRemovesIsolatedDir verifies the isolated directory is
// removed when cleanup is requested.
func TestRecover_CleanupRemovesIsolatedDir(t *testing.T) {
	meta, restoreEngine, _ := realBackup(t)
	base := t.TempDir()
	orch := cleanroom.NewOrchestrator(restoreEngine, ransomware.NewDetector(nil))

	report, err := orch.Recover(context.Background(), meta, &cleanroom.Options{
		BaseDir:        base,
		ExpectedTables: []string{"t"},
		Cleanup:        true,
	})
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if report.Verdict != cleanroom.VerdictPromotable {
		t.Fatalf("expected PROMOTABLE, got %s", report.Verdict)
	}
	if _, statErr := os.Stat(filepath.Dir(report.RecoveredPath)); !os.IsNotExist(statErr) {
		t.Errorf("expected isolated dir to be removed, stat err = %v", statErr)
	}
}

// highThreatScanner is a Scanner stub that always reports a HIGH threat, used to
// prove fail-closed quarantine behavior.
type highThreatScanner struct {
	calls int
}

func (s *highThreatScanner) ScanFile(_ context.Context, path string) (*ransomware.ThreatReport, error) {
	s.calls++
	return &ransomware.ThreatReport{
		ThreatLevel: ransomware.ThreatLevelHigh,
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		FilePath:    path,
		Description: "simulated high-severity detection",
	}, nil
}

// TestRecover_HighThreatQuarantined proves that when the scanner reports a threat
// at or above the threshold, the backup is quarantined (fail-closed) and the
// production target is never written.
func TestRecover_HighThreatQuarantined(t *testing.T) {
	meta, restoreEngine, sourceDB := realBackup(t)
	sourceSum := fileChecksum(t, sourceDB)

	base := t.TempDir()
	scanner := &highThreatScanner{}
	orch := cleanroom.NewOrchestrator(restoreEngine, scanner)

	// A production target path that must never be created by the orchestrator.
	productionTarget := filepath.Join(t.TempDir(), "production.db")

	report, err := orch.Recover(context.Background(), meta, &cleanroom.Options{
		BaseDir:        base,
		ExpectedTables: []string{"t"},
	})
	if err != nil {
		t.Fatalf("Recover returned error: %v", err)
	}
	if report.Verdict != cleanroom.VerdictQuarantined {
		t.Fatalf("expected QUARANTINED, got %s (steps: %+v)", report.Verdict, report.Steps)
	}
	if report.ThreatLevel != ransomware.ThreatLevelHigh {
		t.Errorf("expected HIGH threat level, got %s", report.ThreatLevel)
	}
	if scanner.calls != 1 {
		t.Errorf("expected scanner to run once, ran %d", scanner.calls)
	}

	// Integrity must not run once quarantined: only isolate, restore, scan.
	for _, s := range report.Steps {
		if s.Name == "integrity" {
			t.Errorf("integrity step must not run after quarantine")
		}
	}

	// The recovered artifact must stay inside the isolated dir, never the
	// production target, and the production target must not exist at all.
	if !strings.HasPrefix(report.RecoveredPath, base) {
		t.Errorf("recovered path %q escaped isolated base %q", report.RecoveredPath, base)
	}
	if _, statErr := os.Stat(productionTarget); !os.IsNotExist(statErr) {
		t.Errorf("production target must never be written, stat err = %v", statErr)
	}
	if got := fileChecksum(t, sourceDB); got != sourceSum {
		t.Errorf("production source was modified during quarantined recovery")
	}
}

// TestRecover_ScanErrorFailsClosed proves a scan error yields a Failed verdict
// and a non-nil error (fail-closed), never Promotable.
func TestRecover_ScanErrorFailsClosed(t *testing.T) {
	meta, restoreEngine, _ := realBackup(t)
	base := t.TempDir()
	orch := cleanroom.NewOrchestrator(restoreEngine, erroringScanner{})

	report, err := orch.Recover(context.Background(), meta, &cleanroom.Options{BaseDir: base})
	if err == nil {
		t.Fatal("expected fail-closed error on scan failure")
	}
	if report.Verdict != cleanroom.VerdictFailed {
		t.Errorf("expected FAILED verdict, got %s", report.Verdict)
	}
}

type erroringScanner struct{}

func (erroringScanner) ScanFile(context.Context, string) (*ransomware.ThreatReport, error) {
	return nil, os.ErrPermission
}

// TestRecover_RestoreFailureFails proves a restore failure yields a Failed
// verdict without scanning or integrity checking.
func TestRecover_RestoreFailureFails(t *testing.T) {
	base := t.TempDir()
	orch := cleanroom.NewOrchestrator(failingRestorer{}, ransomware.NewDetector(nil))

	report, err := orch.Recover(context.Background(), &models.BackupMetadata{ID: "missing"}, &cleanroom.Options{BaseDir: base})
	if err == nil {
		t.Fatal("expected error on restore failure")
	}
	if report.Verdict != cleanroom.VerdictFailed {
		t.Errorf("expected FAILED verdict, got %s", report.Verdict)
	}
	for _, s := range report.Steps {
		if s.Name == "scan" || s.Name == "integrity" {
			t.Errorf("scan/integrity must not run after restore failure, saw %q", s.Name)
		}
	}
}

type failingRestorer struct{}

func (failingRestorer) RestoreBackup(context.Context, *models.BackupMetadata, *restore.RestoreOptions) (*restore.RestoreResult, error) {
	return &restore.RestoreResult{Status: database.RestoreStatusFailed}, os.ErrNotExist
}
