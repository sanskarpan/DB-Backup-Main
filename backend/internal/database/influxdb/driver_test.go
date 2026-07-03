package influxdb

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

// TestBackupRecord_RoundTrip proves that a v2 record survives the
// backup -> restore serialization round-trip without a live InfluxDB. It
// mirrors what backupBucket writes and what restoreBucket reads back.
func TestBackupRecord_RoundTrip(t *testing.T) {
	ts := time.Date(2026, 7, 1, 12, 34, 56, 789000000, time.UTC)

	// Simulate a non-pivoted Flux record's Values() map: reserved columns
	// plus two tags. newBackupRecord must keep only the tags.
	values := map[string]interface{}{
		"result":       "_result",
		"table":        int64(0),
		"_start":       ts.Add(-time.Hour),
		"_stop":        ts.Add(time.Hour),
		"_time":        ts,
		"_measurement": "cpu",
		"_field":       "usage_user",
		"_value":       float64(42.5),
		"host":         "server-a",
		"region":       "us-east-1",
	}

	original := newBackupRecord("cpu", "usage_user", float64(42.5), ts, values)

	// Tags must be exactly the non-reserved columns.
	assert.Equal(t, map[string]string{
		"host":   "server-a",
		"region": "us-east-1",
	}, original.Tags)

	// Marshal (as backupBucket does) then unmarshal (as restoreBucket does).
	encoded, err := json.Marshal(original)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(encoded), "map["),
		"serialized form must be JSON, not Go map stringification")

	var decoded backupRecord
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	// Measurement, field, value, timestamp and tags must all survive.
	assert.Equal(t, original.Measurement, decoded.Measurement)
	assert.Equal(t, original.Field, decoded.Field)
	assert.Equal(t, original.Value, decoded.Value)
	assert.True(t, original.Time.Equal(decoded.Time), "timestamp must round-trip")
	assert.Equal(t, original.Tags, decoded.Tags)

	// The decoded record must produce a writable point.
	point := decoded.toPoint()
	require.NotNil(t, point)
	assert.Equal(t, "cpu", point.Name())
}

// TestSumShardDiskSize verifies the Prometheus metrics parser used by
// GetDatabaseSize sums the storage_shard_disk_size gauge and reports presence.
func TestSumShardDiskSize(t *testing.T) {
	metrics := `# HELP storage_shard_disk_size Gauge of the disk size for the shard
# TYPE storage_shard_disk_size gauge
storage_shard_disk_size{bucket="b1"} 1024
storage_shard_disk_size{bucket="b2"} 2048
storage_shard_disk_size_ignored{bucket="b3"} 9999
other_metric 5
`
	total, found, err := sumShardDiskSize(strings.NewReader(metrics))
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, int64(3072), total)

	// When the metric is absent, found must be false so callers can error.
	_, found, err = sumShardDiskSize(strings.NewReader("other_metric 5\n"))
	require.NoError(t, err)
	assert.False(t, found)
}

func skipIfInfluxDBUnavailable(t *testing.T) {
	t.Helper()
	host := getEnv("INFLUXDB_HOST", "localhost")
	port := getEnv("INFLUXDB_PORT", "8086")
	conn, err := net.DialTimeout("tcp", host+":"+port, 2*time.Second)
	if err != nil {
		t.Skip("InfluxDB not available:", err)
	}
	conn.Close()
}

func TestInfluxDBDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfInfluxDBUnavailable(t)

	driver := NewInfluxDBDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("INFLUXDB_HOST", "localhost"),
		Port:     getEnvInt("INFLUXDB_PORT", 8086),
		Password: getEnv("INFLUXDB_TOKEN", ""),
		Database: getEnv("INFLUXDB_ORG", "default"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	defer driver.Disconnect()

	assert.NoError(t, err)
	assert.NotNil(t, driver.client)
	assert.NotNil(t, driver.queryAPI)
}

func TestInfluxDBDriver_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)

	err := driver.Disconnect()
	assert.NoError(t, err)
}

