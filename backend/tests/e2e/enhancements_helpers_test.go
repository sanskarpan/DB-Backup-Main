package e2e

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
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"

	// Register the platform's real SQLite driver so backup/restore engines can
	// resolve it via database.CreateDriver.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	// Register the go-sqlite3 SQL driver used to seed and verify test databases.
	_ "github.com/mattn/go-sqlite3"
)

// sha256Hex returns the hex-encoded SHA-256 checksum of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// seedUsersDB creates a real SQLite database at path containing a single
// "users" table populated with the returned names, in id order.
func seedUsersDB(t *testing.T, path string) []string {
	t.Helper()
	names := []string{"alice", "bob", "carol"}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);`); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	for _, name := range names {
		if _, err := db.Exec(`INSERT INTO users (name) VALUES (?);`, name); err != nil {
			t.Fatalf("insert user %q: %v", name, err)
		}
	}
	return names
}

// createTable executes a CREATE TABLE + INSERT pair against the SQLite database
// at path, used to add a second table for selective-restore coverage.
func createTable(t *testing.T, path, ddl, insert string) {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("ddl %q: %v", ddl, err)
	}
	if _, err := db.Exec(insert); err != nil {
		t.Fatalf("insert %q: %v", insert, err)
	}
}

// queryColumn opens the SQLite database at path and returns the string values
// of the given query's first column, in row order.
func queryColumn(t *testing.T, path, query string) []string {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	return out
}

// tableExists reports whether the named table is present in the SQLite database.
func tableExists(t *testing.T, path, table string) bool {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	var name string
	err = db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?;`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("sqlite_master lookup: %v", err)
	}
	return name == table
}

// newLocalStorage builds a real filesystem-backed storage provider rooted at dir.
func newLocalStorage(t *testing.T, dir string) storage.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&storage.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// readObject downloads the object at remotePath from the provider and returns
// its raw bytes.
func readObject(t *testing.T, p storage.Provider, remotePath string) []byte {
	t.Helper()
	rc, err := p.DownloadStream(context.Background(), remotePath)
	if err != nil {
		t.Fatalf("download stream %q: %v", remotePath, err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read object %q: %v", remotePath, err)
	}
	return data
}

// fakeScanner is a restore.ArtifactScanner test double returning a preconfigured
// report and/or error and recording how many times it was invoked.
type fakeScanner struct {
	report *ransomware.ThreatReport
	err    error
	calls  int
}

// ScanFile records the call and returns the configured report/error.
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

// retentionRecord captures a SetRetention call on the fake immutable provider.
type retentionRecord struct {
	until time.Time
	mode  string
}

// immutableFakeProvider is an in-memory storage.Provider that ALSO satisfies
// storage.ImmutableProvider. It records object-lock operations so tests can
// assert the backup engine drove them correctly, standing in for a real
// object-lock capable backend (e.g. S3/Wasabi) without external dependencies.
type immutableFakeProvider struct {
	objects    map[string][]byte
	retentions map[string]retentionRecord
	legalHolds map[string]bool

	setRetentionCalls int
	setLegalHoldCalls int
}

// newImmutableFakeProvider builds an empty in-memory immutable provider.
func newImmutableFakeProvider() *immutableFakeProvider {
	return &immutableFakeProvider{
		objects:    make(map[string][]byte),
		retentions: make(map[string]retentionRecord),
		legalHolds: make(map[string]bool),
	}
}

// Upload stores the contents of localPath under remotePath.
func (p *immutableFakeProvider) Upload(_ context.Context, localPath, remotePath string, _ *storage.UploadOptions) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	p.objects[remotePath] = data
	return nil
}

// UploadStream stores everything read from reader under remotePath.
func (p *immutableFakeProvider) UploadStream(_ context.Context, reader io.Reader, remotePath string, _ *storage.UploadOptions) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	p.objects[remotePath] = data
	return nil
}

// Download writes the stored object at remotePath to localPath.
func (p *immutableFakeProvider) Download(_ context.Context, remotePath, localPath string) error {
	data, ok := p.objects[remotePath]
	if !ok {
		return errors.New("object not found")
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0o600)
}

