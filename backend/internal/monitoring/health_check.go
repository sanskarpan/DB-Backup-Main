package monitoring

import (
	"context"
	"time"
)

// HealthCheckFunc is a function that performs a health check.
type HealthCheckFunc func(ctx context.Context) (healthy bool, message string, err error)

// HealthCheck represents a health check.
type HealthCheck struct {
	Name       string
	Check      HealthCheckFunc
	Critical   bool
	Interval   time.Duration
	Timeout    time.Duration
	LastResult *HealthCheckResult
}

// HealthCheckResult represents the result of a health check.
type HealthCheckResult struct {
	Healthy   bool          `json:"healthy"`
	Message   string        `json:"message"`
	Error     error         `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// Execute executes the health check.
func (hc *HealthCheck) Execute(ctx context.Context) *HealthCheckResult {
	start := time.Now()

	// Create context with timeout
	timeout := hc.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the check
	healthy, message, err := hc.Check(checkCtx)

	result := &HealthCheckResult{
		Healthy:   healthy,
		Message:   message,
		Error:     err,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}

	hc.LastResult = result
	return result
}
