package restore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	stor "github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"
)

// fakeScanner is a test ArtifactScanner returning a preconfigured report/error.
type fakeScanner struct {
	report *ransomware.ThreatReport
	err    error
	calls  int
}

func (f *fakeScanner) ScanFile(_ context.Context, path string) (*ransomware.ThreatReport, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.report != nil {
		r := *f.report
		r.FilePath = path
		return &r, nil
	}
	return &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelNone, FilePath: path}, nil
}

// fakeDriver is a minimal database.Driver used to exercise the full restore
// path without a real database. It records the RestoreOptions it received.
type fakeDriver struct {
	gotTables []string
	restored  []string
}

func (d *fakeDriver) Connect(context.Context, *database.ConnectionConfig) error { return nil }
func (d *fakeDriver) Disconnect() error                                         { return nil }
func (d *fakeDriver) Ping(context.Context) error                                { return nil }
func (d *fakeDriver) Backup(context.Context, *database.BackupOptions) (*database.BackupResult, error) {
	return nil, nil
}
func (d *fakeDriver) StreamBackup(context.Context, *database.BackupOptions, io.Writer) error {
	return nil
}
func (d *fakeDriver) GetBackupSize(context.Context, *database.BackupOptions) (int64, error) {
	return 0, nil
}
func (d *fakeDriver) Restore(_ context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	d.gotTables = opts.Tables
	restored := opts.Tables
	if len(restored) == 0 {
		restored = []string{"all"}
	}
	d.restored = restored
	return &database.RestoreResult{RestoredTables: restored, RowsRestored: 42}, nil
}
func (d *fakeDriver) StreamRestore(context.Context, *database.RestoreOptions, io.Reader) error {
	return nil
}
func (d *fakeDriver) ValidateRestore(context.Context, *database.RestoreOptions) error { return nil }
func (d *fakeDriver) GetDatabases(context.Context) ([]string, error) {
	return []string{"targetdb"}, nil
}
func (d *fakeDriver) GetTables(context.Context, string) ([]string, error)         { return nil, nil }
func (d *fakeDriver) GetTableSize(context.Context, string, string) (int64, error) { return 0, nil }
func (d *fakeDriver) GetDatabaseSize(context.Context) (int64, error)              { return 0, nil }
func (d *fakeDriver) GetVersion(context.Context) (string, error)                  { return "1.0", nil }
func (d *fakeDriver) GetType() database.DatabaseType                              { return "fakedb" }
func (d *fakeDriver) SupportsIncremental() bool                                   { return false }
func (d *fakeDriver) SupportsPITR() bool                                          { return false }

// registerFakeDriver registers a fresh fakeDriver under a test database type and
// returns it for assertions.
func registerFakeDriver(t *testing.T) *fakeDriver {
	t.Helper()
	drv := &fakeDriver{}
	database.RegisterDriver("fakedb", func() database.Driver { return drv })
	return drv
}

// writeLocalBackup writes a local backup file and returns metadata pointing at it.
func writeLocalBackup(t *testing.T, content []byte) *models.BackupMetadata {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.sql")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return &models.BackupMetadata{
		ID:              "fake-backup",
		DatabaseType:    "fakedb",
		BackupPath:      path,
		StorageLocation: path,
		Checksum:        sha256Hex(content),
	}
}

func baseRestoreOpts() *RestoreOptions {
	return &RestoreOptions{TargetDatabase: "targetdb"}
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func newLocalProvider(t *testing.T, dir string) stor.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&stor.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// uploadArtifact writes content to a temp source file and uploads it to the
// provider at remotePath, returning the checksum.
func uploadArtifact(t *testing.T, provider stor.Provider, remotePath string, content []byte) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "src")
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := provider.Upload(context.Background(), tmp, remotePath, nil); err != nil {
		t.Fatalf("upload: %v", err)
	}
	return sha256Hex(content)
}

// TestDownloadBackup_RemoteRoundTrip proves an uploaded artifact can be
// downloaded and its checksum verifies.
func TestDownloadBackup_RemoteRoundTrip(t *testing.T) {
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "temp")
	provider := newLocalProvider(t, storeDir)

	content := []byte("BACKUP-CONTENT-round-trip")
	remotePath := "backups/abc123/abc123.sql"
	checksum := uploadArtifact(t, provider, remotePath, content)

	meta := &models.BackupMetadata{
		ID:              "abc123",
		BackupPath:      filepath.Join(tempDir, "abc123.sql"),
		StorageLocation: "local://" + remotePath,
		Checksum:        checksum,
	}

	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	dest := filepath.Join(root, "downloaded.sql")
	if err := engine.DownloadBackup(context.Background(), meta, dest); err != nil {
		t.Fatalf("DownloadBackup: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q", string(got))
	}
}

