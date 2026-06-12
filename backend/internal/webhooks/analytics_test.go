package webhooks

import (
	"testing"
	"time"
)

func TestNewAnalytics(t *testing.T) {
	analytics := NewAnalytics()

	if analytics == nil {
		t.Fatal("Analytics should not be nil")
	}

	if analytics.metrics == nil {
		t.Fatal("Metrics map should be initialized")
	}

	t.Log("✓ Analytics created successfully")
}

func TestRecordDelivery(t *testing.T) {
	analytics := NewAnalytics()

	// Record successful delivery
	analytics.RecordDelivery("sub-1", "event-1", true, 100*time.Millisecond, nil)

	metrics := analytics.GetSubscriptionMetrics("sub-1")
	if metrics == nil {
		t.Fatal("Metrics should exist for sub-1")
	}

	if metrics.TotalDeliveries != 1 {
		t.Errorf("Expected 1 delivery, got %d", metrics.TotalDeliveries)
	}

	if metrics.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", metrics.SuccessCount)
	}

	if metrics.FailureCount != 0 {
		t.Errorf("Expected 0 failures, got %d", metrics.FailureCount)
	}

	t.Log("✓ Record delivery works correctly")
}

func TestRecordFailure(t *testing.T) {
	analytics := NewAnalytics()

	// Record failed delivery
	analytics.RecordDelivery("sub-1", "event-1", false, 50*time.Millisecond, &WebhookError{Message: "test error"})

	metrics := analytics.GetSubscriptionMetrics("sub-1")
	if metrics == nil {
		t.Fatal("Metrics should exist for sub-1")
	}

	if metrics.SuccessCount != 0 {
		t.Errorf("Expected 0 successes, got %d", metrics.SuccessCount)
	}

	if metrics.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", metrics.FailureCount)
	}

	if metrics.LastError == "" {
		t.Error("Expected error message to be recorded")
	}

	t.Log("✓ Record failure works correctly")
}

func TestAverageLatency(t *testing.T) {
	analytics := NewAnalytics()

	// Record multiple deliveries
	analytics.RecordDelivery("sub-1", "event-1", true, 100*time.Millisecond, nil)
	analytics.RecordDelivery("sub-1", "event-2", true, 200*time.Millisecond, nil)
	analytics.RecordDelivery("sub-1", "event-3", true, 300*time.Millisecond, nil)

	metrics := analytics.GetSubscriptionMetrics("sub-1")
	if metrics == nil {
		t.Fatal("Metrics should exist for sub-1")
	}

	// Average should be calculated (using exponential moving average)
	if metrics.AverageLatency == 0 {
		t.Error("Average latency should be calculated")
	}

	t.Logf("Average latency: %v", metrics.AverageLatency)
	t.Log("✓ Average latency calculation works")
}

func TestGetAggregateMetrics(t *testing.T) {
	analytics := NewAnalytics()

	// Record deliveries for multiple subscriptions
	analytics.RecordDelivery("sub-1", "event-1", true, 100*time.Millisecond, nil)
	analytics.RecordDelivery("sub-1", "event-2", true, 150*time.Millisecond, nil)
	analytics.RecordDelivery("sub-2", "event-3", false, 200*time.Millisecond, &WebhookError{Message: "error"})
	analytics.RecordDelivery("sub-3", "event-4", true, 120*time.Millisecond, nil)

	agg := analytics.GetAggregateMetrics()

	if agg.TotalSubscriptions != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", agg.TotalSubscriptions)
	}

	if agg.TotalDeliveries != 4 {
		t.Errorf("Expected 4 deliveries, got %d", agg.TotalDeliveries)
	}

	if agg.TotalSuccesses != 3 {
		t.Errorf("Expected 3 successes, got %d", agg.TotalSuccesses)
	}

	if agg.TotalFailures != 1 {
		t.Errorf("Expected 1 failure, got %d", agg.TotalFailures)
	}

	expectedSuccessRate := 75.0 // 3/4 * 100
	if agg.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedSuccessRate, agg.SuccessRate)
	}

	t.Log("✓ Aggregate metrics work correctly")
}

func TestGetSuccessRate(t *testing.T) {
	analytics := NewAnalytics()

	// Record 7 successes and 3 failures
	for i := 0; i < 7; i++ {
		analytics.RecordDelivery("sub-1", "event", true, 100*time.Millisecond, nil)
	}
	for i := 0; i < 3; i++ {
		analytics.RecordDelivery("sub-1", "event", false, 100*time.Millisecond, &WebhookError{Message: "error"})
	}

	successRate := analytics.GetSuccessRate("sub-1")
	expectedRate := 70.0 // 7/10 * 100

	if successRate != expectedRate {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedRate, successRate)
	}

	t.Log("✓ Success rate calculation works correctly")
}

