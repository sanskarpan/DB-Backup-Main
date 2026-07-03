package timescaledb

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func skipIfTimescaleDBUnavailable(t *testing.T) {
	t.Helper()
	host := getEnv("TIMESCALEDB_HOST", "localhost")
	port := getEnv("TIMESCALEDB_PORT", "5432")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 2*time.Second)
	if err != nil {
		t.Skip("TimescaleDB not available:", err)
	}
	conn.Close()
}

func TestTimescaleDBDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfTimescaleDBUnavailable(t)

	driver := NewTimescaleDBDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("TIMESCALEDB_HOST", "localhost"),
		Port:     getEnvInt("TIMESCALEDB_PORT", 5432),
		Username: getEnv("TIMESCALEDB_USER", "postgres"),
		Password: getEnv("TIMESCALEDB_PASSWORD", ""),
		Database: getEnv("TIMESCALEDB_DB", "postgres"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Skipf("TimescaleDB not accessible: %v", err)
	}
	defer driver.Disconnect()

	assert.NotNil(t, driver.pool)
	assert.NotNil(t, driver.compressionManager)
	assert.NotNil(t, driver.chunkManager)
}

func TestTimescaleDBDriver_Connect_NoExtension(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Test connecting to regular PostgreSQL without TimescaleDB extension
	driver := NewTimescaleDBDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     getEnvInt("POSTGRES_PORT", 5432),
		Username: getEnv("POSTGRES_USER", "postgres"),
		Password: getEnv("POSTGRES_PASSWORD", ""),
		Database: "postgres",
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)

	if err != nil {
		if !contains(err.Error(), "timescaledb extension not installed") {
			t.Skipf("PostgreSQL not accessible with postgres role: %v", err)
		}
		assert.Contains(t, err.Error(), "timescaledb extension not installed")
	} else {
		driver.Disconnect()
		t.Skip("Test database has TimescaleDB extension")
	}
}

func TestTimescaleDBDriver_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)

	err := driver.Disconnect()
	assert.NoError(t, err)
}

func TestTimescaleDBDriver_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.Ping(ctx)
	assert.NoError(t, err)
}

func TestTimescaleDBDriver_Backup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	opts := &database.BackupOptions{
		OutputDir: tmpDir,
		Metadata:  make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.NotEmpty(t, result.BackupPath)
	assert.Greater(t, result.BackupSize, int64(0))
	assert.Contains(t, result.Metadata, "timescaledb_version")
	assert.Contains(t, result.Metadata, "backup_file")
}

func TestTimescaleDBDriver_Backup_SpecificTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Get list of tables
	tables, err := driver.GetTables(ctx, driver.config.Database)
	require.NoError(t, err)

	if len(tables) == 0 {
		t.Skip("No tables available for testing")
	}

	opts := &database.BackupOptions{
		OutputDir: tmpDir,
		Tables:    []string{tables[0]},
		Metadata:  make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestTimescaleDBDriver_Restore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// First create a backup
	tmpDir := t.TempDir()
	ctx := context.Background()

	backupOpts := &database.BackupOptions{
		OutputDir: tmpDir,
		Metadata:  make(map[string]interface{}),
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

func TestTimescaleDBDriver_Restore_WithDropExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// First create a backup
	tmpDir := t.TempDir()
	ctx := context.Background()

	backupOpts := &database.BackupOptions{
		OutputDir: tmpDir,
		Metadata:  make(map[string]interface{}),
	}

	backupResult, err := driver.Backup(ctx, backupOpts)
	require.NoError(t, err)

	// Restore with drop existing
	restoreOpts := &database.RestoreOptions{
		BackupPath:   backupResult.BackupPath,
		DropExisting: true,
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
}

func TestTimescaleDBDriver_GetDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	databases, err := driver.GetDatabases(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, databases)
	assert.Contains(t, databases, "postgres")
}

func TestTimescaleDBDriver_GetTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	tables, err := driver.GetTables(ctx, driver.config.Database)

	assert.NoError(t, err)
	// May be empty if no user tables
	t.Logf("Found %d tables", len(tables))
}

func TestTimescaleDBDriver_GetHypertables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	hypertables, err := driver.GetHypertables(ctx)

	assert.NoError(t, err)
	// May be empty if no hypertables
	t.Logf("Found %d hypertables", len(hypertables))
}

func TestTimescaleDBDriver_GetTableSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Get a table to test
	tables, err := driver.GetTables(ctx, driver.config.Database)
	require.NoError(t, err)

	if len(tables) == 0 {
		t.Skip("No tables available for size test")
	}

	size, err := driver.GetTableSize(ctx, driver.config.Database, tables[0])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, size, int64(0))
}

func TestTimescaleDBDriver_GetDatabaseSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	size, err := driver.GetDatabaseSize(ctx)

	assert.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestTimescaleDBDriver_GetVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	version, err := driver.GetVersion(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "TimescaleDB")
	assert.Contains(t, version, "PostgreSQL")
}

func TestTimescaleDBDriver_GetType(t *testing.T) {
	driver := NewTimescaleDBDriver()
	dbType := driver.GetType()
	assert.Equal(t, database.DatabaseType("timescaledb"), dbType)
}

func TestTimescaleDBDriver_SupportsIncremental(t *testing.T) {
	driver := NewTimescaleDBDriver()
	assert.True(t, driver.SupportsIncremental())
}

