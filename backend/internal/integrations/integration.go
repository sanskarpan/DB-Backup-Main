package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeJira       IntegrationType = "jira"
	IntegrationTypeServiceNow IntegrationType = "servicenow"
	IntegrationTypePagerDuty  IntegrationType = "pagerduty"
	IntegrationTypeOpsgenie   IntegrationType = "opsgenie"
	IntegrationTypeTeams      IntegrationType = "teams"
	IntegrationTypeSlack      IntegrationType = "slack"
	IntegrationTypeCustom     IntegrationType = "custom"
)

// IntegrationStatus represents the status of an integration
type IntegrationStatus string

const (
	StatusActive     IntegrationStatus = "active"
	StatusInactive   IntegrationStatus = "inactive"
	StatusError      IntegrationStatus = "error"
	StatusConfigured IntegrationStatus = "configured"
)

// Integration defines the interface that all integrations must implement
type Integration interface {
	// GetType returns the integration type
	GetType() IntegrationType

	// GetName returns the integration name
	GetName() string

	// Configure configures the integration with given settings
	Configure(config *Config) error

	// Validate validates the integration configuration
	Validate() error

	// GetStatus returns the current status of the integration
	GetStatus() IntegrationStatus

	// HealthCheck performs a health check on the integration
	HealthCheck(ctx context.Context) error

	// CreateIncident creates an incident/ticket
	CreateIncident(ctx context.Context, incident *Incident) (*IncidentResponse, error)

	// UpdateIncident updates an existing incident/ticket
	UpdateIncident(ctx context.Context, incidentID string, update *IncidentUpdate) error

	// GetIncident retrieves an incident/ticket
	GetIncident(ctx context.Context, incidentID string) (*IncidentResponse, error)

	// CloseIncident closes an incident/ticket
	CloseIncident(ctx context.Context, incidentID string, resolution string) error

	// SendNotification sends a notification
	SendNotification(ctx context.Context, notification *Notification) error

	// GetMetrics returns integration metrics
	GetMetrics() *Metrics
}

// Config holds integration configuration
type Config struct {
	Type        IntegrationType        `json:"type"`
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	BaseURL     string                 `json:"base_url"`
	APIKey      string                 `json:"api_key,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Password    string                 `json:"password,omitempty"`
	Token       string                 `json:"token,omitempty"`
	OAuth       *OAuthConfig           `json:"oauth,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	RateLimit   *RateLimitConfig       `json:"rate_limit,omitempty"`
	Retry       *RetryConfig           `json:"retry,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	MaxRetries  int                    `json:"max_retries,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// OAuthConfig holds OAuth2 configuration
type OAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	TokenURL     string   `json:"token_url"`
	AuthURL      string   `json:"auth_url"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AccessToken  string   `json:"access_token,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	BurstSize         int           `json:"burst_size"`
	RetryAfter        time.Duration `json:"retry_after"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts  int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
}

// Incident represents an incident/ticket to be created
type Incident struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    Priority               `json:"priority"`
	Severity    Severity               `json:"severity"`
	Status      string                 `json:"status,omitempty"`
	Assignee    string                 `json:"assignee,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Priority represents incident priority
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// Severity represents incident severity
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// IncidentResponse represents the response after creating an incident
type IncidentResponse struct {
	ID            string                 `json:"id"`
	Key           string                 `json:"key"`
	URL           string                 `json:"url"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CustomFields  map[string]interface{} `json:"custom_fields,omitempty"`
}

// IncidentUpdate represents an update to an incident
type IncidentUpdate struct {
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Priority     Priority               `json:"priority,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Assignee     string                 `json:"assignee,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
	Comment      string                 `json:"comment,omitempty"`
}

// Notification represents a notification to be sent
type Notification struct {
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Priority    Priority               `json:"priority"`
	Recipients  []string               `json:"recipients,omitempty"`
	Channel     string                 `json:"channel,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Metrics holds integration metrics
type Metrics struct {
	TotalRequests    int64         `json:"total_requests"`
	SuccessfulCalls  int64         `json:"successful_calls"`
	FailedCalls      int64         `json:"failed_calls"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastCallTime     time.Time     `json:"last_call_time"`
	LastError        string        `json:"last_error,omitempty"`
	LastErrorTime    time.Time     `json:"last_error_time,omitempty"`
	RateLimitHits    int64         `json:"rate_limit_hits"`
	mu               sync.RWMutex  `json:"-"`
}

// RecordRequest records a successful request
func (m *Metrics) RecordRequest(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.SuccessfulCalls++
	m.LastCallTime = time.Now()

	// Calculate average latency using exponential moving average
	if m.TotalRequests == 1 {
		m.AverageLatency = duration
	} else {
		alpha := 0.2
		m.AverageLatency = time.Duration(
			alpha*float64(duration) + (1-alpha)*float64(m.AverageLatency),
		)
	}
}

// RecordError records a failed request
func (m *Metrics) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.FailedCalls++
	m.LastError = err.Error()
	m.LastErrorTime = time.Now()
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RateLimitHits++
}

