package webhooks

import (
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()

	if cb == nil {
		t.Fatal("Circuit breaker should not be nil")
	}

	if cb.circuits == nil {
		t.Fatal("Circuits map should be initialized")
	}

	if cb.config == nil {
		t.Fatal("Config should be initialized")
	}

	t.Log("✓ Circuit breaker created successfully")
}

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker()

	// New circuit should start closed
	state := cb.GetState("sub-1")
	if state != StateClosed {
		t.Errorf("Expected state closed, got %s", state.String())
	}

	// Should allow requests
	if !cb.AllowRequest("sub-1") {
		t.Error("Closed circuit should allow requests")
	}

	t.Log("✓ Circuit starts in closed state")
}

func TestCircuitBreakerOpens(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenTimeout:      5 * time.Second,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Record failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure("sub-1")
	}

	// Circuit should be open
	state := cb.GetState("sub-1")
	if state != StateOpen {
		t.Errorf("Expected state open after %d failures, got %s", config.FailureThreshold, state.String())
	}

	// Should not allow requests
	if cb.AllowRequest("sub-1") {
		t.Error("Open circuit should not allow requests")
	}

	t.Log("✓ Circuit opens after threshold failures")
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Open the circuit
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	if cb.GetState("sub-1") != StateOpen {
		t.Fatal("Circuit should be open")
	}

	// Wait for open timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	allowed := cb.AllowRequest("sub-1")
	if !allowed {
		t.Error("Circuit should allow request after timeout (half-open)")
	}

	state := cb.GetState("sub-1")
	if state != StateHalfOpen {
		t.Errorf("Expected state half-open, got %s", state.String())
	}

	t.Log("✓ Circuit transitions to half-open after timeout")
}

func TestCircuitBreakerCloses(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Open the circuit
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	if cb.GetState("sub-1") != StateOpen {
		t.Fatal("Circuit should be open")
	}

	// Wait and transition to half-open
	time.Sleep(150 * time.Millisecond)
	cb.AllowRequest("sub-1")

	// Record successes to close circuit
	cb.RecordSuccess("sub-1")
	cb.RecordSuccess("sub-1")

	state := cb.GetState("sub-1")
	if state != StateClosed {
		t.Errorf("Expected state closed after %d successes in half-open, got %s", config.SuccessThreshold, state.String())
	}

	t.Log("✓ Circuit closes after threshold successes in half-open")
}

func TestCircuitBreakerReopens(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Open the circuit
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	// Wait and transition to half-open
	time.Sleep(150 * time.Millisecond)
	cb.AllowRequest("sub-1")

	if cb.GetState("sub-1") != StateHalfOpen {
		t.Fatal("Circuit should be half-open")
	}

	// Record failure in half-open
	cb.RecordFailure("sub-1")

	// Should reopen
	state := cb.GetState("sub-1")
	if state != StateOpen {
		t.Errorf("Expected state open after failure in half-open, got %s", state.String())
	}

	t.Log("✓ Circuit reopens on failure in half-open state")
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker()

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure("sub-1")
	}

	if cb.GetState("sub-1") != StateOpen {
		t.Fatal("Circuit should be open")
	}

	// Reset circuit
	cb.Reset("sub-1")

	// Should be closed
	state := cb.GetState("sub-1")
	if state != StateClosed {
		t.Errorf("Expected state closed after reset, got %s", state.String())
	}

	// Should allow requests
	if !cb.AllowRequest("sub-1") {
		t.Error("Reset circuit should allow requests")
	}

	t.Log("✓ Circuit reset works correctly")
}

func TestCircuitBreakerResetAll(t *testing.T) {
	cb := NewCircuitBreaker()

	// Open multiple circuits
	for i := 0; i < 5; i++ {
		cb.RecordFailure("sub-1")
		cb.RecordFailure("sub-2")
		cb.RecordFailure("sub-3")
	}

	// Reset all
	cb.ResetAll()

	// All should be closed
	for _, subID := range []string{"sub-1", "sub-2", "sub-3"} {
		state := cb.GetState(subID)
		if state != StateClosed {
			t.Errorf("Circuit %s should be closed after reset all, got %s", subID, state.String())
		}
	}

	t.Log("✓ Reset all circuits works correctly")
}

func TestGetCircuitInfo(t *testing.T) {
	cb := NewCircuitBreaker()

	// Record some activity
	cb.RecordSuccess("sub-1")
	cb.RecordSuccess("sub-1")
	cb.RecordFailure("sub-1")

	info := cb.GetCircuitInfo("sub-1")
	if info == nil {
		t.Fatal("Circuit info should not be nil")
	}

	if info.SubscriptionID != "sub-1" {
		t.Errorf("Expected subscription ID 'sub-1', got '%s'", info.SubscriptionID)
	}

	if info.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", info.SuccessCount)
	}

	if info.FailureCount != 1 {
		t.Errorf("Expected 1 failure, got %d", info.FailureCount)
	}

	if info.ConsecutiveFails != 1 {
		t.Errorf("Expected 1 consecutive failure, got %d", info.ConsecutiveFails)
	}

	t.Log("✓ Circuit info works correctly")
}

