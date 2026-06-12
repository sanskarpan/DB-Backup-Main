// Package dr provides disaster recovery testing and validation
package dr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
)

// TestSchedule represents a DR test schedule
type TestSchedule struct {
	ID               string
	Name             string
	DatabaseName     string
	Enabled          bool
	CronExpression   string // e.g., "0 2 * * 0" for every Sunday at 2 AM
	LastRun          *time.Time
	NextRun          *time.Time
	TestConfig       *TestConfig
	NotifyOnSuccess  bool
	NotifyOnFailure  bool
	NotificationEmails []string
	NotificationSlackWebhook string
}

// TestConfig represents configuration for a DR test
type TestConfig struct {
	// Environment settings
	IsolatedNetwork    bool
	EphemeralDatabase  bool
	AutoCleanup        bool
	MaxTestDuration    time.Duration

	// Validation settings
	ValidateSchema     bool
	ValidateRowCounts  bool
	ValidateSampleData bool
	SampleDataPercent  float64 // 0-100

	// Performance thresholds
	MaxRestoreTime     time.Duration // RTO
	MaxDataLoss        time.Duration // RPO

	// Smoke tests
	RunSmokeTests      bool
	CustomQueries      []string
}

// DefaultTestConfig returns default DR test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		IsolatedNetwork:    true,
		EphemeralDatabase:  true,
		AutoCleanup:        true,
		MaxTestDuration:    2 * time.Hour,
		ValidateSchema:     true,
		ValidateRowCounts:  true,
		ValidateSampleData: true,
		SampleDataPercent:  5.0,
		MaxRestoreTime:     1 * time.Hour,
		MaxDataLoss:        5 * time.Minute,
		RunSmokeTests:      true,
		CustomQueries:      []string{},
	}
}

// Scheduler manages DR test schedules
type Scheduler struct {
	mu                sync.RWMutex
	schedules         map[string]*TestSchedule
	executor          *TestExecutor
	running           bool
	stopChan          chan struct{}
	notificationRouter *notification.Router
}

// NewScheduler creates a new DR test scheduler
func NewScheduler(executor *TestExecutor, notificationRouter *notification.Router) *Scheduler {
	if notificationRouter == nil {
		notificationRouter = notification.NewRouter()
	}

	return &Scheduler{
		notificationRouter: notificationRouter,
		schedules: make(map[string]*TestSchedule),
		executor:  executor,
		stopChan:  make(chan struct{}),
	}
}

// AddSchedule adds a new test schedule
func (s *Scheduler) AddSchedule(schedule *TestSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if schedule.ID == "" {
		schedule.ID = generateScheduleID()
	}

	// Calculate next run time
	nextRun, err := calculateNextRun(schedule.CronExpression, time.Now())
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	schedule.NextRun = &nextRun

	s.schedules[schedule.ID] = schedule
	return nil
}

// RemoveSchedule removes a test schedule
func (s *Scheduler) RemoveSchedule(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[scheduleID]; !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	delete(s.schedules, scheduleID)
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *Scheduler) GetSchedule(scheduleID string) (*TestSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return schedule, nil
}

// ListSchedules returns all schedules
func (s *Scheduler) ListSchedules() []*TestSchedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make([]*TestSchedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	go s.run(ctx)
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopChan)
		s.running = false
	}
}

// run is the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case now := <-ticker.C:
			s.checkAndExecuteTests(ctx, now)
		}
	}
}

// checkAndExecuteTests checks for due tests and executes them
func (s *Scheduler) checkAndExecuteTests(ctx context.Context, now time.Time) {
	s.mu.RLock()
	dueSchedules := make([]*TestSchedule, 0)

	for _, schedule := range s.schedules {
		if schedule.Enabled && schedule.NextRun != nil && now.After(*schedule.NextRun) {
			dueSchedules = append(dueSchedules, schedule)
		}
	}
	s.mu.RUnlock()

	// Execute due tests
	for _, schedule := range dueSchedules {
		go s.executeScheduledTest(ctx, schedule)
	}
}

// executeScheduledTest executes a scheduled DR test
func (s *Scheduler) executeScheduledTest(ctx context.Context, schedule *TestSchedule) {
	// Update last run time
	now := time.Now()
	s.mu.Lock()
	schedule.LastRun = &now
	s.mu.Unlock()

	// Execute test
	result, err := s.executor.ExecuteTest(ctx, schedule.DatabaseName, schedule.TestConfig)

	// Calculate next run
	if schedule.Enabled {
		nextRun, err := calculateNextRun(schedule.CronExpression, now)
		if err == nil {
			s.mu.Lock()
			schedule.NextRun = &nextRun
			s.mu.Unlock()
		}
	}

	// Send notifications using the notification service if configured
	if s.executor.notifier != nil {
		notifyErr := s.executor.notifier.NotifyTestResult(ctx, schedule, result)
		if notifyErr != nil {
			// Log error but don't fail the test
			fmt.Printf("Failed to send notification: %v\n", notifyErr)
		}
	} else {
		// Fallback to simple notification
		if err != nil || !result.Success {
			if schedule.NotifyOnFailure {
				s.sendNotification(schedule, result, err)
			}
		} else {
			if schedule.NotifyOnSuccess {
				s.sendNotification(schedule, result, nil)
			}
		}
	}
}