// GetSuccessRate returns the success rate percentage
func (m *Metrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return float64(m.SuccessfulCalls) / float64(m.TotalRequests) * 100
}

// Registry manages all integrations
type Registry struct {
	integrations map[string]Integration
	configs      map[string]*Config
	mu           sync.RWMutex
}

// NewRegistry creates a new integration registry
func NewRegistry() *Registry {
	return &Registry{
		integrations: make(map[string]Integration),
		configs:      make(map[string]*Config),
	}
}

// Register registers a new integration
func (r *Registry) Register(integration Integration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := integration.GetName()
	if _, exists := r.integrations[name]; exists {
		return fmt.Errorf("integration already registered: %s", name)
	}

	r.integrations[name] = integration

	log.Info().
		Str("name", name).
		Str("type", string(integration.GetType())).
		Msg("Integration registered")

	return nil
}

// Unregister removes an integration
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.integrations[name]; !exists {
		return fmt.Errorf("integration not found: %s", name)
	}

	delete(r.integrations, name)
	delete(r.configs, name)

	log.Info().Str("name", name).Msg("Integration unregistered")

	return nil
}

// Get retrieves an integration by name
func (r *Registry) Get(name string) (Integration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integration, exists := r.integrations[name]
	if !exists {
		return nil, fmt.Errorf("integration not found: %s", name)
	}

	return integration, nil
}

// List returns all registered integrations
func (r *Registry) List() []Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integrations := make([]Integration, 0, len(r.integrations))
	for _, integration := range r.integrations {
		integrations = append(integrations, integration)
	}

	return integrations
}

// ListByType returns all integrations of a specific type
func (r *Registry) ListByType(integrationType IntegrationType) []Integration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integrations := make([]Integration, 0)
	for _, integration := range r.integrations {
		if integration.GetType() == integrationType {
			integrations = append(integrations, integration)
		}
	}

	return integrations
}

// Configure configures an integration
func (r *Registry) Configure(name string, config *Config) error {
	integration, err := r.Get(name)
	if err != nil {
		return err
	}

	if err := integration.Configure(config); err != nil {
		return fmt.Errorf("failed to configure integration: %w", err)
	}

	r.mu.Lock()
	r.configs[name] = config
	r.mu.Unlock()

	return nil
}

// GetConfig retrieves the configuration for an integration
func (r *Registry) GetConfig(name string) (*Config, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.configs[name]
	if !exists {
		return nil, fmt.Errorf("configuration not found for integration: %s", name)
	}

	return config, nil
}

// HealthCheckAll performs health checks on all integrations
func (r *Registry) HealthCheckAll(ctx context.Context) map[string]error {
	integrations := r.List()
	results := make(map[string]error)

	for _, integration := range integrations {
		name := integration.GetName()
		if err := integration.HealthCheck(ctx); err != nil {
			results[name] = err
		} else {
			results[name] = nil
		}
	}

	return results
}

// GetMetricsAll returns metrics for all integrations
func (r *Registry) GetMetricsAll() map[string]*Metrics {
	integrations := r.List()
	metrics := make(map[string]*Metrics)

	for _, integration := range integrations {
		metrics[integration.GetName()] = integration.GetMetrics()
	}

	return metrics
}

// BaseIntegration provides common functionality for all integrations
type BaseIntegration struct {
	config  *Config
	metrics *Metrics
	status  IntegrationStatus
	mu      sync.RWMutex
}

// NewBaseIntegration creates a new base integration
func NewBaseIntegration() *BaseIntegration {
	return &BaseIntegration{
		metrics: &Metrics{},
		status:  StatusInactive,
	}
}

// Configure configures the base integration
func (b *BaseIntegration) Configure(config *Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.config = config
	b.status = StatusConfigured

	return nil
}

// GetConfig returns the configuration
func (b *BaseIntegration) GetConfig() *Config {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.config
}

// GetStatus returns the current status
func (b *BaseIntegration) GetStatus() IntegrationStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.status
}

// SetStatus sets the integration status
func (b *BaseIntegration) SetStatus(status IntegrationStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.status = status
}

// GetMetrics returns the metrics
func (b *BaseIntegration) GetMetrics() *Metrics {
	return b.metrics
}

// Helper functions

// MarshalJSON marshals config to JSON, omitting sensitive fields
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		*Alias
		APIKey   string `json:"api_key,omitempty"`
		Password string `json:"password,omitempty"`
		Token    string `json:"token,omitempty"`
	}{
		Alias:    (*Alias)(c),
		APIKey:   maskSensitive(c.APIKey),
		Password: maskSensitive(c.Password),
		Token:    maskSensitive(c.Token),
	})
}

func maskSensitive(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   3600,
		BurstSize:         10,
		RetryAfter:        60 * time.Second,
	}
}
