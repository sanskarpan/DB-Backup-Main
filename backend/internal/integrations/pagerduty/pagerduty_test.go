package pagerduty

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewPagerDutyIntegration(t *testing.T) {
	pd := NewPagerDutyIntegration("test-pagerduty")

	if pd == nil {
		t.Fatal("PagerDuty integration should not be nil")
	}

	if pd.GetType() != integrations.IntegrationTypePagerDuty {
		t.Errorf("Expected type %s, got %s", integrations.IntegrationTypePagerDuty, pd.GetType())
	}

	if pd.GetName() != "pagerduty" {
		t.Errorf("Expected name 'pagerduty', got '%s'", pd.GetName())
	}

	t.Log("✓ PagerDuty integration created successfully")
}

func TestPagerDutyConfiguration(t *testing.T) {
	pd := NewPagerDutyIntegration("test-pd")

	config := &integrations.Config{
		Name:  "my-pagerduty",
		Token: "test-token",
		Settings: map[string]interface{}{
			"integration_key": "test-integration-key",
			"service_id":      "test-service-id",
		},
	}

	err := pd.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure PagerDuty: %v", err)
	}

	if pd.GetName() != "my-pagerduty" {
		t.Errorf("Expected name 'my-pagerduty', got '%s'", pd.GetName())
	}

	if pd.GetStatus() != integrations.StatusActive {
		t.Errorf("Expected status %s, got %s", integrations.StatusActive, pd.GetStatus())
	}

	if pd.integrationKey != "test-integration-key" {
		t.Errorf("Expected integration key 'test-integration-key', got '%s'", pd.integrationKey)
	}

	if pd.serviceID != "test-service-id" {
		t.Errorf("Expected service ID 'test-service-id', got '%s'", pd.serviceID)
	}

	t.Log("✓ PagerDuty configuration works")
}

func TestPagerDutyValidation(t *testing.T) {
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
			name: "No token or integration key",
			config: &integrations.Config{
				Settings: map[string]interface{}{},
			},
			expectErr: true,
		},
		{
			name: "Valid with token",
			config: &integrations.Config{
				Token: "test-token",
			},
			expectErr: false,
		},
		{
			name: "Valid with integration key",
			config: &integrations.Config{
				Settings: map[string]interface{}{
					"integration_key": "test-key",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := NewPagerDutyIntegration("test")
			if tt.config != nil {
				pd.Configure(tt.config)
			}

			err := pd.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}

	t.Log("✓ PagerDuty validation works")
}

func TestPagerDutySeverityMapping(t *testing.T) {
	pd := NewPagerDutyIntegration("test")

	tests := []struct {
		severity integrations.Severity
		expected string
	}{
		{integrations.SeverityCritical, "critical"},
		{integrations.SeverityHigh, "error"},
		{integrations.SeverityMedium, "warning"},
		{integrations.SeverityLow, "info"},
		{integrations.Severity("unknown"), "info"},
	}

	for _, tt := range tests {
		result := pd.mapSeverity(tt.severity)
		if result != tt.expected {
			t.Errorf("For severity %s, expected %s, got %s", tt.severity, tt.expected, result)
		}
	}

	t.Log("✓ PagerDuty severity mapping works")
}

func TestPagerDutyMetrics(t *testing.T) {
	pd := NewPagerDutyIntegration("test")

	metrics := pd.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Record some metrics
	metrics.RecordRequest(150 * time.Millisecond)
	metrics.RecordRequest(250 * time.Millisecond)
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

	t.Log("✓ PagerDuty metrics tracking works")
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
