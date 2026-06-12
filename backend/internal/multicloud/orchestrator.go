// Package multicloud provides multi-cloud intelligence and portability features
// including simultaneous backups, intelligent provider selection, and automatic failover.
package multicloud

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StorageProvider represents a cloud storage provider
type StorageProvider string

const (
	ProviderAWS   StorageProvider = "aws"
	ProviderAzure StorageProvider = "azure"
	ProviderGCP   StorageProvider = "gcp"
	ProviderLocal StorageProvider = "local"
)

// BackupDestination represents a backup destination configuration
type BackupDestination struct {
	// Provider is the storage provider
	Provider StorageProvider

	// Region is the cloud region
	Region string

	// Bucket is the storage bucket/container name
	Bucket string

	// Path is the path within the bucket
	Path string

	// Credentials are provider-specific credentials
	Credentials map[string]string

	// Priority for failover (higher = preferred)
	Priority int

	// Enabled indicates if this destination is active
	Enabled bool

	// MaxRetries for upload operations
	MaxRetries int

	// Timeout for operations
	Timeout time.Duration
}

// BackupJob represents a backup job to be executed
type BackupJob struct {
	// ID is the unique job identifier
	ID string

	// DatabaseName is the name of the database
	DatabaseName string

	// BackupData is the backup data to upload
	BackupData []byte

	// Metadata is backup metadata
	Metadata map[string]interface{}

	// Destinations are the target destinations
	Destinations []*BackupDestination

	// RequireAll requires all destinations to succeed
	RequireAll bool

	// MinSuccessful is the minimum number of successful uploads required
	MinSuccessful int

	// CreatedAt is the job creation time
	CreatedAt time.Time
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	// JobID is the backup job ID
	JobID string

	// Provider is the storage provider
	Provider StorageProvider

	// Success indicates if the backup succeeded
	Success bool

	// Error contains error details if failed
	Error error

	// Duration is the upload duration
	Duration time.Duration

	// BytesUploaded is the number of bytes uploaded
	BytesUploaded int64

	// Location is the final backup location
	Location string

	// Timestamp is when the backup completed
	Timestamp time.Time
}

// MultiCloudOrchestrator manages multi-cloud backup operations
type MultiCloudOrchestrator struct {
	mu            sync.RWMutex
	destinations  map[string]*BackupDestination
	activeJobs    map[string]*BackupJob
	results       map[string][]*BackupResult
	uploaders     map[StorageProvider]Uploader
	maxConcurrent int
	semaphore     chan struct{}
}

// Uploader is an interface for uploading backups to a provider
type Uploader interface {
	Upload(ctx context.Context, destination *BackupDestination, data []byte, metadata map[string]interface{}) (string, error)
	Delete(ctx context.Context, destination *BackupDestination, location string) error
	Verify(ctx context.Context, destination *BackupDestination, location string) error
}

