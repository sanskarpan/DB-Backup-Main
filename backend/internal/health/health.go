// Package health provides health check functionality
package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Status represents the health status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Check represents a health check function.
type Check interface {
	Name() string
	Check(ctx context.Context) *CheckResult
}

// Checker manages health checks.
type Checker struct {
	checks []Check
	mu     sync.RWMutex
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checks: make([]Check, 0),
	}
}

// Register registers a new health check.
func (c *Checker) Register(check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks = append(c.checks, check)
}

// CheckAll runs all registered health checks.
func (c *Checker) CheckAll(ctx context.Context) *HealthReport {
	c.mu.RLock()
	checks := make([]Check, len(c.checks))
	copy(checks, c.checks)
	c.mu.RUnlock()

	results := make([]*CheckResult, 0, len(checks))
	var wg sync.WaitGroup
	resultsChan := make(chan *CheckResult, len(checks))

	for _, check := range checks {
		wg.Add(1)
		go func(ch Check) {
			defer wg.Done()
			resultsChan <- ch.Check(ctx)
		}(check)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		results = append(results, result)
	}

	return newHealthReport(results)
}

// CheckByName runs a specific health check by name.
func (c *Checker) CheckByName(ctx context.Context, name string) *CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, check := range c.checks {
		if check.Name() == name {
			return check.Check(ctx)
		}
	}

	return &CheckResult{
		Name:      name,
		Status:    StatusUnhealthy,
		Error:     "check not found",
		Timestamp: time.Now(),
	}
}

// HealthReport represents the overall health status.
type HealthReport struct {
	Status    Status         `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Checks    []*CheckResult `json:"checks"`
	Summary   *HealthSummary `json:"summary"`
}

// HealthSummary provides a summary of health checks.
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Unhealthy int `json:"unhealthy"`
}

// newHealthReport creates a new health report from check results.
func newHealthReport(results []*CheckResult) *HealthReport {
	summary := &HealthSummary{
		Total: len(results),
	}

	overallStatus := StatusHealthy

	for _, result := range results {
		switch result.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusDegraded:
			summary.Degraded++
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		case StatusUnhealthy:
			summary.Unhealthy++
			overallStatus = StatusUnhealthy
		}
	}

	return &HealthReport{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
		Summary:   summary,
	}
}

// DatabaseCheck checks database connectivity.
type DatabaseCheck struct {
	name     string
	pingFunc func(context.Context) error
	timeout  time.Duration
}

// NewDatabaseCheck creates a new database health check.
func NewDatabaseCheck(name string, pingFunc func(context.Context) error, timeout time.Duration) *DatabaseCheck {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &DatabaseCheck{
		name:     name,
		pingFunc: pingFunc,
		timeout:  timeout,
	}
}

// Name returns the check name.
func (d *DatabaseCheck) Name() string {
	return d.name
}

// Check performs the health check.
func (d *DatabaseCheck) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	result := &CheckResult{
		Name:      d.name,
		Timestamp: time.Now(),
	}

	err := d.pingFunc(ctx)
	result.Duration = time.Since(start)

	if err == nil {
		result.Status = StatusHealthy
		result.Message = "database connection successful"
	} else {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
	}

	return result
}

// StorageCheck checks storage accessibility.
type StorageCheck struct {
	name      string
	checkFunc func(context.Context) error
	timeout   time.Duration
}

// NewStorageCheck creates a new storage health check.
func NewStorageCheck(name string, checkFunc func(context.Context) error, timeout time.Duration) *StorageCheck {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &StorageCheck{
		name:      name,
		checkFunc: checkFunc,
		timeout:   timeout,
	}
}

// Name returns the check name.
func (s *StorageCheck) Name() string {
	return s.name
}

// Check performs the health check.
func (s *StorageCheck) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	result := &CheckResult{
		Name:      s.name,
		Timestamp: time.Now(),
	}

	err := s.checkFunc(ctx)
	result.Duration = time.Since(start)

	if err == nil {
		result.Status = StatusHealthy
		result.Message = "storage accessible"
	} else {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
	}

	return result
}

// DiskSpaceCheck checks available disk space.
type DiskSpaceCheck struct {
	name           string
	path           string
	thresholdBytes int64
	checkFunc      func(string) (int64, error)
}

// NewDiskSpaceCheck creates a new disk space health check.
func NewDiskSpaceCheck(name, path string, thresholdBytes int64, checkFunc func(string) (int64, error)) *DiskSpaceCheck {
	return &DiskSpaceCheck{
		name:           name,
		path:           path,
		thresholdBytes: thresholdBytes,
		checkFunc:      checkFunc,
	}
}

// Name returns the check name.
func (d *DiskSpaceCheck) Name() string {
	return d.name
}

// Check performs the health check.
func (d *DiskSpaceCheck) Check(ctx context.Context) *CheckResult {
	start := time.Now()

	result := &CheckResult{
		Name:      d.name,
		Timestamp: time.Now(),
	}

	available, err := d.checkFunc(d.path)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		return result
	}

	result.Metadata = map[string]interface{}{
		"available_bytes": available,
		"threshold_bytes": d.thresholdBytes,
		"path":            d.path,
	}

	if available >= d.thresholdBytes {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("%.2f GB available", float64(available)/(1024*1024*1024))
	} else if available >= d.thresholdBytes/2 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("low disk space: %.2f GB available", float64(available)/(1024*1024*1024))
	} else {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("critical: only %.2f GB available", float64(available)/(1024*1024*1024))
	}

	return result
}

// CustomCheck allows custom health check functions.
type CustomCheck struct {
	name      string
	checkFunc func(context.Context) *CheckResult
}

// NewCustomCheck creates a new custom health check.
func NewCustomCheck(name string, checkFunc func(context.Context) *CheckResult) *CustomCheck {
	return &CustomCheck{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Name returns the check name.
func (c *CustomCheck) Name() string {
	return c.name
}

// Check performs the health check.
func (c *CustomCheck) Check(ctx context.Context) *CheckResult {
	return c.checkFunc(ctx)
}
