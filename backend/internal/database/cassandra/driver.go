// Package cassandra provides Cassandra/ScyllaDB database driver implementation
package cassandra

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gocql/gocql"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
	"github.com/sanskarpan/db-backup/pkg/validation"
)

const (
	snapshotBackupType = "snapshot"
	cassandraDataDir   = "/var/lib/cassandra/data"
	scyllaDataDir      = "/var/lib/scylla/data"

	// dataDirOption is the ConnectionConfig.Options key that overrides the
	// on-disk data directory (useful for non-default installs or when the
	// backup tool runs on the same host with a custom layout).
	dataDirOption = "data_dir"
)

// tableUUIDSuffix matches the "-<32 hex>" table-directory suffix Cassandra
// appends to table data directories (e.g. "users-a1b2...ef").
var tableUUIDSuffix = regexp.MustCompile(`-[0-9a-fA-F]{32}$`)

// keyspaceTable identifies a single Cassandra table by keyspace and name.
type keyspaceTable struct {
	keyspace string
	table    string
}

// CassandraDriver implements the database.Driver interface for Cassandra/ScyllaDB.
//
//nolint:revive // keeps public name stable across dependent packages
type CassandraDriver struct {
	session        *gocql.Session
	config         *database.ConnectionConfig
	cluster        *gocql.ClusterConfig
	isScyllaDB     bool
	clusterManager *ClusterManager
}

func init() {
	database.RegisterDriver("cassandra", func() database.Driver {
		return NewCassandraDriver()
	})
	database.RegisterDriver("scylladb", func() database.Driver {
		return NewCassandraDriver()
	})
}

// NewCassandraDriver creates a new Cassandra driver instance.
func NewCassandraDriver() *CassandraDriver {
	return &CassandraDriver{}
}

// Connect establishes a connection to the Cassandra database.
func (d *CassandraDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Create cluster configuration
	cluster := gocql.NewCluster(config.Host)
	cluster.Port = config.Port
	cluster.Keyspace = config.Database
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	if config.Username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: config.Username,
			Password: config.Password,
		}
	}

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Detect if ScyllaDB
	d.isScyllaDB = d.detectScyllaDB(session)

	d.session = session
	d.config = config
	d.cluster = cluster
	d.clusterManager = NewClusterManager(d)
	return nil
}

// Disconnect closes the database connection.
func (d *CassandraDriver) Disconnect() error {
	if d.session != nil {
		d.session.Close()
	}
	return nil
}

// Ping tests the database connection.
func (d *CassandraDriver) Ping(ctx context.Context) error {
	if d.session == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}

	// Execute simple query to test connection
	iter := d.session.Query("SELECT now() FROM system.local").Iter()
	defer iter.Close()

	var now gocql.UUID
	if !iter.Scan(&now) {
		return fmt.Errorf("failed to ping Cassandra")
	}

	return iter.Close()
}