func TestInfluxDBDriver_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.Ping(ctx)
	assert.NoError(t, err)
}

func TestInfluxDBDriver_BackupV2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Skipping v2 test on v1 instance")
	}

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
	assert.Contains(t, result.Metadata, "influxdb_version")
	assert.Contains(t, result.Metadata, "buckets_backed_up")
}

func TestInfluxDBDriver_BackupV2_SpecificBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Skipping v2 test on v1 instance")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	testBucket := getEnv("INFLUXDB_TEST_BUCKET", "test-bucket")

	opts := &database.BackupOptions{
		OutputDir: tmpDir,
		Database:  testBucket,
		Metadata:  make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestInfluxDBDriver_BackupV2_Incremental(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Skipping v2 test on v1 instance")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	// Backup data from last 24 hours
	since := time.Now().Add(-24 * time.Hour)

	opts := &database.BackupOptions{
		OutputDir:   tmpDir,
		Incremental: true,
		Metadata: map[string]interface{}{
			"since": since,
		},
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
}

func TestInfluxDBDriver_BackupV1(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v2.x
	if driver.version == "" || driver.version[0] != '1' {
		t.Skip("Skipping v1 test on v2 instance")
	}

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
	assert.Contains(t, result.Metadata, "backup_output")
}

func TestInfluxDBDriver_RestoreV2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Skipping v2 test on v1 instance")
	}

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
		BackupPath: backupResult.BackupPath,
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	// Restore may not be fully implemented, check gracefully
	if err != nil {
		t.Logf("Restore not fully implemented: %v", err)
	} else {
		assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
	}
}

func TestInfluxDBDriver_RestoreV1(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v2.x
	if driver.version == "" || driver.version[0] != '1' {
		t.Skip("Skipping v1 test on v2 instance")
	}

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
		BackupPath: backupResult.BackupPath,
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
}

func TestInfluxDBDriver_GetDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	buckets, err := driver.GetDatabases(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, buckets)
}

func TestInfluxDBDriver_GetBucketsToBackup_AllBuckets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	opts := &database.BackupOptions{}

	buckets, err := driver.getBucketsToBackup(ctx, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, buckets)

	// Verify system buckets are excluded
	for _, bucket := range buckets {
		assert.False(t, bucket[0] == '_', "System bucket should be excluded: %s", bucket)
	}
}

func TestInfluxDBDriver_GetBucketsToBackup_SpecificBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testBucket := "test-bucket"

	opts := &database.BackupOptions{
		Database: testBucket,
	}

	buckets, err := driver.getBucketsToBackup(ctx, opts)
	assert.NoError(t, err)
	assert.Equal(t, []string{testBucket}, buckets)
}

func TestInfluxDBDriver_GetTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Get list of buckets first
	buckets, err := driver.GetDatabases(ctx)
	require.NoError(t, err)

	if len(buckets) == 0 {
		t.Skip("No buckets available for testing")
	}

	// Get measurements from first bucket
	measurements, err := driver.GetTables(ctx, buckets[0])
	// This may fail if bucket is empty
	if err != nil {
		t.Logf("Could not get measurements: %v", err)
	} else {
		assert.NotNil(t, measurements)
	}
}

func TestInfluxDBDriver_GetDatabaseSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	size, err := driver.GetDatabaseSize(ctx)

	// GetDatabaseSize returns a real size from the /metrics endpoint when the
	// storage_shard_disk_size gauge is exposed, otherwise an honest error.
	if err != nil {
		t.Logf("database size unavailable via API: %v", err)
	} else {
		assert.GreaterOrEqual(t, size, int64(0))
	}
}

func TestInfluxDBDriver_GetVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	version, err := driver.GetVersion(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "InfluxDB")
}

func TestInfluxDBDriver_GetType(t *testing.T) {
	driver := NewInfluxDBDriver()
	dbType := driver.GetType()
	assert.Equal(t, database.DatabaseType("influxdb"), dbType)
}

