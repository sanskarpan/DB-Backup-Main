package redis

import (
	"context"
	"os"
	"testing"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := NewRedisDriver()
	config := &database.ConnectionConfig{
		Host:     "localhost",
		Port:     6379,
		Database: "0",
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	defer driver.Disconnect()

	assert.NoError(t, err)
	assert.NotNil(t, driver.client)
}

func TestRedisDriver_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.Ping(ctx)
	assert.NoError(t, err)
}

func TestRedisDriver_BackupRDB(t *testing.T) {
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
		BackupType: "rdb",
		Metadata:   make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.NotEmpty(t, result.BackupPath)
	assert.Greater(t, result.BackupSize, int64(0))
	assert.Equal(t, "rdb", result.Metadata["backup_type"])
}

func TestRedisDriver_BackupAOF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()

	ctx := context.Background()
	opts := &database.BackupOptions{
		OutputDir:  tmpDir,
		BackupType: "aof",
		Metadata:   make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.NotEmpty(t, result.BackupPath)
	assert.Equal(t, "aof", result.Metadata["backup_type"])
}

func TestRedisDriver_BackupBoth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()

	ctx := context.Background()
	opts := &database.BackupOptions{
		OutputDir:  tmpDir,
		BackupType: "both",
		Metadata:   make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestRedisDriver_Restore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// First, create a backup
	tmpDir := t.TempDir()
	ctx := context.Background()

	backupOpts := &database.BackupOptions{
		OutputDir:  tmpDir,
		BackupType: "rdb",
		Metadata:   make(map[string]interface{}),
	}

	backupResult, err := driver.Backup(ctx, backupOpts)
	require.NoError(t, err)

	// Then restore
	restoreOpts := &database.RestoreOptions{
		BackupPath:   backupResult.BackupPath,
		DropExisting: true,
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
}

func TestRedisDriver_GetDatabaseSize(t *testing.T) {
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

func TestRedisDriver_GetVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	version, err := driver.GetVersion(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}

func TestPITRManager_EnablePITR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.pitrManager.EnablePITR(ctx)
	assert.NoError(t, err)
}

func TestPITRManager_CreateCheckpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	checkpointPath, err := driver.pitrManager.CreatePITRCheckpoint(ctx, tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, checkpointPath)

	// Verify checkpoint file exists
	_, err = os.Stat(checkpointPath)
	assert.NoError(t, err)
}

func TestPITRManager_GetRecoveryRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	start, end, err := driver.pitrManager.GetRecoveryRange(ctx)
	assert.NoError(t, err)
	assert.True(t, end.After(start))
}

func TestClusterDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Skip("Cluster tests require Redis cluster setup")

	cluster := NewClusterDriver()
	config := &database.ConnectionConfig{
		Host:     "localhost:7000,localhost:7001,localhost:7002",
		Password: "",
	}

	ctx := context.Background()
	err := cluster.Connect(ctx, config)
	defer cluster.Disconnect()

	assert.NoError(t, err)
}

func TestClusterDriver_BackupCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Skip("Cluster tests require Redis cluster setup")
}

// Helper functions

func setupTestDriver(t *testing.T) *RedisDriver {
	driver := NewRedisDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     6379,
		Database: "0",
		Password: getEnv("REDIS_PASSWORD", ""),
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
