// Package timescaledb provides TimescaleDB database driver implementation
package timescaledb

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sanskarpan/db-backup/internal/database"
	internalUtils "github.com/sanskarpan/db-backup/internal/utils"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// TimescaleDBDriver implements the database.Driver interface for TimescaleDB.
type TimescaleDBDriver struct {
	pool               *pgxpool.Pool
	config             *database.ConnectionConfig
	compressionManager *CompressionManager
	chunkManager       *ChunkManager
	version            string
}

func init() {
	database.RegisterDriver("timescaledb", func() database.Driver {
		return NewTimescaleDBDriver()
	})
}

// NewTimescaleDBDriver creates a new TimescaleDB driver instance.
func NewTimescaleDBDriver() *TimescaleDBDriver {
	return &TimescaleDBDriver{}
}

// Connect establishes a connection to TimescaleDB.
func (d *TimescaleDBDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Build connection string
	connStr := d.buildConnectionString(config)

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Set pool settings
	if config.MaxConnections > 0 {
		poolConfig.MaxConns = int32(config.MaxConnections)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Verify TimescaleDB extension is installed
	var extInstalled bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&extInstalled)
	if err != nil {
		pool.Close()
		return pkgErrors.ErrDatabaseConnection(err)
	}

	if !extInstalled {
		pool.Close()
		return pkgErrors.ErrDatabaseConnection(fmt.Errorf("timescaledb extension not installed"))
	}

	// Get TimescaleDB version
	err = pool.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'").Scan(&d.version)
	if err != nil {
		d.version = "unknown"
	}

	d.pool = pool
	d.config = config
	d.compressionManager = NewCompressionManager(d)
	d.chunkManager = NewChunkManager(d)

	return nil
}

// buildConnectionString builds PostgreSQL connection string.
func (d *TimescaleDBDriver) buildConnectionString(config *database.ConnectionConfig) string {
	if config.ConnectionString != "" {
		return config.ConnectionString
	}

	var parts []string

	if config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	}
	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
	}
	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}
	if config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", config.Database))
	}
	if config.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", config.SSLMode))
	} else {
		parts = append(parts, "sslmode=disable")
	}

	if config.ConnectionTimeout > 0 {
		parts = append(parts, fmt.Sprintf("connect_timeout=%d", int(config.ConnectionTimeout.Seconds())))
	}

	return strings.Join(parts, " ")
}

// Disconnect closes the database connection.
func (d *TimescaleDBDriver) Disconnect() error {
	if d.pool != nil {
		d.pool.Close()
	}
	return nil
}

// Ping tests the database connection.
func (d *TimescaleDBDriver) Ping(ctx context.Context) error {
	if d.pool == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}
	return d.pool.Ping(ctx)
}

// Backup creates a backup of the TimescaleDB database.
func (d *TimescaleDBDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Create backup directory
	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Backup hypertable metadata first
	if err := d.backupHypertableMetadata(ctx, backupDir); err != nil {
		result.Metadata["metadata_backup_error"] = err.Error()
	}

	// Backup compression policies
	if err := d.compressionManager.BackupCompressionPolicies(ctx, backupDir); err != nil {
		result.Metadata["compression_backup_error"] = err.Error()
	}

	// Backup continuous aggregates
	if err := d.backupContinuousAggregates(ctx, backupDir); err != nil {
		result.Metadata["caggs_backup_error"] = err.Error()
	}

	// Perform pg_dump backup
	backupFile := filepath.Join(backupDir, "timescaledb_dump.sql")
	if err := d.pgDump(ctx, backupFile, opts); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Get backup size
	info, err := os.Stat(backupFile)
	if err != nil {
		result.BackupSize = 0
	} else {
		result.BackupSize = info.Size()
	}

	// Add directory size for metadata
	dirSize, _ := internalUtils.GetDirectorySize(backupDir)
	result.BackupSize = dirSize

	result.BackupPath = backupDir
	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["timescaledb_version"] = d.version
	result.Metadata["backup_file"] = backupFile

	return result, nil
}

