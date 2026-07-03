package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/security/siem"
	"github.com/sanskarpan/db-backup/internal/storage/replication"
)

// TestE2E_BackupRestoreRoundTrip proves the core lifecycle end to end using the
// real SQLite driver and the real local filesystem storage provider: a seeded
// database is backed up, the artifact + metadata land in storage with a
// checksum, and a restore into a fresh target reproduces the original rows.
func TestE2E_BackupRestoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "source.db")
	storeDir := filepath.Join(root, "store")
	backupTemp := filepath.Join(root, "backup-temp")
	restoreTemp := filepath.Join(root, "restore-temp")
	targetDB := filepath.Join(root, "restored.db")

	want := seedUsersDB(t, dbPath)
	provider := newLocalStorage(t, storeDir)

	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:   backupTemp,
		StorageProvider: provider,
	})

	metadata, err := backupEngine.CreateBackup(ctx, &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "roundtrip",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if metadata.Status != database.BackupStatusSuccess {
		t.Fatalf("expected success status, got %s", metadata.Status)
	}
	if metadata.Checksum == "" {
		t.Errorf("expected a non-empty artifact checksum")
	}
	if !strings.HasPrefix(metadata.StorageLocation, "local://backups/") {
		t.Errorf("unexpected storage location %q", metadata.StorageLocation)
	}

	remotePath := strings.TrimPrefix(metadata.StorageLocation, "local://")
	exists, err := provider.Exists(ctx, remotePath)
	if err != nil {
		t.Fatalf("Exists artifact: %v", err)
	}
	if !exists {
		t.Fatalf("backup artifact missing from storage at %q", remotePath)
	}
	metaExists, err := provider.Exists(ctx, filepath.Join(filepath.Dir(remotePath), "metadata.json"))
	if err != nil {
		t.Fatalf("Exists metadata: %v", err)
	}
	if !metaExists {
		t.Errorf("metadata.json missing from storage")
	}

	// The stored artifact's bytes must match the recorded checksum.
	if got := sha256Hex(readObject(t, provider, remotePath)); got != metadata.Checksum {
		t.Errorf("stored artifact checksum mismatch: got %s want %s", got, metadata.Checksum)
	}

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   restoreTemp,
		StorageProvider: provider,
	})
	result, err := restoreEngine.RestoreBackup(ctx, metadata, &restore.RestoreOptions{
		TargetDatabase: targetDB,
	})
	if err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	if result.Status != database.RestoreStatusSuccess {
		t.Fatalf("expected restore success, got %s (%v)", result.Status, result.Error)
	}

	got := queryColumn(t, targetDB, "SELECT name FROM users ORDER BY id;")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("restored rows mismatch: got %v want %v", got, want)
	}
}

// TestE2E_StorageToStorageReplication proves an artifact can be streamed from
// one real local provider to a second one with byte-for-byte fidelity, using
// the production Replicator and its post-copy verification.
func TestE2E_StorageToStorageReplication(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "source.db")
	srcDir := filepath.Join(root, "src-store")
	dstDir := filepath.Join(root, "dst-store")

	seedUsersDB(t, dbPath)
	src := newLocalStorage(t, srcDir)
	dst := newLocalStorage(t, dstDir)

	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:   filepath.Join(root, "backup-temp"),
		StorageProvider: src,
	})
	metadata, err := backupEngine.CreateBackup(ctx, &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "replication",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	remotePath := strings.TrimPrefix(metadata.StorageLocation, "local://")
	srcBytes := readObject(t, src, remotePath)

	replicator := replication.NewReplicator()
	res, err := replicator.Replicate(ctx, src, dst, remotePath, replication.Options{
		VerifyAfterCopy: true,
	})
	if err != nil {
		t.Fatalf("Replicate: %v", err)
	}
	if res.Status != replication.StatusCopied {
		t.Fatalf("expected StatusCopied, got %s (%v)", res.Status, res.Err)
	}

	dstExists, err := dst.Exists(ctx, remotePath)
	if err != nil {
		t.Fatalf("Exists at destination: %v", err)
	}
	if !dstExists {
		t.Fatalf("replicated object missing at destination %q", remotePath)
	}

	dstBytes := readObject(t, dst, remotePath)
	if !bytes.Equal(srcBytes, dstBytes) {
		t.Errorf("replicated bytes differ from source")
	}
	wantSum := sha256Hex(srcBytes)
	if res.Checksum != wantSum {
		t.Errorf("replicator checksum %s != source checksum %s", res.Checksum, wantSum)
	}
	if sha256Hex(dstBytes) != wantSum {
		t.Errorf("destination checksum mismatch")
	}
	if res.BytesCopied != int64(len(srcBytes)) {
		t.Errorf("bytes copied %d != source size %d", res.BytesCopied, len(srcBytes))
	}
}