func TestGetEventMetrics(t *testing.T) {
	analytics := NewAnalytics()

	analytics.RecordDelivery("sub-1", "event-123", true, 150*time.Millisecond, nil)

	eventMetrics := analytics.GetEventMetrics("sub-1", "event-123")
	if eventMetrics == nil {
		t.Fatal("Event metrics should exist")
	}

	if eventMetrics.EventID != "event-123" {
		t.Errorf("Expected event ID 'event-123', got '%s'", eventMetrics.EventID)
	}

	if !eventMetrics.Success {
		t.Error("Event should be marked as successful")
	}

	if eventMetrics.Duration != 150*time.Millisecond {
		t.Errorf("Expected duration 150ms, got %v", eventMetrics.Duration)
	}

	if eventMetrics.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", eventMetrics.Attempts)
	}

	t.Log("✓ Event metrics work correctly")
}

func TestRetryAttempts(t *testing.T) {
	analytics := NewAnalytics()

	// Simulate retry attempts for same event
	analytics.RecordDelivery("sub-1", "event-123", false, 100*time.Millisecond, &WebhookError{Message: "error 1"})
	analytics.RecordDelivery("sub-1", "event-123", false, 120*time.Millisecond, &WebhookError{Message: "error 2"})
	analytics.RecordDelivery("sub-1", "event-123", true, 110*time.Millisecond, nil)

	eventMetrics := analytics.GetEventMetrics("sub-1", "event-123")
	if eventMetrics == nil {
		t.Fatal("Event metrics should exist")
	}

	if eventMetrics.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", eventMetrics.Attempts)
	}

	if !eventMetrics.Success {
		t.Error("Final attempt should be successful")
	}

	t.Log("✓ Retry attempt tracking works correctly")
}

func TestResetSubscription(t *testing.T) {
	analytics := NewAnalytics()

	analytics.RecordDelivery("sub-1", "event-1", true, 100*time.Millisecond, nil)
	analytics.RecordDelivery("sub-1", "event-2", false, 150*time.Millisecond, &WebhookError{Message: "error"})

	// Verify metrics exist
	if analytics.GetSubscriptionMetrics("sub-1") == nil {
		t.Fatal("Metrics should exist before reset")
	}

	// Reset
	analytics.ResetSubscription("sub-1")

	// Verify metrics are gone
	if analytics.GetSubscriptionMetrics("sub-1") != nil {
		t.Error("Metrics should be removed after reset")
	}

	t.Log("✓ Reset subscription works correctly")
}

func TestResetAll(t *testing.T) {
	analytics := NewAnalytics()

	analytics.RecordDelivery("sub-1", "event-1", true, 100*time.Millisecond, nil)
	analytics.RecordDelivery("sub-2", "event-2", true, 150*time.Millisecond, nil)
	analytics.RecordDelivery("sub-3", "event-3", false, 200*time.Millisecond, &WebhookError{Message: "error"})

	agg := analytics.GetAggregateMetrics()
	if agg.TotalSubscriptions != 3 {
		t.Fatalf("Expected 3 subscriptions before reset, got %d", agg.TotalSubscriptions)
	}

	// Reset all
	analytics.Reset()

	agg = analytics.GetAggregateMetrics()
	if agg.TotalSubscriptions != 0 {
		t.Errorf("Expected 0 subscriptions after reset, got %d", agg.TotalSubscriptions)
	}

	if agg.TotalDeliveries != 0 {
		t.Errorf("Expected 0 deliveries after reset, got %d", agg.TotalDeliveries)
	}

	t.Log("✓ Reset all works correctly")
}

func TestHourlyStats(t *testing.T) {
	analytics := NewAnalytics()

	// Record deliveries
	for i := 0; i < 5; i++ {
		analytics.RecordDelivery("sub-1", "event", true, 100*time.Millisecond, nil)
	}
	for i := 0; i < 3; i++ {
		analytics.RecordDelivery("sub-1", "event", false, 150*time.Millisecond, &WebhookError{Message: "error"})
	}

	stats := analytics.GetHourlyStats("sub-1", 1)
	if len(stats) == 0 {
		t.Fatal("Expected hourly stats")
	}

	currentHourStat := stats[len(stats)-1]
	if currentHourStat.Deliveries != 8 {
		t.Errorf("Expected 8 deliveries in current hour, got %d", currentHourStat.Deliveries)
	}

	if currentHourStat.Successes != 5 {
		t.Errorf("Expected 5 successes in current hour, got %d", currentHourStat.Successes)
	}

	if currentHourStat.Failures != 3 {
		t.Errorf("Expected 3 failures in current hour, got %d", currentHourStat.Failures)
	}

	t.Log("✓ Hourly stats work correctly")
}

// WebhookError is a simple error type for testing
type WebhookError struct {
	Message string
}

func (e *WebhookError) Error() string {
	return e.Message
}