// sendNotification sends test result notifications via email and Slack
func (s *Scheduler) sendNotification(schedule *TestSchedule, result *TestResult, err error) {
	// Determine notification level and status
	var level notification.NotificationLevel
	status := "SUCCESS"

	if err != nil || (result != nil && !result.Success) {
		status = "FAILED"
		level = notification.LevelError
	} else {
		level = notification.LevelSuccess
	}

	// Build notification message
	title := fmt.Sprintf("DR Test %s - %s", status, schedule.Name)
	message := fmt.Sprintf("Disaster Recovery test for database '%s' has %s", schedule.DatabaseName, status)

	// Add error details if failed
	if err != nil {
		message += fmt.Sprintf("\n\nError: %v", err)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"schedule_id":   schedule.ID,
		"schedule_name": schedule.Name,
		"database_name": schedule.DatabaseName,
		"status":        status,
	}

	if result != nil {
		metadata["duration_seconds"] = result.Duration.Seconds()
		metadata["start_time"] = result.StartTime.Format(time.RFC3339)
		metadata["end_time"] = result.EndTime.Format(time.RFC3339)

		if result.RestoreStats != nil {
			metadata["restore_duration"] = result.RestoreStats.Duration.Seconds()
			metadata["restore_size_bytes"] = result.RestoreStats.SizeBytes
		}
	}

	// Build fields for structured display
	var fields []*notification.Field
	if result != nil {
		fields = append(fields,
			&notification.Field{
				Title: "Database",
				Value: schedule.DatabaseName,
				Short: true,
			},
			&notification.Field{
				Title: "Duration",
				Value: fmt.Sprintf("%.2f seconds", result.Duration.Seconds()),
				Short: true,
			},
			&notification.Field{
				Title: "Start Time",
				Value: result.StartTime.Format(time.RFC3339),
				Short: true,
			},
			&notification.Field{
				Title: "End Time",
				Value: result.EndTime.Format(time.RFC3339),
				Short: true,
			},
		)

		if result.RestoreStats != nil {
			fields = append(fields,
				&notification.Field{
					Title: "Restore Duration",
					Value: fmt.Sprintf("%.2f seconds", result.RestoreStats.Duration.Seconds()),
					Short: true,
				},
				&notification.Field{
					Title: "Restore Size",
					Value: fmt.Sprintf("%.2f MB", float64(result.RestoreStats.SizeBytes)/(1024*1024)),
					Short: true,
				},
			)
		}

		// Add validation results
		if len(result.Validations) > 0 {
			validationSummary := ""
			for _, v := range result.Validations {
				status := "✓"
				if !v.Success {
					status = "✗"
				}
				validationSummary += fmt.Sprintf("%s %s\n", status, v.Name)
			}
			fields = append(fields, &notification.Field{
				Title: "Validations",
				Value: validationSummary,
				Short: false,
			})
		}
	}

	// Set color based on status
	color := "#36a64f" // Green for success
	if status == "FAILED" {
		color = "#ff0000" // Red for failure
	}

	// Create notification
	notif := &notification.Notification{
		Title:     title,
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
		Metadata:  metadata,
		Tags:      []string{"dr-test", status},
		Attachments: []*notification.Attachment{
			{
				Title:  title,
				Text:   message,
				Color:  color,
				Fields: fields,
			},
		},
	}

	// Send notification
	ctx := context.Background()
	if err := s.notificationRouter.Send(ctx, notif); err != nil {
		// Log error but don't fail the DR test
		fmt.Printf("Failed to send DR test notification: %v\n", err)
	}
}

// calculateNextRun calculates the next run time based on cron expression
// Simplified cron parser - in production, use a proper cron library
func calculateNextRun(cronExpr string, from time.Time) (time.Time, error) {
	// Simplified implementation - add 1 week by default
	// In production, use github.com/robfig/cron or similar
	return from.Add(7 * 24 * time.Hour), nil
}

// generateScheduleID generates a unique schedule ID
func generateScheduleID() string {
	return fmt.Sprintf("schedule-%d", time.Now().UnixNano())
}

// RunTest manually triggers a DR test for a schedule
func (s *Scheduler) RunTest(ctx context.Context, scheduleID string) (*TestResult, error) {
	schedule, err := s.GetSchedule(scheduleID)
	if err != nil {
		return nil, err
	}

	return s.executor.ExecuteTest(ctx, schedule.DatabaseName, schedule.TestConfig)
}
