package integrations

import (
	"context"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}

	if registry.integrations == nil {
		t.Fatal("Integrations map should be initialized")
	}

	if registry.configs == nil {
		t.Fatal("Configs map should be initialized")
	}

	t.Log("✓ Registry created successfully")
}

func TestBaseIntegration(t *testing.T) {
	base := NewBaseIntegration()

	if base == nil {
		t.Fatal("BaseIntegration should not be nil")
	}

	if base.GetStatus() != StatusInactive {
		t.Errorf("Expected status %s, got %s", StatusInactive, base.GetStatus())
	}

	if base.GetMetrics() == nil {
		t.Fatal("Metrics should be initialized")
	}

	t.Log("✓ BaseIntegration created successfully")
}

func TestConfigureBaseIntegration(t *testing.T) {
	base := NewBaseIntegration()

	config := &Config{
		Type:    IntegrationTypeJira,
		Name:    "test-integration",
		Enabled: true,
		BaseURL: "https://example.com",
		APIKey:  "test-key",
	}

	err := base.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}

	if base.GetStatus() != StatusConfigured {
		t.Errorf("Expected status %s, got %s", StatusConfigured, base.GetStatus())
	}

	retrievedConfig := base.GetConfig()
	if retrievedConfig.Name != "test-integration" {
		t.Errorf("Expected name 'test-integration', got '%s'", retrievedConfig.Name)
	}

	t.Log("✓ BaseIntegration configured successfully")
}

func TestMetricsRecordRequest(t *testing.T) {
	metrics := &Metrics{}

	duration := 100 * time.Millisecond
	metrics.RecordRequest(duration)

	if metrics.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulCalls != 1 {
		t.Errorf("Expected 1 successful call, got %d", metrics.SuccessfulCalls)
	}

	if metrics.AverageLatency != duration {
		t.Errorf("Expected latency %v, got %v", duration, metrics.AverageLatency)
	}

	t.Log("✓ Metrics record request works")
}

func TestMetricsRecordError(t *testing.T) {
	metrics := &Metrics{}

	err := &testError{msg: "test error"}
	metrics.RecordError(err)

	if metrics.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", metrics.TotalRequests)
	}

	if metrics.FailedCalls != 1 {
		t.Errorf("Expected 1 failed call, got %d", metrics.FailedCalls)
	}

	if metrics.LastError != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", metrics.LastError)
	}

	t.Log("✓ Metrics record error works")
}

func TestMetricsSuccessRate(t *testing.T) {
	metrics := &Metrics{}

	// Record 7 successes and 3 failures
	for i := 0; i < 7; i++ {
		metrics.RecordRequest(100 * time.Millisecond)
	}

	for i := 0; i < 3; i++ {
		metrics.RecordError(&testError{msg: "test"})
	}

	successRate := metrics.GetSuccessRate()
	expected := 70.0 // 7/10 * 100

	if successRate != expected {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expected, successRate)
	}

	t.Log("✓ Metrics success rate calculation works")
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	mockIntegration := &MockIntegration{
		typ:  IntegrationTypeJira,
		name: "test-jira",
	}

	err := registry.Register(mockIntegration)
	if err != nil {
		t.Fatalf("Failed to register integration: %v", err)
	}

	// Try to register same integration again
	err = registry.Register(mockIntegration)
	if err == nil {
		t.Error("Expected error when registering duplicate integration")
	}

	t.Log("✓ Registry register works")
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	mockIntegration := &MockIntegration{
		typ:  IntegrationTypeJira,
		name: "test-jira",
	}

	registry.Register(mockIntegration)

	retrieved, err := registry.Get("test-jira")
	if err != nil {
		t.Fatalf("Failed to get integration: %v", err)
	}

	if retrieved.GetName() != "test-jira" {
		t.Errorf("Expected name 'test-jira', got '%s'", retrieved.GetName())
	}

	// Try to get non-existent integration
	_, err = registry.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent integration")
	}

	t.Log("✓ Registry get works")
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	integrations := []Integration{
		&MockIntegration{typ: IntegrationTypeJira, name: "jira-1"},
		&MockIntegration{typ: IntegrationTypePagerDuty, name: "pagerduty-1"},
		&MockIntegration{typ: IntegrationTypeTeams, name: "teams-1"},
	}

	for _, integration := range integrations {
		registry.Register(integration)
	}

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 integrations, got %d", len(list))
	}

	t.Log("✓ Registry list works")
}