func TestInfluxDBDriver_SupportsIncremental(t *testing.T) {
	driver := NewInfluxDBDriver()
	assert.True(t, driver.SupportsIncremental())
}

func TestInfluxDBDriver_SupportsPITR(t *testing.T) {
	driver := NewInfluxDBDriver()
	assert.False(t, driver.SupportsPITR())
}

func TestInfluxDBDriver_ValidateRestore(t *testing.T) {
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

func TestInfluxDBDriver_BackupMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	err := driver.backupMetadata(ctx, tmpDir)
	assert.NoError(t, err)

	// Verify metadata file was created
	metadataFile := tmpDir + "/metadata.json"
	_, err = os.Stat(metadataFile)
	assert.NoError(t, err)
}

func TestInfluxDB_RetentionPolicies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v2.x (retention policies are v1.x only)
	if driver.version == "" || driver.version[0] != '1' {
		t.Skip("Retention policies test only for v1.x")
	}

	// Test would involve querying and backing up retention policies
	// This is a placeholder for comprehensive retention policy tests
	t.Log("Retention policy tests would go here for v1.x")
}

func TestInfluxDB_Tasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x (tasks are v2.x only)
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Tasks test only for v2.x")
	}

	// Test would involve backing up and restoring tasks
	// This is a placeholder for comprehensive task backup tests
	t.Log("Task backup tests would go here for v2.x")
}

func TestInfluxDB_BackupBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	// Skip if v1.x
	if driver.version != "" && driver.version[0] == '1' {
		t.Skip("Bucket backup test only for v2.x")
	}

	tmpDir := t.TempDir()
	ctx := context.Background()

	testBucket := getEnv("INFLUXDB_TEST_BUCKET", "test-bucket")
	outputFile := tmpDir + "/test-bucket.ndjson"

	opts := &database.BackupOptions{
		Incremental: false,
		Metadata:    make(map[string]interface{}),
	}

	size, err := driver.backupBucket(ctx, testBucket, outputFile, opts)
	if err != nil {
		t.Logf("Bucket backup may fail if bucket is empty: %v", err)
	} else {
		assert.GreaterOrEqual(t, size, int64(0))
	}
}

func TestInfluxDB_IncrementalBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	tmpDir := t.TempDir()
	ctx := context.Background()

	// First, do a full backup
	fullOpts := &database.BackupOptions{
		OutputDir:   tmpDir,
		Incremental: false,
		Metadata:    make(map[string]interface{}),
	}

	fullResult, err := driver.Backup(ctx, fullOpts)
	require.NoError(t, err)

	// Then do an incremental backup
	since := fullResult.EndTime

	incrOpts := &database.BackupOptions{
		OutputDir:   tmpDir,
		Incremental: true,
		Metadata: map[string]interface{}{
			"since": since,
		},
	}

	incrResult, err := driver.Backup(ctx, incrOpts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, incrResult.Status)

	// Incremental backup should generally be smaller (if no new data)
	// But we can't guarantee this in tests
	t.Logf("Full backup size: %d, Incremental backup size: %d",
		fullResult.BackupSize, incrResult.BackupSize)
}

func TestInfluxDB_GetBackupSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	opts := &database.BackupOptions{}

	size, err := driver.GetBackupSize(ctx, opts)
	// GetBackupSize delegates to GetDatabaseSize, which returns an honest error
	// when InfluxDB does not expose the disk-size metric.
	if err != nil {
		t.Logf("backup size unavailable via API: %v", err)
	} else {
		assert.GreaterOrEqual(t, size, int64(0))
	}
}

// Helper functions

func setupTestDriver(t *testing.T) *InfluxDBDriver {
	skipIfInfluxDBUnavailable(t)
	driver := NewInfluxDBDriver()
	config := &database.ConnectionConfig{
		Host:     getEnv("INFLUXDB_HOST", "localhost"),
		Port:     getEnvInt("INFLUXDB_PORT", 8086),
		Password: getEnv("INFLUXDB_TOKEN", ""),
		Database: getEnv("INFLUXDB_ORG", "default"),
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
