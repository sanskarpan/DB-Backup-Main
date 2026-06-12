package webhooks

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed - circuit is closed, requests are allowed
	StateClosed CircuitState = iota
	// StateOpen - circuit is open, requests are blocked
	StateOpen
	// StateHalfOpen - circuit is testing if service recovered
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for webhooks
type CircuitBreaker struct {
	mu       sync.RWMutex
	circuits map[string]*Circuit
	config   *CircuitBreakerConfig
}

// Circuit represents a single circuit for a subscription
type Circuit struct {
	subscriptionID  string
	state           CircuitState
	failureCount    int64
	successCount    int64
	lastFailureTime time.Time
	lastSuccessTime time.Time
	lastStateChange time.Time
	consecutiveFails int
	consecutiveSuccs int
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open before closing
	SuccessThreshold int
	// OpenTimeout is how long to wait before transitioning to half-open
	OpenTimeout time.Duration
	// HalfOpenTimeout is how long to stay in half-open before going back to open
	HalfOpenTimeout time.Duration
	// MonitoringWindow is the time window for tracking failures
	MonitoringWindow time.Duration
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenTimeout:      60 * time.Second,
		HalfOpenTimeout:  30 * time.Second,
		MonitoringWindow: 5 * time.Minute,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreakerWithConfig(DefaultCircuitBreakerConfig())
}

// NewCircuitBreakerWithConfig creates a circuit breaker with custom config
func NewCircuitBreakerWithConfig(config *CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		circuits: make(map[string]*Circuit),
		config:   config,
	}

	// Start monitoring goroutine
	go cb.monitor()

	return cb
}

// AllowRequest checks if a request should be allowed for a subscription
func (cb *CircuitBreaker) AllowRequest(subscriptionID string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit := cb.getOrCreateCircuit(subscriptionID)

	switch circuit.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(circuit.lastStateChange) >= cb.config.OpenTimeout {
			circuit.state = StateHalfOpen
			circuit.lastStateChange = time.Now()
			circuit.consecutiveSuccs = 0
			log.Info().
				Str("subscription_id", subscriptionID).
				Msg("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(subscriptionID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit := cb.getOrCreateCircuit(subscriptionID)
	circuit.successCount++
	circuit.lastSuccessTime = time.Now()
	circuit.consecutiveFails = 0
	circuit.consecutiveSuccs++

	switch circuit.state {
	case StateHalfOpen:
		// Check if we should close the circuit
		if circuit.consecutiveSuccs >= cb.config.SuccessThreshold {
			circuit.state = StateClosed
			circuit.lastStateChange = time.Now()
			circuit.consecutiveSuccs = 0
			log.Info().
				Str("subscription_id", subscriptionID).
				Int64("success_count", circuit.successCount).
				Msg("Circuit breaker closed after successful recovery")
		}
	case StateOpen:
		// Should not happen, but reset if it does
		circuit.state = StateClosed
		circuit.lastStateChange = time.Now()
		log.Warn().
			Str("subscription_id", subscriptionID).
			Msg("Circuit breaker was open but received success, closing")
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(subscriptionID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit := cb.getOrCreateCircuit(subscriptionID)
	circuit.failureCount++
	circuit.lastFailureTime = time.Now()
	circuit.consecutiveSuccs = 0
	circuit.consecutiveFails++

	switch circuit.state {
	case StateClosed:
		// Check if we should open the circuit
		if circuit.consecutiveFails >= cb.config.FailureThreshold {
			circuit.state = StateOpen
			circuit.lastStateChange = time.Now()
			log.Warn().
				Str("subscription_id", subscriptionID).
				Int("consecutive_failures", circuit.consecutiveFails).
				Msg("Circuit breaker opened due to failures")
		}
	case StateHalfOpen:
		// Any failure in half-open goes back to open
		circuit.state = StateOpen
		circuit.lastStateChange = time.Now()
		circuit.consecutiveFails = 1
		log.Warn().
			Str("subscription_id", subscriptionID).
			Msg("Circuit breaker reopened due to failure in half-open state")
	}
}

// GetState returns the current state of a circuit
func (cb *CircuitBreaker) GetState(subscriptionID string) CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	circuit, exists := cb.circuits[subscriptionID]
	if !exists {
		return StateClosed
	}

	return circuit.state
}

// GetCircuitInfo returns detailed information about a circuit
func (cb *CircuitBreaker) GetCircuitInfo(subscriptionID string) *CircuitInfo {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	circuit, exists := cb.circuits[subscriptionID]
	if !exists {
		return &CircuitInfo{
			SubscriptionID: subscriptionID,
			State:          StateClosed,
		}
	}

	return &CircuitInfo{
		SubscriptionID:   subscriptionID,
		State:            circuit.state,
		FailureCount:     circuit.failureCount,
		SuccessCount:     circuit.successCount,
		LastFailureTime:  circuit.lastFailureTime,
		LastSuccessTime:  circuit.lastSuccessTime,
		LastStateChange:  circuit.lastStateChange,
		ConsecutiveFails: circuit.consecutiveFails,
		ConsecutiveSuccs: circuit.consecutiveSuccs,
	}
}

// CircuitInfo holds information about a circuit
type CircuitInfo struct {
	SubscriptionID   string       `json:"subscription_id"`
	State            CircuitState `json:"state"`
	FailureCount     int64        `json:"failure_count"`
	SuccessCount     int64        `json:"success_count"`
	LastFailureTime  time.Time    `json:"last_failure_time"`
	LastSuccessTime  time.Time    `json:"last_success_time"`
	LastStateChange  time.Time    `json:"last_state_change"`
	ConsecutiveFails int          `json:"consecutive_fails"`
	ConsecutiveSuccs int          `json:"consecutive_succs"`
}

// Reset resets a circuit for a subscription
func (cb *CircuitBreaker) Reset(subscriptionID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit, exists := cb.circuits[subscriptionID]
	if !exists {
		return
	}

	circuit.state = StateClosed
	circuit.consecutiveFails = 0
	circuit.consecutiveSuccs = 0
	circuit.lastStateChange = time.Now()

	log.Info().
		Str("subscription_id", subscriptionID).
		Msg("Circuit breaker manually reset")
}

// ResetAll resets all circuits
func (cb *CircuitBreaker) ResetAll() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for _, circuit := range cb.circuits {
		circuit.state = StateClosed
		circuit.consecutiveFails = 0
		circuit.consecutiveSuccs = 0
		circuit.lastStateChange = time.Now()
	}

	log.Info().Msg("All circuit breakers reset")
}

// GetAllCircuits returns information about all circuits
func (cb *CircuitBreaker) GetAllCircuits() []*CircuitInfo {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	infos := make([]*CircuitInfo, 0, len(cb.circuits))
	for _, circuit := range cb.circuits {
		infos = append(infos, &CircuitInfo{
			SubscriptionID:   circuit.subscriptionID,
			State:            circuit.state,
			FailureCount:     circuit.failureCount,
			SuccessCount:     circuit.successCount,
			LastFailureTime:  circuit.lastFailureTime,
			LastSuccessTime:  circuit.lastSuccessTime,
			LastStateChange:  circuit.lastStateChange,
			ConsecutiveFails: circuit.consecutiveFails,
			ConsecutiveSuccs: circuit.consecutiveSuccs,
		})
	}

	return infos
}

// getOrCreateCircuit gets or creates a circuit for a subscription
// Caller must hold lock
func (cb *CircuitBreaker) getOrCreateCircuit(subscriptionID string) *Circuit {
	circuit, exists := cb.circuits[subscriptionID]
	if !exists {
		circuit = &Circuit{
			subscriptionID:  subscriptionID,
			state:           StateClosed,
			lastStateChange: time.Now(),
		}
		cb.circuits[subscriptionID] = circuit
	}

	return circuit
}

// monitor periodically checks circuits and performs maintenance
func (cb *CircuitBreaker) monitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cb.performMaintenance()
	}
}

