// Package sqlite provides SQLite database driver implementation
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
	"github.com/sanskarpan/db-backup/pkg/validation"
)

// SQLiteDriver implements the database.Driver interface for SQLite.
type SQLiteDriver struct {
	db     *sql.DB
	config *database.ConnectionConfig
}

func init() {
	database.RegisterDriver(database.DatabaseTypeSQLite, func() database.Driver {
		return NewSQLiteDriver()
	})
}

// NewSQLiteDriver creates a new SQLite driver instance.
func NewSQLiteDriver() *SQLiteDriver {
	return &SQLiteDriver{}
}

// Connect establishes a connection to the SQLite database.
func (d *SQLiteDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// For SQLite, the database field contains the file path
	dbPath := config.Database
	if dbPath == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "database file path is required")
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Set connection pool settings (SQLite only supports single writer)
	db.SetMaxOpenConns(1)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return pkgErrors.ErrDatabaseConnection(err)
	}

	d.db = db
	d.config = config
	return nil
}

// Disconnect closes the database connection.
func (d *SQLiteDriver) Disconnect() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping tests the database connection.
func (d *SQLiteDriver) Ping(ctx context.Context) error {
	if d.db == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}
	return d.db.PingContext(ctx)
}

// Backup creates a backup of the SQLite database.
func (d *SQLiteDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Validate and sanitize output path to prevent SQL injection
	// Get the base directory from the output path
	baseDir := filepath.Dir(filepath.Dir(opts.OutputPath))
	sanitizedPath, err := validation.SanitizePath(opts.OutputPath, baseDir)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = fmt.Errorf("invalid output path: %w", err)
		return result, pkgErrors.ErrDatabaseBackup(result.Error)
	}

	// SQLite online backup using VACUUM INTO (SQLite 3.27.0+)
	// or file copy for compatibility

	// Try VACUUM INTO first (preferred method)
	// Note: VACUUM INTO doesn't support parameterized queries, but we've validated the path
	query := fmt.Sprintf("VACUUM INTO '%s'", sanitizedPath)
	_, err = d.db.ExecContext(ctx, query)
	if err != nil {
		// Fallback to file copy method
		if err := d.fileCopyBackup(ctx, sanitizedPath); err != nil {
			result.Status = database.BackupStatusFailed
			result.Error = err
			return result, pkgErrors.ErrDatabaseBackup(err)
		}
	}

	// Get file info
	fileInfo, err := os.Stat(sanitizedPath)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, err
	}

	// Get database version
	version, _ := d.GetVersion(ctx)

	// Get table information
	tables, _ := d.getTableInfo(ctx)

	// Complete result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Size = fileInfo.Size()
	result.DatabaseVersion = version
	result.Tables = tables
	result.Status = database.BackupStatusSuccess

	return result, nil
}

// StreamBackup streams a backup to the provided writer.
func (d *SQLiteDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	// SQLite doesn't natively support streaming
	// We would need to backup to a temp file first, then stream it
	return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "SQLite streaming backup not implemented - use file-based backup")
}

// GetBackupSize estimates the size of a backup.
func (d *SQLiteDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	// Get database file size
	fileInfo, err := os.Stat(d.config.Database)
	if err != nil {
		return 0, err
	}

	return fileInfo.Size(), nil
}

// Restore restores a SQLite database from backup.
func (d *SQLiteDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Validate backup file exists
	if _, err := os.Stat(opts.SourceBackup); os.IsNotExist(err) {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseRestore(err).WithMetadata("backup_file", opts.SourceBackup)
	}

	// Close existing connection
	if err := d.Disconnect(); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	// Copy backup file to target location
	targetPath := opts.Database
	if targetPath == "" {
		targetPath = d.config.Database
	}

	if err := copyFile(opts.SourceBackup, targetPath); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Reopen connection
	if err := d.Connect(ctx, d.config); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	// Verify integrity
	var integrityCheck string
	if err := d.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrityCheck); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	if integrityCheck != "ok" {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("integrity check failed: %s", integrityCheck)
		return result, pkgErrors.ErrDatabaseRestore(result.Error)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = database.RestoreStatusSuccess

	return result, nil
}