// TestE2E_ImmutableBackup proves WORM/object-lock behavior. Against a provider
// that supports object lock the engine drives SetRetention/SetLegalHold and
// records the protection on the metadata; against a plain provider an immutable
// backup fails closed.
func TestE2E_ImmutableBackup(t *testing.T) {
	t.Run("object_lock_applied", testImmutableObjectLockApplied)
	t.Run("plain_provider_fails_closed", testImmutablePlainProviderFailsClosed)
}

func testImmutableObjectLockApplied(t *testing.T) {
	ctx := context.Background()

	{
		root := t.TempDir()
		dbPath := filepath.Join(root, "source.db")
		seedUsersDB(t, dbPath)

		provider := newImmutableFakeProvider()
		engine := backup.NewEngine(&backup.Config{
			TempDirectory:   filepath.Join(root, "backup-temp"),
			StorageProvider: provider,
		})

		retentionUntil := time.Now().Add(48 * time.Hour).UTC()
		metadata, err := engine.CreateBackup(ctx, &backup.CreateOptions{
			DatabaseType:   database.DatabaseTypeSQLite,
			Database:       dbPath,
			Name:           "immutable",
			Immutable:      true,
			RetentionUntil: retentionUntil,
			LockMode:       "COMPLIANCE",
		})
		if err != nil {
			t.Fatalf("CreateBackup(immutable): %v", err)
		}

		remotePath := strings.TrimPrefix(metadata.StorageLocation, "s3://")
		if provider.setRetentionCalls != 1 {
			t.Fatalf("expected exactly one SetRetention call, got %d", provider.setRetentionCalls)
		}
		rec, ok := provider.retentions[remotePath]
		if !ok {
			t.Fatalf("no retention recorded for %q", remotePath)
		}
		if rec.mode != "COMPLIANCE" {
			t.Errorf("expected COMPLIANCE mode, got %q", rec.mode)
		}
		if !rec.until.Equal(retentionUntil) {
			t.Errorf("retention until %v != requested %v", rec.until, retentionUntil)
		}

		if !metadata.Immutable {
			t.Errorf("metadata.Immutable should be true")
		}
		if metadata.ImmutableUntil == nil || !metadata.ImmutableUntil.Equal(retentionUntil) {
			t.Errorf("metadata.ImmutableUntil = %v, want %v", metadata.ImmutableUntil, retentionUntil)
		}
		if metadata.LockMode != "COMPLIANCE" {
			t.Errorf("metadata.LockMode = %q, want COMPLIANCE", metadata.LockMode)
		}

		// ApplyLegalHold must flip the hold on the provider and the metadata.
		if err = engine.ApplyLegalHold(ctx, metadata, true); err != nil {
			t.Fatalf("ApplyLegalHold: %v", err)
		}
		if provider.setLegalHoldCalls != 1 || !provider.legalHolds[remotePath] {
			t.Errorf("expected legal hold applied on provider")
		}
		if !metadata.LegalHold {
			t.Errorf("metadata.LegalHold should be true after ApplyLegalHold")
		}

		// GetBackupImmutability must report the provider's authoritative state.
		until, mode, legalHold, err := engine.GetBackupImmutability(ctx, metadata)
		if err != nil {
			t.Fatalf("GetBackupImmutability: %v", err)
		}
		if !until.Equal(retentionUntil) {
			t.Errorf("reported retention %v != %v", until, retentionUntil)
		}
		if mode != "COMPLIANCE" {
			t.Errorf("reported mode %q, want COMPLIANCE", mode)
		}
		if !legalHold {
			t.Errorf("expected legal hold reported on")
		}
	}
}