// pgDump performs a pg_dump backup.
func (d *TimescaleDBDriver) pgDump(ctx context.Context, outputFile string, opts *database.BackupOptions) error {
	args := []string{
		"-h", d.config.Host,
		"-p", fmt.Sprintf("%d", d.config.Port),
		"-U", d.config.Username,
		"-F", "c", // Custom format for better compression and parallel restore
		"-f", outputFile,
	}

	// Add specific database
	if opts.Database != "" {
		args = append(args, opts.Database)
	} else if d.config.Database != "" {
		args = append(args, d.config.Database)
	}

	// Add table filters if specified
	if len(opts.Tables) > 0 {
		for _, table := range opts.Tables {
			args = append(args, "-t", table)
		}
	}

	// Set password via environment variable
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", d.config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	return nil
}

// backupHypertableMetadata backs up hypertable metadata.
func (d *TimescaleDBDriver) backupHypertableMetadata(ctx context.Context, backupDir string) error {
	query := `
		SELECT
			ht.table_name,
			ht.schema_name,
			dim.column_name,
			dim.interval_length,
			ht.num_dimensions,
			ht.num_chunks
		FROM _timescaledb_catalog.hypertable ht
		LEFT JOIN _timescaledb_catalog.dimension dim ON ht.id = dim.hypertable_id
		WHERE dim.dimension_number = 1
	`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	metadataFile := filepath.Join(backupDir, "hypertables_metadata.sql")
	file, err := os.Create(metadataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("-- TimescaleDB Hypertable Metadata\n\n")

	for rows.Next() {
		var tableName, schemaName, columnName string
		var intervalLength, numDimensions, numChunks int64

		if err := rows.Scan(&tableName, &schemaName, &columnName, &intervalLength, &numDimensions, &numChunks); err != nil {
			continue
		}

		// Generate CREATE HYPERTABLE statement
		createStmt := fmt.Sprintf(
			"-- Hypertable: %s.%s (chunks: %d, dimensions: %d)\n"+
				"SELECT create_hypertable('%s.%s', '%s', chunk_time_interval => INTERVAL '%d microseconds');\n\n",
			schemaName, tableName, numChunks, numDimensions,
			schemaName, tableName, columnName, intervalLength,
		)
		file.WriteString(createStmt)
	}

	return rows.Err()
}

// backupContinuousAggregates backs up continuous aggregates metadata.
func (d *TimescaleDBDriver) backupContinuousAggregates(ctx context.Context, backupDir string) error {
	query := `
		SELECT
			user_view_schema,
			user_view_name,
			view_definition
		FROM timescaledb_information.continuous_aggregates
	`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	caggFile := filepath.Join(backupDir, "continuous_aggregates.sql")
	file, err := os.Create(caggFile)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("-- TimescaleDB Continuous Aggregates\n\n")

	for rows.Next() {
		var schema, name, definition string
		if err := rows.Scan(&schema, &name, &definition); err != nil {
			continue
		}

		file.WriteString(fmt.Sprintf("-- Continuous Aggregate: %s.%s\n", schema, name))
		file.WriteString(definition + ";\n\n")
	}

	return rows.Err()
}

// Restore restores TimescaleDB from a backup.
func (d *TimescaleDBDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	backupPath := opts.BackupPath
	if backupPath == "" {
		backupPath = opts.SourceBackup
	}

	// Find the dump file
	dumpFile := filepath.Join(backupPath, "timescaledb_dump.sql")
	if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("backup file not found: %s", dumpFile)
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(result.Error)
	}

	// Restore using pg_restore
	if err := d.pgRestore(ctx, dumpFile, opts); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Restore hypertable metadata
	metadataFile := filepath.Join(backupPath, "hypertables_metadata.sql")
	if _, err := os.Stat(metadataFile); err == nil {
		if err := d.restoreHypertableMetadata(ctx, metadataFile); err != nil {
			result.Metadata = map[string]interface{}{
				"metadata_restore_error": err.Error(),
			}
		}
	}

	// Restore compression policies
	if err := d.compressionManager.RestoreCompressionPolicies(ctx, backupPath); err != nil {
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["compression_restore_error"] = err.Error()
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()

	return result, nil
}

// pgRestore performs a pg_restore operation.
func (d *TimescaleDBDriver) pgRestore(ctx context.Context, inputFile string, opts *database.RestoreOptions) error {
	args := []string{
		"-h", d.config.Host,
		"-p", fmt.Sprintf("%d", d.config.Port),
		"-U", d.config.Username,
		"-d", d.config.Database,
	}

	if opts.DropExisting {
		args = append(args, "-c") // Clean (drop) database objects before recreating
	}

	args = append(args, inputFile)

	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", d.config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w, output: %s", err, string(output))
	}

	return nil
}

