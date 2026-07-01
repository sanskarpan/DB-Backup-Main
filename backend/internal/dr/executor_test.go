package dr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestExecutor(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.provisioner)
	assert.NotNil(t, executor.validator)
}

func TestTestExecutor_ExecuteTest(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	ctx := context.Background()

	result, err := executor.ExecuteTest(ctx, "test-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.TestID)
	assert.Equal(t, "test-db", result.DatabaseName)
	assert.True(t, result.Success)
	assert.NotZero(t, result.Duration)
	assert.NotEmpty(t, result.TestEnvironmentID)
	assert.Greater(t, result.RestoreSize, int64(0))
}

func TestTestExecutor_ExecuteTestNoTarget(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	// A DR test with no configured target must fail with a real error, not
	// simulate success.
	config := DefaultTestConfig() // Target is nil
	result, err := executor.ExecuteTest(context.Background(), "test-db", config)

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "no test target configured")
}

func TestTestExecutor_ExecuteTestMissingBackup(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	target := newSQLiteTarget(t)
	target.BackupPath = "/nonexistent/backup.sql"
	config := testConfigForTarget(target)

	result, err := executor.ExecuteTest(context.Background(), "test-db", config)
	assert.Error(t, err)
	assert.False(t, result.Success)
}

func TestTestExecutor_ExecuteTestWithValidation(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	config.MaxTestDuration = 30 * time.Minute
	config.SampleDataPercent = 100.0
	config.MaxRestoreTime = 10 * time.Minute
	config.MaxDataLoss = 1 * time.Hour
	config.CustomQueries = []string{"SELECT COUNT(*) FROM users"}

	result, err := executor.ExecuteTest(context.Background(), "prod-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.True(t, result.SchemaValid)
	assert.True(t, result.RowCountValid)
	assert.True(t, result.SampleDataValid)
	assert.True(t, result.RTOMet)
	assert.True(t, result.RPOMet)
	assert.Greater(t, result.SmokeTestsPassed, 0)
	assert.Equal(t, 0, result.SmokeTestsFailed)
	// RPO is derived from real timestamps (backup taken ~30s ago).
	assert.Greater(t, result.RPO, time.Duration(0))
}

func TestTestExecutor_ExecuteTestBadCustomQuery(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	config.CustomQueries = []string{"SELECT * FROM table_that_does_not_exist"}

	result, err := executor.ExecuteTest(context.Background(), "prod-db", config)
	assert.NoError(t, err)
	assert.Greater(t, result.SmokeTestsFailed, 0)
	assert.False(t, result.Success)
}

func TestTestExecutor_ExecuteTestAutoCleanup(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	config.AutoCleanup = true

	result, err := executor.ExecuteTest(context.Background(), "test-db", config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.CleanedUp)
}

func TestTestExecutor_ExecuteTestNoCleanup(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	config.AutoCleanup = false

	result, err := executor.ExecuteTest(context.Background(), "test-db", config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.CleanedUp)
}

func TestTestExecutor_RTOValidation(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := testConfigForTarget(newSQLiteTarget(t))
	config.MaxRestoreTime = 1 * time.Nanosecond // Impossibly short RTO threshold
	config.MaxDataLoss = 1 * time.Hour

	result, err := executor.ExecuteTest(context.Background(), "test-db", config)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.RTOMet)
	assert.Greater(t, result.RTO, config.MaxRestoreTime)
}

func TestTestExecutor_RPOFromRealTimestamps(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	target := newSQLiteTarget(t)
	target.BackupTime = time.Now().Add(-2 * time.Hour) // stale backup
	config := testConfigForTarget(target)
	config.MaxDataLoss = 5 * time.Minute // RPO of ~2h must fail

	result, err := executor.ExecuteTest(context.Background(), "test-db", config)
	assert.NoError(t, err)
	assert.False(t, result.RPOMet)
	assert.Greater(t, result.RPO, 1*time.Hour)
	assert.False(t, result.Success)
}

func TestTestResult_Fields(t *testing.T) {
	now := time.Now()
	result := &TestResult{
		TestID:            "dr-test-123",
		DatabaseName:      "prod-db",
		StartTime:         now,
		EndTime:           now.Add(5 * time.Minute),
		Duration:          5 * time.Minute,
		Success:           true,
		RestoreStartTime:  now.Add(1 * time.Minute),
		RestoreEndTime:    now.Add(4 * time.Minute),
		RestoreDuration:   3 * time.Minute,
		RestoreSize:       1000000000,
		SchemaValid:       true,
		RowCountValid:     true,
		SampleDataValid:   true,
		RTO:               3 * time.Minute,
		RPO:               1 * time.Minute,
		RTOThreshold:      5 * time.Minute,
		RPOThreshold:      2 * time.Minute,
		RTOMet:            true,
		RPOMet:            true,
		SmokeTestsPassed:  5,
		SmokeTestsFailed:  0,
		TestEnvironmentID: "env-123",
		CleanedUp:         true,
	}

	assert.Equal(t, "dr-test-123", result.TestID)
	assert.Equal(t, "prod-db", result.DatabaseName)
	assert.Equal(t, now, result.StartTime)
	assert.Equal(t, now.Add(5*time.Minute), result.EndTime)
	assert.Equal(t, 5*time.Minute, result.Duration)
	assert.True(t, result.Success)
	assert.Equal(t, now.Add(1*time.Minute), result.RestoreStartTime)
	assert.Equal(t, now.Add(4*time.Minute), result.RestoreEndTime)
	assert.Equal(t, 3*time.Minute, result.RestoreDuration)
	assert.Equal(t, int64(1000000000), result.RestoreSize)
	assert.True(t, result.SchemaValid)
	assert.True(t, result.RowCountValid)
	assert.True(t, result.SampleDataValid)
	assert.Equal(t, 3*time.Minute, result.RTO)
	assert.Equal(t, 1*time.Minute, result.RPO)
	assert.Equal(t, 5*time.Minute, result.RTOThreshold)
	assert.Equal(t, 2*time.Minute, result.RPOThreshold)
	assert.True(t, result.RTOMet)
	assert.True(t, result.RPOMet)
	assert.Equal(t, 5, result.SmokeTestsPassed)
	assert.Equal(t, 0, result.SmokeTestsFailed)
	assert.Equal(t, "env-123", result.TestEnvironmentID)
	assert.True(t, result.CleanedUp)
}

