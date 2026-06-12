package webhooks

import (
	"sync"
	"time"
)

// Analytics tracks webhook delivery metrics
type Analytics struct {
	mu      sync.RWMutex
	metrics map[string]*SubscriptionMetrics
}

// SubscriptionMetrics holds metrics for a single subscription
type SubscriptionMetrics struct {
	SubscriptionID   string
	TotalDeliveries  int64
	SuccessCount     int64
	FailureCount     int64
	AverageLatency   time.Duration
	LastDeliveryTime time.Time
	LastSuccessTime  time.Time
	LastFailureTime  time.Time
	LastError        string
	EventMetrics     map[string]*EventMetrics // Event ID -> metrics
	hourlyStats      []HourlyStat
}

// EventMetrics holds metrics for a single event
type EventMetrics struct {
	EventID      string
	DeliveryTime time.Time
	Success      bool
	Duration     time.Duration
	Error        string
	Attempts     int
}

// HourlyStat represents stats for an hour window
type HourlyStat struct {
	Hour         time.Time
	Deliveries   int64
	Successes    int64
	Failures     int64
	AvgDuration  time.Duration
}

// AggregateMetrics holds overall metrics across all subscriptions
type AggregateMetrics struct {
	TotalSubscriptions int
	ActiveSubscriptions int
	TotalDeliveries    int64
	TotalSuccesses     int64
	TotalFailures      int64
	SuccessRate        float64
	AverageLatency     time.Duration
	SubscriptionMetrics []*SubscriptionMetrics
}

// NewAnalytics creates a new analytics tracker
func NewAnalytics() *Analytics {
	return &Analytics{
		metrics: make(map[string]*SubscriptionMetrics),
	}
}

// RecordDelivery records a delivery attempt
func (a *Analytics) RecordDelivery(subscriptionID, eventID string, success bool, duration time.Duration, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get or create subscription metrics
	metrics, exists := a.metrics[subscriptionID]
	if !exists {
		metrics = &SubscriptionMetrics{
			SubscriptionID: subscriptionID,
			EventMetrics:   make(map[string]*EventMetrics),
			hourlyStats:    make([]HourlyStat, 0),
		}
		a.metrics[subscriptionID] = metrics
	}

	// Update totals
	metrics.TotalDeliveries++
	if success {
		metrics.SuccessCount++
		metrics.LastSuccessTime = time.Now()
	} else {
		metrics.FailureCount++
		metrics.LastFailureTime = time.Now()
		if err != nil {
			metrics.LastError = err.Error()
		}
	}

	metrics.LastDeliveryTime = time.Now()

	// Update average latency
	if metrics.TotalDeliveries == 1 {
		metrics.AverageLatency = duration
	} else {
		// Exponential moving average
		alpha := 0.2
		metrics.AverageLatency = time.Duration(
			alpha*float64(duration) + (1-alpha)*float64(metrics.AverageLatency),
		)
	}

	// Record event metrics
	eventMetric := &EventMetrics{
		EventID:      eventID,
		DeliveryTime: time.Now(),
		Success:      success,
		Duration:     duration,
		Attempts:     1,
	}
	if err != nil {
		eventMetric.Error = err.Error()
	}

	// Update existing event metric if it exists (retry case)
	if existing, exists := metrics.EventMetrics[eventID]; exists {
		eventMetric.Attempts = existing.Attempts + 1
	}

	metrics.EventMetrics[eventID] = eventMetric

	// Update hourly stats
	a.updateHourlyStats(metrics, success, duration)
}

// updateHourlyStats updates hourly statistics
func (a *Analytics) updateHourlyStats(metrics *SubscriptionMetrics, success bool, duration time.Duration) {
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)

	// Find or create current hour stat
	var currentStat *HourlyStat
	for i := range metrics.hourlyStats {
		if metrics.hourlyStats[i].Hour.Equal(currentHour) {
			currentStat = &metrics.hourlyStats[i]
			break
		}
	}

	if currentStat == nil {
		newStat := HourlyStat{Hour: currentHour}
		metrics.hourlyStats = append(metrics.hourlyStats, newStat)
		currentStat = &metrics.hourlyStats[len(metrics.hourlyStats)-1]
	}

	// Update stats
	currentStat.Deliveries++
	if success {
		currentStat.Successes++
	} else {
		currentStat.Failures++
	}

	// Update average duration
	if currentStat.Deliveries == 1 {
		currentStat.AvgDuration = duration
	} else {
		currentStat.AvgDuration = time.Duration(
			(float64(currentStat.AvgDuration)*float64(currentStat.Deliveries-1) + float64(duration)) / float64(currentStat.Deliveries),
		)
	}

	// Keep only last 24 hours of stats
	if len(metrics.hourlyStats) > 24 {
		metrics.hourlyStats = metrics.hourlyStats[len(metrics.hourlyStats)-24:]
	}
}

