// Package scheduler provides cron-based backup scheduling
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/logger"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// Scheduler manages scheduled backup jobs
type Scheduler struct {
	cron     *cron.Cron
	jobs     map[string]*ScheduledJob
	jobsMux  sync.RWMutex
	engine   *backup.Engine
	logger   *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
	runMux   sync.Mutex
}

// ScheduledJob represents a scheduled backup job
type ScheduledJob struct {
	ID          string
	Name        string
	Schedule    string // Cron expression
	BackupOpts  *backup.CreateOptions
	Enabled     bool
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int64
	FailCount   int64
	EntryID     cron.EntryID
	Retries     int
	RetryDelay  time.Duration
	Timeout     time.Duration
	Tags        map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JobResult represents the result of a scheduled job execution
type JobResult struct {
	JobID      string
	Success    bool
	StartTime  time.Time
	EndTime    time.Time
	BackupID   string
	Error      error
	Duration   time.Duration
	RetryCount int
}

// Config holds scheduler configuration
type Config struct {
	MaxConcurrentJobs int
	JobTimeout        time.Duration
	DefaultRetries    int
	DefaultRetryDelay time.Duration
	EnablePersistence bool
	PersistencePath   string
}

// NewScheduler creates a new scheduler instance
func NewScheduler(engine *backup.Engine, log *logger.Logger, cfg *Config) (*Scheduler, error) {
	if engine == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "backup engine is required")
	}
	if log == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "logger is required")
	}

	if cfg == nil {
		cfg = &Config{
			MaxConcurrentJobs: 5,
			JobTimeout:        24 * time.Hour,
			DefaultRetries:    3,
			DefaultRetryDelay: 5 * time.Minute,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create cron with seconds support and logger
	c := cron.New(cron.WithSeconds(),
		cron.WithLogger(cron.VerbosePrintfLogger(&cronLogger{log: log})))

	return &Scheduler{
		cron:   c,
		jobs:   make(map[string]*ScheduledJob),
		engine: engine,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.runMux.Lock()
	defer s.runMux.Unlock()

	if s.running {
		return pkgErrors.New(pkgErrors.ErrorTypeOperation, "scheduler already running")
	}

	s.logger.Info("Starting backup scheduler")
	s.cron.Start()
	s.running = true

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.runMux.Lock()
	defer s.runMux.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping backup scheduler")

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Cancel all running jobs
	s.cancel()

	// Wait for all jobs to finish (with timeout)
	select {
	case <-ctx.Done():
		s.logger.Info("All scheduled jobs completed")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Scheduler stop timeout, some jobs may still be running")
	}

	s.running = false
	return nil
}

// AddJob adds a new scheduled job
func (s *Scheduler) AddJob(job *ScheduledJob) error {
	if job.ID == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "job ID is required")
	}
	if job.Schedule == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "job schedule is required")
	}
	if job.BackupOpts == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "backup options are required")
	}

	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	// Check if job already exists
	if _, exists := s.jobs[job.ID]; exists {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, fmt.Sprintf("job with ID %s already exists", job.ID))
	}

	// Validate cron expression
	if _, err := cron.ParseStandard(job.Schedule); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeValidation, "invalid cron expression")
	}

	// Set defaults
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	if job.Retries == 0 {
		job.Retries = 3
	}
	if job.RetryDelay == 0 {
		job.RetryDelay = 5 * time.Minute
	}
	if job.Timeout == 0 {
		job.Timeout = 24 * time.Hour
	}

	// Add to cron if enabled
	if job.Enabled {
		entryID, err := s.cron.AddFunc(job.Schedule, s.createJobFunc(job))
		if err != nil {
			return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to add job to cron")
		}
		job.EntryID = entryID

		// Get next run time
		entry := s.cron.Entry(entryID)
		job.NextRun = entry.Next
	}

	s.jobs[job.ID] = job
	s.logger.Info(fmt.Sprintf("Added scheduled job: %s (%s)", job.ID, job.Schedule))

	return nil
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(jobID string) error {
	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("job %s not found", jobID))
	}

	// Remove from cron if it was scheduled
	if job.EntryID > 0 {
		s.cron.Remove(job.EntryID)
	}

	delete(s.jobs, jobID)
	s.logger.Info(fmt.Sprintf("Removed scheduled job: %s", jobID))

	return nil
}

// GetJob retrieves a scheduled job by ID
func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.jobsMux.RLock()
	defer s.jobsMux.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("job %s not found", jobID))
	}

	return job, nil
}

// ListJobs returns all scheduled jobs
func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.jobsMux.RLock()
	defer s.jobsMux.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// UpdateJob updates an existing scheduled job