// TestDownloadBackup_ChecksumMismatch ensures a corrupted/incorrect checksum
// causes the download verification to fail.
func TestDownloadBackup_ChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "temp")
	provider := newLocalProvider(t, storeDir)

	remotePath := "backups/xyz/xyz.sql"
	uploadArtifact(t, provider, remotePath, []byte("real content"))

	meta := &models.BackupMetadata{
		ID:              "xyz",
		StorageLocation: "local://" + remotePath,
		Checksum:        "0000000000000000000000000000000000000000000000000000000000000000",
	}
	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	if err := engine.DownloadBackup(context.Background(), meta, filepath.Join(root, "out.sql")); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
}

// TestValidateBackup_LocalChecksumMismatch checks local validation fails on
// checksum mismatch.
func TestValidateBackup_LocalChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "b.sql")
	if err := os.WriteFile(localPath, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	meta := &models.BackupMetadata{Checksum: "wrongsum"}
	engine := NewEngine(&Config{TempDirectory: root})

	if err := engine.validateBackup(context.Background(), meta, localPath); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}

	// Correct checksum should pass.
	meta.Checksum = sha256Hex([]byte("data"))
	if err := engine.validateBackup(context.Background(), meta, localPath); err != nil {
		t.Errorf("expected validation to pass, got %v", err)
	}
}

// TestValidateBackup_RemoteMissing ensures validation fails when a remote
// artifact does not exist.
func TestValidateBackup_RemoteMissing(t *testing.T) {
	root := t.TempDir()
	provider := newLocalProvider(t, filepath.Join(root, "store"))
	engine := NewEngine(&Config{TempDirectory: root, StorageProvider: provider})

	meta := &models.BackupMetadata{StorageLocation: "local://backups/missing/missing.sql"}
	if err := engine.validateBackup(context.Background(), meta, ""); err == nil {
		t.Fatalf("expected error for missing remote artifact")
	}
}

// TestDownloadBackup_LocalNilProvider verifies local backups are copied without
// a provider configured.
func TestDownloadBackup_LocalNilProvider(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "local.sql")
	content := []byte("local-only-backup")
	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatal(err)
	}
	meta := &models.BackupMetadata{
		BackupPath:      src,
		StorageLocation: src,
		Checksum:        sha256Hex(content),
	}
	engine := NewEngine(&Config{TempDirectory: root})

	dest := filepath.Join(root, "copy.sql")
	if err := engine.DownloadBackup(context.Background(), meta, dest); err != nil {
		t.Fatalf("DownloadBackup: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch")
	}
}

// TestRestoreBackup_MalwareDetectedAborts ensures a high-severity detection
// aborts the restore before the driver is ever reached.
func TestRestoreBackup_MalwareDetectedAborts(t *testing.T) {
	drv := registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("infected-backup"))

	scanner := &fakeScanner{report: &ransomware.ThreatReport{
		ThreatLevel: ransomware.ThreatLevelCritical,
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		Description: "known ransomware signature",
	}}
	engine := NewEngine(&Config{TempDirectory: t.TempDir(), ArtifactScanner: scanner})

	result, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
	if err == nil {
		t.Fatal("expected restore to abort on malware detection")
	}
	if !errors.Is(err, ErrMalwareDetected) {
		t.Fatalf("expected ErrMalwareDetected, got %v", err)
	}
	if result.Status != database.RestoreStatusFailed {
		t.Errorf("expected failed status, got %v", result.Status)
	}
	if result.ThreatReport == nil || result.ThreatReport.ThreatLevel != ransomware.ThreatLevelCritical {
		t.Errorf("expected threat report surfaced in result")
	}
	if scanner.calls != 1 {
		t.Errorf("expected scanner to run once, ran %d times", scanner.calls)
	}
	if drv.restored != nil {
		t.Errorf("driver must not be invoked when malware is detected")
	}
}

// TestRestoreBackup_CleanScanProceeds ensures a clean scan lets the restore run.
func TestRestoreBackup_CleanScanProceeds(t *testing.T) {
	drv := registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("clean-backup"))

	scanner := &fakeScanner{} // defaults to ThreatLevelNone
	engine := NewEngine(&Config{TempDirectory: t.TempDir(), ArtifactScanner: scanner})

	result, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
	if err != nil {
		t.Fatalf("expected clean restore, got %v", err)
	}
	if result.Status != database.RestoreStatusSuccess {
		t.Errorf("expected success, got %v", result.Status)
	}
	if scanner.calls != 1 {
		t.Errorf("expected scanner to run once, ran %d times", scanner.calls)
	}
	if drv.restored == nil {
		t.Errorf("expected driver restore to run after clean scan")
	}
}

