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

	config := DefaultTestConfig()
	ctx := context.Background()

	result, err := executor.ExecuteTest(ctx, "test-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.TestID)
	assert.Equal(t, "test-db", result.DatabaseName)
	assert.True(t, result.Success)
	assert.NotZero(t, result.Duration)
	assert.NotEmpty(t, result.TestEnvironmentID)
}

func TestTestExecutor_ExecuteTestWithValidation(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := &TestConfig{
		IsolatedNetwork:    true,
		EphemeralDatabase:  true,
		AutoCleanup:        true,
		MaxTestDuration:    30 * time.Minute,
		ValidateSchema:     true,
		ValidateRowCounts:  true,
		ValidateSampleData: true,
		SampleDataPercent:  10.0,
		MaxRestoreTime:     10 * time.Minute,
		MaxDataLoss:        1 * time.Minute,
		RunSmokeTests:      true,
		CustomQueries:      []string{"SELECT COUNT(*) FROM users"},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTest(ctx, "prod-db", config)

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
}

func TestTestExecutor_ExecuteTestAutoCleanup(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := &TestConfig{
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
		AutoCleanup:       true,
		MaxTestDuration:   30 * time.Minute,
		MaxRestoreTime:    10 * time.Minute,
		MaxDataLoss:       1 * time.Minute,
	}

	ctx := context.Background()
	result, err := executor.ExecuteTest(ctx, "test-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.CleanedUp)
}

func TestTestExecutor_ExecuteTestNoCleanup(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := &TestConfig{
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
		AutoCleanup:       false, // Don't cleanup
		MaxTestDuration:   30 * time.Minute,
		MaxRestoreTime:    10 * time.Minute,
		MaxDataLoss:       1 * time.Minute,
	}

	ctx := context.Background()
	result, err := executor.ExecuteTest(ctx, "test-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.CleanedUp)
}

func TestTestExecutor_RTOValidation(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)

	config := &TestConfig{
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
		AutoCleanup:       true,
		MaxTestDuration:   30 * time.Minute,
		MaxRestoreTime:    1 * time.Millisecond, // Very short RTO threshold
		MaxDataLoss:       1 * time.Hour,
	}

	ctx := context.Background()
	result, err := executor.ExecuteTest(ctx, "test-db", config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// RTO likely not met due to very short threshold
	assert.False(t, result.RTOMet)
	assert.Greater(t, result.RTO, config.MaxRestoreTime)
}

func TestTestResult_Fields(t *testing.T) {
	now := time.Now()
	result := &TestResult{
		TestID:           "dr-test-123",
		DatabaseName:     "prod-db",
		StartTime:        now,
		EndTime:          now.Add(5 * time.Minute),
		Duration:         5 * time.Minute,
		Success:          true,
		Error:            nil,
		RestoreStartTime: now.Add(1 * time.Minute),
		RestoreEndTime:   now.Add(4 * time.Minute),
		RestoreDuration:  3 * time.Minute,
		RestoreSize:      1000000000,
		SchemaValid:      true,
		SchemaErrors:     []string{},
		RowCountValid:    true,
		RowCountErrors:   []string{},
		SampleDataValid:  true,
		SampleDataErrors: []string{},
		RTO:              3 * time.Minute,
		RPO:              1 * time.Minute,
		RTOThreshold:     5 * time.Minute,
		RPOThreshold:     2 * time.Minute,
		RTOMet:           true,
		RPOMet:           true,
		SmokeTestsPassed: 5,
		SmokeTestsFailed: 0,
		SmokeTestErrors:  []string{},
		TestEnvironmentID: "env-123",
		CleanedUp:         true,
	}

	assert.Equal(t, "dr-test-123", result.TestID)
	assert.Equal(t, "prod-db", result.DatabaseName)
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)
	assert.Equal(t, 5*time.Minute, result.Duration)
	assert.Equal(t, 3*time.Minute, result.RestoreDuration)
	assert.Equal(t, int64(1000000000), result.RestoreSize)
	assert.True(t, result.SchemaValid)
	assert.True(t, result.RowCountValid)
	assert.True(t, result.SampleDataValid)
	assert.True(t, result.RTOMet)
	assert.True(t, result.RPOMet)
	assert.Equal(t, 5, result.SmokeTestsPassed)
	assert.Equal(t, 0, result.SmokeTestsFailed)
	assert.True(t, result.CleanedUp)
}

func TestGenerateTestID(t *testing.T) {
	id1 := generateTestID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateTestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "dr-test-")
	assert.Contains(t, id2, "dr-test-")
}

func TestNewEnvironmentProvisioner(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()

	assert.NotNil(t, provisioner)
	assert.NotNil(t, provisioner.environments)
	assert.Empty(t, provisioner.environments)
}

func TestEnvironmentProvisioner_ProvisionEnvironment(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	config := &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	}

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, config)

	assert.NoError(t, err)
	assert.NotNil(t, env)
	assert.NotEmpty(t, env.ID)
	assert.Equal(t, "test-db-test", env.DatabaseName)
	assert.Contains(t, env.ConnectionString, "test-db-test")
	assert.True(t, env.IsolatedNetwork)
	assert.True(t, env.EphemeralDatabase)
	assert.Equal(t, "ready", env.Status)
	assert.NotZero(t, env.CreatedAt)
}

func TestEnvironmentProvisioner_CleanupEnvironment(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	config := &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	}

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, config)
	require.NoError(t, err)

	// Cleanup the environment
	err = provisioner.CleanupEnvironment(ctx, env.ID)
	assert.NoError(t, err)

	// Environment should be removed
	_, exists := provisioner.environments[env.ID]
	assert.False(t, exists)
}

func TestEnvironmentProvisioner_CleanupNonExistent(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	ctx := context.Background()

	err := provisioner.CleanupEnvironment(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment not found")
}

func TestEnvironmentProvisioner_MultipleEnvironments(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	ctx := context.Background()

	// Provision multiple environments
	env1, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "db1",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	env2, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "db2",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	assert.NotEqual(t, env1.ID, env2.ID)
	assert.Equal(t, 2, len(provisioner.environments))

	// Cleanup one environment
	err = provisioner.CleanupEnvironment(ctx, env1.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(provisioner.environments))

	// Other environment should still exist
	_, exists := provisioner.environments[env2.ID]
	assert.True(t, exists)
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
	assert.Contains(t, env.ConnectionString, "test-db-test")
	assert.True(t, env.IsolatedNetwork)
	assert.True(t, env.EphemeralDatabase)
	assert.Equal(t, "ready", env.Status)
	assert.Equal(t, now, env.CreatedAt)
}

func TestGenerateEnvironmentID(t *testing.T) {
	id1 := generateEnvironmentID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateEnvironmentID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "env-")
	assert.Contains(t, id2, "env-")
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