func TestRegistryListByType(t *testing.T) {
	registry := NewRegistry()

	integrations := []Integration{
		&MockIntegration{typ: IntegrationTypeJira, name: "jira-1"},
		&MockIntegration{typ: IntegrationTypeJira, name: "jira-2"},
		&MockIntegration{typ: IntegrationTypePagerDuty, name: "pagerduty-1"},
	}

	for _, integration := range integrations {
		registry.Register(integration)
	}

	jiraIntegrations := registry.ListByType(IntegrationTypeJira)
	if len(jiraIntegrations) != 2 {
		t.Errorf("Expected 2 Jira integrations, got %d", len(jiraIntegrations))
	}

	pdIntegrations := registry.ListByType(IntegrationTypePagerDuty)
	if len(pdIntegrations) != 1 {
		t.Errorf("Expected 1 PagerDuty integration, got %d", len(pdIntegrations))
	}

	t.Log("✓ Registry list by type works")
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	mockIntegration := &MockIntegration{
		typ:  IntegrationTypeJira,
		name: "test-jira",
	}

	registry.Register(mockIntegration)

	err := registry.Unregister("test-jira")
	if err != nil {
		t.Fatalf("Failed to unregister integration: %v", err)
	}

	// Verify it's removed
	_, err = registry.Get("test-jira")
	if err == nil {
		t.Error("Expected error when getting unregistered integration")
	}

	// Try to unregister non-existent integration
	err = registry.Unregister("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent integration")
	}

	t.Log("✓ Registry unregister works")
}

func TestRegistryConfigure(t *testing.T) {
	registry := NewRegistry()

	mockIntegration := &MockIntegration{
		typ:             IntegrationTypeJira,
		name:            "test-jira",
		BaseIntegration: NewBaseIntegration(),
	}

	registry.Register(mockIntegration)

	config := &Config{
		Type:    IntegrationTypeJira,
		Name:    "test-jira",
		Enabled: true,
		BaseURL: "https://example.com",
	}

	err := registry.Configure("test-jira", config)
	if err != nil {
		t.Fatalf("Failed to configure integration: %v", err)
	}

	retrievedConfig, err := registry.GetConfig("test-jira")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if retrievedConfig.BaseURL != "https://example.com" {
		t.Errorf("Expected base URL 'https://example.com', got '%s'", retrievedConfig.BaseURL)
	}

	t.Log("✓ Registry configure works")
}

func TestRegistryHealthCheckAll(t *testing.T) {
	registry := NewRegistry()

	goodIntegration := &MockIntegration{
		typ:             IntegrationTypeJira,
		name:            "good-integration",
		BaseIntegration: NewBaseIntegration(),
		healthCheckErr:  nil,
	}

	badIntegration := &MockIntegration{
		typ:             IntegrationTypePagerDuty,
		name:            "bad-integration",
		BaseIntegration: NewBaseIntegration(),
		healthCheckErr:  &testError{msg: "health check failed"},
	}

	registry.Register(goodIntegration)
	registry.Register(badIntegration)

	results := registry.HealthCheckAll(context.Background())

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results["good-integration"] != nil {
		t.Errorf("Expected good integration to pass health check, got error: %v", results["good-integration"])
	}

	if results["bad-integration"] == nil {
		t.Error("Expected bad integration to fail health check")
	}

	t.Log("✓ Registry health check all works")
}