// TestRestoreBackup_ScanFailsClosed ensures a scan error aborts by default.
func TestRestoreBackup_ScanFailsClosed(t *testing.T) {
	registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("some-backup"))

	scanner := &fakeScanner{err: errors.New("scanner exploded")}
	engine := NewEngine(&Config{TempDirectory: t.TempDir(), ArtifactScanner: scanner})

	_, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
	if err == nil {
		t.Fatal("expected scan failure to abort restore (fail closed)")
	}
}

// TestRestoreBackup_ScanWarnOnly ensures warn-only mode does not abort on a
// detection or a scan error.
func TestRestoreBackup_ScanWarnOnly(t *testing.T) {
	registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("warn-backup"))

	scanner := &fakeScanner{report: &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelCritical}}
	engine := NewEngine(&Config{
		TempDirectory:       t.TempDir(),
		ArtifactScanner:     scanner,
		MalwareScanWarnOnly: true,
	})

	result, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
	if err != nil {
		t.Fatalf("warn-only mode should not abort, got %v", err)
	}
	if result.Status != database.RestoreStatusSuccess {
		t.Errorf("expected success in warn-only mode, got %v", result.Status)
	}
}

// TestRestoreBackup_ScanDisabledAndNilSafe verifies scanning is skipped when
// opted out or when no scanner is configured.
func TestRestoreBackup_ScanDisabledAndNilSafe(t *testing.T) {
	t.Run("skip flag", func(t *testing.T) {
		registerFakeDriver(t)
		meta := writeLocalBackup(t, []byte("skip-backup"))
		scanner := &fakeScanner{report: &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelCritical}}
		engine := NewEngine(&Config{TempDirectory: t.TempDir(), ArtifactScanner: scanner})

		opts := baseRestoreOpts()
		opts.SkipMalwareScan = true
		result, err := engine.RestoreBackup(context.Background(), meta, opts)
		if err != nil {
			t.Fatalf("expected restore to proceed when scan skipped, got %v", err)
		}
		if result.Status != database.RestoreStatusSuccess {
			t.Errorf("expected success, got %v", result.Status)
		}
		if scanner.calls != 0 {
			t.Errorf("scanner must not run when SkipMalwareScan is set")
		}
	})

	t.Run("nil scanner", func(t *testing.T) {
		registerFakeDriver(t)
		meta := writeLocalBackup(t, []byte("nil-backup"))
		engine := NewEngine(&Config{TempDirectory: t.TempDir()})

		result, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
		if err != nil {
			t.Fatalf("expected restore to proceed with nil scanner, got %v", err)
		}
		if result.Status != database.RestoreStatusSuccess {
			t.Errorf("expected success, got %v", result.Status)
		}
	})
}

// TestRestoreBackup_BelowThresholdProceeds ensures a detection below the abort
// threshold does not stop the restore but is still surfaced.
func TestRestoreBackup_BelowThresholdProceeds(t *testing.T) {
	registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("low-risk-backup"))

	scanner := &fakeScanner{report: &ransomware.ThreatReport{ThreatLevel: ransomware.ThreatLevelMedium}}
	engine := NewEngine(&Config{
		TempDirectory:        t.TempDir(),
		ArtifactScanner:      scanner,
		MalwareScanThreshold: ransomware.ThreatLevelHigh,
	})

	result, err := engine.RestoreBackup(context.Background(), meta, baseRestoreOpts())
	if err != nil {
		t.Fatalf("medium threat below HIGH threshold should proceed, got %v", err)
	}
	if result.ThreatReport == nil || result.ThreatReport.ThreatLevel != ransomware.ThreatLevelMedium {
		t.Errorf("expected medium threat surfaced in result")
	}
}

// TestRestoreTables_PassesTablesThrough verifies selective restore flows the
// requested tables to the driver and reports the restored subset.
func TestRestoreTables_PassesTablesThrough(t *testing.T) {
	drv := registerFakeDriver(t)
	meta := writeLocalBackup(t, []byte("selective-backup"))
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})

	tables := []string{"users", "orders"}
	result, err := engine.RestoreTables(context.Background(), meta, tables, baseRestoreOpts())
	if err != nil {
		t.Fatalf("RestoreTables: %v", err)
	}
	if len(drv.gotTables) != 2 || drv.gotTables[0] != "users" || drv.gotTables[1] != "orders" {
		t.Errorf("driver did not receive requested tables: %v", drv.gotTables)
	}
	if len(result.RestoredTables) != 2 {
		t.Errorf("expected restored tables reported, got %v", result.RestoredTables)
	}
}

// TestRestoreTables_RequiresTables ensures an empty table list is rejected.
func TestRestoreTables_RequiresTables(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	if _, err := engine.RestoreTables(context.Background(), &models.BackupMetadata{}, nil, nil); err == nil {
		t.Fatal("expected error for empty table list")
	}
}
