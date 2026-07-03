package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Manager manages webhook subscriptions and delivery.
type Manager struct {
	subscriptions  map[string]*Subscription
	deliveryQueue  chan *DeliveryTask
	analytics      *Analytics
	templates      *TemplateEngine
	circuitBreaker *CircuitBreaker
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	workers        int
}

// Subscription represents a webhook subscription.
type Subscription struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Events      []EventType       `json:"events"`       // Which events to subscribe to
	Filters     []EventFilter     `json:"filters"`      // Additional filtering rules
	Template    string            `json:"template"`     // Template name to use
	Headers     map[string]string `json:"headers"`      // Custom headers
	Secret      string            `json:"secret"`       // Secret for HMAC signing
	Enabled     bool              `json:"enabled"`      // Whether subscription is active
	RetryConfig *RetryConfig      `json:"retry_config"` // Retry configuration
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata"` // Additional metadata
}

// EventType represents the type of event.
type EventType string

const (
	EventBackupStarted    EventType = "backup.started"
	EventBackupCompleted  EventType = "backup.completed"
	EventBackupFailed     EventType = "backup.failed"
	EventRestoreStarted   EventType = "restore.started"
	EventRestoreCompleted EventType = "restore.completed"
	EventRestoreFailed    EventType = "restore.failed"
	EventScheduleCreated  EventType = "schedule.created"
	EventScheduleUpdated  EventType = "schedule.updated"
	EventScheduleDeleted  EventType = "schedule.deleted"
	EventAlertTriggered   EventType = "alert.triggered"
	EventHealthChanged    EventType = "health.changed"
	EventAll              EventType = "*" // Subscribe to all events
)