func TestRegistryGetMetricsAll(t *testing.T) {
	registry := NewRegistry()

	integration1 := &MockIntegration{
		typ:             IntegrationTypeJira,
		name:            "integration-1",
		BaseIntegration: NewBaseIntegration(),
	}

	integration2 := &MockIntegration{
		typ:             IntegrationTypePagerDuty,
		name:            "integration-2",
		BaseIntegration: NewBaseIntegration(),
	}

	registry.Register(integration1)
	registry.Register(integration2)

	// Record some metrics
	integration1.GetMetrics().RecordRequest(100 * time.Millisecond)
	integration2.GetMetrics().RecordError(&testError{msg: "test"})

	metrics := registry.GetMetricsAll()

	if len(metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(metrics))
	}

	if metrics["integration-1"].SuccessfulCalls != 1 {
		t.Errorf("Expected 1 successful call for integration-1, got %d", metrics["integration-1"].SuccessfulCalls)
	}

	if metrics["integration-2"].FailedCalls != 1 {
		t.Errorf("Expected 1 failed call for integration-2, got %d", metrics["integration-2"].FailedCalls)
	}

	t.Log("✓ Registry get metrics all works")
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", config.MaxAttempts)
	}

	if config.InitialDelay != 1*time.Second {
		t.Errorf("Expected initial delay 1s, got %v", config.InitialDelay)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected multiplier 2.0, got %.2f", config.Multiplier)
	}

	t.Log("✓ Default retry config works")
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.RequestsPerMinute != 60 {
		t.Errorf("Expected 60 requests per minute, got %d", config.RequestsPerMinute)
	}

	if config.RequestsPerHour != 3600 {
		t.Errorf("Expected 3600 requests per hour, got %d", config.RequestsPerHour)
	}

	if config.BurstSize != 10 {
		t.Errorf("Expected burst size 10, got %d", config.BurstSize)
	}

	t.Log("✓ Default rate limit config works")
}

func TestConfigMarshalJSON(t *testing.T) {
	config := &Config{
		Type:     IntegrationTypeJira,
		Name:     "test",
		APIKey:   "secret-api-key-12345",
		Password: "secret-password-67890",
		Token:    "secret-token-abcdef",
	}

	data, err := config.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Verify sensitive data is masked
	dataStr := string(data)

	if contains(dataStr, "secret-api-key-12345") {
		t.Error("API key should be masked")
	}

	if contains(dataStr, "secret-password-67890") {
		t.Error("Password should be masked")
	}

	if contains(dataStr, "secret-token-abcdef") {
		t.Error("Token should be masked")
	}

	t.Log("✓ Config JSON marshaling with masking works")
}

// Helper types and functions

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

type MockIntegration struct {
	*BaseIntegration
	typ            IntegrationType
	name           string
	healthCheckErr error
}

func (m *MockIntegration) GetType() IntegrationType {
	return m.typ
}

func (m *MockIntegration) GetName() string {
	return m.name
}

func (m *MockIntegration) Validate() error {
	return nil
}

func (m *MockIntegration) HealthCheck(ctx context.Context) error {
	return m.healthCheckErr
}

func (m *MockIntegration) CreateIncident(ctx context.Context, incident *Incident) (*IncidentResponse, error) {
	return &IncidentResponse{
		ID:        "test-incident",
		Status:    "created",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockIntegration) UpdateIncident(ctx context.Context, incidentID string, update *IncidentUpdate) error {
	return nil
}

func (m *MockIntegration) GetIncident(ctx context.Context, incidentID string) (*IncidentResponse, error) {
	return &IncidentResponse{
		ID:     incidentID,
		Status: "open",
	}, nil
}

func (m *MockIntegration) CloseIncident(ctx context.Context, incidentID, resolution string) error {
	return nil
}

func (m *MockIntegration) SendNotification(ctx context.Context, notification *Notification) error {
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