// Backup creates a backup of the Cassandra database.
func (d *CassandraDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Determine backup type
	backupType := opts.BackupType
	if backupType == "" {
		backupType = snapshotBackupType
	}

	var err error
	switch backupType {
	case snapshotBackupType:
		err = d.backupSnapshot(ctx, opts, result)
	case "incremental":
		err = d.backupIncremental(opts, result)
	default:
		err = fmt.Errorf("unsupported backup type: %s", backupType)
	}

	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// backupSnapshot creates a snapshot backup using nodetool.
func (d *CassandraDriver) backupSnapshot(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) error {
	snapshotName := fmt.Sprintf("backup_%s", result.ID)

	// Execute nodetool snapshot command
	args := []string{"snapshot"}

	// Add keyspace if specified
	if len(opts.IncludeSchemas) > 0 {
		args = append(args, "-kt", strings.Join(opts.IncludeSchemas, ","))
	}

	// Add tag
	args = append(args, "-t", snapshotName)

	cmd := exec.CommandContext(ctx, "nodetool", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodetool snapshot failed: %w, output: %s", err, string(output))
	}

	// Copy snapshot files to backup location
	dataDir := d.resolveDataDir()

	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err = os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Find and copy snapshot files
	err = d.copySnapshotFiles(dataDir, snapshotName, backupDir)
	if err != nil {
		return fmt.Errorf("failed to copy snapshot files: %w", err)
	}

	// Clean up snapshot from Cassandra
	clearCmd := exec.CommandContext(ctx, "nodetool", "clearsnapshot", "-t", snapshotName)
	if err := clearCmd.Run(); err != nil {
		// Log error but don't fail the backup
		fmt.Printf("Warning: failed to clear snapshot: %v\n", err)
	}

	// Get backup size
	var totalSize int64
	if walkErr := filepath.Walk(backupDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	}); walkErr != nil {
		return fmt.Errorf("failed to calculate backup size: %w", walkErr)
	}

	result.BackupPath = backupDir
	result.BackupSize = totalSize
	result.Metadata["backup_type"] = "snapshot"
	result.Metadata["snapshot_name"] = snapshotName
	result.Metadata["is_scylladb"] = d.isScyllaDB

	return nil
}

// backupIncremental creates an incremental backup.
func (d *CassandraDriver) backupIncremental(opts *database.BackupOptions, result *database.BackupResult) error {
	// Cassandra incremental backups are enabled via configuration
	// and create hard links in the backups directory

	// Get incremental backup directory
	dataDir := d.resolveDataDir()

	backupsDir := filepath.Join(dataDir, "backups")
	outputDir := filepath.Join(opts.OutputDir, result.ID)

	// Copy incremental backup files
	if err := d.copyDirectory(backupsDir, outputDir); err != nil {
		return fmt.Errorf("failed to copy incremental backups: %w", err)
	}

	// Get backup size
	var totalSize int64
	if walkErr := filepath.Walk(outputDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	}); walkErr != nil {
		return fmt.Errorf("failed to calculate backup size: %w", walkErr)
	}

	result.BackupPath = outputDir
	result.BackupSize = totalSize
	result.Metadata["backup_type"] = "incremental"
	result.Metadata["is_scylladb"] = d.isScyllaDB

	return nil
}

// GetBackupSize estimates the size of a backup.
func (d *CassandraDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	// For Cassandra, estimate based on current data size
	return d.GetDatabaseSize(ctx)
}

// StreamBackup streams a backup to the provided writer.
//
// Cassandra backups are SSTable snapshots on disk and cannot be produced as a
// single stream, so streaming is intentionally unsupported.
func (d *CassandraDriver) StreamBackup(_ context.Context, _ *database.BackupOptions, _ io.Writer) error {
	return fmt.Errorf("streaming backup is not supported for Cassandra; use a file-based snapshot backup")
}

// StreamRestore restores from a reader.
//
// Cassandra restores load SSTable files from disk, so streaming restore is
// intentionally unsupported.
func (d *CassandraDriver) StreamRestore(_ context.Context, _ *database.RestoreOptions, _ io.Reader) error {
	return fmt.Errorf("streaming restore is not supported for Cassandra; use a file-based snapshot restore")
}

// ValidateRestore validates that a restore can be performed.
func (d *CassandraDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" {
		return fmt.Errorf("backup path is required")
	}
	return nil
}

// GetDatabases returns list of keyspaces.
func (d *CassandraDriver) GetDatabases(ctx context.Context) ([]string, error) {
	query := "SELECT keyspace_name FROM system_schema.keyspaces"
	iter := d.session.Query(query).Iter()
	defer iter.Close()

	var keyspaces []string
	var keyspace string
	for iter.Scan(&keyspace) {
		keyspaces = append(keyspaces, keyspace)
	}

	return keyspaces, iter.Close()
}

