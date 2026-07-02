package redis

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfRedisUnavailable(t *testing.T) {
	t.Helper()
	host := getEnv("REDIS_HOST", "localhost")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:6379", host), 2*time.Second)
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	conn.Close()
}

func TestRedisDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfRedisUnavailable(t)

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

	// Skip if Redis RDB dump file doesn't exist at the default path
	if _, err := os.Stat("/data/dump.rdb"); err != nil {
		t.Skip("Redis RDB file not available at /data/dump.rdb")
	}

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

	if _, err := os.Stat("/data/appendonly.aof"); err != nil {
		t.Skip("Redis AOF file not available at /data/appendonly.aof")
	}

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

	if _, err := os.Stat("/data/dump.rdb"); err != nil {
		t.Skip("Redis data files not available at /data/")
	}

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

	if _, err := os.Stat("/data/dump.rdb"); err != nil {
		t.Skip("Redis data files not available at /data/")
	}

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

	if _, err := os.Stat("/data/appendonly.aof"); err != nil {
		t.Skip("Redis AOF file not available at /data/appendonly.aof")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	checkpointPath, err := driver.pitrManager.CreatePITRCheckpoint(ctx, tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, checkpointPath)

	// Verify checkpoint file exists
	_, err = os.Stat(checkpointPath)
	assert.NoError(t, err)
}

// TestPITRManager_GetRecoveryRange_Honest verifies that GetRecoveryRange no
// longer fabricates a now/now-24h range and instead returns an honest error,
// because Redis maintains no continuous recovery window. This is a pure unit
// test and needs no live Redis (the method returns before touching the client).
func TestPITRManager_GetRecoveryRange_Honest(t *testing.T) {
	p := NewPITRManager(NewRedisDriver())

	start, end, err := p.GetRecoveryRange(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPITRNotSupported)
	assert.True(t, start.IsZero())
	assert.True(t, end.IsZero())
}

// TestRedisDriver_SupportsPITR verifies the driver honestly reports that it
// cannot perform true time-based point-in-time recovery.
func TestRedisDriver_SupportsPITR(t *testing.T) {
	driver := NewRedisDriver()
	assert.False(t, driver.SupportsPITR())
}

// TestPITRManager_RestoreToPIT_SnapshotNewerThanTarget verifies that a request
// whose only snapshot is newer than the target time returns an honest
// granularity error instead of silently restoring the wrong data. The method
// returns before touching the Redis client, so no live Redis is required.
func TestPITRManager_RestoreToPIT_SnapshotNewerThanTarget(t *testing.T) {
	p := NewPITRManager(NewRedisDriver())

	backupPath := filepath.Join(t.TempDir(), "snapshot.rdb")
	require.NoError(t, os.WriteFile(backupPath, []byte("REDIS0011"), 0o600))

	// Force the snapshot's modification time to be well after the target.
	snapshotTime := time.Now()
	require.NoError(t, os.Chtimes(backupPath, snapshotTime, snapshotTime))
	targetTime := snapshotTime.Add(-1 * time.Hour)

	err := p.RestoreToPIT(context.Background(), targetTime, backupPath)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPITRGranularity)
}

// TestPITRManager_RestoreToPIT_MissingBackup verifies an honest error when the
// referenced backup artifact does not exist.
func TestPITRManager_RestoreToPIT_MissingBackup(t *testing.T) {
	p := NewPITRManager(NewRedisDriver())

	missing := filepath.Join(t.TempDir(), "does-not-exist.rdb")
	err := p.RestoreToPIT(context.Background(), time.Now(), missing)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrPITRGranularity)
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
	skipIfRedisUnavailable(t)
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
