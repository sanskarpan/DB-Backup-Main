package optimization

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// SlowQueryDetector detects and tracks slow queries.
type SlowQueryDetector struct {
	threshold   time.Duration
	slowQueries []*SlowQuery
	queryStats  map[string]*slowQueryStats
	mu          sync.RWMutex
	maxEntries  int
}

// slowQueryStats tracks statistics for a specific query pattern.
type slowQueryStats struct {
	count         int64
	totalDuration time.Duration
	maxDuration   time.Duration
	minDuration   time.Duration
	lastSeen      time.Time
}

// NewSlowQueryDetector creates a new slow query detector.
func NewSlowQueryDetector(threshold time.Duration) *SlowQueryDetector {
	return &SlowQueryDetector{
		threshold:   threshold,
		slowQueries: make([]*SlowQuery, 0),
		queryStats:  make(map[string]*slowQueryStats),
		maxEntries:  1000, // Keep last 1000 slow queries
	}
}

// Detect detects if a query is slow and records it.
func (sqd *SlowQueryDetector) Detect(query string, executionTime time.Duration) *SlowQuery {
	if executionTime < sqd.threshold {
		return nil
	}

	sqd.mu.Lock()
	defer sqd.mu.Unlock()

	// Calculate query hash
	queryHash := calculateQueryHash(query)

	// Determine severity
	severity := sqd.calculateSeverity(executionTime)

	// Create slow query entry
	slowQuery := &SlowQuery{
		Query:           query,
		QueryHash:       queryHash,
		ExecutionTime:   executionTime,
		Timestamp:       time.Now(),
		Severity:        severity,
		Recommendations: sqd.generateRecommendations(executionTime),
	}

	// Add to slow queries list
	sqd.slowQueries = append(sqd.slowQueries, slowQuery)

	// Trim if exceeds max entries
	if len(sqd.slowQueries) > sqd.maxEntries {
		sqd.slowQueries = sqd.slowQueries[len(sqd.slowQueries)-sqd.maxEntries:]
	}

	// Update statistics
	stats, exists := sqd.queryStats[queryHash]
	if !exists {
		stats = &slowQueryStats{
			minDuration: executionTime,
			maxDuration: executionTime,
		}
		sqd.queryStats[queryHash] = stats
	}

	stats.count++
	stats.totalDuration += executionTime
	stats.lastSeen = time.Now()

	if executionTime > stats.maxDuration {
		stats.maxDuration = executionTime
	}
	if executionTime < stats.minDuration {
		stats.minDuration = executionTime
	}

	return slowQuery
}

// GetSlowQueries returns the most recent slow queries.
func (sqd *SlowQueryDetector) GetSlowQueries(limit int) []*SlowQuery {
	sqd.mu.RLock()
	defer sqd.mu.RUnlock()

	if limit <= 0 || limit > len(sqd.slowQueries) {
		limit = len(sqd.slowQueries)
	}

	// Return most recent queries
	start := len(sqd.slowQueries) - limit
	result := make([]*SlowQuery, limit)
	copy(result, sqd.slowQueries[start:])

	return result
}

// GetQueryStats returns statistics for a specific query pattern.
func (sqd *SlowQueryDetector) GetQueryStats(queryHash string) *QueryStatsSummary {
	sqd.mu.RLock()
	defer sqd.mu.RUnlock()

	stats, exists := sqd.queryStats[queryHash]
	if !exists {
		return nil
	}

	avgDuration := time.Duration(0)
	if stats.count > 0 {
		avgDuration = stats.totalDuration / time.Duration(stats.count)
	}

	return &QueryStatsSummary{
		QueryHash:      queryHash,
		ExecutionCount: stats.count,
		TotalDuration:  stats.totalDuration,
		AvgDuration:    avgDuration,
		MinDuration:    stats.minDuration,
		MaxDuration:    stats.maxDuration,
		LastSeen:       stats.lastSeen,
	}
}

// GetTopSlowQueries returns the top N slowest query patterns.
func (sqd *SlowQueryDetector) GetTopSlowQueries(limit int) []*QueryStatsSummary {
	sqd.mu.RLock()
	defer sqd.mu.RUnlock()

	summaries := make([]*QueryStatsSummary, 0, len(sqd.queryStats))

	for queryHash, stats := range sqd.queryStats {
		avgDuration := time.Duration(0)
		if stats.count > 0 {
			avgDuration = stats.totalDuration / time.Duration(stats.count)
		}

		summaries = append(summaries, &QueryStatsSummary{
			QueryHash:      queryHash,
			ExecutionCount: stats.count,
			TotalDuration:  stats.totalDuration,
			AvgDuration:    avgDuration,
			MinDuration:    stats.minDuration,
			MaxDuration:    stats.maxDuration,
			LastSeen:       stats.lastSeen,
		})
	}

	// Sort by average duration (descending)
	sortQueryStatsByAvgDuration(summaries)

	if limit > 0 && limit < len(summaries) {
		summaries = summaries[:limit]
	}

	return summaries
}

// Clear clears all slow query data.
func (sqd *SlowQueryDetector) Clear() {
	sqd.mu.Lock()
	defer sqd.mu.Unlock()

	sqd.slowQueries = make([]*SlowQuery, 0)
	sqd.queryStats = make(map[string]*slowQueryStats)
}

// GetThreshold returns the slow query threshold.
func (sqd *SlowQueryDetector) GetThreshold() time.Duration {
	return sqd.threshold
}

// SetThreshold sets the slow query threshold.
func (sqd *SlowQueryDetector) SetThreshold(threshold time.Duration) {
	sqd.mu.Lock()
	defer sqd.mu.Unlock()
	sqd.threshold = threshold
}

// GetCount returns the total number of slow queries detected.
func (sqd *SlowQueryDetector) GetCount() int {
	sqd.mu.RLock()
	defer sqd.mu.RUnlock()
	return len(sqd.slowQueries)
}

// calculateSeverity determines the severity based on execution time.
func (sqd *SlowQueryDetector) calculateSeverity(executionTime time.Duration) Severity {
	ratio := float64(executionTime) / float64(sqd.threshold)

	switch {
	case ratio >= 10:
		return SeverityCritical
	case ratio >= 5:
		return SeverityHigh
	case ratio >= 2:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// generateRecommendations generates recommendations for slow queries.
func (sqd *SlowQueryDetector) generateRecommendations(executionTime time.Duration) []string {
	recommendations := make([]string, 0)

	if executionTime > 10*time.Second {
		recommendations = append(recommendations,
			"Consider adding appropriate indexes",
			"Review query for N+1 problems")
	}

	if executionTime > 30*time.Second {
		recommendations = append(recommendations,
			"Consider query pagination",
			"Review table statistics and vacuum")
	}

	if executionTime > 60*time.Second {
		recommendations = append(recommendations,
			"Consider breaking query into smaller parts",
			"Review execution plan for table scans")
	}

	return recommendations
}

// QueryStatsSummary summarizes query statistics.
type QueryStatsSummary struct {
	QueryHash      string
	ExecutionCount int64
	TotalDuration  time.Duration
	AvgDuration    time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
	LastSeen       time.Time
}

// calculateQueryHash calculates a hash for a query.
func calculateQueryHash(query string) string {
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", hash)
}

// sortQueryStatsByAvgDuration sorts query stats by average duration (descending).
func sortQueryStatsByAvgDuration(summaries []*QueryStatsSummary) {
	// Simple bubble sort for demonstration
	n := len(summaries)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if summaries[j].AvgDuration < summaries[j+1].AvgDuration {
				summaries[j], summaries[j+1] = summaries[j+1], summaries[j]
			}
		}
	}
}
