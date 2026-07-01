package dr

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

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
	scheduler := NewScheduler(executor, nil)

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
	scheduler := NewScheduler(executor, nil)

	schedule := &TestSchedule{
		Name:           "Invalid Schedule",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "", // Invalid empty cron
		TestConfig:     DefaultTestConfig(),
	}

	err := scheduler.AddSchedule(schedule)
	// The real cron parser rejects an empty/invalid expression.
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestScheduler_AddScheduleMalformedCron(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

	err := scheduler.AddSchedule(&TestSchedule{
		Name:           "Bad Schedule",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "not a cron",
		TestConfig:     DefaultTestConfig(),
	})
	assert.Error(t, err)
}

func TestScheduler_RemoveSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

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
	scheduler := NewScheduler(executor, nil)

	err := scheduler.RemoveSchedule("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestScheduler_GetSchedule(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

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
	scheduler := NewScheduler(executor, nil)

	_, err := scheduler.GetSchedule("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schedule not found")
}

func TestScheduler_ListSchedules(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

	// Add multiple schedules with explicit IDs to avoid nanosecond collision
	for i := 0; i < 3; i++ {
		schedule := &TestSchedule{
			ID:             fmt.Sprintf("schedule-test-%d", i),
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
	scheduler := NewScheduler(executor, nil)

	schedules := scheduler.ListSchedules()
	assert.Empty(t, schedules)
}

func TestScheduler_StartStop(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

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
	scheduler := NewScheduler(executor, nil)

	schedule := &TestSchedule{
		Name:           "Manual Test",
		DatabaseName:   "test-db",
		Enabled:        true,
		CronExpression: "0 2 * * 0",
		TestConfig:     testConfigForTarget(newSQLiteTarget(t)),
	}

	err := scheduler.AddSchedule(schedule)
	require.NoError(t, err)

	ctx := context.Background()

	// Manually run the test against a real restored target.
	result, err := scheduler.RunTest(ctx, schedule.ID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-db", result.DatabaseName)
	assert.True(t, result.Success)
}

func TestScheduler_RunTestNonExistent(t *testing.T) {
	provisioner := NewEnvironmentProvisioner()
	validator := NewValidator()
	executor := NewTestExecutor(provisioner, validator)
	scheduler := NewScheduler(executor, nil)

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
		ID:                       "schedule-123",
		Name:                     "Weekly DR Test",
		DatabaseName:             "prod-db",
		Enabled:                  true,
		CronExpression:           "0 2 * * 0",
		LastRun:                  &now,
		NextRun:                  &nextRun,
		TestConfig:               DefaultTestConfig(),
		NotifyOnSuccess:          true,
		NotifyOnFailure:          true,
		NotificationEmails:       []string{"admin@example.com"},
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
	// "0 2 * * 0" => every Sunday at 02:00.
	now := time.Now()
	nextRun, err := calculateNextRun("0 2 * * 0", now)
	assert.NoError(t, err)
	assert.True(t, nextRun.After(now))
	assert.Equal(t, time.Sunday, nextRun.Weekday())
	assert.Equal(t, 2, nextRun.Hour())
	assert.Equal(t, 0, nextRun.Minute())
}

func TestCalculateNextRun_Invalid(t *testing.T) {
	_, err := calculateNextRun("", time.Now())
	assert.Error(t, err)

	_, err = calculateNextRun("bogus expression", time.Now())
	assert.Error(t, err)
}

func TestCalculateNextRun_Interval(t *testing.T) {
	// Standard cron supports @every style descriptors via ParseStandard.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	next, err := calculateNextRun("@daily", from)
	assert.NoError(t, err)
	assert.Equal(t, from.Add(24*time.Hour), next)
}