// GetTables returns list of tables in a keyspace.
func (d *CassandraDriver) GetTables(ctx context.Context, keyspace string) ([]string, error) {
	query := "SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?"
	iter := d.session.Query(query, keyspace).Iter()
	defer iter.Close()

	var tables []string
	var table string
	for iter.Scan(&table) {
		tables = append(tables, table)
	}

	return tables, iter.Close()
}

// GetTableSize returns the size of a table.
func (d *CassandraDriver) GetTableSize(ctx context.Context, keyspace, table string) (int64, error) {
	// Cassandra doesn't provide direct table size - estimate from size_estimates
	query := "SELECT mean_partition_size, partitions_count FROM system.size_estimates WHERE keyspace_name = ? AND table_name = ?"
	iter := d.session.Query(query, keyspace, table).Iter()
	defer iter.Close()

	var meanSize, partitionCount int64
	if iter.Scan(&meanSize, &partitionCount) {
		return meanSize * partitionCount, nil
	}

	return 0, fmt.Errorf("could not determine table size")
}

// GetType returns the database type.
func (d *CassandraDriver) GetType() database.DatabaseType {
	if d.isScyllaDB {
		return "scylladb"
	}
	return "cassandra"
}

// SupportsIncremental returns whether incremental backups are supported.
func (d *CassandraDriver) SupportsIncremental() bool {
	return true
}

// SupportsPITR returns whether point-in-time recovery is supported.
func (d *CassandraDriver) SupportsPITR() bool {
	return false // Cassandra doesn't support PITR natively
}

// Restore restores the Cassandra database from a backup.
//
// It stages the backed-up SSTable files into the (configurable) data directory
// and then reloads them into the running node with "nodetool refresh" per
// restored table, so a manual service restart is not required. If a refresh
// fails, an actionable error is returned instead of silently reporting success.
func (d *CassandraDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
		Metadata:  make(map[string]interface{}),
	}

	fail := func(err error) (*database.RestoreResult, error) {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	if opts.BackupPath == "" {
		return fail(fmt.Errorf("backup path is required"))
	}

	dataDir := d.resolveDataDir()
	result.Metadata["data_dir"] = dataDir

	// Stage SSTable files into the live table directories.
	tables, err := d.stageRestoreFiles(opts.BackupPath, dataDir)
	if err != nil {
		return fail(fmt.Errorf("failed to stage restore files: %w", err))
	}

	// Reload the staged SSTables into the running node.
	if err := d.refreshTables(ctx, tables); err != nil {
		return fail(err)
	}

	// Verify the node is still reachable after the reload.
	if err := d.Ping(ctx); err != nil {
		return fail(err)
	}

	for _, kt := range tables {
		result.RestoredTables = append(result.RestoredTables, kt.keyspace+"."+kt.table)
	}
	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// resolveDataDir returns the on-disk data directory, honoring the "data_dir"
// connection option and falling back to the engine default when unset.
func (d *CassandraDriver) resolveDataDir() string {
	if d.config != nil {
		if dir := d.config.Options[dataDirOption]; dir != "" {
			return dir
		}
	}
	if d.isScyllaDB {
		return scyllaDataDir
	}
	return cassandraDataDir
}

// stripTableUUID removes the "-<uuid>" suffix Cassandra appends to table data
// directories, returning the bare table name.
func stripTableUUID(dir string) string {
	return tableUUIDSuffix.ReplaceAllString(dir, "")
}

// planRestoreEntry maps a backup-relative file path to the destination path
// (relative to the data directory) and the keyspace/table it belongs to.
//
// SSTable files stored under a "snapshots/<name>/" (or "backups/") segment are
// flattened into the live table directory so that "nodetool refresh" can load
// them. ok is false for paths that are not recognizable table data.
func planRestoreEntry(rel string) (destRel string, kt keyspaceTable, ok bool) {
	segs := strings.Split(filepath.ToSlash(rel), "/")
	if len(segs) < 3 {
		return "", keyspaceTable{}, false
	}

	keyspace, tableDir := segs[0], segs[1]
	kt = keyspaceTable{keyspace: keyspace, table: stripTableUUID(tableDir)}

	rest := segs[2:]
	switch {
	case len(rest) >= 3 && rest[0] == "snapshots":
		// rest = ["snapshots", "<name>", <file...>]
		rest = rest[2:]
	case len(rest) >= 2 && rest[0] == "backups":
		// rest = ["backups", <file...>]
		rest = rest[1:]
	}

	destRel = filepath.Join(append([]string{keyspace, tableDir}, rest...)...)
	return destRel, kt, true
}

// stageRestoreFiles copies the backup files into the data directory, flattening
// snapshot/backup subdirectories into the live table directories. It returns
// the distinct keyspace/table pairs that were restored, in first-seen order.
func (d *CassandraDriver) stageRestoreFiles(backupPath, dataDir string) ([]keyspaceTable, error) {
	seen := make(map[keyspaceTable]struct{})
	var tables []keyspaceTable

	walkErr := filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(backupPath, path)
		if err != nil {
			return err
		}

		destRel, kt, ok := planRestoreEntry(rel)
		if !ok {
			destRel = rel
		}

		destPath := filepath.Join(dataDir, destRel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		if err := d.copyFile(path, destPath); err != nil {
			return err
		}

		if ok {
			if _, dup := seen[kt]; !dup {
				seen[kt] = struct{}{}
				tables = append(tables, kt)
			}
		}
		return nil
	})

	return tables, walkErr
}