func (s *Scheduler) UpdateJob(jobID string, updates *ScheduledJob) error {
	s.jobsMux.Lock()
	defer s.jobsMux.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("job %s not found", jobID))
	}

	// Remove old cron entry
	if job.EntryID > 0 {
		s.cron.Remove(job.EntryID)
		job.EntryID = 0
	}

	// Update fields
	if updates.Name != "" {
		job.Name = updates.Name
	}
	if updates.Schedule != "" {
		job.Schedule = updates.Schedule
	}
	if updates.BackupOpts != nil {
		job.BackupOpts = updates.BackupOpts
	}
	if updates.Tags != nil {
		job.Tags = updates.Tags
	}
	if updates.Retries > 0 {
		job.Retries = updates.Retries
	}
	if updates.RetryDelay > 0 {
		job.RetryDelay = updates.RetryDelay
	}
	if updates.Timeout > 0 {
		job.Timeout = updates.Timeout
	}

	job.Enabled = updates.Enabled
	job.UpdatedAt = time.Now()

	// Re-add to cron if enabled
	if job.Enabled {
		entryID, err := s.cron.AddFunc(job.Schedule, s.createJobFunc(job))
		if err != nil {
			return pkgErrors.Wrap(err, pkgErrors.ErrorTypeOperation, "failed to update job in cron")
		}
		job.EntryID = entryID

		entry := s.cron.Entry(entryID)
		job.NextRun = entry.Next
	}

	s.logger.Info(fmt.Sprintf("Updated scheduled job: %s", jobID))

	return nil
}

// EnableJob enables a scheduled job
func (s *Scheduler) EnableJob(jobID string) error {
	return s.UpdateJob(jobID, &ScheduledJob{Enabled: true})
}

// DisableJob disables a scheduled job
func (s *Scheduler) DisableJob(jobID string) error {
	return s.UpdateJob(jobID, &ScheduledJob{Enabled: false})
}

// RunJobNow executes a job immediately (outside its schedule)
func (s *Scheduler) RunJobNow(jobID string) (*JobResult, error) {
	job, err := s.GetJob(jobID)
	if err != nil {
		return nil, err
	}

	return s.executeJob(job), nil
}

// createJobFunc creates a function to be executed by cron
func (s *Scheduler) createJobFunc(job *ScheduledJob) func() {
	return func() {
		result := s.executeJob(job)

		// Update job statistics
		s.jobsMux.Lock()
		job.LastRun = result.StartTime
		job.RunCount++
		if !result.Success {
			job.FailCount++
		}
		// Update next run time
		if job.EntryID > 0 {
			entry := s.cron.Entry(job.EntryID)
			job.NextRun = entry.Next
		}
		s.jobsMux.Unlock()

		// Log result
		if result.Success {
			s.logger.Info(fmt.Sprintf("Scheduled job %s completed successfully (backup ID: %s, duration: %v)",
				job.ID, result.BackupID, result.Duration))
		} else {
			s.logger.Error(fmt.Sprintf("Scheduled job %s failed (duration: %v, retries: %d)",
				job.ID, result.Duration, result.RetryCount), result.Error)
		}
	}
}

// executeJob executes a backup job with retry logic
func (s *Scheduler) executeJob(job *ScheduledJob) *JobResult {
	result := &JobResult{
		JobID:     job.ID,
		StartTime: time.Now(),
	}

	ctx, cancel := context.WithTimeout(s.ctx, job.Timeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= job.Retries; attempt++ {
		if attempt > 0 {
			s.logger.Info(fmt.Sprintf("Retrying job %s (attempt %d/%d)", job.ID, attempt+1, job.Retries+1))
			time.Sleep(job.RetryDelay)
			result.RetryCount++
		}

		// Execute backup
		metadata, err := s.engine.CreateBackup(ctx, job.BackupOpts)
		if err == nil {
			result.Success = true
			result.BackupID = metadata.ID
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		lastErr = err

		// Check if context was cancelled
		if ctx.Err() != nil {
			break
		}
	}

	result.Success = false
	result.Error = lastErr
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.runMux.Lock()
	defer s.runMux.Unlock()
	return s.running
}

// GetJobCount returns the number of scheduled jobs
func (s *Scheduler) GetJobCount() int {
	s.jobsMux.RLock()
	defer s.jobsMux.RUnlock()
	return len(s.jobs)
}

// cronLogger adapts our logger to cron's logger interface
type cronLogger struct {
	log *logger.Logger
}

func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log.Debug(msg)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.log.Error(msg, err)
}

// Printf implements the Printf interface for VerbosePrintfLogger
func (l *cronLogger) Printf(format string, args ...interface{}) {
	l.log.Debug(fmt.Sprintf(format, args...))
}
