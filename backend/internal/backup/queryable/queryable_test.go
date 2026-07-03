package queryable

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	// Register the go-sqlite3 driver for seeding the source database.
	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"

	// Register the SQLite platform driver used by the backup engine.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	stor "github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"
)

// seedSQLiteDB creates a small SQLite database file with a users table holding
// three known rows.
func seedSQLiteDB(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (1,'alice'),(2,'bob'),(3,'carol');`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

// makeBackup seeds a real SQLite DB, backs it up to a local storage provider,
// and returns the metadata plus the provider and its on-disk root directory.
func makeBackup(t *testing.T) (*models.BackupMetadata, stor.Provider, string) {
	t.Helper()
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "temp")
	dbPath := filepath.Join(root, "source.db")

	seedSQLiteDB(t, dbPath)

	provider, err := local.NewLocalProvider(&stor.LocalConfig{Path: storeDir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}

	engine := backup.NewEngine(&backup.Config{
		TempDirectory:   tempDir,
		StorageProvider: provider,
	})

	meta, err := engine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "queryable-test",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if meta.Status != database.BackupStatusSuccess {
		t.Fatalf("expected success status, got %s", meta.Status)
	}

	return meta, provider, storeDir
}

// artifactBytes reads the raw stored backup artifact from the local provider's
// on-disk location.
func artifactBytes(t *testing.T, storeDir string, meta *models.BackupMetadata) []byte {
	t.Helper()
	remote := strings.TrimPrefix(meta.StorageLocation, "local://")
	b, err := os.ReadFile(filepath.Join(storeDir, remote))
	if err != nil {
		t.Fatalf("read stored artifact: %v", err)
	}
	return b
}

func TestMount_QueryRealBackup(t *testing.T) {
	meta, provider, storeDir := makeBackup(t)
	type userRow struct {
		id   int64
		name string
	}
	want := []userRow{
		{1, "alice"},
		{2, "bob"},
		{3, "carol"},
	}

	before := artifactBytes(t, storeDir, meta)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(t.TempDir(), "restore"),
		StorageProvider: provider,
	})

	mounter := NewMounter(restoreEngine)
	ctx := context.Background()

	mount, err := mounter.Mount(ctx, meta, &MountOptions{TempDirectory: filepath.Join(t.TempDir(), "mounts")})
	if err != nil {
		t.Fatalf("Mount: %v", err)
	}
	defer func() {
		if unmountErr := mount.Unmount(); unmountErr != nil {
			t.Errorf("Unmount: %v", unmountErr)
		}
	}()

	// Tables must include the seeded users table.
	tables, err := mount.Tables(ctx)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if !containsString(tables, "users") {
		t.Fatalf("expected users table, got %v", tables)
	}

	// A SELECT must return exactly the seeded rows.
	res, err := mount.Query(ctx, "SELECT id, name FROM users ORDER BY id")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.RowCount != len(want) {
		t.Fatalf("expected %d rows, got %d", len(want), res.RowCount)
	}
	if len(res.Columns) != 2 || res.Columns[0] != "id" || res.Columns[1] != "name" {
		t.Fatalf("unexpected columns: %v", res.Columns)
	}
	for i, row := range res.Rows {
		if len(row) != 2 {
			t.Fatalf("row %d: expected 2 cols, got %d", i, len(row))
		}
		gotID, ok := row[0].(int64)
		if !ok {
			t.Fatalf("row %d: id not int64: %T", i, row[0])
		}
		gotName, ok := row[1].(string)
		if !ok {
			t.Fatalf("row %d: name not string: %T", i, row[1])
		}
		if gotID != want[i].id || gotName != want[i].name {
			t.Fatalf("row %d mismatch: got (%v,%v) want %v", i, gotID, gotName, want[i])
		}
	}

	// Mutating statements must be rejected.
	for _, mut := range []string{
		"UPDATE users SET name='x' WHERE id=1",
		"DELETE FROM users",
		"DROP TABLE users",
		"INSERT INTO users (id, name) VALUES (99,'mallory')",
	} {
		if _, mutErr := mount.Query(ctx, mut); !errors.Is(mutErr, ErrNotReadOnly) {
			t.Fatalf("expected ErrNotReadOnly for %q, got %v", mut, mutErr)
		}
	}

	// An empty query is rejected too.
	if _, emptyErr := mount.Query(ctx, "   "); !errors.Is(emptyErr, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", emptyErr)
	}

	// The stored backup artifact must be byte-for-byte unchanged after querying.
	after := artifactBytes(t, storeDir, meta)
	if len(before) != len(after) {
		t.Fatalf("artifact size changed: before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("artifact bytes changed at offset %d", i)
		}
	}

	// The row count in the mount must still be 3 (proving no mutation slipped in).
	res2, err := mount.Query(ctx, "SELECT COUNT(*) FROM users")
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if got, ok := res2.Rows[0][0].(int64); !ok || got != 3 {
		t.Fatalf("expected 3 rows in users, got %v", res2.Rows[0][0])
	}
}

func TestMount_UnsupportedType(t *testing.T) {
	mounter := NewMounter(nopMaterializer{})
	_, err := mounter.Mount(context.Background(), &models.BackupMetadata{DatabaseType: "postgres"}, nil)
	if !errors.Is(err, ErrUnsupportedDatabaseType) {
		t.Fatalf("expected ErrUnsupportedDatabaseType, got %v", err)
	}
}

func TestMount_QueryAfterUnmount(t *testing.T) {
	meta, provider, _ := makeBackup(t)
	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(t.TempDir(), "restore"),
		StorageProvider: provider,
	})
	mount, err := NewMounter(restoreEngine).Mount(context.Background(), meta, nil)
	if err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := mount.Unmount(); err != nil {
		t.Fatalf("Unmount: %v", err)
	}
	if _, err := mount.Query(context.Background(), "SELECT 1"); !errors.Is(err, ErrNotMounted) {
		t.Fatalf("expected ErrNotMounted, got %v", err)
	}
	// A second Unmount is a no-op.
	if err := mount.Unmount(); err != nil {
		t.Fatalf("second Unmount: %v", err)
	}
}

// nopMaterializer is a Materializer used only for the unsupported-type path,
// where materialization is never reached.
type nopMaterializer struct{}

func (nopMaterializer) DownloadBackup(_ context.Context, _ *models.BackupMetadata, _ string) error {
	return errors.New("should not be called")
}

func containsString(items []string, target string) bool {
	for _, s := range items {
		if s == target {
			return true
		}
	}
	return false
}