// DownloadStream returns a reader over the stored object at remotePath.
func (p *immutableFakeProvider) DownloadStream(_ context.Context, remotePath string) (io.ReadCloser, error) {
	data, ok := p.objects[remotePath]
	if !ok {
		return nil, errors.New("object not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete removes the stored object at remotePath.
func (p *immutableFakeProvider) Delete(_ context.Context, remotePath string) error {
	delete(p.objects, remotePath)
	return nil
}

// Exists reports whether an object is stored at remotePath.
func (p *immutableFakeProvider) Exists(_ context.Context, remotePath string) (bool, error) {
	_, ok := p.objects[remotePath]
	return ok, nil
}

// GetMetadata returns size and checksum metadata for the stored object.
func (p *immutableFakeProvider) GetMetadata(_ context.Context, remotePath string) (*storage.FileMetadata, error) {
	data, ok := p.objects[remotePath]
	if !ok {
		return nil, errors.New("object not found")
	}
	return &storage.FileMetadata{
		Path:     remotePath,
		Size:     int64(len(data)),
		Checksum: sha256Hex(data),
	}, nil
}

// List returns metadata for stored objects whose path starts with prefix.
func (p *immutableFakeProvider) List(_ context.Context, prefix string) ([]*storage.FileMetadata, error) {
	var out []*storage.FileMetadata
	for path, data := range p.objects {
		if prefix == "" || (len(path) >= len(prefix) && path[:len(prefix)] == prefix) {
			out = append(out, &storage.FileMetadata{Path: path, Size: int64(len(data))})
		}
	}
	return out, nil
}

// GetType identifies this as an object-lock capable (S3-style) provider.
func (p *immutableFakeProvider) GetType() storage.ProviderType { return storage.ProviderTypeS3 }

// ValidateConfig always succeeds for the in-memory provider.
func (p *immutableFakeProvider) ValidateConfig() error { return nil }

// SetRetention records an object-lock retention for remotePath.
func (p *immutableFakeProvider) SetRetention(_ context.Context, remotePath string, until time.Time, mode string) error {
	p.setRetentionCalls++
	p.retentions[remotePath] = retentionRecord{until: until, mode: mode}
	return nil
}

// GetRetention returns the recorded retention, or ErrNoRetention when none.
func (p *immutableFakeProvider) GetRetention(_ context.Context, remotePath string) (time.Time, string, error) {
	rec, ok := p.retentions[remotePath]
	if !ok {
		return time.Time{}, "", storage.ErrNoRetention
	}
	return rec.until, rec.mode, nil
}

// SetLegalHold records a legal-hold change for remotePath.
func (p *immutableFakeProvider) SetLegalHold(_ context.Context, remotePath string, on bool) error {
	p.setLegalHoldCalls++
	p.legalHolds[remotePath] = on
	return nil
}

// GetLegalHold reports the recorded legal-hold state for remotePath.
func (p *immutableFakeProvider) GetLegalHold(_ context.Context, remotePath string) (bool, error) {
	return p.legalHolds[remotePath], nil
}

// echoDriver is a minimal database.Driver that echoes the requested tables back
// through RestoreResult.RestoredTables, letting a selective-restore test assert
// the engine passes the table subset through to the driver and reports it.
type echoDriver struct {
	gotTables []string
}

// Connect is a no-op for the echo driver.
func (d *echoDriver) Connect(context.Context, *database.ConnectionConfig) error { return nil }

// Disconnect is a no-op for the echo driver.
func (d *echoDriver) Disconnect() error { return nil }

// Ping is a no-op for the echo driver.
func (d *echoDriver) Ping(context.Context) error { return nil }

// errUnusedDriverOp is returned by echoDriver methods that are not exercised by
// the restore path, so callers get a sentinel error instead of a nil/nil result.
var errUnusedDriverOp = errors.New("echoDriver: operation not supported")

// Backup is unused by the restore path.
func (d *echoDriver) Backup(context.Context, *database.BackupOptions) (*database.BackupResult, error) {
	return nil, errUnusedDriverOp
}

// StreamBackup is unused by the restore path.
func (d *echoDriver) StreamBackup(context.Context, *database.BackupOptions, io.Writer) error {
	return nil
}

// GetBackupSize is unused by the restore path.
func (d *echoDriver) GetBackupSize(context.Context, *database.BackupOptions) (int64, error) {
	return 0, nil
}

// Restore echoes the requested tables back as the restored tables.
func (d *echoDriver) Restore(_ context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	d.gotTables = opts.Tables
	restored := opts.Tables
	if len(restored) == 0 {
		restored = []string{"all"}
	}
	return &database.RestoreResult{RestoredTables: restored, RowsRestored: int64(len(restored))}, nil
}

// StreamRestore is unused by the restore path.
func (d *echoDriver) StreamRestore(context.Context, *database.RestoreOptions, io.Reader) error {
	return nil
}

// ValidateRestore is a no-op for the echo driver.
func (d *echoDriver) ValidateRestore(context.Context, *database.RestoreOptions) error { return nil }

// GetDatabases returns the single target database expected by verifyRestore.
func (d *echoDriver) GetDatabases(context.Context) ([]string, error) {
	return []string{"targetdb"}, nil
}

// GetTables is unused by the restore path.
func (d *echoDriver) GetTables(context.Context, string) ([]string, error) { return nil, nil }

// GetTableSize is unused by the restore path.
func (d *echoDriver) GetTableSize(context.Context, string, string) (int64, error) { return 0, nil }

// GetDatabaseSize is unused by the restore path.
func (d *echoDriver) GetDatabaseSize(context.Context) (int64, error) { return 0, nil }

// GetVersion returns a static version string.
func (d *echoDriver) GetVersion(context.Context) (string, error) { return "1.0", nil }

// GetType returns the echo driver's synthetic database type.
func (d *echoDriver) GetType() database.DatabaseType { return "e2eecho" }

// SupportsIncremental reports no incremental support.
func (d *echoDriver) SupportsIncremental() bool { return false }

// SupportsPITR reports no point-in-time recovery support.
func (d *echoDriver) SupportsPITR() bool { return false }
