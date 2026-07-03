// Package queryable provides queryable / mountable backups: it "mounts" a
// backup artifact into a temporary read-only workspace and runs read-only SQL
// queries against it without performing a full production restore.
package queryable

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Register the go-sqlite3 SQL driver used to open mounted SQLite artifacts.
	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
)

const (
	// sqliteDriver is the database/sql driver name for SQLite.
	sqliteDriver = "sqlite3"
	// artifactFileName is the file name the backup artifact is materialized to
	// inside a mount's workspace.
	artifactFileName = "artifact.db"
)

// ErrUnsupportedDatabaseType indicates the backup's database type cannot be
// mounted for querying (only SQLite is currently supported).
var ErrUnsupportedDatabaseType = errors.New("queryable: unsupported database type for mounting")

// ErrEmptyQuery indicates an empty query string was supplied.
var ErrEmptyQuery = errors.New("queryable: empty query")

// ErrNotReadOnly indicates a query was rejected because it is not a read-only
// statement; a mount must never be able to mutate the underlying backup.
var ErrNotReadOnly = errors.New("queryable: only read-only statements (SELECT/PRAGMA/WITH) are allowed")

// ErrNotMounted indicates the mount has already been unmounted (or was never
// successfully opened) and can no longer be queried.
var ErrNotMounted = errors.New("queryable: mount is not open")

// Materializer materializes a backup artifact onto the local filesystem. It is
// satisfied by *restore.Engine via its DownloadBackup method.
type Materializer interface {
	// DownloadBackup writes the backup artifact for meta to dst on the local
	// filesystem, verifying it against the recorded checksum.
	DownloadBackup(ctx context.Context, meta *models.BackupMetadata, dst string) error
}

// Mounter mounts backups for read-only querying using a Materializer to fetch
// the underlying artifact.
type Mounter struct {
	materializer Materializer
}

// NewMounter creates a Mounter backed by the given Materializer.
func NewMounter(materializer Materializer) *Mounter {
	return &Mounter{materializer: materializer}
}

// MountOptions configures a Mount operation.
type MountOptions struct {
	// TempDirectory is the base directory under which the mount's private
	// workspace is created. When empty, the system temporary directory is used.
	TempDirectory string
}

// Mount is an opened, read-only view of a backup artifact. It owns a temporary
// workspace and an open read-only database handle; call Unmount to release them.
type Mount struct {
	db      *sql.DB
	workDir string
	// artifactPath is the on-disk path of the materialized backup artifact.
	artifactPath string
}

// QueryResult holds the columns and rows returned by a read-only query.
type QueryResult struct {
	// Columns are the result column names in order.
	Columns []string
	// Rows are the result rows; each row has one value per column.
	Rows [][]any
	// RowCount is the number of rows returned.
	RowCount int
}

// Mount materializes the backup artifact read-only into a private temporary
// workspace and opens it for querying. For SQLite the artifact is opened with
// database/sql in read-only mode so a mount can never mutate the backup.
func (m *Mounter) Mount(ctx context.Context, meta *models.BackupMetadata, opts *MountOptions) (*Mount, error) {
	if meta == nil {
		return nil, errors.New("queryable: backup metadata is nil")
	}
	if meta.DatabaseType != database.DatabaseTypeSQLite {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDatabaseType, meta.DatabaseType)
	}

	base := ""
	if opts != nil {
		base = opts.TempDirectory
	}
	if base != "" {
		if err := os.MkdirAll(base, 0o700); err != nil {
			return nil, fmt.Errorf("queryable: create temp base: %w", err)
		}
	}

	workDir, err := os.MkdirTemp(base, "mount-*")
	if err != nil {
		return nil, fmt.Errorf("queryable: create workspace: %w", err)
	}

	artifactPath := filepath.Join(workDir, artifactFileName)
	if dlErr := m.materializer.DownloadBackup(ctx, meta, artifactPath); dlErr != nil {
		cleanupDir(workDir)
		return nil, fmt.Errorf("queryable: materialize artifact: %w", dlErr)
	}

	db, err := openReadOnlySQLite(ctx, artifactPath)
	if err != nil {
		cleanupDir(workDir)
		return nil, err
	}

	return &Mount{db: db, workDir: workDir, artifactPath: artifactPath}, nil
}

