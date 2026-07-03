// Package instant provides instant recovery / run-from-backup: it materializes
// a backup artifact locally and exposes it as an immediately runnable database,
// measuring the time-to-ready.
package instant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	// Register the database/sql sqlite3 driver used to open the materialized DB.
	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/database"

	// Register the platform-level SQLite driver so backup/restore engines that
	// rely on it resolve correctly when this package is used end-to-end.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	"github.com/sanskarpan/db-backup/internal/models"
)

// sqlDriverName is the database/sql driver name for SQLite.
const sqlDriverName = "sqlite3"

// materializedFileName is the fixed name of the runnable artifact inside the
// isolated working directory.
const materializedFileName = "instant.db"

var (
	// ErrNilMetadata is returned when PrepareInstant is called without backup
	// metadata to recover from.
	ErrNilMetadata = errors.New("instant: backup metadata is required")

	// ErrUnsupportedDatabaseType is returned when instant recovery is requested
	// for a database type that cannot be run directly from its artifact.
	ErrUnsupportedDatabaseType = errors.New("instant: unsupported database type for instant recovery")

	// ErrNotUsable is returned when the materialized artifact fails its
	// post-materialization usability check and is not safe to run.
	ErrNotUsable = errors.New("instant: materialized database is not usable")

	// ErrHandleClosed is returned when an operation is attempted on a handle
	// that has already been closed.
	ErrHandleClosed = errors.New("instant: handle is closed")
)

// Materializer materializes a backup artifact onto local disk. It is satisfied
// by *restore.Engine via its DownloadBackup method.
type Materializer interface {
	// DownloadBackup writes the backup artifact described by meta to
	// destinationPath, verifying it against the recorded checksum when present.
	DownloadBackup(ctx context.Context, meta *models.BackupMetadata, destinationPath string) error
}

// Recoverer prepares instant, run-from-backup databases from backups.
type Recoverer struct {
	materializer Materializer
}

// NewRecoverer builds a Recoverer that materializes artifacts using the given
// Materializer (for example a *restore.Engine).
func NewRecoverer(materializer Materializer) *Recoverer {
	return &Recoverer{materializer: materializer}
}

// Options controls how an instant database is prepared.
type Options struct {
	// WorkDir is the isolated directory the artifact is materialized into. When
	// empty a fresh temporary directory is created and owned by the handle
	// (removed on Cleanup).
	WorkDir string

	// RemoveWorkDir, when true, forces Cleanup to remove WorkDir even when the
	// caller supplied it. A handle-owned temporary directory is always removed.
	RemoveWorkDir bool
}

// Handle is a ready-to-use database materialized from a backup.
type Handle struct {
	// Path is the absolute path to the runnable database file on disk.
	Path string
	// DSN is the database/sql data source name for opening the database.
	DSN string
	// ReadyAt is the instant the database became usable.
	ReadyAt time.Time
	// Elapsed is the wall-clock time it took to make the database ready.
	Elapsed time.Duration
	// DatabaseType is the type of the materialized database.
	DatabaseType database.DatabaseType

	workDir       string
	removeWorkDir bool

	mu     sync.Mutex
	db     *sql.DB
	closed bool
}

// PrepareInstant materializes the backup artifact into an isolated working
// directory and returns a handle to a database that is immediately runnable.
// For SQLite the materialized file is the runnable database; it is opened
// read-write and verified with PRAGMA quick_check before the handle is
// returned.
func (r *Recoverer) PrepareInstant(
	ctx context.Context,
	meta *models.BackupMetadata,
	opts *Options,
) (*Handle, error) {
	start := time.Now()

	if meta == nil {
		return nil, ErrNilMetadata
	}
	if meta.DatabaseType != database.DatabaseTypeSQLite {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDatabaseType, meta.DatabaseType)
	}
	if opts == nil {
		opts = &Options{}
	}

	workDir, ownWorkDir, err := prepareWorkDir(opts.WorkDir)
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(workDir, materializedFileName)
	if err = r.materializer.DownloadBackup(ctx, meta, dbPath); err != nil {
		return nil, errors.Join(
			fmt.Errorf("instant: materialize backup: %w", err),
			cleanupWorkDir(ownWorkDir, workDir),
		)
	}

	if err = quickCheck(ctx, dbPath); err != nil {
		return nil, errors.Join(err, cleanupWorkDir(ownWorkDir, workDir))
	}

	now := time.Now()
	return &Handle{
		Path:          dbPath,
		DSN:           dbPath,
		ReadyAt:       now,
		Elapsed:       now.Sub(start),
		DatabaseType:  meta.DatabaseType,
		workDir:       workDir,
		removeWorkDir: ownWorkDir || opts.RemoveWorkDir,
	}, nil
}

// prepareWorkDir resolves the working directory, creating a temporary one when
// none was supplied. It reports whether the directory is owned by the handle.
func prepareWorkDir(requested string) (dir string, owned bool, err error) {
	if requested == "" {
		created, mkErr := os.MkdirTemp("", "instant-recovery-*")
		if mkErr != nil {
			return "", false, fmt.Errorf("instant: create work dir: %w", mkErr)
		}
		return created, true, nil
	}
	if mkErr := os.MkdirAll(requested, 0o700); mkErr != nil {
		return "", false, fmt.Errorf("instant: create work dir: %w", mkErr)
	}
	return requested, false, nil
}

// cleanupWorkDir removes an owned working directory. It returns any removal
// error so callers on a failing path can surface it alongside the primary
// error.
func cleanupWorkDir(owned bool, dir string) error {
	if !owned || dir == "" {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("instant: remove work dir: %w", err)
	}
	return nil
}

// quickCheck opens the SQLite database at path read-write and runs PRAGMA
// quick_check to confirm it is usable.
func quickCheck(ctx context.Context, path string) (err error) {
	db, err := sql.Open(sqlDriverName, path)
	if err != nil {
		return fmt.Errorf("instant: open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("instant: close probe database: %w", closeErr)
		}
	}()

	var result string
	if err = db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return fmt.Errorf("%w: quick_check failed: %w", ErrNotUsable, err)
	}
	if result != "ok" {
		return fmt.Errorf("%w: %s", ErrNotUsable, result)
	}
	return nil
}

// Open returns a read-write *sql.DB for the materialized database. Repeated
// calls return the same pooled handle, which Close and Cleanup release.
func (h *Handle) Open() (*sql.DB, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHandleClosed
	}
	if h.db != nil {
		return h.db, nil
	}

	db, err := sql.Open(sqlDriverName, h.DSN)
	if err != nil {
		return nil, fmt.Errorf("instant: open database: %w", err)
	}
	h.db = db
	return db, nil
}

// Close releases the pooled database connection without removing the working
// directory.
func (h *Handle) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeLocked()
}

// closeLocked closes the pooled connection. The caller must hold h.mu.
func (h *Handle) closeLocked() error {
	h.closed = true
	if h.db == nil {
		return nil
	}
	err := h.db.Close()
	h.db = nil
	if err != nil {
		return fmt.Errorf("instant: close database: %w", err)
	}
	return nil
}

// Cleanup releases the database connection and, when the working directory is
// owned by the handle (or removal was requested), removes it from disk.
func (h *Handle) Cleanup() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	err := h.closeLocked()
	if h.removeWorkDir && h.workDir != "" {
		if rerr := os.RemoveAll(h.workDir); rerr != nil && err == nil {
			err = fmt.Errorf("instant: remove work dir: %w", rerr)
		}
	}
	return err
}