func TestGenerateTestID(t *testing.T) {
	id1 := generateTestID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateTestID()

	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "dr-test-")
}

func TestNewEnvironmentProvisioner(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	assert.NotNil(t, provisioner)
	assert.NotNil(t, provisioner.environments)
	assert.Empty(t, provisioner.environments)
}

func TestEnvironmentProvisioner_ProvisionEnvironment(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	target := newSQLiteTarget(t)
	config := &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
		Target:            target,
	}

	env, err := provisioner.ProvisionEnvironment(context.Background(), config)
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.NotEmpty(t, env.ID)
	assert.Equal(t, "test-db", env.DatabaseName)
	assert.Equal(t, target.DSN, env.ConnectionString)
	assert.NotNil(t, env.db)
	assert.Equal(t, "ready", env.Status)
	assert.NotZero(t, env.CreatedAt)

	require.NoError(t, provisioner.CleanupEnvironment(context.Background(), env.ID))
}

func TestEnvironmentProvisioner_ProvisionNoTarget(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	_, err := provisioner.ProvisionEnvironment(context.Background(), &ProvisionConfig{
		DatabaseName: "test-db",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no test target configured")
}

func TestEnvironmentProvisioner_CleanupEnvironment(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	env, err := provisioner.ProvisionEnvironment(context.Background(), &ProvisionConfig{
		DatabaseName: "test-db",
		Target:       newSQLiteTarget(t),
	})
	require.NoError(t, err)

	err = provisioner.CleanupEnvironment(context.Background(), env.ID)
	assert.NoError(t, err)

	_, exists := provisioner.environments[env.ID]
	assert.False(t, exists)
}

func TestEnvironmentProvisioner_CleanupNonExistent(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	err := provisioner.CleanupEnvironment(context.Background(), "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment not found")
}

func TestEnvironmentProvisioner_CleanupScrubsTables(t *testing.T) {
	// After cleanup, re-opening the target must show no restored tables.
	target := newSQLiteTarget(t)
	prov, env := provisionRestored(t, target)

	tables, err := listTables(context.Background(), env.db, env.driver, env.DatabaseName)
	require.NoError(t, err)
	assert.NotEmpty(t, tables)

	require.NoError(t, prov.CleanupEnvironment(context.Background(), env.ID))

	// Re-open the same on-disk database and confirm the tables are gone.
	db, err := openTarget(context.Background(), target)
	require.NoError(t, err)
	defer db.Close()
	remaining, err := listTables(context.Background(), db, "sqlite3", "testdb")
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestEnvironmentProvisioner_MultipleEnvironments(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	ctx := context.Background()

	env1, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{DatabaseName: "db1", Target: newSQLiteTarget(t)})
	require.NoError(t, err)
	env2, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{DatabaseName: "db2", Target: newSQLiteTarget(t)})
	require.NoError(t, err)

	assert.NotEqual(t, env1.ID, env2.ID)
	assert.Equal(t, 2, len(provisioner.environments))

	require.NoError(t, provisioner.CleanupEnvironment(ctx, env1.ID))
	assert.Equal(t, 1, len(provisioner.environments))

	_, exists := provisioner.environments[env2.ID]
	assert.True(t, exists)
	require.NoError(t, provisioner.CleanupEnvironment(ctx, env2.ID))
}

func TestTestEnvironment_Fields(t *testing.T) {
	now := time.Now()
	env := &TestEnvironment{
		ID:                "env-123",
		DatabaseName:      "test-db-test",
		ConnectionString:  "postgresql://localhost:5432/test-db-test",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
		CreatedAt:         now,
		Status:            "ready",
	}

	assert.Equal(t, "env-123", env.ID)
	assert.Equal(t, "test-db-test", env.DatabaseName)
	assert.Equal(t, "ready", env.Status)
	assert.Equal(t, now, env.CreatedAt)
}

func TestGenerateEnvironmentID(t *testing.T) {
	id1 := generateEnvironmentID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateEnvironmentID()

	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "env-")
}

func TestProvisionConfig_Fields(t *testing.T) {
	config := &ProvisionConfig{
		DatabaseName:      "prod-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	}

	assert.Equal(t, "prod-db", config.DatabaseName)
	assert.True(t, config.IsolatedNetwork)
	assert.True(t, config.EphemeralDatabase)
}