// nodetoolRefreshArgs builds the "nodetool refresh <keyspace> <table>" args.
func nodetoolRefreshArgs(kt keyspaceTable) []string {
	return []string{"refresh", kt.keyspace, kt.table}
}

// refreshTables reloads the staged SSTables for each restored table via
// "nodetool refresh". Keyspace/table names are validated before use so the
// exec arguments cannot be attacker-controlled shell/flag injection.
func (d *CassandraDriver) refreshTables(ctx context.Context, tables []keyspaceTable) error {
	for _, kt := range tables {
		if err := validation.ValidateDatabaseName(kt.keyspace); err != nil {
			return fmt.Errorf("invalid keyspace name %q in backup: %w", kt.keyspace, err)
		}
		if err := validation.ValidateTableName(kt.table); err != nil {
			return fmt.Errorf("invalid table name %q in backup: %w", kt.table, err)
		}

		//nolint:gosec // G204: keyspace/table validated by validation.Validate* above
		cmd := exec.CommandContext(ctx, "nodetool", nodetoolRefreshArgs(kt)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf(
				"nodetool refresh for %s.%s failed; SSTables were copied to the data directory but not loaded — "+
					"run 'nodetool refresh %s %s' on the node or restart it: %w (output: %s)",
				kt.keyspace, kt.table, kt.keyspace, kt.table, err, strings.TrimSpace(string(output)),
			)
		}
	}
	return nil
}

// GetDatabaseSize returns the total size of the database.
func (d *CassandraDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	// Query system tables for size information
	query := "SELECT sum(total_disk_space_used) as size FROM system.size_estimates"
	iter := d.session.Query(query).Iter()
	defer iter.Close()

	var size int64
	if iter.Scan(&size) {
		return size, nil
	}

	return 0, fmt.Errorf("failed to get database size")
}

// GetVersion returns the Cassandra/ScyllaDB version.
func (d *CassandraDriver) GetVersion(ctx context.Context) (string, error) {
	query := "SELECT release_version FROM system.local"
	iter := d.session.Query(query).Iter()
	defer iter.Close()

	var version string
	if iter.Scan(&version) {
		if d.isScyllaDB {
			return fmt.Sprintf("ScyllaDB %s", version), nil
		}
		return fmt.Sprintf("Cassandra %s", version), nil
	}

	return "unknown", nil
}

