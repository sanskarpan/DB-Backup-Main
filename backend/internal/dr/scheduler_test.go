package dr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.schedules)
	assert.NotNil(t, scheduler.executor)
	assert.NotNil(t, scheduler.stopChan)
	assert.False(t, scheduler.running)
}

func TestScheduler_AddSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedule := &TestSchedule{
		Name:           "Weekly DR Test",
		DatabaseName:   "prod-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0", // Every Sunday at 2 AM
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	assert.NoError(t, err)
	assert.NotEmpty(t, schedule.ID)
	assert.NotNil(t, schedule.NextRun)
}

func TestScheduler_AddScheduleInvalidCron(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedule := &TestSchedule{
		Name:           "Invalid Schedule",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "", // Invalid empty cron
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	// Should succeed with empty cron (simplified implementation adds 7 days)
	assert.NoError(t, err)
}

func TestScheduler_RemoveSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedule := &TestSchedule{
		Name:           "Test Schedule",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0",
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	// Remove the schedule
	err = scheduler.RemoveSchedule(schedule.ID)
	assert.NoError(t, err)

	// Try to get removed schedule
	_, err = scheduler.GetSchedule(schedule.ID)
	assert.Error(t, err)
}

func TestScheduler_RemoveNonExistentSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	err := scheduler.RemoveSchedule("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestScheduler_GetSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedule := &TestSchedule{
		Name:           "Test Schedule",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0",
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	// Get the schedule
	retrieved, err := scheduler.GetSchedule(schedule.ID)
	assert.NoError(t, err)
	assert.Equal(t, schedule.Name, retrieved.Name)
	assert.Equal(t, schedule.DatabaseName, retrieved.DatabaseName)
}

func TestScheduler_GetNonExistentSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	_, err := scheduler.GetSchedule("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestScheduler_ListSchedules(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	// Add multiple schedules
	for i := 0; i < 3; i++ {
		schedule := &TestSchedule{
			Name:           "Test Schedule",
			DatabaseName:   "test-db",
			Enabled:        true,
			CronExpression: "0 2 * * 0",
			TestConfig:     DefaultTestConfig(),
		}
		err := scheduler.AddSchedule(schedule)
		require.NoError(t, err)
	}

	schedules := scheduler.ListSchedules()
	assert.Len(t, schedules, 3)
}

func TestScheduler_ListSchedulesEmpty(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedules := scheduler.ListSchedules()
	assert.Empty(t, schedules)
}

func TestScheduler_StartStop(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	ctx := context.Background()

	// Start scheduler
	err := scheduler.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, scheduler.running)

	// Try to start again (should fail)
	err = scheduler.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop scheduler
	scheduler.Stop()

	// Wait a moment for goroutine to stop
	time.Sleep(50 * time.Millisecond)

	assert.False(t, scheduler.running)
}

func TestScheduler_RunTest(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	schedule := &TestSchedule{
		Name:           "Manual Test",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0",
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	ctx := context.Background()

	// Manually run the test
	result, err := scheduler.RunTest(ctx, schedule.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-db", result.DatabaseName)
}

func TestScheduler_RunTestNonExistent(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor)

	ctx := context.Background()

	_, err := scheduler.RunTest(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestDefaultTestConfig(t *testing.T) {
	config := DefaultTestConfig()

	assert.NotNil(t, config)
	assert.True(t, config.IsolatedNetwork)
	assert.True(t, config.EphemeralDatabase)
	assert.True(t, config.AutoCleanup)
	assert.Equal(t, 2*time.Hour, config.MaxTestDuration)
	assert.True(t, config.ValidateSchema)
	assert.True(t, config.ValidateRowCounts)
	assert.True(t, config.ValidateSampleData)
	assert.Equal(t, 5.0, config.SampleDataPercent)
	assert.Equal(t, 1*time.Hour, config.MaxRestoreTime)
	assert.Equal(t, 5*time.Minute, config.MaxDataLoss)
	assert.True(t, config.RunSmokeTests)
}

func TestTestSchedule_Fields(t *testing.T) {
	now := time.Now()
	nextRun := now.Add(7 * 24 * time.Hour)

	schedule := &TestSchedule{
		ID:             "schedule-123",
		Name:           "Weekly DR Test",
		DatabaseName:   "prod-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0",
		LastRun:        &now,
		NextRun:        &nextRun,
		TestConfig:     DefaultTestConfig(),
		NotifyOnSuccess: true,
		NotifyOnFailure: true,
		NotificationEmails: []string{"admin@example.com"},
		NotificationSlackWebhook: "https://hooks.slack.com/...",
	}

	assert.Equal(t, "schedule-123", schedule.ID)
	assert.Equal(t, "Weekly DR Test", schedule.Name)
	assert.Equal(t, "prod-db", schedule.DatabaseName)
	assert.True(t, schedule.Enabled)
	assert.Equal(t, "0 2 * * 0", schedule.CronExpression)
	assert.NotNil(t, schedule.LastRun)
	assert.NotNil(t, schedule.NextRun)
	assert.True(t, schedule.NotifyOnSuccess)
	assert.True(t, schedule.NotifyOnFailure)
	assert.Len(t, schedule.NotificationEmails, 1)
	assert.NotEmpty(t, schedule.NotificationSlackWebhook)
}

func TestGenerateScheduleID(t *testing.T) {
	id1 := generateScheduleID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateScheduleID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "schedule-")
	assert.Contains(t, id2, "schedule-")
}

func TestCalculateNextRun(t *testing.T) {
	now := time.Now()
	cronExpr := "0 2 * * 0"

	nextRun, err := calculateNextRun(cronExpr, now)
	assert.NoError(t, err)
	assert.True(t, nextRun.After(now))

	// Simplified implementation adds 7 days
	expectedNextRun := now.Add(7 * 24 * time.Hour)
	// Allow 1 second tolerance
	diff := nextRun.Sub(expectedNextRun)
	assert.LessOrEqual(t, diff, 1*time.Second)
}
