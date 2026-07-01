package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/storage/local"

	// Register the SQLite driver used by the end-to-end backup test.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	// Register the go-sqlite3 driver for seeding the test database.
	_ "github.com/mattn/go-sqlite3"

	stor "github.com/sanskarpan/db-backup/internal/storage"
)

// seedSQLiteDB creates a small SQLite database file with one table.
func seedSQLiteDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO t (name) VALUES ('alice'), ('bob');`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func newLocalProvider(t *testing.T, dir string) stor.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&stor.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// TestCreateBackup_UploadsToProvider proves the backup is durably stored in the
// storage provider and that the recorded StorageLocation points at it.
func TestCreateBackup_UploadsToProvider(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	storeDir := filepath.Join(root, "store")
	dbPath := filepath.Join(root, "source.db")

	seedSQLiteDB(t, dbPath)
	provider := newLocalProvider(t, storeDir)

	engine := NewEngine(&Config{
		TempDirectory:   tempDir,
		StorageProvider: provider,
	})

	ctx := context.Background()
	var sawUploading bool
	metadata, err := engine.CreateBackup(ctx, &CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "test-backup",
		ProgressCallback: func(p Progress) {
			if p.Stage == "uploading" {
				sawUploading = true
			}
		},
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	if metadata.Status != database.BackupStatusSuccess {
		t.Fatalf("expected success, got %s", metadata.Status)
	}
	if !sawUploading {
		t.Errorf("expected an 'uploading' progress stage")
	}
	if !strings.HasPrefix(metadata.StorageLocation, "local://backups/") {
		t.Errorf("unexpected storage location: %q", metadata.StorageLocation)
	}

	// The artifact must exist in the provider at the recorded remote path.
	remotePath := strings.TrimPrefix(metadata.StorageLocation, "local://")
	exists, err := provider.Exists(ctx, remotePath)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Errorf("backup artifact not found in storage at %q", remotePath)
	}

	// Metadata JSON must also be uploaded alongside the artifact.
	metaExists, err := provider.Exists(ctx, filepath.Join(filepath.Dir(remotePath), "metadata.json"))
	if err != nil {
		t.Fatalf("Exists metadata: %v", err)
	}
	if !metaExists {
		t.Errorf("metadata.json not uploaded to storage")
	}

	// ValidateBackup must pass against the stored (and still-local) artifact.
	if err := engine.ValidateBackup(ctx, metadata.ID); err != nil {
		t.Errorf("ValidateBackup: %v", err)
	}
}

// TestCreateBackup_UploadFailureFailsBackup ensures a backup whose upload fails
// is never reported as successful.
func TestCreateBackup_UploadFailureFailsBackup(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	engine := NewEngine(&Config{
		TempDirectory:   tempDir,
		StorageProvider: failingProvider{},
	})

	metadata, err := engine.CreateBackup(context.Background(), &CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
	})
	if err == nil {
		t.Fatalf("expected error when upload fails")
	}
	if metadata.Status == database.BackupStatusSuccess {
		t.Errorf("backup must not be marked successful when upload fails")
	}
}

// TestCreateBackup_NoProviderLocalOnly verifies local-only behavior when no
// provider is configured.
func TestCreateBackup_NoProviderLocalOnly(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	engine := NewEngine(&Config{TempDirectory: tempDir})

	metadata, err := engine.CreateBackup(context.Background(), &CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if metadata.StorageLocation != metadata.BackupPath {
		t.Errorf("expected StorageLocation to equal local BackupPath, got %q vs %q", metadata.StorageLocation, metadata.BackupPath)
	}
	if _, err := os.Stat(metadata.BackupPath); err != nil {
		t.Errorf("local backup file missing: %v", err)
	}
}

// TestValidateBackup_ChecksumMismatch ensures validation fails on tamper.
func TestValidateBackup_ChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	engine := NewEngine(&Config{TempDirectory: tempDir})

	// Write a backup file and matching-but-wrong metadata checksum.
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(tempDir, "b1.sql")
	if err := os.WriteFile(backupPath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}
	meta := &models.BackupMetadata{
		ID:              "b1",
		BackupPath:      backupPath,
		StorageLocation: backupPath,
		Checksum:        "deadbeef",
	}
	if err := engine.saveMetadata(meta); err != nil {
		t.Fatal(err)
	}

	if err := engine.ValidateBackup(context.Background(), "b1"); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
}

// failingProvider is a Provider whose uploads always fail.
type failingProvider struct{ stor.Provider }

func (failingProvider) Upload(ctx context.Context, localPath, remotePath string, opts *stor.UploadOptions) error {
	return os.ErrPermission
}

func (failingProvider) GetType() stor.ProviderType { return stor.ProviderTypeLocal }
