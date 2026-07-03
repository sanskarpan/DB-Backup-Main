package analytics

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CompletionEvent represents a completion usage event.
type CompletionEvent struct {
	Timestamp    time.Time         `json:"timestamp"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Accepted     bool              `json:"accepted"`
	Source       string            `json:"source"` // "cache", "live", "fuzzy", "history"
	ResponseTime float64           `json:"response_time_ms"`
	ResultCount  int               `json:"result_count"`
	Context      map[string]string `json:"context"`
}

// QualityMetrics tracks completion quality.
type QualityMetrics struct {
	TotalCompletions int     `json:"total_completions"`
	AcceptedCount    int     `json:"accepted_count"`
	RejectedCount    int     `json:"rejected_count"`
	AcceptanceRate   float64 `json:"acceptance_rate"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	FuzzyMatchRate   float64 `json:"fuzzy_match_rate"`
	HistoryMatchRate float64 `json:"history_match_rate"`
	ErrorCount       int     `json:"error_count"`
}

// Analytics tracks completion usage and quality.
type Analytics struct {
	mu           sync.RWMutex
	dataFile     string
	events       []*CompletionEvent
	metrics      *QualityMetrics
	maxEvents    int
	flushEnabled bool
}

// NewAnalytics creates a new analytics tracker.
func NewAnalytics(dataFile string) *Analytics {
	return &Analytics{
		dataFile:     dataFile,
		events:       make([]*CompletionEvent, 0),
		metrics:      &QualityMetrics{},
		maxEvents:    10000,
		flushEnabled: true,
	}
}

// Load loads analytics data from file.
func (a *Analytics) Load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	dir := filepath.Dir(a.dataFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(a.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type state struct {
		Events  []*CompletionEvent `json:"events"`
		Metrics *QualityMetrics    `json:"metrics"`
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	a.events = s.Events
	a.metrics = s.Metrics

	return nil
}

// Save saves analytics data to file.
func (a *Analytics) Save() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	type state struct {
		Events  []*CompletionEvent `json:"events"`
		Metrics *QualityMetrics    `json:"metrics"`
	}

	s := state{
		Events:  a.events,
		Metrics: a.metrics,
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.dataFile, data, 0o600)
}

// TrackCompletion records a completion event.
func (a *Analytics) TrackCompletion(event *CompletionEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	event.Timestamp = time.Now()
	a.events = append(a.events, event)

	// Update metrics
	a.updateMetrics(event)

	// Trim events if exceeding max
	if len(a.events) > a.maxEvents {
		a.events = a.events[len(a.events)-a.maxEvents:]
	}

	// Auto-save if enabled
	if a.flushEnabled {
		go func() {
			if err := a.Save(); err != nil {
				log.Printf("analytics: auto-save failed: %v", err)
			}
		}()
	}
}

// TrackError records a completion error.
func (a *Analytics) TrackError(command, errorMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.metrics.ErrorCount++

	event := &CompletionEvent{
		Timestamp: time.Now(),
		Command:   command,
		Accepted:  false,
		Context: map[string]string{
			"error": errorMsg,
		},
	}

	a.events = append(a.events, event)
}

// GetMetrics returns current quality metrics.
func (a *Analytics) GetMetrics() *QualityMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy
	metricsCopy := *a.metrics
	return &metricsCopy
}

// GetRecentEvents returns recent completion events.
func (a *Analytics) GetRecentEvents(limit int) []*CompletionEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	start := len(a.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]*CompletionEvent, len(a.events)-start)
	copy(events, a.events[start:])

	return events
}

