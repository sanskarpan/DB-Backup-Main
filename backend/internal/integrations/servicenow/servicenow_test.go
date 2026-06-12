package servicenow

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewServiceNowIntegration(t *testing.T) {
	sn := NewServiceNowIntegration("test-servicenow")

	if sn == nil {
		t.Fatal("ServiceNow integration should not be nil")
	}

	if sn.GetType() != integrations.IntegrationTypeServiceNow {
		t.Errorf("Expected type %s, got %s", integrations.IntegrationTypeServiceNow, sn.GetType())
	}

	if sn.GetName() != "servicenow" {
		t.Errorf("Expected name 'servicenow', got '%s'", sn.GetName())
	}

	t.Log("✓ ServiceNow integration created successfully")
}

func TestServiceNowConfiguration(t *testing.T) {
	sn := NewServiceNowIntegration("test-sn")

	config := &integrations.Config{
		Name:     "my-servicenow",
		BaseURL:  "https://dev12345.service-now.com",
		Username: "admin",
		Password: "test-password",
		Settings: map[string]interface{}{
			"assignment_group": "Database Team",
			"category":         "Software",
			"subcategory":      "Database",
		},
	}

	err := sn.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure ServiceNow: %v", err)
	}

	if sn.GetName() != "my-servicenow" {
		t.Errorf("Expected name 'my-servicenow', got '%s'", sn.GetName())
	}

	if sn.GetStatus() != integrations.StatusActive {
		t.Errorf("Expected status %s, got %s", integrations.StatusActive, sn.GetStatus())
	}

	if sn.assignmentGroup != "Database Team" {
		t.Errorf("Expected assignment group 'Database Team', got '%s'", sn.assignmentGroup)
	}

	if sn.category != "Software" {
		t.Errorf("Expected category 'Software', got '%s'", sn.category)
	}

	if sn.subcategory != "Database" {
		t.Errorf("Expected subcategory 'Database', got '%s'", sn.subcategory)
	}

	t.Log("✓ ServiceNow configuration works")
}

func TestServiceNowValidation(t *testing.T) {
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
			name: "Missing base URL",
			config: &integrations.Config{
				Username: "admin",
				Password: "password",
			},
			expectErr: true,
		},
		{
			name: "Missing authentication",
			config: &integrations.Config{
				BaseURL: "https://dev12345.service-now.com",
			},
			expectErr: true,
		},
		{
			name: "Valid with basic auth",
			config: &integrations.Config{
				BaseURL:  "https://dev12345.service-now.com",
				Username: "admin",
				Password: "password",
			},
			expectErr: false,
		},
		{
			name: "Valid with OAuth",
			config: &integrations.Config{
				BaseURL: "https://dev12345.service-now.com",
				OAuth: &integrations.OAuthConfig{
					AccessToken: "test-token",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sn := NewServiceNowIntegration("test")
			if tt.config != nil {
				sn.Configure(tt.config)
			}

			err := sn.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}

	t.Log("✓ ServiceNow validation works")
}

func TestServiceNowPriorityMapping(t *testing.T) {
	sn := NewServiceNowIntegration("test")

	tests := []struct {
		priority         integrations.Priority
		expectedPriority string
		expectedUrgency  string
		expectedImpact   string
	}{
		{integrations.PriorityCritical, "1", "1", "1"},
		{integrations.PriorityHigh, "2", "2", "2"},
		{integrations.PriorityMedium, "3", "2", "3"},
		{integrations.PriorityLow, "4", "3", "3"},
		{integrations.Priority("unknown"), "3", "2", "3"},
	}

	for _, tt := range tests {
		priority, urgency, impact := sn.mapPriority(tt.priority)
		if priority != tt.expectedPriority {
			t.Errorf("For priority %s, expected priority %s, got %s", tt.priority, tt.expectedPriority, priority)
		}
		if urgency != tt.expectedUrgency {
			t.Errorf("For priority %s, expected urgency %s, got %s", tt.priority, tt.expectedUrgency, urgency)
		}
		if impact != tt.expectedImpact {
			t.Errorf("For priority %s, expected impact %s, got %s", tt.priority, tt.expectedImpact, impact)
		}
	}

	t.Log("✓ ServiceNow priority mapping works")
}

func TestServiceNowStateMapping(t *testing.T) {
	sn := NewServiceNowIntegration("test")

	tests := []struct {
		state          string
		expectedStatus string
	}{
		{StateNew, "new"},
		{StateInProgress, "in_progress"},
		{StateOnHold, "on_hold"},
		{StateResolved, "resolved"},
		{StateClosed, "closed"},
		{StateCanceled, "canceled"},
		{"99", "unknown"},
	}

	for _, tt := range tests {
		status := sn.mapStateToStatus(tt.state)
		if status != tt.expectedStatus {
			t.Errorf("For state %s, expected status %s, got %s", tt.state, tt.expectedStatus, status)
		}
	}

	t.Log("✓ ServiceNow state to status mapping works")
}

func TestServiceNowStatusMapping(t *testing.T) {
	sn := NewServiceNowIntegration("test")

	tests := []struct {
		status        string
		expectedState string
	}{
		{"new", StateNew},
		{"in_progress", StateInProgress},
		{"on_hold", StateOnHold},
		{"resolved", StateResolved},
		{"closed", StateClosed},
		{"canceled", StateCanceled},
		{"unknown", StateInProgress},
	}

	for _, tt := range tests {
		state := sn.mapStatusToState(tt.status)
		if state != tt.expectedState {
			t.Errorf("For status %s, expected state %s, got %s", tt.status, tt.expectedState, state)
		}
	}

	t.Log("✓ ServiceNow status to state mapping works")
}

func TestServiceNowMetrics(t *testing.T) {
	sn := NewServiceNowIntegration("test")

	metrics := sn.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Record some metrics
	metrics.RecordRequest(130 * time.Millisecond)
	metrics.RecordRequest(270 * time.Millisecond)
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

	t.Log("✓ ServiceNow metrics tracking works")
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
