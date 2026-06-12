package optimization

import (
	"sync"
	"time"
)

// StatsCollector collects and maintains query statistics
type StatsCollector struct {
	statistics       map[string]*QueryStatistics
	retentionPeriod  time.Duration
	mu               sync.RWMutex
	cleanupTicker    *time.Ticker
	stopCleanup      chan struct{}
}

// NewStatsCollector creates a new statistics collector
func NewStatsCollector(retentionPeriod time.Duration) *StatsCollector {
	collector := &StatsCollector{
		statistics:      make(map[string]*QueryStatistics),
		retentionPeriod: retentionPeriod,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	if retentionPeriod > 0 {
		collector.cleanupTicker = time.NewTicker(retentionPeriod / 2)
		go collector.cleanupLoop()
	}

	return collector
}

// RecordQuery records a query execution
func (sc *StatsCollector) RecordQuery(query string, executionTime time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	queryHash := calculateQueryHash(query)

	stats, exists := sc.statistics[queryHash]
	if !exists {
		stats = &QueryStatistics{
			Query:         query,
			QueryHash:     queryHash,
			MinTime:       executionTime,
			MaxTime:       executionTime,
			FirstExecuted: time.Now(),
		}
		sc.statistics[queryHash] = stats
	}

	// Update statistics
	stats.ExecutionCount++
	stats.TotalTime += executionTime
	stats.LastExecuted = time.Now()

	// Update min/max
	if executionTime < stats.MinTime {
		stats.MinTime = executionTime
	}
	if executionTime > stats.MaxTime {
		stats.MaxTime = executionTime
	}

	// Calculate mean
	stats.MeanTime = stats.TotalTime / time.Duration(stats.ExecutionCount)

	// Update standard deviation (simplified)
	if stats.ExecutionCount > 1 {
		variance := float64(executionTime-stats.MeanTime) * float64(executionTime-stats.MeanTime)
		stats.StdDevTime = time.Duration(variance / float64(stats.ExecutionCount))
	}
}

// RecordQueryWithDetails records a query execution with additional details
func (sc *StatsCollector) RecordQueryWithDetails(query string, executionTime time.Duration, rowsReturned, rowsAffected int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	queryHash := calculateQueryHash(query)

	stats, exists := sc.statistics[queryHash]
	if !exists {
		stats = &QueryStatistics{
			Query:         query,
			QueryHash:     queryHash,
			MinTime:       executionTime,
			MaxTime:       executionTime,
			FirstExecuted: time.Now(),
		}
		sc.statistics[queryHash] = stats
	}

	// Update statistics
	stats.ExecutionCount++
	stats.TotalTime += executionTime
	stats.LastExecuted = time.Now()
	stats.RowsReturned += rowsReturned
	stats.RowsAffected += rowsAffected

	// Update min/max
	if executionTime < stats.MinTime {
		stats.MinTime = executionTime
	}
	if executionTime > stats.MaxTime {
		stats.MaxTime = executionTime
	}

	// Calculate mean
	stats.MeanTime = stats.TotalTime / time.Duration(stats.ExecutionCount)
}

// GetStatistics returns all collected statistics
func (sc *StatsCollector) GetStatistics() map[string]*QueryStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*QueryStatistics, len(sc.statistics))
	for k, v := range sc.statistics {
		statsCopy := *v
		result[k] = &statsCopy
	}

	return result
}

// GetQueryStatistics returns statistics for a specific query
func (sc *StatsCollector) GetQueryStatistics(queryHash string) *QueryStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats, exists := sc.statistics[queryHash]
	if !exists {
		return nil
	}

	statsCopy := *stats
	return &statsCopy
}

// GetTopQueries returns the top N queries by total execution time
func (sc *StatsCollector) GetTopQueries(limit int) []*QueryStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := make([]*QueryStatistics, 0, len(sc.statistics))
	for _, s := range sc.statistics {
		statsCopy := *s
		stats = append(stats, &statsCopy)
	}

	// Sort by total time (descending)
	sortStatsByTotalTime(stats)

	if limit > 0 && limit < len(stats) {
		stats = stats[:limit]
	}

	return stats
}

// GetSlowestQueries returns the top N queries by average execution time
func (sc *StatsCollector) GetSlowestQueries(limit int) []*QueryStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := make([]*QueryStatistics, 0, len(sc.statistics))
	for _, s := range sc.statistics {
		statsCopy := *s
		stats = append(stats, &statsCopy)
	}

	// Sort by mean time (descending)
	sortStatsByMeanTime(stats)

	if limit > 0 && limit < len(stats) {
		stats = stats[:limit]
	}

	return stats
}

// GetMostFrequentQueries returns the top N most frequently executed queries
func (sc *StatsCollector) GetMostFrequentQueries(limit int) []*QueryStatistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := make([]*QueryStatistics, 0, len(sc.statistics))
	for _, s := range sc.statistics {
		statsCopy := *s
		stats = append(stats, &statsCopy)
	}

	// Sort by execution count (descending)
	sortStatsByExecutionCount(stats)

	if limit > 0 && limit < len(stats) {
		stats = stats[:limit]
	}

	return stats
}

// Clear clears all statistics
func (sc *StatsCollector) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.statistics = make(map[string]*QueryStatistics)
}

// GetTotalQueries returns the total number of unique queries tracked
func (sc *StatsCollector) GetTotalQueries() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.statistics)
}

// GetTotalExecutions returns the total number of query executions
func (sc *StatsCollector) GetTotalExecutions() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var total int64
	for _, stats := range sc.statistics {
		total += stats.ExecutionCount
	}
	return total
}

// Stop stops the statistics collector
func (sc *StatsCollector) Stop() {
	if sc.cleanupTicker != nil {
		sc.cleanupTicker.Stop()
		close(sc.stopCleanup)
	}
}

// cleanupLoop periodically removes old statistics
func (sc *StatsCollector) cleanupLoop() {
	for {
		select {
		case <-sc.cleanupTicker.C:
			sc.cleanup()
		case <-sc.stopCleanup:
			return
		}
	}
}

// cleanup removes statistics older than retention period
func (sc *StatsCollector) cleanup() {
	if sc.retentionPeriod <= 0 {
		return
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	cutoff := time.Now().Add(-sc.retentionPeriod)

	for hash, stats := range sc.statistics {
		if stats.LastExecuted.Before(cutoff) {
			delete(sc.statistics, hash)
		}
	}
}

// Sorting functions

func sortStatsByTotalTime(stats []*QueryStatistics) {
	n := len(stats)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if stats[j].TotalTime < stats[j+1].TotalTime {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}
}

func sortStatsByMeanTime(stats []*QueryStatistics) {
	n := len(stats)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if stats[j].MeanTime < stats[j+1].MeanTime {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}
}

func sortStatsByExecutionCount(stats []*QueryStatistics) {
	n := len(stats)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if stats[j].ExecutionCount < stats[j+1].ExecutionCount {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}
}