// performMaintenance performs periodic maintenance on circuits
func (cb *CircuitBreaker) performMaintenance() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	for subscriptionID, circuit := range cb.circuits {
		// Auto-transition half-open back to open if timeout exceeded
		if circuit.state == StateHalfOpen {
			if now.Sub(circuit.lastStateChange) >= cb.config.HalfOpenTimeout {
				circuit.state = StateOpen
				circuit.lastStateChange = now
				log.Warn().
					Str("subscription_id", subscriptionID).
					Msg("Circuit breaker reopened due to half-open timeout")
			}
		}

		// Reset consecutive failure count if outside monitoring window
		if circuit.state == StateClosed {
			if !circuit.lastFailureTime.IsZero() && now.Sub(circuit.lastFailureTime) >= cb.config.MonitoringWindow {
				if circuit.consecutiveFails > 0 {
					log.Debug().
						Str("subscription_id", subscriptionID).
						Int("consecutive_fails", circuit.consecutiveFails).
						Msg("Resetting consecutive failure count outside monitoring window")
					circuit.consecutiveFails = 0
				}
			}
		}
	}
}

// GetConfig returns the circuit breaker configuration
func (cb *CircuitBreaker) GetConfig() *CircuitBreakerConfig {
	return cb.config
}

// UpdateConfig updates the circuit breaker configuration
func (cb *CircuitBreaker) UpdateConfig(config *CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.config = config

	log.Info().
		Int("failure_threshold", config.FailureThreshold).
		Int("success_threshold", config.SuccessThreshold).
		Dur("open_timeout", config.OpenTimeout).
		Msg("Circuit breaker configuration updated")
}
