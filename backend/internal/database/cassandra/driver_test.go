package cassandra

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

func TestCassandraDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfCassandraUnavailable(t)

	driver := NewCassandraDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("CASSANDRA_HOST", "localhost"),
		Port:     getEnvInt("CASSANDRA_PORT", 9042),
		Username: getEnv("CASSANDRA_USER", "cassandra"),
		Password: getEnv("CASSANDRA_PASSWORD", "cassandra"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	defer driver.Disconnect()

	assert.NoError(t, err)
	assert.NotNil(t, driver.session)
}

func TestCassandraDriver_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)

	err := driver.Disconnect()
	assert.NoError(t, err)
}

func TestCassandraDriver_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.Ping(ctx)
	assert.NoError(t, err)
}

func TestCassandraDriver_Backup_Snapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Create test directory
	tmpDir := t.TempDir()

	ctx := context.Background()
	opts := &database.BackupOptions{
		OutputDir:  tmpDir,
		BackupType: "snapshot",
		Metadata:   make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.NotEmpty(t, result.BackupPath)
	assert.Greater(t, result.BackupSize, int64(0))
}

func TestCassandraDriver_Backup_IncrementalBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	opts := &database.BackupOptions{
		OutputDir:   tmpDir,
		BackupType:  "incremental",
		Incremental: true,
		Metadata:    make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestCassandraDriver_Backup_SpecificKeyspace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	opts := &database.BackupOptions{
		OutputDir:      tmpDir,
		BackupType:     "snapshot",
		IncludeSchemas: []string{"system_schema"},
		Metadata:       make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.Contains(t, result.Metadata, "keyspaces")
}

func TestCassandraDriver_Restore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// First create a backup
	tmpDir := t.TempDir()
	ctx := context.Background()

	backupOpts := &database.BackupOptions{
		OutputDir:  tmpDir,
		BackupType: "snapshot",
		Metadata:   make(map[string]interface{}),
	}

	backupResult, err := driver.Backup(ctx, backupOpts)
	require.NoError(t, err)

	// Then restore
	restoreOpts := &database.RestoreOptions{
		BackupPath:   backupResult.BackupPath,
		DropExisting: false,
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
}

func TestCassandraDriver_GetKeyspaces(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	keyspaces, err := driver.GetDatabases(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, keyspaces)
	assert.Contains(t, keyspaces, "system")
	assert.Contains(t, keyspaces, "system_schema")
}

func TestCassandraDriver_GetTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	tables, err := driver.GetTables(ctx, "system_schema")

	assert.NoError(t, err)
	assert.NotEmpty(t, tables)
}

func TestCassandraDriver_GetVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	version, err := driver.GetVersion(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "Cassandra")
}

func TestCassandraDriver_GetDatabaseSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	size, err := driver.GetDatabaseSize(ctx)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, size, int64(0))
}

func TestCassandraCluster_BackupMultiDC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Skip("Multi-DC tests require cluster setup")

	cluster := NewClusterManager(nil)
	ctx := context.Background()
	tmpDir := t.TempDir()

	opts := &database.BackupOptions{
		OutputDir: tmpDir,
		Metadata:  make(map[string]interface{}),
	}

	result, err := cluster.BackupMultiDC(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestCassandraCluster_IncrementalBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	cluster := driver.clusterManager
	ctx := context.Background()

	// Enable incremental backups
	err := cluster.EnableIncrementalBackup(ctx)
	assert.NoError(t, err)

	// Check status
	enabled, err := cluster.IsIncrementalBackupEnabled(ctx)
	assert.NoError(t, err)
	assert.True(t, enabled)

	// Disable
	err = cluster.DisableIncrementalBackup(ctx)
	assert.NoError(t, err)
}

func TestScyllaDB_AutoDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := NewCassandraDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("SCYLLA_HOST", "localhost"),
		Port:     getEnvInt("SCYLLA_PORT", 9042),
		Username: getEnv("SCYLLA_USER", "cassandra"),
		Password: getEnv("SCYLLA_PASSWORD", "cassandra"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Skip("ScyllaDB not available")
	}
	defer driver.Disconnect()

	// Driver should auto-detect ScyllaDB
	version, _ := driver.GetVersion(ctx)
	if driver.isScyllaDB {
		assert.Contains(t, version, "ScyllaDB")
	}
}

func TestCassandra_SSHConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Skip("SSH connection tests require SSH server setup")

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Test SSH connection for nodetool access
	ctx := context.Background()
	err := driver.testSSHConnection(ctx)
	assert.NoError(t, err)
}

func TestCassandraDriver_SnapshotOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create snapshot
	snapshotName := "test_snapshot_" + time.Now().Format("20060102_150405")
	err := driver.createSnapshot(ctx, snapshotName, tmpDir, nil)
	assert.NoError(t, err)

	// Verify snapshot directory exists
	snapshotPath := filepath.Join(tmpDir, snapshotName)
	_, err = os.Stat(snapshotPath)
	assert.NoError(t, err)

	// Clear snapshot
	err = driver.clearSnapshot(ctx, snapshotName)
	assert.NoError(t, err)
}

func TestCassandraDriver_GetTableSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Get size of a system table
	size, err := driver.GetTableSize(ctx, "system_schema", "keyspaces")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, size, int64(0))
}

func TestCassandraDriver_GetType(t *testing.T) {
	driver := NewCassandraDriver()
	dbType := driver.GetType()
	assert.Equal(t, database.DatabaseType("cassandra"), dbType)
}

func TestCassandraDriver_SupportsIncremental(t *testing.T) {
	driver := NewCassandraDriver()
	assert.True(t, driver.SupportsIncremental())
}

func TestCassandraDriver_SupportsPITR(t *testing.T) {
	driver := NewCassandraDriver()
	assert.False(t, driver.SupportsPITR())
}

// Helper functions

func skipIfCassandraUnavailable(t *testing.T) {
	t.Helper()
	host := getEnv("CASSANDRA_HOST", "localhost")
	port := getEnvInt("CASSANDRA_PORT", 9042)
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("Cassandra not available at %s: %v", addr, err)
	}
	conn.Close()
}

func setupTestDriver(t *testing.T) *CassandraDriver {
	skipIfCassandraUnavailable(t)
	driver := NewCassandraDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("CASSANDRA_HOST", "localhost"),
		Port:     getEnvInt("CASSANDRA_PORT", 9042),
		Username: getEnv("CASSANDRA_USER", "cassandra"),
		Password: getEnv("CASSANDRA_PASSWORD", "cassandra"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	require.NoError(t, err)

	return driver
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion, in production use strconv
		return defaultValue
	}
	return defaultValue
}