// NewMultiCloudOrchestrator creates a new multi-cloud orchestrator
func NewMultiCloudOrchestrator(maxConcurrent int) *MultiCloudOrchestrator {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	return &MultiCloudOrchestrator{
		destinations:  make(map[string]*BackupDestination),
		activeJobs:    make(map[string]*BackupJob),
		results:       make(map[string][]*BackupResult),
		uploaders:     make(map[StorageProvider]Uploader),
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// RegisterDestination registers a backup destination
func (mco *MultiCloudOrchestrator) RegisterDestination(id string, destination *BackupDestination) error {
	if id == "" {
		return fmt.Errorf("destination ID cannot be empty")
	}
	if destination == nil {
		return fmt.Errorf("destination cannot be nil")
	}

	mco.mu.Lock()
	defer mco.mu.Unlock()

	mco.destinations[id] = destination
	return nil
}

// RegisterUploader registers an uploader for a provider
func (mco *MultiCloudOrchestrator) RegisterUploader(provider StorageProvider, uploader Uploader) error {
	if uploader == nil {
		return fmt.Errorf("uploader cannot be nil")
	}

	mco.mu.Lock()
	defer mco.mu.Unlock()

	mco.uploaders[provider] = uploader
	return nil
}

// GetDestination retrieves a destination by ID
func (mco *MultiCloudOrchestrator) GetDestination(id string) (*BackupDestination, error) {
	mco.mu.RLock()
	defer mco.mu.RUnlock()

	dest, ok := mco.destinations[id]
	if !ok {
		return nil, fmt.Errorf("destination %s not found", id)
	}

	return dest, nil
}

// ListDestinations returns all registered destinations
func (mco *MultiCloudOrchestrator) ListDestinations() []*BackupDestination {
	mco.mu.RLock()
	defer mco.mu.RUnlock()

	destinations := make([]*BackupDestination, 0, len(mco.destinations))
	for _, dest := range mco.destinations {
		destinations = append(destinations, dest)
	}

	return destinations
}

// ExecuteBackup executes a multi-cloud backup job
func (mco *MultiCloudOrchestrator) ExecuteBackup(ctx context.Context, job *BackupJob) ([]*BackupResult, error) {
	if job == nil {
		return nil, fmt.Errorf("job cannot be nil")
	}
	if len(job.Destinations) == 0 {
		return nil, fmt.Errorf("no destinations specified")
	}

	// Register active job
	mco.mu.Lock()
	mco.activeJobs[job.ID] = job
	mco.mu.Unlock()

	// Execute uploads concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan *BackupResult, len(job.Destinations))
	errorsChan := make(chan error, len(job.Destinations))

	for _, dest := range job.Destinations {
		if !dest.Enabled {
			continue
		}

		wg.Add(1)
		go func(destination *BackupDestination) {
			defer wg.Done()

			// Acquire semaphore for concurrency control
			select {
			case mco.semaphore <- struct{}{}:
				defer func() { <-mco.semaphore }()
			case <-ctx.Done():
				errorsChan <- ctx.Err()
				return
			}

			// Execute upload with retry
			result := mco.executeUpload(ctx, job, destination)
			resultsChan <- result
		}(dest)
	}

	// Wait for all uploads to complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	results := make([]*BackupResult, 0)
	for result := range resultsChan {
		results = append(results, result)
	}

	// Store results
	mco.mu.Lock()
	mco.results[job.ID] = results
	delete(mco.activeJobs, job.ID)
	mco.mu.Unlock()

	// Check if requirements are met
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	if job.RequireAll && successCount < len(job.Destinations) {
		return results, fmt.Errorf("not all destinations succeeded: %d/%d", successCount, len(job.Destinations))
	}

	if successCount < job.MinSuccessful {
		return results, fmt.Errorf("minimum successful uploads not met: %d/%d", successCount, job.MinSuccessful)
	}

	return results, nil
}

// executeUpload executes a single upload with retry logic
func (mco *MultiCloudOrchestrator) executeUpload(ctx context.Context, job *BackupJob, dest *BackupDestination) *BackupResult {
	startTime := time.Now()
	result := &BackupResult{
		JobID:     job.ID,
		Provider:  dest.Provider,
		Timestamp: time.Now(),
	}

	// Get uploader for provider
	mco.mu.RLock()
	uploader, ok := mco.uploaders[dest.Provider]
	mco.mu.RUnlock()

	if !ok {
		result.Error = fmt.Errorf("no uploader registered for provider %s", dest.Provider)
		return result
	}

	// Set default max retries
	maxRetries := dest.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Retry loop
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result
			}
		}

		// Create upload context with timeout
		uploadCtx := ctx
		if dest.Timeout > 0 {
			var cancel context.CancelFunc
			uploadCtx, cancel = context.WithTimeout(ctx, dest.Timeout)
			defer cancel()
		}

		// Execute upload
		location, err := uploader.Upload(uploadCtx, dest, job.BackupData, job.Metadata)
		if err == nil {
			// Success
			result.Success = true
			result.Location = location
			result.BytesUploaded = int64(len(job.BackupData))
			result.Duration = time.Since(startTime)
			return result
		}

		lastErr = err
	}

	// All retries failed
	result.Error = fmt.Errorf("upload failed after %d retries: %w", maxRetries, lastErr)
	result.Duration = time.Since(startTime)
	return result
}

// GetJobResults retrieves results for a backup job
func (mco *MultiCloudOrchestrator) GetJobResults(jobID string) ([]*BackupResult, error) {
	mco.mu.RLock()
	defer mco.mu.RUnlock()

	results, ok := mco.results[jobID]
	if !ok {
		return nil, fmt.Errorf("no results found for job %s", jobID)
	}

	return results, nil
}

