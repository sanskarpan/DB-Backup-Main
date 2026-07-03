package instant

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"
)

// seedSQLiteDB creates a real SQLite database with a known table and rows.
func seedSQLiteDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Fatalf("close sqlite: %v", cerr)
		}
	}()
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO t (name) VALUES ('alice'), ('bob');`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

// newLocalProvider builds a real local storage provider rooted at dir.
func newLocalProvider(t *testing.T, dir string) storage.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&storage.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// backupSeededDB backs up a freshly seeded SQLite DB to a local provider and
// returns the provider and the resulting metadata.
func backupSeededDB(t *testing.T, root string) (storage.Provider, *models.BackupMetadata) {
	t.Helper()

	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	provider := newLocalProvider(t, filepath.Join(root, "store"))
	engine := backup.NewEngine(&backup.Config{
		TempDirectory:   filepath.Join(root, "backup-temp"),
		StorageProvider: provider,
	})

	meta, err := engine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "instant-test",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if meta.Status != database.BackupStatusSuccess {
		t.Fatalf("expected backup success, got %s", meta.Status)
	}
	return provider, meta
}

func TestPrepareInstant_RunsFromBackup(t *testing.T) {
	root := t.TempDir()
	provider, meta := backupSeededDB(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	recoverer := NewRecoverer(restoreEngine)
	handle, err := recoverer.PrepareInstant(context.Background(), meta, &Options{
		WorkDir: filepath.Join(root, "work"),
	})
	if err != nil {
		t.Fatalf("PrepareInstant: %v", err)
	}
	t.Cleanup(func() {
		if cerr := handle.Cleanup(); cerr != nil {
			t.Errorf("Cleanup: %v", cerr)
		}
	})

	if handle.Elapsed <= 0 {
		t.Errorf("expected Elapsed > 0, got %v", handle.Elapsed)
	}
	if handle.ReadyAt.IsZero() {
		t.Error("expected ReadyAt to be set")
	}
	if handle.DatabaseType != database.DatabaseTypeSQLite {
		t.Errorf("unexpected DatabaseType: %s", handle.DatabaseType)
	}
	if _, statErr := os.Stat(handle.Path); statErr != nil {
		t.Fatalf("materialized db missing: %v", statErr)
	}

	db, err := handle.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	names := map[int]string{}
	rows, err := db.QueryContext(context.Background(), "SELECT id, name FROM t ORDER BY id")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			t.Errorf("rows close: %v", cerr)
		}
	}()
	for rows.Next() {
		var id int
		var name string
		if scanErr := rows.Scan(&id, &name); scanErr != nil {
			t.Fatalf("scan: %v", scanErr)
		}
		names[id] = name
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("rows err: %v", rowsErr)
	}

	if len(names) != 2 || names[1] != "alice" || names[2] != "bob" {
		t.Fatalf("unexpected rows: %v", names)
	}

	// Prove the database is genuinely read-write, not a read-only mount.
	if _, err := db.ExecContext(context.Background(),
		"INSERT INTO t (name) VALUES ('carol')"); err != nil {
		t.Fatalf("insert into instant db: %v", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM t").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows after insert, got %d", count)
	}
}

func TestPrepareInstant_OwnedWorkDirRemovedOnCleanup(t *testing.T) {
	root := t.TempDir()
	provider, meta := backupSeededDB(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	handle, err := NewRecoverer(restoreEngine).PrepareInstant(context.Background(), meta, nil)
	if err != nil {
		t.Fatalf("PrepareInstant: %v", err)
	}

	workDir := filepath.Dir(handle.Path)
	if _, statErr := os.Stat(workDir); statErr != nil {
		t.Fatalf("work dir missing before cleanup: %v", statErr)
	}
	if cerr := handle.Cleanup(); cerr != nil {
		t.Fatalf("Cleanup: %v", cerr)
	}
	if _, statErr := os.Stat(workDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected owned work dir removed, stat err: %v", statErr)
	}
}

func TestPrepareInstant_MissingArtifactReturnsError(t *testing.T) {
	root := t.TempDir()
	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory: filepath.Join(root, "restore-temp"),
	})

	meta := &models.BackupMetadata{
		ID:           "missing",
		DatabaseType: database.DatabaseTypeSQLite,
		BackupPath:   filepath.Join(root, "does-not-exist.db"),
	}

	handle, err := NewRecoverer(restoreEngine).PrepareInstant(context.Background(), meta, nil)
	if err == nil {
		t.Fatal("expected error for missing artifact, got nil")
	}
	if handle != nil {
		t.Fatalf("expected nil handle on error, got %+v", handle)
	}
}

// corruptMaterializer writes non-SQLite bytes to simulate a corrupt artifact.
type corruptMaterializer struct{}

func (corruptMaterializer) DownloadBackup(_ context.Context, _ *models.BackupMetadata, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte("this is not a sqlite database"), 0o600)
}

func TestPrepareInstant_CorruptArtifactReturnsNotUsable(t *testing.T) {
	root := t.TempDir()
	meta := &models.BackupMetadata{
		ID:           "corrupt",
		DatabaseType: database.DatabaseTypeSQLite,
	}

	handle, err := NewRecoverer(corruptMaterializer{}).PrepareInstant(
		context.Background(), meta, &Options{WorkDir: filepath.Join(root, "work")},
	)
	if err == nil {
		t.Fatal("expected error for corrupt artifact, got nil")
	}
	if !errors.Is(err, ErrNotUsable) {
		t.Fatalf("expected ErrNotUsable, got %v", err)
	}
	if handle != nil {
		t.Fatalf("expected nil handle, got %+v", handle)
	}
}

func TestPrepareInstant_NilMetadata(t *testing.T) {
	handle, err := NewRecoverer(corruptMaterializer{}).PrepareInstant(context.Background(), nil, nil)
	if !errors.Is(err, ErrNilMetadata) {
		t.Fatalf("expected ErrNilMetadata, got %v", err)
	}
	if handle != nil {
		t.Fatalf("expected nil handle, got %+v", handle)
	}
}

func TestPrepareInstant_UnsupportedType(t *testing.T) {
	meta := &models.BackupMetadata{ID: "x", DatabaseType: database.DatabaseType("postgres")}
	handle, err := NewRecoverer(corruptMaterializer{}).PrepareInstant(context.Background(), meta, nil)
	if !errors.Is(err, ErrUnsupportedDatabaseType) {
		t.Fatalf("expected ErrUnsupportedDatabaseType, got %v", err)
	}
	if handle != nil {
		t.Fatalf("expected nil handle, got %+v", handle)
	}
}

func TestHandleClosedAfterClose(t *testing.T) {
	root := t.TempDir()
	provider, meta := backupSeededDB(t, root)
	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	handle, err := NewRecoverer(restoreEngine).PrepareInstant(context.Background(), meta,
		&Options{WorkDir: filepath.Join(root, "work")})
	if err != nil {
		t.Fatalf("PrepareInstant: %v", err)
	}
	if _, err := handle.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := handle.Open(); !errors.Is(err, ErrHandleClosed) {
		t.Fatalf("expected ErrHandleClosed after Close, got %v", err)
	}
}
