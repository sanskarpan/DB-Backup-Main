package multicloud

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
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
	activeProvider  StorageProvider
	monitoring      bool
	stopChan        chan struct{}
}

// HealthChecker probes the reachability of a storage backend.
type HealthChecker interface {
	// Probe performs a lightweight reachability check against the backend,
	// returning a non-nil error when the backend is unreachable.
	Probe(ctx context.Context) error
}

// storageProber adapts a storage.Provider to the HealthChecker interface by
// performing a lightweight existence check against a probe path.
type storageProber struct {
	provider  storage.Provider
	probePath string
}

// NewStorageHealthChecker returns a HealthChecker that probes the given storage
// provider for reachability using a lightweight existence check. An empty
// probePath falls back to a default sentinel path.
func NewStorageHealthChecker(provider storage.Provider, probePath string) HealthChecker {
	if probePath == "" {
		probePath = ".multicloud-healthcheck"
	}
	return &storageProber{provider: provider, probePath: probePath}
}

// Probe checks the reachability of the underlying storage provider. A missing
// probe object is treated as reachable; only transport-level errors are treated
// as unreachable.
func (p *storageProber) Probe(ctx context.Context) error {
	if p.provider == nil {
		return errors.New("multicloud: storage provider is nil")
	}
	if _, err := p.provider.Exists(ctx, p.probePath); err != nil {
		return fmt.Errorf("multicloud: probe of %q failed: %w", p.probePath, err)
	}
	return nil
}

// HealthCheckConfig configures health checks for a provider.
type HealthCheckConfig struct {
	Provider         StorageProvider
	Checker          HealthChecker
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold int
	SuccessThreshold int
	CurrentFailures  int
	Healthy          bool
	LastCheck        time.Time
	LastLatency      time.Duration
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

// SetActiveProvider sets the currently active (primary) provider that failover
// re-routes away from when it becomes unhealthy.
func (fm *FailoverManager) SetActiveProvider(provider StorageProvider) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.activeProvider = provider
}

