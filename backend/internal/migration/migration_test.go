package migration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	stor "github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"

	// Register the SQLite platform driver used for the real backup/restore.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	// Register the go-sqlite3 SQL driver for seeding and verifying databases.
	_ "github.com/mattn/go-sqlite3"
)

const localScheme = "local://"

// seedSQLiteDB creates a small SQLite database file with one seeded table.
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

// readNames returns the names in table t of the SQLite database at path.
func readNames(t *testing.T, path string) []string {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name FROM t ORDER BY id;`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return names
}

func newLocalProvider(t *testing.T, dir string) stor.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&stor.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// sha256File returns the hex SHA-256 of the file at path.
func sha256File(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("hash %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// makeBackup performs a real SQLite backup into the source provider and returns
// its metadata plus the source provider.
func makeBackup(t *testing.T) (meta *models.BackupMetadata, src stor.Provider) {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	srcStore := filepath.Join(root, "src-store")
	src = newLocalProvider(t, srcStore)

	engine := backup.NewEngine(&backup.Config{
		TempDirectory:   filepath.Join(root, "backup-temp"),
		StorageProvider: src,
	})

	meta, err := engine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "migration-source",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if meta.Status != database.BackupStatusSuccess {
		t.Fatalf("backup not successful: %s", meta.Status)
	}
	if !strings.HasPrefix(meta.StorageLocation, localScheme) {
		t.Fatalf("unexpected storage location: %q", meta.StorageLocation)
	}
	return meta, src
}

// TestMigrateArtifact_CrossProvider proves an artifact and its metadata.json are
// streamed from one local provider to a second provider with a different root,
// and that the destination bytes and checksum match the recorded backup.
func TestMigrateArtifact_CrossProvider(t *testing.T) {
	meta, src := makeBackup(t)

	dstStore := filepath.Join(t.TempDir(), "dst-store")
	dst := newLocalProvider(t, dstStore)

	migrator := NewMigrator(src, nil)
	res, err := migrator.MigrateArtifact(context.Background(), meta, dst, nil)
	if err != nil {
		t.Fatalf("MigrateArtifact: %v", err)
	}

	if !res.ChecksumVerified {
		t.Errorf("expected checksum to be verified")
	}
	if res.BytesCopied <= 0 {
		t.Errorf("expected bytes copied, got %d", res.BytesCopied)
	}
	if !res.MetadataMigrated {
		t.Errorf("expected metadata.json to be migrated")
	}
	if res.SourceType != stor.ProviderTypeLocal || res.DestType != stor.ProviderTypeLocal {
		t.Errorf("unexpected provider types: src=%s dst=%s", res.SourceType, res.DestType)
	}

	remotePath := strings.TrimPrefix(meta.StorageLocation, localScheme)

	// The artifact must exist on the destination and its bytes must hash to the
	// recorded checksum.
	exists, err := dst.Exists(context.Background(), remotePath)
	if err != nil {
		t.Fatalf("dst Exists: %v", err)
	}
	if !exists {
		t.Fatalf("artifact missing on destination at %q", remotePath)
	}
	destChecksum := sha256File(t, filepath.Join(dstStore, remotePath))
	if destChecksum != meta.Checksum {
		t.Errorf("destination checksum %s != backup checksum %s", destChecksum, meta.Checksum)
	}

	// metadata.json must be present alongside the artifact on the destination.
	metaRemote := filepath.Join(filepath.Dir(remotePath), "metadata.json")
	metaExists, err := dst.Exists(context.Background(), metaRemote)
	if err != nil {
		t.Fatalf("dst metadata Exists: %v", err)
	}
	if !metaExists {
		t.Errorf("metadata.json missing on destination at %q", metaRemote)
	}
}

// TestMigrateAndRestoreToTarget proves a backup taken against one system can be
// migrated to a different storage provider and then restored into a fresh,
// different target database path, with the original data intact.
func TestMigrateAndRestoreToTarget(t *testing.T) {
	meta, src := makeBackup(t)

	dstStore := filepath.Join(t.TempDir(), "dst-store")
	dst := newLocalProvider(t, dstStore)

	// Migrate the artifact to the destination provider.
	if _, err := NewMigrator(src, nil).MigrateArtifact(context.Background(), meta, dst, nil); err != nil {
		t.Fatalf("MigrateArtifact: %v", err)
	}

	// Restore, using the destination provider as the backing store, into a fresh
	// target path that is different from the origin database.
	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(t.TempDir(), "restore-temp"),
		StorageProvider: dst,
	})
	migrator := NewMigrator(nil, restoreEngine)

	targetPath := filepath.Join(t.TempDir(), "restored.db")
	res, err := migrator.RestoreToTarget(context.Background(), meta, Target{
		DatabaseType: database.DatabaseTypeSQLite,
		DSNorPath:    targetPath,
	}, nil)
	if err != nil {
		t.Fatalf("RestoreToTarget: %v", err)
	}
	if res.RestoreStatus != database.RestoreStatusSuccess {
		t.Fatalf("expected restore success, got %s", res.RestoreStatus)
	}

	names := readNames(t, targetPath)
	want := []string{"alice", "bob"}
	if len(names) != len(want) {
		t.Fatalf("restored rows = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, names[i], want[i])
		}
	}
}

// TestMigrateArtifact_ChecksumMismatch ensures a mismatch between the migrated
// bytes and the recorded checksum is reported as an error rather than success.
func TestMigrateArtifact_ChecksumMismatch(t *testing.T) {
	meta, src := makeBackup(t)

	// Corrupt the expected checksum so the honest destination bytes no longer
	// match what the metadata claims.
	tampered := *meta
	tampered.Checksum = "0000000000000000000000000000000000000000000000000000000000000000"

	dst := newLocalProvider(t, filepath.Join(t.TempDir(), "dst-store"))
	_, err := NewMigrator(src, nil).MigrateArtifact(context.Background(), &tampered, dst, nil)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

// TestMigrateArtifact_NoSource ensures artifact migration requires a source.
func TestMigrateArtifact_NoSource(t *testing.T) {
	dst := newLocalProvider(t, filepath.Join(t.TempDir(), "dst"))
	meta := &models.BackupMetadata{StorageLocation: "local://backups/x/x.sql", Checksum: "abc"}
	_, err := NewMigrator(nil, nil).MigrateArtifact(context.Background(), meta, dst, nil)
	if !errors.Is(err, ErrNoSource) {
		t.Fatalf("expected ErrNoSource, got %v", err)
	}
}

// TestMigrateArtifact_NoStorageLocation ensures a local-only backup with no
// remote reference cannot be migrated between providers.
func TestMigrateArtifact_NoStorageLocation(t *testing.T) {
	src := newLocalProvider(t, filepath.Join(t.TempDir(), "src"))
	dst := newLocalProvider(t, filepath.Join(t.TempDir(), "dst"))
	meta := &models.BackupMetadata{StorageLocation: "/var/tmp/local.sql", Checksum: "abc"}
	_, err := NewMigrator(src, nil).MigrateArtifact(context.Background(), meta, dst, nil)
	if !errors.Is(err, ErrNoStorageLocation) {
		t.Fatalf("expected ErrNoStorageLocation, got %v", err)
	}
}

// TestMigrateArtifact_NoChecksum ensures migration refuses to run when there is
// no recorded checksum to verify the destination against.
func TestMigrateArtifact_NoChecksum(t *testing.T) {
	src := newLocalProvider(t, filepath.Join(t.TempDir(), "src"))
	dst := newLocalProvider(t, filepath.Join(t.TempDir(), "dst"))
	meta := &models.BackupMetadata{StorageLocation: "local://backups/x/x.sql"}
	_, err := NewMigrator(src, nil).MigrateArtifact(context.Background(), meta, dst, nil)
	if !errors.Is(err, ErrNoChecksum) {
		t.Fatalf("expected ErrNoChecksum, got %v", err)
	}
}

// TestRestoreToTarget_NoRestorer ensures target restore requires a restorer.
func TestRestoreToTarget_NoRestorer(t *testing.T) {
	meta := &models.BackupMetadata{ID: "x", DatabaseType: database.DatabaseTypeSQLite}
	_, err := NewMigrator(nil, nil).RestoreToTarget(context.Background(), meta, Target{
		DSNorPath: "/tmp/x.db",
	}, nil)
	if !errors.Is(err, ErrNoRestorer) {
		t.Fatalf("expected ErrNoRestorer, got %v", err)
	}
}

// TestRestoreToTarget_DatabaseTypeMismatch ensures a target database type that
// differs from the backup's is rejected.
func TestRestoreToTarget_DatabaseTypeMismatch(t *testing.T) {
	restoreEngine := restore.NewEngine(&restore.Config{TempDirectory: t.TempDir()})
	meta := &models.BackupMetadata{ID: "x", DatabaseType: database.DatabaseTypeSQLite}
	_, err := NewMigrator(nil, restoreEngine).RestoreToTarget(context.Background(), meta, Target{
		DatabaseType: database.DatabaseTypePostgreSQL,
		DSNorPath:    "host=example",
	}, nil)
	if !errors.Is(err, ErrDatabaseTypeMismatch) {
		t.Fatalf("expected ErrDatabaseTypeMismatch, got %v", err)
	}
}

// TestChecksumRemote verifies the destination checksum helper hashes streamed
// bytes correctly.
func TestChecksumRemote(t *testing.T) {
	dir := t.TempDir()
	provider := newLocalProvider(t, dir)
	content := []byte("hello-migration-bytes")
	if err := provider.UploadStream(context.Background(), bytes.NewReader(content), "obj/data.bin", nil); err != nil {
		t.Fatalf("upload: %v", err)
	}
	got, err := checksumRemote(context.Background(), provider, "obj/data.bin")
	if err != nil {
		t.Fatalf("checksumRemote: %v", err)
	}
	want := sha256.Sum256(content)
	if got != hex.EncodeToString(want[:]) {
		t.Errorf("checksum = %s, want %s", got, hex.EncodeToString(want[:]))
	}
}