func testImmutablePlainProviderFailsClosed(t *testing.T) {
	ctx := context.Background()

	{
		root := t.TempDir()
		dbPath := filepath.Join(root, "source.db")
		seedUsersDB(t, dbPath)

		// A real local provider does NOT implement storage.ImmutableProvider.
		provider := newLocalStorage(t, filepath.Join(root, "store"))
		engine := backup.NewEngine(&backup.Config{
			TempDirectory:   filepath.Join(root, "backup-temp"),
			StorageProvider: provider,
		})

		metadata, err := engine.CreateBackup(ctx, &backup.CreateOptions{
			DatabaseType:   database.DatabaseTypeSQLite,
			Database:       dbPath,
			Name:           "immutable-unsupported",
			Immutable:      true,
			RetentionUntil: time.Now().Add(time.Hour),
		})
		if err == nil {
			t.Fatalf("expected immutable backup to fail on non-immutable provider")
		}
		if metadata != nil && metadata.Status == database.BackupStatusSuccess {
			t.Errorf("backup must not be reported successful when object lock is unavailable")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "object lock") {
			t.Errorf("expected an object-lock error, got %v", err)
		}
	}
}

// TestE2E_PreRestoreMalwareScan proves the pre-restore scan fails closed: a
// high-severity detection aborts the restore with restore.ErrMalwareDetected and
// never touches the target, while a clean scan (or a skip) lets the restore
// proceed and still reproduces the data.
func TestE2E_PreRestoreMalwareScan(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "source.db")
	storeDir := filepath.Join(root, "store")

	want := seedUsersDB(t, dbPath)
	provider := newLocalStorage(t, storeDir)

	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:   filepath.Join(root, "backup-temp"),
		StorageProvider: provider,
	})
	metadata, err := backupEngine.CreateBackup(ctx, &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "scan",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	t.Run("malware_detected_aborts", func(t *testing.T) {
		targetDB := filepath.Join(t.TempDir(), "restored.db")
		scanner := &fakeScanner{report: &ransomware.ThreatReport{
			ThreatLevel: ransomware.ThreatLevelCritical,
			ThreatType:  ransomware.ThreatTypeSignatureMatch,
			Description: "known ransomware signature",
		}}
		engine := restore.NewEngine(&restore.Config{
			TempDirectory:   filepath.Join(t.TempDir(), "restore-temp"),
			StorageProvider: provider,
			ArtifactScanner: scanner,
		})

		result, err := engine.RestoreBackup(ctx, metadata, &restore.RestoreOptions{
			TargetDatabase: targetDB,
		})
		if err == nil {
			t.Fatal("expected restore to abort on malware detection")
		}
		if !errors.Is(err, restore.ErrMalwareDetected) {
			t.Fatalf("expected ErrMalwareDetected, got %v", err)
		}
		if result.Status != database.RestoreStatusFailed {
			t.Errorf("expected failed status, got %s", result.Status)
		}
		if result.ThreatReport == nil || result.ThreatReport.ThreatLevel != ransomware.ThreatLevelCritical {
			t.Errorf("expected critical threat report surfaced on result")
		}
		if scanner.calls != 1 {
			t.Errorf("expected scanner to run once, ran %d", scanner.calls)
		}
		if _, statErr := os.Stat(targetDB); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("target database must not be created when malware is detected")
		}
	})

	t.Run("clean_scan_proceeds", func(t *testing.T) {
		targetDB := filepath.Join(t.TempDir(), "restored.db")
		scanner := &fakeScanner{} // ThreatLevelNone
		engine := restore.NewEngine(&restore.Config{
			TempDirectory:   filepath.Join(t.TempDir(), "restore-temp"),
			StorageProvider: provider,
			ArtifactScanner: scanner,
		})

		result, err := engine.RestoreBackup(ctx, metadata, &restore.RestoreOptions{
			TargetDatabase: targetDB,
		})
		if err != nil {
			t.Fatalf("expected clean restore, got %v", err)
		}
		if result.Status != database.RestoreStatusSuccess {
			t.Fatalf("expected success, got %s", result.Status)
		}
		if scanner.calls != 1 {
			t.Errorf("expected scanner to run once, ran %d", scanner.calls)
		}
		got := queryColumn(t, targetDB, "SELECT name FROM users ORDER BY id;")
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("restored rows mismatch: got %v want %v", got, want)
		}
	})

	t.Run("skip_scan_bypasses_detector", func(t *testing.T) {
		targetDB := filepath.Join(t.TempDir(), "restored.db")
		scanner := &fakeScanner{report: &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelCritical}}
		engine := restore.NewEngine(&restore.Config{
			TempDirectory:   filepath.Join(t.TempDir(), "restore-temp"),
			StorageProvider: provider,
			ArtifactScanner: scanner,
		})

		result, err := engine.RestoreBackup(ctx, metadata, &restore.RestoreOptions{
			TargetDatabase:  targetDB,
			SkipMalwareScan: true,
		})
		if err != nil {
			t.Fatalf("expected restore to proceed when scan skipped, got %v", err)
		}
		if result.Status != database.RestoreStatusSuccess {
			t.Fatalf("expected success, got %s", result.Status)
		}
		if scanner.calls != 0 {
			t.Errorf("scanner must not run when SkipMalwareScan is set, ran %d", scanner.calls)
		}
	})
}

