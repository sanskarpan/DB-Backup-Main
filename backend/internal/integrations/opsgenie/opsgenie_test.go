package opsgenie

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewOpsgenieIntegration(t *testing.T) {
	og := NewOpsgenieIntegration("test-opsgenie")

	if og == nil {
		t.Fatal("Opsgenie integration should not be nil")
	}

	if og.GetType() != integrations.IntegrationTypeOpsgenie {
		t.Errorf("Expected type %s, got %s", integrations.IntegrationTypeOpsgenie, og.GetType())
	}

	if og.GetName() != "opsgenie" {
		t.Errorf("Expected name 'opsgenie', got '%s'", og.GetName())
	}

	if og.apiURL != "https://api.opsgenie.com/v2" {
		t.Errorf("Expected default API URL 'https://api.opsgenie.com/v2', got '%s'", og.apiURL)
	}

	t.Log("✓ Opsgenie integration created successfully")
}

func TestOpsgenieConfiguration(t *testing.T) {
	og := NewOpsgenieIntegration("test-og")

	config := &integrations.Config{
		Name:   "my-opsgenie",
		APIKey: "test-api-key",
		Settings: map[string]interface{}{
			"api_url": "https://api.eu.opsgenie.com/v2",
			"responders": []interface{}{
				map[string]interface{}{
					"id":   "team-123",
					"type": "team",
				},
				map[string]interface{}{
					"id":   "user-456",
					"type": "user",
				},
			},
			"tags": []interface{}{"database", "backup", "production"},
		},
	}

	err := og.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure Opsgenie: %v", err)
	}

	if og.GetName() != "my-opsgenie" {
		t.Errorf("Expected name 'my-opsgenie', got '%s'", og.GetName())
	}

	if og.GetStatus() != integrations.StatusActive {
		t.Errorf("Expected status %s, got %s", integrations.StatusActive, og.GetStatus())
	}

	if og.apiURL != "https://api.eu.opsgenie.com/v2" {
		t.Errorf("Expected API URL 'https://api.eu.opsgenie.com/v2', got '%s'", og.apiURL)
	}

	if len(og.responders) != 2 {
		t.Errorf("Expected 2 responders, got %d", len(og.responders))
	}

	if len(og.tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(og.tags))
	}

	if og.tags[0] != "database" {
		t.Errorf("Expected first tag 'database', got '%s'", og.tags[0])
	}

	t.Log("✓ Opsgenie configuration works")
}

func TestOpsgenieValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *integrations.Config
		expectErr bool
	}{
		{
			name:      "No configuration",
			config:    nil,
			expectErr: true,
		},
		{
			name: "Missing API key",
			config: &integrations.Config{
				Settings: map[string]interface{}{},
			},
			expectErr: true,
		},
		{
			name: "Valid configuration",
			config: &integrations.Config{
				APIKey: "test-api-key",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			og := NewOpsgenieIntegration("test")
			if tt.config != nil {
				og.Configure(tt.config)
			}

			err := og.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}

	t.Log("✓ Opsgenie validation works")
}

func TestOpsgeniePriorityMapping(t *testing.T) {
	og := NewOpsgenieIntegration("test")

	tests := []struct {
		priority integrations.Priority
		expected string
	}{
		{integrations.PriorityCritical, PriorityP1},
		{integrations.PriorityHigh, PriorityP2},
		{integrations.PriorityMedium, PriorityP3},
		{integrations.PriorityLow, PriorityP4},
		{integrations.Priority("unknown"), PriorityP3},
	}

	for _, tt := range tests {
		result := og.mapPriority(tt.priority)
		if result != tt.expected {
			t.Errorf("For priority %s, expected %s, got %s", tt.priority, tt.expected, result)
		}
	}

	t.Log("✓ Opsgenie priority mapping works")
}

func TestOpsgenieResponders(t *testing.T) {
	og := NewOpsgenieIntegration("test")

	config := &integrations.Config{
		APIKey: "test-key",
		Settings: map[string]interface{}{
			"responders": []interface{}{
				map[string]interface{}{
					"id":   "database-team",
					"type": "team",
				},
			},
		},
	}

	err := og.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}

	if len(og.responders) != 1 {
		t.Errorf("Expected 1 responder, got %d", len(og.responders))
	}

	if og.responders[0]["id"] != "database-team" {
		t.Errorf("Expected responder ID 'database-team', got '%s'", og.responders[0]["id"])
	}

	if og.responders[0]["type"] != "team" {
		t.Errorf("Expected responder type 'team', got '%s'", og.responders[0]["type"])
	}

	t.Log("✓ Opsgenie responders configuration works")
}

func TestOpsgenieTags(t *testing.T) {
	og := NewOpsgenieIntegration("test")

	config := &integrations.Config{
		APIKey: "test-key",
		Settings: map[string]interface{}{
			"tags": []interface{}{"production", "database", "critical"},
		},
	}

	err := og.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}

	if len(og.tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(og.tags))
	}

	expectedTags := []string{"production", "database", "critical"}
	for i, expected := range expectedTags {
		if og.tags[i] != expected {
			t.Errorf("Expected tag[%d] '%s', got '%s'", i, expected, og.tags[i])
		}
	}

	t.Log("✓ Opsgenie tags configuration works")
}

func TestOpsgenieMetrics(t *testing.T) {
	og := NewOpsgenieIntegration("test")

	metrics := og.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Record some metrics
	metrics.RecordRequest(140 * time.Millisecond)
	metrics.RecordRequest(260 * time.Millisecond)
	metrics.RecordError(&testError{msg: "test error"})

	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulCalls != 2 {
		t.Errorf("Expected 2 successful calls, got %d", metrics.SuccessfulCalls)
	}

	if metrics.FailedCalls != 1 {
		t.Errorf("Expected 1 failed call, got %d", metrics.FailedCalls)
	}

	t.Log("✓ Opsgenie metrics tracking works")
}

func TestOpsgenieCustomAPIURL(t *testing.T) {
	og := NewOpsgenieIntegration("test")

	// Test EU region
	config := &integrations.Config{
		APIKey: "test-key",
		Settings: map[string]interface{}{
			"api_url": "https://api.eu.opsgenie.com/v2",
		},
	}

	err := og.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure: %v", err)
	}

	if og.apiURL != "https://api.eu.opsgenie.com/v2" {
		t.Errorf("Expected EU API URL, got '%s'", og.apiURL)
	}

	t.Log("✓ Opsgenie custom API URL configuration works")
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
