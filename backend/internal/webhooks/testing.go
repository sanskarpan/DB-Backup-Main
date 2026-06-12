package webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// MockWebhookServer provides a test server for webhook testing
type MockWebhookServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	requests      []*ReceivedRequest
	responseCode  int
	responseDelay time.Duration
	shouldFail    bool
	failCount     int
	failAfter     int
}

// ReceivedRequest represents a received webhook request
type ReceivedRequest struct {
	Headers   http.Header            `json:"headers"`
	Body      map[string]interface{} `json:"body"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature"`
}

// NewMockWebhookServer creates a new mock webhook server
func NewMockWebhookServer() *MockWebhookServer {
	mock := &MockWebhookServer{
		requests:     make([]*ReceivedRequest, 0),
		responseCode: http.StatusOK,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))

	return mock
}

// handleRequest handles incoming webhook requests
func (m *MockWebhookServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate delay if configured
	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Parse JSON body
	var bodyData map[string]interface{}
	if err := json.Unmarshal(body, &bodyData); err != nil {
		log.Debug().Err(err).Msg("Failed to parse webhook body as JSON")
		bodyData = map[string]interface{}{"raw": string(body)}
	}

	// Record request (before checking if should fail, so all requests are counted)
	req := &ReceivedRequest{
		Headers:   r.Header.Clone(),
		Body:      bodyData,
		Timestamp: time.Now(),
		Signature: r.Header.Get("X-Webhook-Signature"),
	}

	m.requests = append(m.requests, req)

	// Check if should fail
	if m.shouldFail {
		if m.failAfter > 0 && len(m.requests)-1 >= m.failAfter {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "simulated failure"}`))
			return
		}
		if m.failCount > 0 {
			m.failCount--
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "simulated failure"}`))
			return
		}
	}

	// Send response
	w.WriteHeader(m.responseCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "received"}`))
}

// URL returns the server URL
func (m *MockWebhookServer) URL() string {
	return m.server.URL
}

// Close closes the server
func (m *MockWebhookServer) Close() {
	m.server.Close()
}

// GetRequests returns all received requests
func (m *MockWebhookServer) GetRequests() []*ReceivedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]*ReceivedRequest{}, m.requests...)
}

// GetRequestCount returns the number of requests received
func (m *MockWebhookServer) GetRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.requests)
}

// Reset resets the server state
func (m *MockWebhookServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = make([]*ReceivedRequest, 0)
	m.responseCode = http.StatusOK
	m.responseDelay = 0
	m.shouldFail = false
	m.failCount = 0
	m.failAfter = 0
}

// SetResponseCode sets the HTTP response code
func (m *MockWebhookServer) SetResponseCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responseCode = code
}

// SetResponseDelay sets a delay before responding
func (m *MockWebhookServer) SetResponseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responseDelay = delay
}

// SetShouldFail sets whether requests should fail
func (m *MockWebhookServer) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shouldFail = shouldFail
}

// SetFailCount sets number of requests that should fail
func (m *MockWebhookServer) SetFailCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failCount = count
	if count > 0 {
		m.shouldFail = true
	}
}

// SetFailAfter sets failure after N successful requests
func (m *MockWebhookServer) SetFailAfter(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failAfter = count
}

// WaitForRequests waits for a specific number of requests
func (m *MockWebhookServer) WaitForRequests(count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		m.mu.Lock()
		current := len(m.requests)
		m.mu.Unlock()

		if current >= count {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %d requests, got %d", count, m.GetRequestCount())
}

// GetLastRequest returns the last received request
func (m *MockWebhookServer) GetLastRequest() *ReceivedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.requests) == 0 {
		return nil
	}

	return m.requests[len(m.requests)-1]
}

// VerifySignature verifies HMAC signature on last request
func (m *MockWebhookServer) VerifySignature(secret string) bool {
	req := m.GetLastRequest()
	if req == nil {
		return false
	}

	if req.Signature == "" {
		return false
	}

	// Re-encode body to verify signature
	body, err := json.Marshal(req.Body)
	if err != nil {
		return false
	}

	expectedSignature := generateHMACSignature(body, secret)
	return req.Signature == expectedSignature
}

// TestHelper provides helper functions for webhook testing
type TestHelper struct {
	manager *Manager
	servers map[string]*MockWebhookServer
	mu      sync.Mutex
}

// NewTestHelper creates a new test helper
func NewTestHelper() *TestHelper {
	config := &ManagerConfig{
		Workers:   2,
		QueueSize: 100,
	}

	return &TestHelper{
		manager: NewManager(config),
		servers: make(map[string]*MockWebhookServer),
	}
}

// CreateMockServer creates a new mock server and returns its URL
func (h *TestHelper) CreateMockServer(name string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	server := NewMockWebhookServer()
	h.servers[name] = server

	return server.URL()
}

// GetMockServer returns a mock server by name
func (h *TestHelper) GetMockServer(name string) *MockWebhookServer {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.servers[name]
}

// CreateSubscription creates a test subscription
func (h *TestHelper) CreateSubscription(name string, events []EventType, serverName string) (*Subscription, error) {
	server := h.GetMockServer(serverName)
	if server == nil {
		return nil, fmt.Errorf("mock server not found: %s", serverName)
	}

	sub := &Subscription{
		Name:        name,
		URL:         server.URL(),
		Events:      events,
		Secret:      "test-secret",
		RetryConfig: DefaultRetryConfig(),
	}

	if err := h.manager.Subscribe(sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// PublishEvent publishes a test event
func (h *TestHelper) PublishEvent(eventType EventType, data map[string]interface{}) error {
	event := &Event{
		Type:   eventType,
		Source: "test",
		Data:   data,
	}

	return h.manager.Publish(event)
}

// GetManager returns the webhook manager
func (h *TestHelper) GetManager() *Manager {
	return h.manager
}

// Cleanup closes all mock servers and stops the manager
func (h *TestHelper) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, server := range h.servers {
		server.Close()
	}

	h.manager.Stop()
}

// WaitForDelivery waits for webhook delivery to complete
func (h *TestHelper) WaitForDelivery(serverName string, expectedCount int, timeout time.Duration) error {
	server := h.GetMockServer(serverName)
	if server == nil {
		return fmt.Errorf("mock server not found: %s", serverName)
	}

	return server.WaitForRequests(expectedCount, timeout)
}

// EventBuilder helps build test events
type EventBuilder struct {
	event *Event
}

// NewEventBuilder creates a new event builder
func NewEventBuilder(eventType EventType) *EventBuilder {
	return &EventBuilder{
		event: &Event{
			Type:      eventType,
			Timestamp: time.Now(),
			Source:    "test",
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]interface{}),
		},
	}
}

// WithSource sets the event source
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithData sets event data
func (b *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	b.event.Data[key] = value
	return b
}

// WithMetadata sets event metadata
func (b *EventBuilder) WithMetadata(key string, value interface{}) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]interface{})
	}
	b.event.Metadata[key] = value
	return b
}

// WithTimestamp sets the event timestamp
func (b *EventBuilder) WithTimestamp(timestamp time.Time) *EventBuilder {
	b.event.Timestamp = timestamp
	return b
}

// Build returns the built event
func (b *EventBuilder) Build() *Event {
	return b.event
}

// SubscriptionBuilder helps build test subscriptions
type SubscriptionBuilder struct {
	subscription *Subscription
}

// NewSubscriptionBuilder creates a new subscription builder
func NewSubscriptionBuilder(name, url string) *SubscriptionBuilder {
	return &SubscriptionBuilder{
		subscription: &Subscription{
			Name:        name,
			URL:         url,
			Events:      []EventType{},
			Filters:     []EventFilter{},
			Headers:     make(map[string]string),
			Metadata:    make(map[string]string),
			RetryConfig: DefaultRetryConfig(),
			Enabled:     true,
		},
	}
}

// WithEvents sets the events to subscribe to
func (b *SubscriptionBuilder) WithEvents(events ...EventType) *SubscriptionBuilder {
	b.subscription.Events = events
	return b
}

// WithFilter adds an event filter
func (b *SubscriptionBuilder) WithFilter(field, operator string, value interface{}) *SubscriptionBuilder {
	b.subscription.Filters = append(b.subscription.Filters, EventFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return b
}

// WithTemplate sets the template name
func (b *SubscriptionBuilder) WithTemplate(template string) *SubscriptionBuilder {
	b.subscription.Template = template
	return b
}

// WithHeader adds a custom header
func (b *SubscriptionBuilder) WithHeader(key, value string) *SubscriptionBuilder {
	b.subscription.Headers[key] = value
	return b
}

// WithSecret sets the HMAC secret
func (b *SubscriptionBuilder) WithSecret(secret string) *SubscriptionBuilder {
	b.subscription.Secret = secret
	return b
}

// WithRetryConfig sets retry configuration
func (b *SubscriptionBuilder) WithRetryConfig(config *RetryConfig) *SubscriptionBuilder {
	b.subscription.RetryConfig = config
	return b
}

// Build returns the built subscription
func (b *SubscriptionBuilder) Build() *Subscription {
	return b.subscription
}