// TestE2E_SelectiveRestore proves the RestoreTables API contract: the requested
// subset is passed through to the driver and surfaced in RestoredTables. It uses
// a registered echo driver for the contract (real SQLite restores the whole
// file regardless, which is verified and documented in the second subtest).
func TestE2E_SelectiveRestore(t *testing.T) {
	ctx := context.Background()

	t.Run("api_contract_reports_requested_tables", func(t *testing.T) {
		drv := &echoDriver{}
		const dbType database.DatabaseType = "e2eecho"
		database.RegisterDriver(dbType, func() database.Driver { return drv })

		content := []byte("echo-backup-artifact")
		backupPath := filepath.Join(t.TempDir(), "backup.sql")
		if err := os.WriteFile(backupPath, content, 0o600); err != nil {
			t.Fatal(err)
		}
		meta := &models.BackupMetadata{
			ID:              "echo-backup",
			DatabaseType:    dbType,
			BackupPath:      backupPath,
			StorageLocation: backupPath,
			Checksum:        sha256Hex(content),
		}

		engine := restore.NewEngine(&restore.Config{TempDirectory: t.TempDir()})
		requested := []string{"users", "orders"}
		result, err := engine.RestoreTables(ctx, meta, requested, &restore.RestoreOptions{
			TargetDatabase: "targetdb",
		})
		if err != nil {
			t.Fatalf("RestoreTables: %v", err)
		}
		if strings.Join(drv.gotTables, ",") != strings.Join(requested, ",") {
			t.Errorf("driver received tables %v, want %v", drv.gotTables, requested)
		}
		if strings.Join(result.RestoredTables, ",") != strings.Join(requested, ",") {
			t.Errorf("RestoredTables = %v, want %v", result.RestoredTables, requested)
		}
	})

	t.Run("sqlite_restores_whole_file", func(t *testing.T) {
		// The real SQLite driver restores the entire database file and ignores a
		// requested table subset. This asserts that documented behavior: a
		// selective restore succeeds and all tables (not just the requested one)
		// are present afterward.
		root := t.TempDir()
		dbPath := filepath.Join(root, "source.db")
		seedUsersDB(t, dbPath)
		createTable(t, dbPath,
			`CREATE TABLE orders (id INTEGER PRIMARY KEY, item TEXT);`,
			`INSERT INTO orders (item) VALUES ('widget'), ('gadget');`)

		provider := newLocalStorage(t, filepath.Join(root, "store"))
		backupEngine := backup.NewEngine(&backup.Config{
			TempDirectory:   filepath.Join(root, "backup-temp"),
			StorageProvider: provider,
		})
		metadata, err := backupEngine.CreateBackup(ctx, &backup.CreateOptions{
			DatabaseType: database.DatabaseTypeSQLite,
			Database:     dbPath,
			Name:         "selective",
		})
		if err != nil {
			t.Fatalf("CreateBackup: %v", err)
		}

		targetDB := filepath.Join(root, "restored.db")
		restoreEngine := restore.NewEngine(&restore.Config{
			TempDirectory:   filepath.Join(root, "restore-temp"),
			StorageProvider: provider,
		})
		result, err := restoreEngine.RestoreTables(ctx, metadata, []string{"users"}, &restore.RestoreOptions{
			TargetDatabase: targetDB,
		})
		if err != nil {
			t.Fatalf("RestoreTables(sqlite): %v", err)
		}
		if result.Status != database.RestoreStatusSuccess {
			t.Fatalf("expected success, got %s (%v)", result.Status, result.Error)
		}
		// Whole-file restore: both tables must be present despite requesting one.
		if !tableExists(t, targetDB, "users") {
			t.Errorf("users table missing after restore")
		}
		if !tableExists(t, targetDB, "orders") {
			t.Errorf("orders table missing: sqlite restores the whole file")
		}
	})
}