// restoreHypertableMetadata restores hypertable metadata.
func (d *TimescaleDBDriver) restoreHypertableMetadata(ctx context.Context, metadataFile string) error {
	// Read and execute the metadata SQL file
	content, err := os.ReadFile(metadataFile)
	if err != nil {
		return err
	}

	_, err = d.pool.Exec(ctx, string(content))
	return err
}

// GetDatabases returns the list of databases.
func (d *TimescaleDBDriver) GetDatabases(ctx context.Context) ([]string, error) {
	query := "SELECT datname FROM pg_database WHERE datistemplate = false"
	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbname string
		if err := rows.Scan(&dbname); err != nil {
			continue
		}
		databases = append(databases, dbname)
	}

	return databases, rows.Err()
}

// GetTables returns the list of tables (including hypertables).
func (d *TimescaleDBDriver) GetTables(ctx context.Context, database string) ([]string, error) {
	query := `
		SELECT table_schema || '.' || table_name as full_name
		FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		AND table_schema NOT IN ('pg_catalog', 'information_schema', '_timescaledb_catalog', '_timescaledb_config', '_timescaledb_cache', '_timescaledb_internal')
	`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// GetTableSize returns the size of a table.
func (d *TimescaleDBDriver) GetTableSize(ctx context.Context, database, table string) (int64, error) {
	var size int64
	query := fmt.Sprintf("SELECT pg_total_relation_size('%s')", pgx.Identifier{table}.Sanitize())
	err := d.pool.QueryRow(ctx, query).Scan(&size)
	return size, err
}

// GetDatabaseSize returns the total database size.
func (d *TimescaleDBDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	var size int64
	query := fmt.Sprintf("SELECT pg_database_size('%s')", d.config.Database)
	err := d.pool.QueryRow(ctx, query).Scan(&size)
	return size, err
}

// GetVersion returns the TimescaleDB version.
func (d *TimescaleDBDriver) GetVersion(ctx context.Context) (string, error) {
	var pgVersion string
	err := d.pool.QueryRow(ctx, "SELECT version()").Scan(&pgVersion)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("TimescaleDB %s on %s", d.version, pgVersion), nil
}

// StreamBackup streams backup data to a writer.
func (d *TimescaleDBDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for TimescaleDB")
}

// StreamRestore streams restore data from a reader.
func (d *TimescaleDBDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for TimescaleDB")
}

// GetBackupSize returns the estimated size of a backup.
func (d *TimescaleDBDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	return d.GetDatabaseSize(ctx)
}

// ValidateRestore validates restore options.
func (d *TimescaleDBDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" && opts.SourceBackup == "" {
		return fmt.Errorf("backup path is required")
	}

	backupPath := opts.BackupPath
	if backupPath == "" {
		backupPath = opts.SourceBackup
	}

	// Check if backup path exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup path does not exist: %s", backupPath)
	}

	return nil
}

// GetType returns the database type.
func (d *TimescaleDBDriver) GetType() database.DatabaseType {
	return "timescaledb"
}

// SupportsIncremental returns whether the driver supports incremental backups.
func (d *TimescaleDBDriver) SupportsIncremental() bool {
	// TimescaleDB can support incremental backups via chunk-based backups
	return true
}

// SupportsPITR returns whether the driver supports point-in-time recovery.
func (d *TimescaleDBDriver) SupportsPITR() bool {
	// PostgreSQL/TimescaleDB supports PITR via WAL archiving
	return true
}

// GetHypertables returns the list of hypertables.
func (d *TimescaleDBDriver) GetHypertables(ctx context.Context) ([]string, error) {
	query := `
		SELECT schema_name || '.' || table_name as full_name
		FROM _timescaledb_catalog.hypertable
	`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hypertables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		hypertables = append(hypertables, name)
	}

	return hypertables, rows.Err()
}