// StreamRestore restores from a reader.
func (d *SQLiteDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	// SQLite doesn't support streaming restore
	// We would need to write to a temp file first
	return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "SQLite streaming restore not implemented - use file-based restore")
}

// ValidateRestore validates that a restore can be performed.
func (d *SQLiteDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	// Check if backup file exists
	if _, err := os.Stat(opts.SourceBackup); os.IsNotExist(err) {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup file not found: %s", opts.SourceBackup))
	}

	return nil
}

// GetDatabases returns list of databases (for SQLite, just the current database).
func (d *SQLiteDriver) GetDatabases(ctx context.Context) ([]string, error) {
	// SQLite has only one database per file
	return []string{d.config.Database}, nil
}

// GetTables returns list of tables in the database.
func (d *SQLiteDriver) GetTables(ctx context.Context, database string) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// GetTableSize returns the size of a table (approximate).
func (d *SQLiteDriver) GetTableSize(ctx context.Context, database, table string) (int64, error) {
	// Validate table name to prevent SQL injection
	if err := validation.ValidateTableName(table); err != nil {
		return 0, fmt.Errorf("invalid table name %q: %w", table, err)
	}

	// SQLite doesn't have a direct way to get table size
	// We can estimate by counting rows and page size
	var rowCount int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := d.db.QueryRowContext(ctx, query).Scan(&rowCount); err != nil {
		return 0, err
	}

	// Get page size
	var pageSize int64
	if err := d.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, err
	}

	// Very rough estimate
	return rowCount * pageSize, nil
}

// GetVersion returns the SQLite version.
func (d *SQLiteDriver) GetVersion(ctx context.Context) (string, error) {
	var version string
	err := d.db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	return version, err
}

// GetType returns the database type.
func (d *SQLiteDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeSQLite
}

// SupportsIncremental returns whether incremental backups are supported.
func (d *SQLiteDriver) SupportsIncremental() bool {
	return false // SQLite doesn't have built-in incremental backup
}

// SupportsPITR returns whether point-in-time recovery is supported.
func (d *SQLiteDriver) SupportsPITR() bool {
	return false // SQLite doesn't have built-in PITR
}

// fileCopyBackup performs a file-based backup.
func (d *SQLiteDriver) fileCopyBackup(ctx context.Context, destPath string) error {
	// Checkpoint WAL first
	if _, err := d.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		// WAL might not be enabled, continue anyway
	}

	// Lock database for consistent backup
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Copy file
	if err := copyFile(d.config.Database, destPath); err != nil {
		return err
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// getTableInfo retrieves information about tables.
func (d *SQLiteDriver) getTableInfo(ctx context.Context) ([]database.TableInfo, error) {
	tables, err := d.GetTables(ctx, d.config.Database)
	if err != nil {
		return nil, err
	}

	var tableInfos []database.TableInfo
	for _, table := range tables {
		// Validate table name to prevent SQL injection
		if err := validation.ValidateTableName(table); err != nil {
			continue // Skip invalid table names
		}

		var rowCount int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := d.db.QueryRowContext(ctx, query).Scan(&rowCount); err != nil {
			continue // Skip tables with errors
		}

		info := database.TableInfo{
			Name:     table,
			RowCount: rowCount,
		}
		tableInfos = append(tableInfos, info)
	}

	return tableInfos, nil
}

// GetDatabaseSize returns the total size of the SQLite database file.
func (d *SQLiteDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	// Get the database file size from disk
	info, err := os.Stat(d.config.Database)
	if err != nil {
		return 0, pkgErrors.ErrOperationFailed("failed to query database size: " + err.Error())
	}

	return info.Size(), nil
}
