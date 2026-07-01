package dynamodb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfAWSUnavailable(t *testing.T) {
	t.Helper()
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
		t.Skip("AWS credentials not available (set AWS_ACCESS_KEY_ID or AWS_PROFILE)")
	}
}

func TestDynamoDBDriver_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfAWSUnavailable(t)

	driver := NewDynamoDBDriver()
	config := &database.ConnectionConfig{
		Host: getEnv("AWS_REGION", "us-east-1"),
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	defer driver.Disconnect()

	assert.NoError(t, err)
	assert.NotNil(t, driver.client)
	assert.NotNil(t, driver.pitrManager)
}

func TestDynamoDBDriver_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)

	err := driver.Disconnect()
	assert.NoError(t, err)
}

func TestDynamoDBDriver_Ping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	err := driver.Ping(ctx)
	assert.NoError(t, err)
}

func TestDynamoDBDriver_Backup_SingleTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	opts := &database.BackupOptions{
		IncludeSchemas: []string{testTable},
		Metadata:       make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.NotEmpty(t, result.BackupPath)
	assert.Contains(t, result.Metadata, "backup_arns")
	assert.Contains(t, result.Metadata, "table_count")
	assert.Equal(t, 1, result.Metadata["table_count"])
}

func TestDynamoDBDriver_Backup_MultiTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Get all tables
	tables, err := driver.getTablesToBackup(ctx, &database.BackupOptions{})
	require.NoError(t, err)

	if len(tables) < 2 {
		t.Skip("Need at least 2 tables for this test")
	}

	opts := &database.BackupOptions{
		IncludeSchemas: tables[:2], // First two tables
		Metadata:       make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, result.Status)
	assert.Equal(t, 2, result.Metadata["table_count"])
}

func TestDynamoDBDriver_CreateTableBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")
	backupID := "test_backup_" + time.Now().Format("20060102150405")

	backupARN, err := driver.createTableBackup(ctx, testTable, backupID)
	assert.NoError(t, err)
	assert.NotEmpty(t, backupARN)
	assert.Contains(t, backupARN, "arn:aws:dynamodb")
}

func TestDynamoDBDriver_WaitForBackup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")
	backupID := "test_wait_" + time.Now().Format("20060102150405")

	// Create backup
	backupARN, err := driver.createTableBackup(ctx, testTable, backupID)
	require.NoError(t, err)

	// Wait for it to complete
	err = driver.waitForBackup(ctx, backupARN)
	assert.NoError(t, err)
}

func TestDynamoDBDriver_GetTablesToBackup_AllTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	opts := &database.BackupOptions{}

	tables, err := driver.getTablesToBackup(ctx, opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, tables)
}

func TestDynamoDBDriver_GetTablesToBackup_SpecificTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	opts := &database.BackupOptions{
		IncludeSchemas: []string{testTable},
	}

	tables, err := driver.getTablesToBackup(ctx, opts)
	assert.NoError(t, err)
	assert.Equal(t, []string{testTable}, tables)
}

func TestDynamoDBDriver_Restore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// First create a backup
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")
	opts := &database.BackupOptions{
		IncludeSchemas: []string{testTable},
		Metadata:       make(map[string]interface{}),
	}

	backupResult, err := driver.Backup(ctx, opts)
	require.NoError(t, err)

	arns, ok := backupResult.Metadata["backup_arns"].([]string)
	require.True(t, ok, "backup result should expose backup_arns")
	require.NotEmpty(t, arns)

	// Then restore into a brand-new table (RestoreTableFromBackup always
	// creates a new table; it cannot overwrite the source).
	restoreOpts := &database.RestoreOptions{
		SourceBackup: arns[0],
		Database:     testTable + "_restored_" + time.Now().Format("20060102_150405"),
	}

	restoreResult, err := driver.Restore(ctx, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, restoreResult.Status)
	assert.NotEmpty(t, restoreResult.RestoredTables)
}

func TestDynamoDBDriver_GetDatabaseSize(t *testing.T) {
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

func TestDynamoDBDriver_GetVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	version, err := driver.GetVersion(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "DynamoDB")
}

func TestDynamoDBDriver_EnablePITR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	err := driver.EnablePITR(ctx, testTable)
	assert.NoError(t, err)

	// Verify PITR is enabled
	enabled, err := driver.pitrManager.IsPITREnabled(ctx, testTable)
	assert.NoError(t, err)
	assert.True(t, enabled)
}