// GetSubscriptionMetrics returns metrics for a specific subscription
func (a *Analytics) GetSubscriptionMetrics(subscriptionID string) *SubscriptionMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics, exists := a.metrics[subscriptionID]
	if !exists {
		return nil
	}

	// Return a copy to prevent race conditions
	return &SubscriptionMetrics{
		SubscriptionID:   metrics.SubscriptionID,
		TotalDeliveries:  metrics.TotalDeliveries,
		SuccessCount:     metrics.SuccessCount,
		FailureCount:     metrics.FailureCount,
		AverageLatency:   metrics.AverageLatency,
		LastDeliveryTime: metrics.LastDeliveryTime,
		LastSuccessTime:  metrics.LastSuccessTime,
		LastFailureTime:  metrics.LastFailureTime,
		LastError:        metrics.LastError,
		EventMetrics:     metrics.EventMetrics,
		hourlyStats:      metrics.hourlyStats,
	}
}

// GetAggregateMetrics returns aggregate metrics across all subscriptions
func (a *Analytics) GetAggregateMetrics() *AggregateMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agg := &AggregateMetrics{
		TotalSubscriptions: len(a.metrics),
		SubscriptionMetrics: make([]*SubscriptionMetrics, 0, len(a.metrics)),
	}

	var totalLatency time.Duration
	subscriptionCount := int64(0)

	for _, metrics := range a.metrics {
		agg.TotalDeliveries += metrics.TotalDeliveries
		agg.TotalSuccesses += metrics.SuccessCount
		agg.TotalFailures += metrics.FailureCount

		if metrics.TotalDeliveries > 0 {
			agg.ActiveSubscriptions++
			totalLatency += metrics.AverageLatency
			subscriptionCount++
		}

		// Add copy of metrics
		agg.SubscriptionMetrics = append(agg.SubscriptionMetrics, &SubscriptionMetrics{
			SubscriptionID:   metrics.SubscriptionID,
			TotalDeliveries:  metrics.TotalDeliveries,
			SuccessCount:     metrics.SuccessCount,
			FailureCount:     metrics.FailureCount,
			AverageLatency:   metrics.AverageLatency,
			LastDeliveryTime: metrics.LastDeliveryTime,
			LastSuccessTime:  metrics.LastSuccessTime,
			LastFailureTime:  metrics.LastFailureTime,
			LastError:        metrics.LastError,
		})
	}

	// Calculate success rate
	if agg.TotalDeliveries > 0 {
		agg.SuccessRate = float64(agg.TotalSuccesses) / float64(agg.TotalDeliveries) * 100
	}

	// Calculate average latency
	if subscriptionCount > 0 {
		agg.AverageLatency = totalLatency / time.Duration(subscriptionCount)
	}

	return agg
}

// GetHourlyStats returns hourly statistics for a subscription
func (a *Analytics) GetHourlyStats(subscriptionID string, hours int) []HourlyStat {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics, exists := a.metrics[subscriptionID]
	if !exists {
		return nil
	}

	if hours <= 0 || hours > len(metrics.hourlyStats) {
		hours = len(metrics.hourlyStats)
	}

	// Return last N hours
	start := len(metrics.hourlyStats) - hours
	if start < 0 {
		start = 0
	}

	stats := make([]HourlyStat, hours)
	copy(stats, metrics.hourlyStats[start:])

	return stats
}

// Reset resets all analytics data
func (a *Analytics) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.metrics = make(map[string]*SubscriptionMetrics)
}

// ResetSubscription resets analytics for a specific subscription
func (a *Analytics) ResetSubscription(subscriptionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.metrics, subscriptionID)
}

// GetSuccessRate returns the success rate for a subscription
func (a *Analytics) GetSuccessRate(subscriptionID string) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics, exists := a.metrics[subscriptionID]
	if !exists || metrics.TotalDeliveries == 0 {
		return 0.0
	}

	return float64(metrics.SuccessCount) / float64(metrics.TotalDeliveries) * 100
}

// GetEventMetrics returns metrics for a specific event
func (a *Analytics) GetEventMetrics(subscriptionID, eventID string) *EventMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics, exists := a.metrics[subscriptionID]
	if !exists {
		return nil
	}

	eventMetric, exists := metrics.EventMetrics[eventID]
	if !exists {
		return nil
	}

	// Return a copy
	return &EventMetrics{
		EventID:      eventMetric.EventID,
		DeliveryTime: eventMetric.DeliveryTime,
		Success:      eventMetric.Success,
		Duration:     eventMetric.Duration,
		Error:        eventMetric.Error,
		Attempts:     eventMetric.Attempts,
	}
}
