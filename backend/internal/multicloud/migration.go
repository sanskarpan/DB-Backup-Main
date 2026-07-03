package multicloud

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// MigrationJob represents a cloud migration job.
type MigrationJob struct {
	// ID is the unique migration job identifier
	ID string

	// BackupID is the backup to migrate
	BackupID string

	// SourceProvider is the source cloud provider
	SourceProvider StorageProvider

	// SourceLocation is the source backup location
	SourceLocation string

	// TargetDestinations are the target destinations
	TargetDestinations []*BackupDestination

	// DeleteSource indicates whether to delete from source after migration
	DeleteSource bool

	// VerifyIntegrity indicates whether to verify after migration
	VerifyIntegrity bool

	// Priority for scheduling (higher = more urgent)
	Priority int

	// CreatedAt is when the migration job was created
	CreatedAt time.Time

	// StartedAt is when the migration started
	StartedAt *time.Time

	// CompletedAt is when the migration completed
	CompletedAt *time.Time

	// Status is the current migration status
	Status MigrationStatus

	// Progress is the migration progress (0-100)
	Progress float64

	// Error contains error details if failed
	Error error
}

// MigrationStatus represents the status of a migration.
type MigrationStatus string

const (
	MigrationPending    MigrationStatus = "pending"
	MigrationInProgress MigrationStatus = "in_progress"
	MigrationCompleted  MigrationStatus = "completed"
	MigrationFailed     MigrationStatus = "failed"
	MigrationCancelled  MigrationStatus = "canceled"
)

// MigrationManager manages cloud migrations.
type MigrationManager struct {
	mu            sync.RWMutex
	orchestrator  *MultiCloudOrchestrator
	jobs          map[string]*MigrationJob
	activeJobs    int
	maxConcurrent int
	downloader    Downloader
}

// Downloader is an interface for downloading backups from a provider.
type Downloader interface {
	Download(ctx context.Context, provider StorageProvider, location string) ([]byte, map[string]interface{}, error)
}

// NewMigrationManager creates a new migration manager.
func NewMigrationManager(orchestrator *MultiCloudOrchestrator, downloader Downloader, maxConcurrent int) *MigrationManager {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}

	return &MigrationManager{
		orchestrator:  orchestrator,
		jobs:          make(map[string]*MigrationJob),
		maxConcurrent: maxConcurrent,
		downloader:    downloader,
	}
}

// CreateMigration creates a new migration job.
func (mm *MigrationManager) CreateMigration(job *MigrationJob) error {
	if job == nil {
		return fmt.Errorf("migration job cannot be nil")
	}
	if job.ID == "" {
		return fmt.Errorf("migration job ID cannot be empty")
	}
	if len(job.TargetDestinations) == 0 {
		return fmt.Errorf("no target destinations specified")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.jobs[job.ID]; exists {
		return fmt.Errorf("migration job %s already exists", job.ID)
	}

	job.Status = MigrationPending
	job.CreatedAt = time.Now()
	mm.jobs[job.ID] = job

	return nil
}

// ExecuteMigration executes a migration job.
func (mm *MigrationManager) ExecuteMigration(ctx context.Context, jobID string) error {
	mm.mu.Lock()
	if mm.activeJobs >= mm.maxConcurrent {
		mm.mu.Unlock()
		return fmt.Errorf("maximum concurrent migrations reached (%d)", mm.maxConcurrent)
	}

	job, ok := mm.jobs[jobID]
	if !ok {
		mm.mu.Unlock()
		return fmt.Errorf("migration job %s not found", jobID)
	}

	if job.Status != MigrationPending {
		mm.mu.Unlock()
		return fmt.Errorf("migration job %s is not pending (status: %s)", jobID, job.Status)
	}

	mm.activeJobs++
	mm.mu.Unlock()

	defer func() {
		mm.mu.Lock()
		mm.activeJobs--
		mm.mu.Unlock()
	}()

	// Update status
	now := time.Now()
	mm.updateJob(jobID, func(j *MigrationJob) {
		j.Status = MigrationInProgress
		j.StartedAt = &now
		j.Progress = 0
	})

	// Download from source
	mm.updateProgress(jobID, 10)
	backupData, metadata, err := mm.downloader.Download(ctx, job.SourceProvider, job.SourceLocation)
	if err != nil {
		mm.updateJob(jobID, func(j *MigrationJob) {
			j.Status = MigrationFailed
			j.Error = fmt.Errorf("download failed: %w", err)
		})
		return err
	}

	mm.updateProgress(jobID, 40)

	// Upload to target destinations
	backupJob := &BackupJob{
		ID:            job.ID,
		DatabaseName:  fmt.Sprintf("migration-%s", job.BackupID),
		BackupData:    backupData,
		Metadata:      metadata,
		Destinations:  job.TargetDestinations,
		RequireAll:    false,
		MinSuccessful: 1,
		CreatedAt:     time.Now(),
	}

	_, err = mm.orchestrator.ExecuteBackup(ctx, backupJob)
	if err != nil {
		mm.updateJob(jobID, func(j *MigrationJob) {
			j.Status = MigrationFailed
			j.Error = fmt.Errorf("upload failed: %w", err)
		})
		return err
	}

	mm.updateProgress(jobID, 80)

	// Verify integrity if requested
	if job.VerifyIntegrity {
		verifyResults, err := mm.orchestrator.VerifyBackup(ctx, job.ID)
		if err != nil {
			mm.updateJob(jobID, func(j *MigrationJob) {
				j.Status = MigrationFailed
				j.Error = fmt.Errorf("verification failed: %w", err)
			})
			return err
		}

		// Check if any verification failed
		for provider, err := range verifyResults {
			if err != nil {
				mm.updateJob(jobID, func(j *MigrationJob) {
					j.Status = MigrationFailed
					j.Error = fmt.Errorf("verification failed for %s: %w", provider, err)
				})
				return err
			}
		}
	}

	mm.updateProgress(jobID, 90)

	// Delete from source if requested
	if job.DeleteSource {
		// This would call the source provider's delete method
		// For now, we'll assume it's part of the downloader interface
	}

	// Mark as completed
	completedAt := time.Now()
	mm.updateJob(jobID, func(j *MigrationJob) {
		j.Status = MigrationCompleted
		j.Progress = 100
		j.CompletedAt = &completedAt
	})

	return nil
}

// GetMigration retrieves a migration job by ID.
func (mm *MigrationManager) GetMigration(jobID string) (*MigrationJob, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	job, ok := mm.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("migration job %s not found", jobID)
	}

	return job, nil
}