func TestDynamoDBDriver_DisablePITR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	// First enable it
	err := driver.EnablePITR(ctx, testTable)
	require.NoError(t, err)

	// Then disable it
	err = driver.DisablePITR(ctx, testTable)
	assert.NoError(t, err)

	// Verify PITR is disabled
	enabled, err := driver.pitrManager.IsPITREnabled(ctx, testTable)
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestDynamoDBDriver_ExportToS3(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")
	s3Bucket := getEnv("S3_BUCKET", "test-backup-bucket")
	s3Prefix := "dynamodb-exports/"

	exportARN, err := driver.ExportToS3(ctx, testTable, s3Bucket, s3Prefix)
	if err != nil {
		// S3 export may not be available in all regions/accounts
		t.Skip("S3 export not available")
	}

	assert.NotEmpty(t, exportARN)
	assert.Contains(t, exportARN, "arn:aws:dynamodb")
}

func TestDynamoDBDriver_GetType(t *testing.T) {
	driver := NewDynamoDBDriver()
	dbType := driver.GetType()
	assert.Equal(t, database.DatabaseType("dynamodb"), dbType)
}

func TestDynamoDBDriver_SupportsIncremental(t *testing.T) {
	driver := NewDynamoDBDriver()
	assert.False(t, driver.SupportsIncremental())
}

func TestDynamoDBDriver_SupportsPITR(t *testing.T) {
	driver := NewDynamoDBDriver()
	assert.True(t, driver.SupportsPITR())
}

// PITR Manager Tests

func TestPITRManager_RestoreToPIT(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	// Enable PITR first
	err := driver.EnablePITR(ctx, testTable)
	require.NoError(t, err)

	// Wait a bit for PITR to be fully enabled
	time.Sleep(5 * time.Second)

	// Get recovery range
	start, end, err := driver.pitrManager.GetRecoveryRange(ctx, testTable)
	if err != nil {
		t.Skip("PITR not fully enabled yet")
	}

	// Restore to a time within the range
	targetTime := start.Add(time.Duration(end.Sub(start).Seconds()/2) * time.Second)

	restoreOpts := &database.RestoreOptions{
		SourceDatabase: testTable,
		PointInTime:    &targetTime,
	}

	result, err := driver.pitrManager.RestoreToPIT(ctx, targetTime, restoreOpts)
	require.NoError(t, err)
	assert.Equal(t, database.RestoreStatusCompleted, result.Status)
	assert.Contains(t, result.Metadata, "source_table")
	assert.Contains(t, result.Metadata, "target_table")
	assert.Contains(t, result.Metadata, "restore_time")
}

func TestPITRManager_GetRecoveryRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	// Enable PITR
	err := driver.EnablePITR(ctx, testTable)
	require.NoError(t, err)

	// Wait a bit for PITR to be fully enabled
	time.Sleep(5 * time.Second)

	start, end, err := driver.pitrManager.GetRecoveryRange(ctx, testTable)
	if err != nil {
		t.Skip("PITR not fully enabled yet")
	}

	assert.NotZero(t, start)
	assert.NotZero(t, end)
	assert.True(t, end.After(start))
}

func TestPITRManager_IsPITREnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	// Check initial state
	enabled, err := driver.pitrManager.IsPITREnabled(ctx, testTable)
	assert.NoError(t, err)

	// Enable PITR
	err = driver.EnablePITR(ctx, testTable)
	require.NoError(t, err)

	// Check it's enabled
	enabled, err = driver.pitrManager.IsPITREnabled(ctx, testTable)
	assert.NoError(t, err)
	assert.True(t, enabled)

	// Disable PITR
	err = driver.DisablePITR(ctx, testTable)
	require.NoError(t, err)

	// Check it's disabled
	enabled, err = driver.pitrManager.IsPITREnabled(ctx, testTable)
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestDynamoDBPITR_RestoreWithInvalidTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()
	testTable := getEnv("DYNAMODB_TEST_TABLE", "test-table")

	// Try to restore to a time in the future (should fail)
	futureTime := time.Now().Add(24 * time.Hour)

	restoreOpts := &database.RestoreOptions{
		SourceDatabase: testTable,
		PointInTime:    &futureTime,
	}

	result, err := driver.pitrManager.RestoreToPIT(ctx, futureTime, restoreOpts)
	assert.Error(t, err)
	assert.Equal(t, database.RestoreStatusFailed, result.Status)
}

func TestDynamoDB_BackupWithNoTables(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	driver := setupTestDriver(t)
	defer driver.Disconnect()

	ctx := context.Background()

	// Try to backup a non-existent table
	opts := &database.BackupOptions{
		IncludeSchemas: []string{"nonexistent_table_12345"},
		Metadata:       make(map[string]interface{}),
	}

	result, err := driver.Backup(ctx, opts)
	assert.Error(t, err)
	assert.Equal(t, database.BackupStatusFailed, result.Status)
}

// Helper functions

func setupTestDriver(t *testing.T) *DynamoDBDriver {
	skipIfAWSUnavailable(t)
	driver := NewDynamoDBDriver()
	config := &database.ConnectionConfig{
		Host: getEnv("AWS_REGION", "us-east-1"),
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