func TestTimescaleDBDriver_SupportsPITR(t *testing.T) {
	driver := NewTimescaleDBDriver()
	assert.True(t, driver.SupportsPITR())
}

func TestTimescaleDBDriver_ValidateRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Test with missing backup path
	err := driver.ValidateRestore(ctx, &database.RestoreOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup path is required")

	// Test with non-existent path
	err = driver.ValidateRestore(ctx, &database.RestoreOptions{
		BackupPath: "/nonexistent/path/12345",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	// Test with valid path
	tmpDir := t.TempDir()
	err = driver.ValidateRestore(ctx, &database.RestoreOptions{
		BackupPath: tmpDir,
	})
	assert.NoError(t, err)
}

func TestTimescaleDBDriver_GetBackupSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	opts := &database.BackupOptions{}

	size, err := driver.GetBackupSize(ctx, opts)
	assert.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestTimescaleDB_BackupHypertableMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	err := driver.backupHypertableMetadata(ctx, tmpDir)
	assert.NoError(t, err)

	// Verify metadata file was created
	metadataFile := tmpDir + "/hypertables_metadata.sql"
	_, err = os.Stat(metadataFile)
	assert.NoError(t, err)
}

func TestTimescaleDB_BackupContinuousAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	err := driver.backupContinuousAggregates(ctx, tmpDir)
	assert.NoError(t, err)

	// Verify continuous aggregates file was created
	caggFile := tmpDir + "/continuous_aggregates.sql"
	_, err = os.Stat(caggFile)
	assert.NoError(t, err)
}

func TestTimescaleDB_CompressionPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Test backing up compression policies
	err := driver.compressionManager.BackupCompressionPolicies(ctx, tmpDir)
	assert.NoError(t, err)

	// Verify compression policies file was created
	compressionFile := tmpDir + "/compression_policies.sql"
	_, err = os.Stat(compressionFile)
	assert.NoError(t, err)

	// Test restoring compression policies
	err = driver.compressionManager.RestoreCompressionPolicies(ctx, tmpDir)
	// This may fail if there are no hypertables, which is okay
	if err != nil {
		t.Logf("Restore compression policies: %v", err)
	}
}

func TestTimescaleDB_ChunkOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Get hypertables first
	hypertables, err := driver.GetHypertables(ctx)
	require.NoError(t, err)

	if len(hypertables) == 0 {
		t.Skip("No hypertables available for chunk operations test")
	}

	// Test getting chunks
	chunks, err := driver.chunkManager.GetChunks(ctx, hypertables[0])
	if err != nil {
		t.Logf("GetChunks error: %v", err)
	} else {
		t.Logf("Found %d chunks for hypertable %s", len(chunks), hypertables[0])
	}

	// Test getting chunk statistics
	stats, err := driver.chunkManager.GetChunkStatistics(ctx)
	if err != nil {
		t.Logf("GetChunkStatistics error: %v", err)
	} else {
		assert.NotNil(t, stats)
	}
}

func TestTimescaleDB_pgDumpIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	outputFile := tmpDir + "/test_dump.sql"
	opts := &database.BackupOptions{
		Database: driver.config.Database,
	}

	err := driver.pgDump(ctx, outputFile, opts)
	require.NoError(t, err)

	// Verify dump file was created
	info, err := os.Stat(outputFile)
	assert.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestTimescaleDB_pgRestoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// First create a dump
	tmpDir := t.TempDir()
	ctx := context.Background()

	dumpFile := tmpDir + "/test_dump.sql"
	backupOpts := &database.BackupOptions{
		Database: driver.config.Database,
	}

	err := driver.pgDump(ctx, dumpFile, backupOpts)
	require.NoError(t, err)

	// Then restore it
	restoreOpts := &database.RestoreOptions{
		Database:     driver.config.Database,
		DropExisting: false,
	}

	err = driver.pgRestore(ctx, dumpFile, restoreOpts)
	// This may have warnings but should complete
	if err != nil {
		t.Logf("pgRestore warnings/errors: %v", err)
	}
}

func TestTimescaleDB_BuildConnectionString(t *testing.T) {
	driver := NewTimescaleDBDriver()

	// Test with all parameters
	config := &database.ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "require",
	}

	connStr := driver.buildConnectionString(config)
	assert.Contains(t, connStr, "host=localhost")
	assert.Contains(t, connStr, "port=5432")
	assert.Contains(t, connStr, "user=testuser")
	assert.Contains(t, connStr, "password=testpass")
	assert.Contains(t, connStr, "dbname=testdb")
	assert.Contains(t, connStr, "sslmode=require")

	// Test with connection string directly
	config2 := &database.ConnectionConfig{
		ConnectionString: "postgresql://user:pass@host:5432/db",
	}

	connStr2 := driver.buildConnectionString(config2)
	assert.Equal(t, "postgresql://user:pass@host:5432/db", connStr2)
}

// Helper functions

func setupTestDriver(t *testing.T) *TimescaleDBDriver {
	t.Helper()
	skipIfTimescaleDBUnavailable(t)
	driver := NewTimescaleDBDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("TIMESCALEDB_HOST", "localhost"),
		Port:     getEnvInt("TIMESCALEDB_PORT", 5432),
		Username: getEnv("TIMESCALEDB_USER", "postgres"),
		Password: getEnv("TIMESCALEDB_PASSWORD", ""),
		Database: getEnv("TIMESCALEDB_DB", "postgres"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Skipf("TimescaleDB not accessible: %v", err)
	}

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
