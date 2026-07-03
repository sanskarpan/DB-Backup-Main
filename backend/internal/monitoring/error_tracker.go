package monitoring

import (
	"sync"
	"time"
)

// ErrorEntry represents a tracked error.
type ErrorEntry struct {
	Error     error                  `json:"error"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context"`
}

// ErrorTracker tracks errors and calculates error rates.
type ErrorTracker struct {
	mu         sync.RWMutex
	errors     []ErrorEntry
	maxErrors  int
	windowSize time.Duration
}

// NewErrorTracker creates a new error tracker.
func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errors:     make([]ErrorEntry, 0),
		maxErrors:  10000,
		windowSize: 5 * time.Minute,
	}
}

// RecordError records an error.
func (et *ErrorTracker) RecordError(err error, context map[string]interface{}) {
	et.mu.Lock()
	defer et.mu.Unlock()

	entry := ErrorEntry{
		Error:     err,
		Timestamp: time.Now(),
		Context:   context,
	}

	et.errors = append(et.errors, entry)
	if len(et.errors) > et.maxErrors {
		et.errors = et.errors[1:]
	}
}

// GetErrorRate calculates the error rate (errors per second).
func (et *ErrorTracker) GetErrorRate() float64 {
	et.mu.RLock()
	defer et.mu.RUnlock()

	cutoff := time.Now().Add(-et.windowSize)
	count := 0

	for _, entry := range et.errors {
		if entry.Timestamp.After(cutoff) {
			count++
		}
	}

	return float64(count) / et.windowSize.Seconds()
}

// GetRecentErrors returns recent errors within the window.
func (et *ErrorTracker) GetRecentErrors(limit int) []ErrorEntry {
	et.mu.RLock()
	defer et.mu.RUnlock()

	cutoff := time.Now().Add(-et.windowSize)
	recent := make([]ErrorEntry, 0)

	for i := len(et.errors) - 1; i >= 0 && len(recent) < limit; i-- {
		if et.errors[i].Timestamp.After(cutoff) {
			recent = append(recent, et.errors[i])
		}
	}

	return recent
}

// ClearOldErrors clears errors older than the window.
func (et *ErrorTracker) ClearOldErrors() int {
	et.mu.Lock()
	defer et.mu.Unlock()

	cutoff := time.Now().Add(-et.windowSize * 2)
	newErrors := make([]ErrorEntry, 0)

	for _, entry := range et.errors {
		if entry.Timestamp.After(cutoff) {
			newErrors = append(newErrors, entry)
		}
	}

	cleared := len(et.errors) - len(newErrors)
	et.errors = newErrors

	return cleared
}