// EventFilter represents filtering criteria.
type EventFilter struct {
	Field    string      `json:"field"`    // Field to filter on (e.g., "database_id", "status")
	Operator string      `json:"operator"` // Operator (eq, ne, contains, regex)
	Value    interface{} `json:"value"`    // Value to compare against
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`      // Maximum number of retries
	InitialDelay    time.Duration `json:"initial_delay"`    // Initial delay before first retry
	MaxDelay        time.Duration `json:"max_delay"`        // Maximum delay between retries
	Multiplier      float64       `json:"multiplier"`       // Backoff multiplier
	Timeout         time.Duration `json:"timeout"`          // Request timeout
	RetryableStatus []int         `json:"retryable_status"` // HTTP status codes that should be retried
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        5 * time.Minute,
		Multiplier:      2.0,
		Timeout:         30 * time.Second,
		RetryableStatus: []int{408, 429, 500, 502, 503, 504},
	}
}

// DeliveryTask represents a webhook delivery task.
type DeliveryTask struct {
	ID             string
	SubscriptionID string
	Event          *Event
	Attempt        int
	MaxAttempts    int
	NextRetry      time.Time
	CreatedAt      time.Time
}

// Event represents an event to be sent.
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ManagerConfig holds manager configuration.
type ManagerConfig struct {
	Workers      int           // Number of delivery workers
	QueueSize    int           // Size of delivery queue
	RetryTimeout time.Duration // Global retry timeout
}

// NewManager creates a new webhook manager.
func NewManager(config *ManagerConfig) *Manager {
	if config.Workers == 0 {
		config.Workers = 10
	}
	if config.QueueSize == 0 {
		config.QueueSize = 1000
	}
	if config.RetryTimeout == 0 {
		config.RetryTimeout = 24 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		subscriptions:  make(map[string]*Subscription),
		deliveryQueue:  make(chan *DeliveryTask, config.QueueSize),
		analytics:      NewAnalytics(),
		templates:      NewTemplateEngine(),
		circuitBreaker: NewCircuitBreaker(),
		ctx:            ctx,
		cancel:         cancel,
		workers:        config.Workers,
	}

	// Start delivery workers
	for i := 0; i < config.Workers; i++ {
		go m.deliveryWorker(i)
	}

	// Start retry processor
	go m.retryProcessor()

	log.Info().
		Int("workers", config.Workers).
		Int("queue_size", config.QueueSize).
		Msg("Webhook manager started")

	return m
}

// Subscribe creates a new webhook subscription.
func (m *Manager) Subscribe(sub *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}

	if sub.Name == "" {
		return fmt.Errorf("subscription name is required")
	}

	if sub.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if len(sub.Events) == 0 {
		return fmt.Errorf("at least one event type is required")
	}

	if sub.RetryConfig == nil {
		sub.RetryConfig = DefaultRetryConfig()
	}

	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	sub.Enabled = true

	m.subscriptions[sub.ID] = sub

	log.Info().
		Str("subscription_id", sub.ID).
		Str("name", sub.Name).
		Str("url", sub.URL).
		Interface("events", sub.Events).
		Msg("Webhook subscription created")

	return nil
}

// Unsubscribe removes a webhook subscription.
func (m *Manager) Unsubscribe(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[id]; !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	delete(m.subscriptions, id)

	log.Info().Str("subscription_id", id).Msg("Webhook subscription removed")

	return nil
}

// UpdateSubscription updates an existing subscription.
func (m *Manager) UpdateSubscription(id string, updates *Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	// Update fields
	if updates.Name != "" {
		sub.Name = updates.Name
	}
	if updates.URL != "" {
		sub.URL = updates.URL
	}
	if len(updates.Events) > 0 {
		sub.Events = updates.Events
	}
	if len(updates.Filters) > 0 {
		sub.Filters = updates.Filters
	}
	if updates.Template != "" {
		sub.Template = updates.Template
	}
	if len(updates.Headers) > 0 {
		sub.Headers = updates.Headers
	}
	if updates.RetryConfig != nil {
		sub.RetryConfig = updates.RetryConfig
	}

	sub.UpdatedAt = time.Now()

	log.Info().Str("subscription_id", id).Msg("Webhook subscription updated")

	return nil
}

// EnableSubscription enables a subscription.
func (m *Manager) EnableSubscription(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	sub.Enabled = true
	sub.UpdatedAt = time.Now()

	log.Info().Str("subscription_id", id).Msg("Webhook subscription enabled")

	return nil
}

// DisableSubscription disables a subscription.
func (m *Manager) DisableSubscription(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	sub.Enabled = false
	sub.UpdatedAt = time.Now()

	log.Info().Str("subscription_id", id).Msg("Webhook subscription disabled")

	return nil
}

// GetSubscription returns a subscription by ID.
func (m *Manager) GetSubscription(id string) (*Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, exists := m.subscriptions[id]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", id)
	}

	return sub, nil
}

// ListSubscriptions returns all subscriptions.
func (m *Manager) ListSubscriptions() []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs := make([]*Subscription, 0, len(m.subscriptions))
	for _, sub := range m.subscriptions {
		subs = append(subs, sub)
	}

	return subs
}

// Publish publishes an event to all matching subscriptions.
func (m *Manager) Publish(event *Event) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	matchCount := 0
	for _, sub := range m.subscriptions {
		if !sub.Enabled {
			continue
		}

		// Check if subscription matches event
		if m.matchesSubscription(sub, event) {
			task := &DeliveryTask{
				ID:             uuid.New().String(),
				SubscriptionID: sub.ID,
				Event:          event,
				Attempt:        0,
				MaxAttempts:    sub.RetryConfig.MaxRetries + 1,
				NextRetry:      time.Now(),
				CreatedAt:      time.Now(),
			}

			select {
			case m.deliveryQueue <- task:
				matchCount++
			default:
				log.Warn().
					Str("event_id", event.ID).
					Str("subscription_id", sub.ID).
					Msg("Delivery queue full, dropping task")
				m.analytics.RecordDelivery(sub.ID, event.ID, false, 0, fmt.Errorf("queue full"))
			}
		}
	}

	log.Debug().
		Str("event_id", event.ID).
		Str("event_type", string(event.Type)).
		Int("matched_subscriptions", matchCount).
		Msg("Event published")

	return nil
}

// matchesSubscription checks if event matches subscription criteria.
func (m *Manager) matchesSubscription(sub *Subscription, event *Event) bool {
	// Check event type
	eventMatches := false
	for _, eventType := range sub.Events {
		if eventType == EventAll || eventType == event.Type {
			eventMatches = true
			break
		}
	}

	if !eventMatches {
		return false
	}

	// Apply filters
	for _, filter := range sub.Filters {
		if !m.applyFilter(filter, event) {
			return false
		}
	}

	return true
}

// applyFilter applies a filter to an event.
func (m *Manager) applyFilter(filter EventFilter, event *Event) bool {
	// Get field value from event data
	value, exists := event.Data[filter.Field]
	if !exists {
		return false
	}

	// Apply operator
	switch filter.Operator {
	case "eq":
		return value == filter.Value
	case "ne":
		return value != filter.Value
	case "contains":
		strValue, ok := value.(string)
		filterValue, ok2 := filter.Value.(string)
		if !ok || !ok2 {
			return false
		}
		return contains(strValue, filterValue)
	case "regex":
		// Implement regex matching for string values
		strValue, ok := value.(string)
		filterValue, ok2 := filter.Value.(string)
		if !ok || !ok2 {
			return false
		}

		// Compile and match regex pattern
		matched, err := regexp.MatchString(filterValue, strValue)
		if err != nil {
			// Log regex compilation error
			log.Warn().Err(err).Str("pattern", filterValue).Msg("Invalid regex pattern in webhook filter")
			return false
		}
		return matched
	default:
		return false
	}
}

// deliveryWorker processes delivery tasks.
func (m *Manager) deliveryWorker(id int) {
	log.Debug().Int("worker_id", id).Msg("Delivery worker started")

	for {
		select {
		case <-m.ctx.Done():
			log.Debug().Int("worker_id", id).Msg("Delivery worker stopped")
			return
		case task, ok := <-m.deliveryQueue:
			if !ok {
				// Channel closed
				log.Debug().Int("worker_id", id).Msg("Delivery worker stopped (channel closed)")
				return
			}
			if task != nil {
				m.processDelivery(task)
			}
		}
	}
}

// processDelivery processes a single delivery task.
func (m *Manager) processDelivery(task *DeliveryTask) {
	m.mu.RLock()
	sub, exists := m.subscriptions[task.SubscriptionID]
	m.mu.RUnlock()

	if !exists || !sub.Enabled {
		log.Warn().
			Str("task_id", task.ID).
			Str("subscription_id", task.SubscriptionID).
			Msg("Subscription not found or disabled")
		return
	}

	// Check circuit breaker
	if !m.circuitBreaker.AllowRequest(sub.ID) {
		log.Warn().
			Str("subscription_id", sub.ID).
			Msg("Circuit breaker open, skipping delivery")
		m.analytics.RecordDelivery(sub.ID, task.Event.ID, false, 0, fmt.Errorf("circuit breaker open"))
		return
	}

	// Render payload using template
	payload, err := m.templates.Render(sub.Template, task.Event)
	if err != nil {
		log.Error().
			Err(err).
			Str("subscription_id", sub.ID).
			Str("template", sub.Template).
			Msg("Failed to render template")
		m.analytics.RecordDelivery(sub.ID, task.Event.ID, false, 0, err)
		return
	}

	// Deliver webhook
	startTime := time.Now()
	err = m.deliver(sub, payload)
	duration := time.Since(startTime)

	if err != nil {
		task.Attempt++

		// Record failure
		m.circuitBreaker.RecordFailure(sub.ID)
		m.analytics.RecordDelivery(sub.ID, task.Event.ID, false, duration, err)

		// Retry if not exceeded max attempts
		if task.Attempt < task.MaxAttempts {
			delay := m.calculateRetryDelay(sub.RetryConfig, task.Attempt)
			task.NextRetry = time.Now().Add(delay)

			log.Warn().
				Err(err).
				Str("subscription_id", sub.ID).
				Str("event_id", task.Event.ID).
				Int("attempt", task.Attempt).
				Dur("retry_in", delay).
				Msg("Webhook delivery failed, will retry")

			// Re-queue for retry
			time.AfterFunc(delay, func() {
				select {
				case m.deliveryQueue <- task:
				default:
					log.Error().Str("task_id", task.ID).Msg("Failed to re-queue task")
				}
			})
		} else {
			log.Error().
				Err(err).
				Str("subscription_id", sub.ID).
				Str("event_id", task.Event.ID).
				Int("attempts", task.Attempt).
				Msg("Webhook delivery failed after all retries")
		}
	} else {
		// Record success
		m.circuitBreaker.RecordSuccess(sub.ID)
		m.analytics.RecordDelivery(sub.ID, task.Event.ID, true, duration, nil)

		log.Info().
			Str("subscription_id", sub.ID).
			Str("event_id", task.Event.ID).
			Dur("duration", duration).
			Msg("Webhook delivered successfully")
	}
}

// deliver sends the webhook HTTP request.
func (m *Manager) deliver(sub *Subscription, payload []byte) error {
	// Create HTTP client with timeout; disable redirects to prevent SSRF
	client := &http.Client{
		Timeout: sub.RetryConfig.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Create request
	req, err := http.NewRequestWithContext(m.ctx, http.MethodPost, sub.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DB-Backup-Webhook/1.0")
	for key, value := range sub.Headers {
		req.Header.Set(key, value)
	}

	// Add HMAC signature if secret is configured
	if sub.Secret != "" {
		signature := generateHMACSignature(payload, sub.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// calculateRetryDelay calculates the delay before next retry with exponential backoff.
func (m *Manager) calculateRetryDelay(config *RetryConfig, attempt int) time.Duration {
	delay := float64(config.InitialDelay) * pow(config.Multiplier, float64(attempt-1))

	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// retryProcessor processes retry tasks.
func (m *Manager) retryProcessor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// This would process delayed retries from a persistent store
			// For now, retries are handled inline in processDelivery
		}
	}
}

// Stop stops the webhook manager.
func (m *Manager) Stop() {
	log.Info().Msg("Stopping webhook manager")
	m.cancel()
	close(m.deliveryQueue)
}

// GetAnalytics returns analytics data.
func (m *Manager) GetAnalytics() *Analytics {
	return m.analytics
}

// Helper functions

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func generateHMACSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