// GetEventsByTimeRange returns events within time range.
func (a *Analytics) GetEventsByTimeRange(start, end time.Time) []*CompletionEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	filtered := make([]*CompletionEvent, 0)

	for _, event := range a.events {
		if event.Timestamp.After(start) && event.Timestamp.Before(end) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// GetTopCommands returns most used commands.
func (a *Analytics) GetTopCommands(limit int) []map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	commandCounts := make(map[string]int)

	for _, event := range a.events {
		if event.Accepted {
			commandCounts[event.Command]++
		}
	}

	type cmdCount struct {
		command string
		count   int
	}

	counts := make([]cmdCount, 0, len(commandCounts))
	for cmd, count := range commandCounts {
		counts = append(counts, cmdCount{cmd, count})
	}

	// Sort by count descending
	for i := 0; i < len(counts); i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	if len(counts) > limit {
		counts = counts[:limit]
	}

	result := make([]map[string]interface{}, len(counts))
	for i, c := range counts {
		result[i] = map[string]interface{}{
			"command": c.command,
			"count":   c.count,
		}
	}

	return result
}

// GetPerformanceStats returns performance statistics.
func (a *Analytics) GetPerformanceStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.events) == 0 {
		return map[string]interface{}{
			"avg_response_time": 0.0,
			"min_response_time": 0.0,
			"max_response_time": 0.0,
			"p50_response_time": 0.0,
			"p95_response_time": 0.0,
			"p99_response_time": 0.0,
		}
	}

	times := make([]float64, 0, len(a.events))
	for _, event := range a.events {
		if event.ResponseTime > 0 {
			times = append(times, event.ResponseTime)
		}
	}

	if len(times) == 0 {
		return map[string]interface{}{
			"avg_response_time": 0.0,
			"min_response_time": 0.0,
			"max_response_time": 0.0,
		}
	}

	// Sort times
	for i := 0; i < len(times); i++ {
		for j := i + 1; j < len(times); j++ {
			if times[j] < times[i] {
				times[i], times[j] = times[j], times[i]
			}
		}
	}

	sum := 0.0
	minTime := times[0]
	maxTime := times[len(times)-1]
	for _, t := range times {
		sum += t
	}
	avg := sum / float64(len(times))

	p50 := times[len(times)*50/100]
	p95 := times[len(times)*95/100]
	p99 := times[len(times)*99/100]

	return map[string]interface{}{
		"avg_response_time": avg,
		"min_response_time": minTime,
		"max_response_time": maxTime,
		"p50_response_time": p50,
		"p95_response_time": p95,
		"p99_response_time": p99,
	}
}

// Clear clears all analytics data.
func (a *Analytics) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = make([]*CompletionEvent, 0)
	a.metrics = &QualityMetrics{}

	return os.Remove(a.dataFile)
}

// updateMetrics updates quality metrics based on event.
func (a *Analytics) updateMetrics(event *CompletionEvent) {
	a.metrics.TotalCompletions++

	if event.Accepted {
		a.metrics.AcceptedCount++
	} else {
		a.metrics.RejectedCount++
	}

	a.metrics.AcceptanceRate = float64(a.metrics.AcceptedCount) / float64(a.metrics.TotalCompletions)

	// Update average response time
	if a.metrics.AvgResponseTime == 0 {
		a.metrics.AvgResponseTime = event.ResponseTime
	} else {
		a.metrics.AvgResponseTime = (a.metrics.AvgResponseTime + event.ResponseTime) / 2
	}

	// Update source-specific rates
	switch event.Source {
	case "cache":
		cacheCount := 0
		for _, e := range a.events {
			if e.Source == "cache" {
				cacheCount++
			}
		}
		a.metrics.CacheHitRate = float64(cacheCount) / float64(a.metrics.TotalCompletions)

	case "fuzzy":
		fuzzyCount := 0
		for _, e := range a.events {
			if e.Source == "fuzzy" {
				fuzzyCount++
			}
		}
		a.metrics.FuzzyMatchRate = float64(fuzzyCount) / float64(a.metrics.TotalCompletions)

	case "history":
		historyCount := 0
		for _, e := range a.events {
			if e.Source == "history" {
				historyCount++
			}
		}
		a.metrics.HistoryMatchRate = float64(historyCount) / float64(a.metrics.TotalCompletions)
	}
}