// detectScyllaDB detects if the database is ScyllaDB.
func (d *CassandraDriver) detectScyllaDB(session *gocql.Session) bool {
	query := "SELECT release_version FROM system.local"
	iter := session.Query(query).Iter()
	defer iter.Close()

	var version string
	if iter.Scan(&version) {
		return strings.Contains(strings.ToLower(version), "scylla")
	}

	return false
}

// copySnapshotFiles copies snapshot files from source to destination.
func (d *CassandraDriver) copySnapshotFiles(dataDir, snapshotName, destDir string) error {
	return filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.Contains(path, "snapshots/"+snapshotName) {
			rel, err := filepath.Rel(dataDir, path)
			if err != nil {
				return err
			}

			destPath := filepath.Join(destDir, rel)
			destPathDir := filepath.Dir(destPath)

			if err := os.MkdirAll(destPathDir, 0o755); err != nil {
				return err
			}

			return d.copyFile(path, destPath)
		}

		return nil
	})
}

// copyDirectory recursively copies a directory.
func (d *CassandraDriver) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return d.copyFile(path, destPath)
	})
}

// copyFile copies a single file.
func (d *CassandraDriver) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}

// ClusterManager manages Cassandra cluster operations.
type ClusterManager struct {
	driver *CassandraDriver
}

// NewClusterManager creates a new cluster manager.
func NewClusterManager(driver *CassandraDriver) *ClusterManager {
	return &ClusterManager{driver: driver}
}

// BackupMultiDC creates a backup across multiple datacenters.
//
// The single-node ClusterManager has no datacenter topology, so this is not
// supported here; use MultiDCDriver (see cluster.go) with an explicit
// datacenter-to-hosts mapping instead.
func (cm *ClusterManager) BackupMultiDC(_ context.Context, _ *database.BackupOptions) (*database.BackupResult, error) {
	return nil, fmt.Errorf(
		"multi-datacenter backup is not supported by the single-node driver; " +
			"use MultiDCDriver with a datacenter topology instead")
}

// EnableIncrementalBackup enables incremental backups on the node via
// "nodetool enablebackup".
func (cm *ClusterManager) EnableIncrementalBackup(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "nodetool", "enablebackup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodetool enablebackup failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// DisableIncrementalBackup disables incremental backups on the node via
// "nodetool disablebackup".
func (cm *ClusterManager) DisableIncrementalBackup(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "nodetool", "disablebackup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodetool disablebackup failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsIncrementalBackupEnabled checks whether incremental backup is enabled via
// "nodetool statusbackup" (which prints "running" or "not running").
func (cm *ClusterManager) IsIncrementalBackupEnabled(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "nodetool", "statusbackup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("nodetool statusbackup failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}
	return strings.EqualFold(strings.TrimSpace(string(output)), "running"), nil
}

// testSSHConnection tests SSH connection for nodetool access.
//
// This driver invokes nodetool on the local host; remote SSH execution is not
// supported in this configuration.
func (d *CassandraDriver) testSSHConnection(_ context.Context) error {
	return fmt.Errorf("SSH-based nodetool access is not supported in this configuration")
}

// createSnapshot creates a Cassandra snapshot.
func (d *CassandraDriver) createSnapshot(ctx context.Context, snapshotName string, keyspaces []string) error {
	// Execute nodetool snapshot command
	args := []string{"snapshot"}

	// Add keyspace if specified
	if len(keyspaces) > 0 {
		args = append(args, "-kt", strings.Join(keyspaces, ","))
	}

	// Add tag
	args = append(args, "-t", snapshotName)

	cmd := exec.CommandContext(ctx, "nodetool", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodetool snapshot failed: %w, output: %s", err, string(output))
	}

	return nil
}

// clearSnapshot clears a Cassandra snapshot.
func (d *CassandraDriver) clearSnapshot(ctx context.Context, snapshotName string) error {
	// Execute nodetool clearsnapshot command
	cmd := exec.CommandContext(ctx, "nodetool", "clearsnapshot", "-t", snapshotName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nodetool clearsnapshot failed: %w, output: %s", err, string(output))
	}
	return nil
}
