package monitoring

import (
	"context"
	"sync"
	"time"
)

// ResponseEntry represents a recorded response.
type ResponseEntry struct {
	Duration  time.Duration `json:"duration"`
	Endpoint  string        `json:"endpoint"`
	Timestamp time.Time     `json:"timestamp"`
}

// PerformanceMonitor monitors performance metrics.
type PerformanceMonitor struct {
	mu         sync.RWMutex
	responses  []ResponseEntry
	maxEntries int
	windowSize time.Duration
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		responses:  make([]ResponseEntry, 0),
		maxEntries: 50000,
		windowSize: 10 * time.Minute,
	}
}

// RecordResponse records a response time.
func (pm *PerformanceMonitor) RecordResponse(duration time.Duration, endpoint string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry := ResponseEntry{
		Duration:  duration,
		Endpoint:  endpoint,
		Timestamp: time.Now(),
	}

	pm.responses = append(pm.responses, entry)
	if len(pm.responses) > pm.maxEntries {
		pm.responses = pm.responses[1:]
	}
}

// GetAverageResponseTime calculates average response time.
func (pm *PerformanceMonitor) GetAverageResponseTime() time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.responses) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-pm.windowSize)
	var total time.Duration
	count := 0

	for _, entry := range pm.responses {
		if entry.Timestamp.After(cutoff) {
			total += entry.Duration
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// GetP95ResponseTime calculates 95th percentile response time.
func (pm *PerformanceMonitor) GetP95ResponseTime() time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	cutoff := time.Now().Add(-pm.windowSize)
	durations := make([]time.Duration, 0)

	for _, entry := range pm.responses {
		if entry.Timestamp.After(cutoff) {
			durations = append(durations, entry.Duration)
		}
	}

	if len(durations) == 0 {
		return 0
	}

	// Simple sorting for P95 calculation
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	p95Index := int(float64(len(durations)) * 0.95)
	return durations[p95Index]
}

// Start starts the performance monitor.
func (pm *PerformanceMonitor) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.cleanup()
		}
	}
}

// cleanup removes old entries.
func (pm *PerformanceMonitor) cleanup() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-pm.windowSize * 2)
	newResponses := make([]ResponseEntry, 0)

	for _, entry := range pm.responses {
		if entry.Timestamp.After(cutoff) {
			newResponses = append(newResponses, entry)
		}
	}

	pm.responses = newResponses
}