// openReadOnlySQLite opens the SQLite artifact at path in read-only mode and
// verifies it is a usable database.
func openReadOnlySQLite(ctx context.Context, path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&_query_only=true", path)
	db, err := sql.Open(sqliteDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("queryable: open read-only sqlite: %w", err)
	}
	// A read-only SQLite database supports a single connection safely.
	db.SetMaxOpenConns(1)
	if pingErr := db.PingContext(ctx); pingErr != nil {
		_ = db.Close() //nolint:errcheck // best-effort close on a failed open
		return nil, fmt.Errorf("queryable: verify sqlite artifact: %w", pingErr)
	}
	return db, nil
}

// Query runs a read-only SELECT (or PRAGMA/WITH) statement and returns its
// result. Non-read-only statements are rejected with ErrNotReadOnly so a mount
// can never mutate the underlying backup.
func (mt *Mount) Query(ctx context.Context, query string, args ...any) (*QueryResult, error) {
	if mt.db == nil {
		return nil, ErrNotMounted
	}
	if err := ensureReadOnly(query); err != nil {
		return nil, err
	}

	rows, err := mt.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("queryable: query: %w", err)
	}
	defer rows.Close()

	cols, data, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	return &QueryResult{Columns: cols, Rows: data, RowCount: len(data)}, nil
}

// Tables lists the user tables in the mounted backup.
func (mt *Mount) Tables(ctx context.Context) ([]string, error) {
	if mt.db == nil {
		return nil, ErrNotMounted
	}

	const q = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	rows, err := mt.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("queryable: list tables: %w", err)
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("queryable: scan table name: %w", scanErr)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryable: iterate tables: %w", err)
	}
	return tables, nil
}

// Unmount closes the database handle and removes the mount's workspace. It is
// safe to call more than once.
func (mt *Mount) Unmount() error {
	var closeErr error
	if mt.db != nil {
		closeErr = mt.db.Close()
		mt.db = nil
	}
	if mt.workDir != "" {
		if rmErr := os.RemoveAll(mt.workDir); rmErr != nil && closeErr == nil {
			closeErr = rmErr
		}
		mt.workDir = ""
	}
	if closeErr != nil {
		return fmt.Errorf("queryable: unmount: %w", closeErr)
	}
	return nil
}

// ensureReadOnly returns nil only when query is a read-only statement, i.e. it
// begins (after trimming) with SELECT, PRAGMA, or WITH.
func ensureReadOnly(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ErrEmptyQuery
	}

	upper := strings.ToUpper(trimmed)
	for _, prefix := range []string{"SELECT", "PRAGMA", "WITH"} {
		if hasWordPrefix(upper, prefix) {
			return nil
		}
	}
	return ErrNotReadOnly
}

// hasWordPrefix reports whether s starts with word followed by end-of-string or
// a non-identifier byte, so "SELECTED" does not match the "SELECT" keyword.
func hasWordPrefix(s, word string) bool {
	if !strings.HasPrefix(s, word) {
		return false
	}
	if len(s) == len(word) {
		return true
	}
	next := s[len(word)]
	isIdent := next == '_' ||
		(next >= 'A' && next <= 'Z') ||
		(next >= '0' && next <= '9')
	return !isIdent
}

// scanRows reads every row from rows, returning the column names and the row
// values. Byte-slice values are converted to strings for readability.
func scanRows(rows *sql.Rows) (cols []string, out [][]any, err error) {
	cols, err = rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("queryable: columns: %w", err)
	}

	out = make([][]any, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		pointers := make([]any, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}
		if scanErr := rows.Scan(pointers...); scanErr != nil {
			return nil, nil, fmt.Errorf("queryable: scan row: %w", scanErr)
		}

		row := make([]any, len(cols))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		out = append(out, row)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("queryable: iterate rows: %w", err)
	}
	return cols, out, nil
}

// cleanupDir removes dir best-effort, ignoring any error (used on failure
// paths where the original error is what matters).
func cleanupDir(dir string) {
	_ = os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup on an error path
}