// GetActiveJobs returns all currently active jobs
func (mco *MultiCloudOrchestrator) GetActiveJobs() []*BackupJob {
	mco.mu.RLock()
	defer mco.mu.RUnlock()

	jobs := make([]*BackupJob, 0, len(mco.activeJobs))
	for _, job := range mco.activeJobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// DeleteBackup deletes a backup from all destinations
func (mco *MultiCloudOrchestrator) DeleteBackup(ctx context.Context, jobID string) error {
	results, err := mco.GetJobResults(jobID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errors := make([]error, 0)
	var errorsMu sync.Mutex

	for _, result := range results {
		if !result.Success {
			continue
		}

		wg.Add(1)
		go func(r *BackupResult) {
			defer wg.Done()

			// Get uploader
			mco.mu.RLock()
			uploader, ok := mco.uploaders[r.Provider]
			mco.mu.RUnlock()

			if !ok {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("no uploader for provider %s", r.Provider))
				errorsMu.Unlock()
				return
			}

			// Find destination
			var dest *BackupDestination
			mco.mu.RLock()
			for _, d := range mco.destinations {
				if d.Provider == r.Provider {
					dest = d
					break
				}
			}
			mco.mu.RUnlock()

			if dest == nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("destination not found for provider %s", r.Provider))
				errorsMu.Unlock()
				return
			}

			// Delete backup
			if err := uploader.Delete(ctx, dest, r.Location); err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			}
		}(result)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete from %d destinations: %v", len(errors), errors)
	}

	return nil
}

// VerifyBackup verifies a backup across all destinations
func (mco *MultiCloudOrchestrator) VerifyBackup(ctx context.Context, jobID string) (map[StorageProvider]error, error) {
	results, err := mco.GetJobResults(jobID)
	if err != nil {
		return nil, err
	}

	verificationResults := make(map[StorageProvider]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, result := range results {
		if !result.Success {
			continue
		}

		wg.Add(1)
		go func(r *BackupResult) {
			defer wg.Done()

			// Get uploader
			mco.mu.RLock()
			uploader, ok := mco.uploaders[r.Provider]
			mco.mu.RUnlock()

			if !ok {
				mu.Lock()
				verificationResults[r.Provider] = fmt.Errorf("no uploader for provider")
				mu.Unlock()
				return
			}

			// Find destination
			var dest *BackupDestination
			mco.mu.RLock()
			for _, d := range mco.destinations {
				if d.Provider == r.Provider {
					dest = d
					break
				}
			}
			mco.mu.RUnlock()

			if dest == nil {
				mu.Lock()
				verificationResults[r.Provider] = fmt.Errorf("destination not found")
				mu.Unlock()
				return
			}

			// Verify backup
			err := uploader.Verify(ctx, dest, r.Location)
			mu.Lock()
			verificationResults[r.Provider] = err
			mu.Unlock()
		}(result)
	}

	wg.Wait()

	return verificationResults, nil
}

// GetStatistics returns orchestrator statistics
func (mco *MultiCloudOrchestrator) GetStatistics() *OrchestratorStats {
	mco.mu.RLock()
	defer mco.mu.RUnlock()

	stats := &OrchestratorStats{
		TotalDestinations: len(mco.destinations),
		ActiveJobs:        len(mco.activeJobs),
		CompletedJobs:     len(mco.results),
		ByProvider:        make(map[StorageProvider]int),
	}

	// Count destinations by provider
	for _, dest := range mco.destinations {
		if dest.Enabled {
			stats.EnabledDestinations++
		}
		stats.ByProvider[dest.Provider]++
	}

	// Calculate success rate
	totalUploads := 0
	successfulUploads := 0
	for _, results := range mco.results {
		for _, result := range results {
			totalUploads++
			if result.Success {
				successfulUploads++
			}
		}
	}

	if totalUploads > 0 {
		stats.SuccessRate = float64(successfulUploads) / float64(totalUploads) * 100
	}

	return stats
}

// OrchestratorStats represents orchestrator statistics
type OrchestratorStats struct {
	TotalDestinations   int                        `json:"total_destinations"`
	EnabledDestinations int                        `json:"enabled_destinations"`
	ActiveJobs          int                        `json:"active_jobs"`
	CompletedJobs       int                        `json:"completed_jobs"`
	SuccessRate         float64                    `json:"success_rate"`
	ByProvider          map[StorageProvider]int    `json:"by_provider"`
}
