// Package cassandra provides Cassandra/ScyllaDB database driver implementation
package cassandra

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocql/gocql"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// CassandraDriver implements the database.Driver interface for Cassandra/ScyllaDB.
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
		backupType = "snapshot"
	}

	var err error
	switch backupType {
	case "snapshot":
		err = d.backupSnapshot(ctx, opts, result)
	case "incremental":
		err = d.backupIncremental(ctx, opts, result)
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
	if opts.IncludeSchemas != nil && len(opts.IncludeSchemas) > 0 {
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
	dataDir := "/var/lib/cassandra/data" // Default Cassandra data directory
	if d.isScyllaDB {
		dataDir = "/var/lib/scylla/data"
	}

	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
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
	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	result.BackupPath = backupDir
	result.BackupSize = totalSize
	result.Metadata["backup_type"] = "snapshot"
	result.Metadata["snapshot_name"] = snapshotName
	result.Metadata["is_scylladb"] = d.isScyllaDB

	return nil
}

// backupIncremental creates an incremental backup.
func (d *CassandraDriver) backupIncremental(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) error {
	// Cassandra incremental backups are enabled via configuration
	// and create hard links in the backups directory

	// Get incremental backup directory
	dataDir := "/var/lib/cassandra/data"
	if d.isScyllaDB {
		dataDir = "/var/lib/scylla/data"
	}

	backupsDir := filepath.Join(dataDir, "backups")
	outputDir := filepath.Join(opts.OutputDir, result.ID)

	// Copy incremental backup files
	if err := d.copyDirectory(backupsDir, outputDir); err != nil {
		return fmt.Errorf("failed to copy incremental backups: %w", err)
	}

	// Get backup size
	var totalSize int64
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

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
func (d *CassandraDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for Cassandra")
}

// StreamRestore restores from a reader.
func (d *CassandraDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for Cassandra")
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
func (d *CassandraDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Stop Cassandra service (in production, use proper service management)
	// This is a simplified implementation

	// Copy backup files to data directory
	dataDir := "/var/lib/cassandra/data"
	if d.isScyllaDB {
		dataDir = "/var/lib/scylla/data"
	}

	// Copy files
	if err := d.copyDirectory(opts.BackupPath, dataDir); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Restart Cassandra service would happen here

	// Verify restore
	if err := d.Ping(ctx); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
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
func (cm *ClusterManager) BackupMultiDC(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	// Stub implementation for multi-DC backup
	return nil, fmt.Errorf("multi-DC backup not implemented")
}

// EnableIncrementalBackup enables incremental backups on the cluster.
func (cm *ClusterManager) EnableIncrementalBackup(ctx context.Context) error {
	// Stub implementation
	return fmt.Errorf("incremental backup enable not implemented")
}

// DisableIncrementalBackup disables incremental backups on the cluster.
func (cm *ClusterManager) DisableIncrementalBackup(ctx context.Context) error {
	// Stub implementation
	return fmt.Errorf("incremental backup disable not implemented")
}

// IsIncrementalBackupEnabled checks if incremental backup is enabled.
func (cm *ClusterManager) IsIncrementalBackupEnabled(ctx context.Context) (bool, error) {
	// Stub implementation
	return false, fmt.Errorf("incremental backup status check not implemented")
}

// testSSHConnection tests SSH connection for nodetool access.
func (d *CassandraDriver) testSSHConnection(ctx context.Context) error {
	// Stub implementation for SSH connection testing
	return fmt.Errorf("SSH connection testing not implemented")
}

// createSnapshot creates a Cassandra snapshot.
func (d *CassandraDriver) createSnapshot(ctx context.Context, snapshotName, outputDir string, keyspaces []string) error {
	// Execute nodetool snapshot command
	args := []string{"snapshot"}

	// Add keyspace if specified
	if keyspaces != nil && len(keyspaces) > 0 {
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