// GetActiveProvider returns the currently active (primary) provider.
func (fm *FailoverManager) GetActiveProvider() StorageProvider {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.activeProvider
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

// performHealthCheck performs a real health probe of a single provider,
// measuring latency and recording healthy/unhealthy state. When the probe
// causes the provider to transition to unhealthy, an automatic failover is
// triggered.
func (fm *FailoverManager) performHealthCheck(ctx context.Context, config *HealthCheckConfig) {
	checkCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	start := time.Now()
	err := fm.checkProviderHealth(checkCtx, config)
	latency := time.Since(start)

	fm.mu.Lock()
	config.LastCheck = time.Now()
	config.LastLatency = latency
	becameUnhealthy := applyProbeResult(config, err)
	provider := config.Provider
	fm.mu.Unlock()

	if becameUnhealthy {
		reason := fmt.Sprintf("provider %s failed health check %d times: %v",
			provider, config.FailureThreshold, err)
		_, _ = fm.triggerFailover(ctx, provider, reason)
	}
}

// applyProbeResult updates a config's failure counters from a probe result and
// reports whether the provider just transitioned to an unhealthy state. The
// caller must hold fm.mu.
func applyProbeResult(config *HealthCheckConfig, err error) bool {
	if err != nil {
		config.LastError = err
		config.CurrentFailures++
		if config.CurrentFailures >= config.FailureThreshold && config.Healthy {
			config.Healthy = false
			return true
		}
		return false
	}

	config.LastError = nil
	if config.Healthy {
		config.CurrentFailures = 0
		return false
	}
	config.CurrentFailures--
	if config.CurrentFailures <= 0 {
		config.Healthy = true
		config.CurrentFailures = 0
	}
	return false
}

// checkProviderHealth probes a provider for reachability using its registered
// HealthChecker, returning a non-nil error when the backend is unreachable or
// when no checker is configured.
func (fm *FailoverManager) checkProviderHealth(ctx context.Context, config *HealthCheckConfig) error {
	if config.Checker == nil {
		return fmt.Errorf("multicloud: no health checker registered for provider %s", config.Provider)
	}
	return config.Checker.Probe(ctx)
}

// failoverTarget is a candidate destination for a failover, optionally sourced
// from a specific failover rule.
type failoverTarget struct {
	provider StorageProvider
	ruleID   string
}

// triggerFailover re-routes away from an unhealthy provider to a healthy
// destination. On success it switches the active provider (when the failed
// provider is the active one, or none is set) and records the transition. It
// returns an honest error when no healthy destination is available.
func (fm *FailoverManager) triggerFailover(ctx context.Context, provider StorageProvider, reason string) (*FailoverEvent, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	event := &FailoverEvent{
		ID:              uid.New("failover"),
		TriggerProvider: provider,
		Reason:          reason,
		Timestamp:       time.Now(),
	}

	target, ok := fm.selectHealthyTarget(ctx, fm.failoverTargets(provider))
	if !ok {
		event.Error = fmt.Errorf("multicloud: no healthy failover target available for provider %s", provider)
		fm.activeFailovers[event.ID] = event
		return event, event.Error
	}

	event.RuleID = target.ruleID
	event.TargetProvider = target.provider
	event.Success = true
	if provider == fm.activeProvider || fm.activeProvider == "" {
		fm.activeProvider = target.provider
	}
	fm.activeFailovers[event.ID] = event
	return event, nil
}

// failoverTargets returns ordered failover candidates for a failed provider.
// Enabled rules matching the provider take precedence; otherwise all other
// registered providers are used as candidates. The caller must hold fm.mu.
func (fm *FailoverManager) failoverTargets(provider StorageProvider) []failoverTarget {
	targets := make([]failoverTarget, 0)
	matched := false
	for _, rule := range fm.failoverRules {
		if !rule.Enabled || rule.TriggerProvider != provider {
			continue
		}
		matched = true
		for _, tp := range rule.TargetProviders {
			targets = append(targets, failoverTarget{provider: tp, ruleID: rule.ID})
		}
	}
	if !matched {
		targets = fm.registeredTargets(provider)
	}
	return targets
}

// registeredTargets returns failover candidates drawn from all registered
// health checks except the failed provider, ordered by the cloud selector's
// score when available. The caller must hold fm.mu.
func (fm *FailoverManager) registeredTargets(failed StorageProvider) []failoverTarget {
	candidates := make([]failoverTarget, 0, len(fm.healthChecks))
	for provider := range fm.healthChecks {
		if provider == failed {
			continue
		}
		candidates = append(candidates, failoverTarget{provider: provider})
	}
	fm.orderBySelector(candidates)
	return candidates
}

// orderBySelector orders candidates by descending cloud-selector score, falling
// back to a deterministic ordering by provider name.
func (fm *FailoverManager) orderBySelector(targets []failoverTarget) {
	if fm.selector == nil {
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].provider < targets[j].provider
		})
		return
	}
	sort.Slice(targets, func(i, j int) bool {
		si, _ := fm.selector.BestScoreForProvider(targets[i].provider)
		sj, _ := fm.selector.BestScoreForProvider(targets[j].provider)
		if si == sj {
			return targets[i].provider < targets[j].provider
		}
		return si > sj
	})
}

// selectHealthyTarget returns the first candidate that is currently healthy,
// verifying reachability with an on-demand probe when a checker is available.
// The caller must hold fm.mu.
func (fm *FailoverManager) selectHealthyTarget(ctx context.Context, targets []failoverTarget) (failoverTarget, bool) {
	for _, t := range targets {
		hc, ok := fm.healthChecks[t.provider]
		if !ok || !hc.Healthy {
			continue
		}
		if hc.Checker != nil && hc.Checker.Probe(ctx) != nil {
			continue
		}
		return t, true
	}
	return failoverTarget{}, false
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