func TestGetAllCircuits(t *testing.T) {
	cb := NewCircuitBreaker()

	// Create multiple circuits
	cb.RecordSuccess("sub-1")
	cb.RecordSuccess("sub-2")
	cb.RecordFailure("sub-3")

	circuits := cb.GetAllCircuits()

	if len(circuits) != 3 {
		t.Errorf("Expected 3 circuits, got %d", len(circuits))
	}

	// Verify all subscription IDs are present
	found := make(map[string]bool)
	for _, circuit := range circuits {
		found[circuit.SubscriptionID] = true
	}

	for _, subID := range []string{"sub-1", "sub-2", "sub-3"} {
		if !found[subID] {
			t.Errorf("Expected to find circuit for %s", subID)
		}
	}

	t.Log("✓ Get all circuits works correctly")
}

func TestConsecutiveFailureCount(t *testing.T) {
	cb := NewCircuitBreaker()

	// Record consecutive failures
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	info := cb.GetCircuitInfo("sub-1")
	if info.ConsecutiveFails != 3 {
		t.Errorf("Expected 3 consecutive failures, got %d", info.ConsecutiveFails)
	}

	// Success should reset consecutive failures
	cb.RecordSuccess("sub-1")

	info = cb.GetCircuitInfo("sub-1")
	if info.ConsecutiveFails != 0 {
		t.Errorf("Expected consecutive failures to be reset, got %d", info.ConsecutiveFails)
	}

	t.Log("✓ Consecutive failure tracking works correctly")
}

func TestConsecutiveSuccessCount(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 3,
		OpenTimeout:      50 * time.Millisecond,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Open circuit
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	// Wait and transition to half-open
	time.Sleep(100 * time.Millisecond)
	cb.AllowRequest("sub-1")

	// Record consecutive successes
	cb.RecordSuccess("sub-1")
	cb.RecordSuccess("sub-1")

	info := cb.GetCircuitInfo("sub-1")
	if info.ConsecutiveSuccs != 2 {
		t.Errorf("Expected 2 consecutive successes, got %d", info.ConsecutiveSuccs)
	}

	// One more success should close circuit
	cb.RecordSuccess("sub-1")

	if cb.GetState("sub-1") != StateClosed {
		t.Error("Circuit should be closed after threshold successes")
	}

	t.Log("✓ Consecutive success tracking works correctly")
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("Expected state string '%s', got '%s'", tt.expected, tt.state.String())
		}
	}

	t.Log("✓ Circuit state string works correctly")
}

func TestUpdateConfig(t *testing.T) {
	cb := NewCircuitBreaker()

	newConfig := &CircuitBreakerConfig{
		FailureThreshold: 10,
		SuccessThreshold: 5,
		OpenTimeout:      2 * time.Minute,
		HalfOpenTimeout:  1 * time.Minute,
		MonitoringWindow: 10 * time.Minute,
	}

	cb.UpdateConfig(newConfig)

	config := cb.GetConfig()
	if config.FailureThreshold != 10 {
		t.Errorf("Expected failure threshold 10, got %d", config.FailureThreshold)
	}

	if config.SuccessThreshold != 5 {
		t.Errorf("Expected success threshold 5, got %d", config.SuccessThreshold)
	}

	t.Log("✓ Update config works correctly")
}

func TestMonitoringWindowReset(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 5,
		MonitoringWindow: 100 * time.Millisecond,
	}

	cb := NewCircuitBreakerWithConfig(config)

	// Record failures
	cb.RecordFailure("sub-1")
	cb.RecordFailure("sub-1")

	info := cb.GetCircuitInfo("sub-1")
	if info.ConsecutiveFails != 2 {
		t.Fatal("Expected 2 consecutive failures")
	}

	// Wait for monitoring window to expire
	time.Sleep(150 * time.Millisecond)

	// Trigger maintenance (would normally be done by monitor goroutine)
	cb.performMaintenance()

	// Consecutive failures should be reset
	info = cb.GetCircuitInfo("sub-1")
	if info.ConsecutiveFails != 0 {
		t.Errorf("Expected consecutive failures to be reset after monitoring window, got %d", info.ConsecutiveFails)
	}

	t.Log("✓ Monitoring window reset works correctly")
}

func TestNonExistentCircuit(t *testing.T) {
	cb := NewCircuitBreaker()

	// Get state of non-existent circuit
	state := cb.GetState("non-existent")
	if state != StateClosed {
		t.Errorf("Non-existent circuit should default to closed, got %s", state.String())
	}

	// Allow request for non-existent circuit
	if !cb.AllowRequest("non-existent") {
		t.Error("Non-existent circuit should allow requests")
	}

	t.Log("✓ Non-existent circuit handling works correctly")
}