// TestE2E_MITREAndSIEM proves detected threats are enriched with MITRE ATT&CK
// techniques and exported to a SIEM as well-formed JSON carrying those
// techniques and the severity, using a real httptest SIEM endpoint.
func TestE2E_MITREAndSIEM(t *testing.T) {
	ctx := context.Background()

	report := &ransomware.ThreatReport{
		ThreatLevel: ransomware.ThreatLevelCritical,
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		Description: "ransomware note dropped; shadow copies deleted",
		FilePath:    "/backups/infected.sql",
		Indicators:  []string{"vssadmin.exe delete shadows /all /quiet"},
	}

	ransomware.EnrichWithMITRE(report)
	if !hasTechnique(report.MITRETechniques, "T1486") {
		t.Errorf("expected T1486 (Data Encrypted for Impact) from signature match")
	}
	if !hasTechnique(report.MITRETechniques, "T1490") {
		t.Errorf("expected T1490 (Inhibit System Recovery) from vssadmin/shadow indicators")
	}

	type siemPayload struct {
		Severity   string                      `json:"severity"`
		ThreatType string                      `json:"threat.type"`
		MITRE      []ransomware.MITRETechnique `json:"threat.mitre.techniques"`
		Indicators []string                    `json:"threat.indicators"`
	}

	received := make(chan siemPayload, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		var payload siemPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		received <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := siem.NewExporter(siem.Config{
		Endpoint: server.URL,
		Format:   siem.FormatWebhook,
		Source:   "db-backup-e2e",
	})
	if err := exporter.Export(ctx, report); err != nil {
		t.Fatalf("Export: %v", err)
	}

	select {
	case payload := <-received:
		if payload.Severity != string(ransomware.ThreatLevelCritical) {
			t.Errorf("severity = %q, want CRITICAL", payload.Severity)
		}
		if payload.ThreatType != string(ransomware.ThreatTypeSignatureMatch) {
			t.Errorf("threat.type = %q, want SIGNATURE_MATCH", payload.ThreatType)
		}
		if !hasTechnique(payload.MITRE, "T1486") || !hasTechnique(payload.MITRE, "T1490") {
			t.Errorf("SIEM payload missing expected ATT&CK techniques: %+v", payload.MITRE)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SIEM endpoint did not receive the exported event")
	}
}

// hasTechnique reports whether the technique list contains the given ATT&CK ID.
func hasTechnique(techniques []ransomware.MITRETechnique, id string) bool {
	for _, tq := range techniques {
		if tq.ID == id {
			return true
		}
	}
	return false
}