// ListMigrations returns all migration jobs.
func (mm *MigrationManager) ListMigrations(status MigrationStatus) []*MigrationJob {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	jobs := make([]*MigrationJob, 0)
	for _, job := range mm.jobs {
		if status == "" || job.Status == status {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

// CancelMigration cancels a pending migration.
func (mm *MigrationManager) CancelMigration(jobID string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	job, ok := mm.jobs[jobID]
	if !ok {
		return fmt.Errorf("migration job %s not found", jobID)
	}

	if job.Status != MigrationPending {
		return fmt.Errorf("cannot cancel migration in status %s", job.Status)
	}

	job.Status = MigrationCancelled
	return nil
}

// updateJob updates a job atomically.
func (mm *MigrationManager) updateJob(jobID string, updateFn func(*MigrationJob)) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if job, ok := mm.jobs[jobID]; ok {
		updateFn(job)
	}
}

// updateProgress updates migration progress.
func (mm *MigrationManager) updateProgress(jobID string, progress float64) {
	mm.updateJob(jobID, func(j *MigrationJob) {
		j.Progress = progress
	})
}

// FailoverManager manages automatic failover between cloud providers.
type FailoverManager struct {
	mu              sync.RWMutex
	orchestrator    *MultiCloudOrchestrator
	selector        *CloudSelector
	healthChecks    map[StorageProvider]*HealthCheckConfig
	failoverRules   []*FailoverRule
	activeFailovers map[string]*FailoverEvent
	monitoring      bool
	stopChan        chan struct{}
}

// HealthCheckConfig configures health checks for a provider.
type HealthCheckConfig struct {
	Provider         StorageProvider
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold int
	SuccessThreshold int
	CurrentFailures  int
	Healthy          bool
	LastCheck        time.Time
	LastError        error
}

// FailoverRule defines when and how to failover.
type FailoverRule struct {
	ID              string
	Name            string
	TriggerProvider StorageProvider
	TargetProviders []StorageProvider
	AutoFailback    bool
	Enabled         bool
}

// FailoverEvent represents a failover event.
type FailoverEvent struct {
	ID              string
	RuleID          string
	TriggerProvider StorageProvider
	TargetProvider  StorageProvider
	Reason          string
	Timestamp       time.Time
	Success         bool
	Error           error
}

// NewFailoverManager creates a new failover manager.
func NewFailoverManager(orchestrator *MultiCloudOrchestrator, selector *CloudSelector) *FailoverManager {
	return &FailoverManager{
		orchestrator:    orchestrator,
		selector:        selector,
		healthChecks:    make(map[StorageProvider]*HealthCheckConfig),
		failoverRules:   make([]*FailoverRule, 0),
		activeFailovers: make(map[string]*FailoverEvent),
		stopChan:        make(chan struct{}),
	}
}

// RegisterHealthCheck registers a health check configuration.
func (fm *FailoverManager) RegisterHealthCheck(config *HealthCheckConfig) error {
	if config == nil {
		return fmt.Errorf("health check config cannot be nil")
	}

	if config.Interval <= 0 {
		config.Interval = 60 * time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 3
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	config.Healthy = true
	fm.healthChecks[config.Provider] = config

	return nil
}

// AddFailoverRule adds a failover rule.
func (fm *FailoverManager) AddFailoverRule(rule *FailoverRule) error {
	if rule == nil {
		return fmt.Errorf("failover rule cannot be nil")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.failoverRules = append(fm.failoverRules, rule)
	return nil
}

// StartMonitoring starts health check monitoring.
func (fm *FailoverManager) StartMonitoring(ctx context.Context) error {
	fm.mu.Lock()
	if fm.monitoring {
		fm.mu.Unlock()
		return fmt.Errorf("monitoring already started")
	}
	fm.monitoring = true
	fm.mu.Unlock()

	go fm.monitorLoop(ctx)

	return nil
}

// StopMonitoring stops health check monitoring.
func (fm *FailoverManager) StopMonitoring() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.monitoring {
		close(fm.stopChan)
		fm.monitoring = false
	}
}

// monitorLoop runs the health check monitoring loop.
func (fm *FailoverManager) monitorLoop(ctx context.Context) {
	for {
		select {
		case <-fm.stopChan:
			return
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second): // Check every 10 seconds
			fm.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks performs health checks on all registered providers.
func (fm *FailoverManager) performHealthChecks(ctx context.Context) {
	fm.mu.Lock()
	configs := make([]*HealthCheckConfig, 0, len(fm.healthChecks))
	for _, config := range fm.healthChecks {
		// Check if it's time to perform health check
		if time.Since(config.LastCheck) >= config.Interval {
			configs = append(configs, config)
		}
	}
	fm.mu.Unlock()

	for _, config := range configs {
		fm.performHealthCheck(ctx, config)
	}
}

// performHealthCheck performs a health check on a single provider.
func (fm *FailoverManager) performHealthCheck(ctx context.Context, config *HealthCheckConfig) {
	checkCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Simulate health check (in real implementation, this would ping the provider)
	err := fm.checkProviderHealth(checkCtx, config.Provider)

	fm.mu.Lock()
	defer fm.mu.Unlock()

	config.LastCheck = time.Now()

	if err != nil {
		config.LastError = err
		config.CurrentFailures++

		if config.CurrentFailures >= config.FailureThreshold && config.Healthy {
			// Provider is now unhealthy
			config.Healthy = false
			fm.triggerFailover(config.Provider, fmt.Sprintf("health check failed %d times: %v", config.CurrentFailures, err))
		}
	} else {
		config.LastError = nil
		if !config.Healthy {
			config.CurrentFailures--
			if config.CurrentFailures <= 0 {
				// Provider is healthy again
				config.Healthy = true
				config.CurrentFailures = 0
			}
		} else {
			config.CurrentFailures = 0
		}
	}
}

// checkProviderHealth checks if a provider is healthy.
func (fm *FailoverManager) checkProviderHealth(ctx context.Context, provider StorageProvider) error {
	// This would implement actual health checks
	// For now, return nil (healthy)
	return nil
}

// triggerFailover triggers a failover event.
func (fm *FailoverManager) triggerFailover(provider StorageProvider, reason string) {
	// Find applicable failover rules
	for _, rule := range fm.failoverRules {
		if !rule.Enabled {
			continue
		}

		if rule.TriggerProvider == provider {
			// Execute failover
			event := &FailoverEvent{
				ID:              uid.New("failover"),
				RuleID:          rule.ID,
				TriggerProvider: provider,
				Reason:          reason,
				Timestamp:       time.Now(),
			}

			// Select best target provider
			if len(rule.TargetProviders) > 0 {
				// Use first healthy target provider
				for _, targetProvider := range rule.TargetProviders {
					if hc, ok := fm.healthChecks[targetProvider]; ok && hc.Healthy {
						event.TargetProvider = targetProvider
						event.Success = true
						break
					}
				}
			}

			fm.activeFailovers[event.ID] = event
		}
	}
}

// GetHealthStatus returns the health status of all providers.
func (fm *FailoverManager) GetHealthStatus() map[StorageProvider]*HealthCheckConfig {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	status := make(map[StorageProvider]*HealthCheckConfig)
	for provider, config := range fm.healthChecks {
		status[provider] = config
	}

	return status
}

// GetFailoverEvents returns recent failover events.
func (fm *FailoverManager) GetFailoverEvents() []*FailoverEvent {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	events := make([]*FailoverEvent, 0, len(fm.activeFailovers))
	for _, event := range fm.activeFailovers {
		events = append(events, event)
	}

	return events
}
