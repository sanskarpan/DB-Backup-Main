package jira

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewJiraIntegration(t *testing.T) {
	jira := NewJiraIntegration("test-jira")

	if jira == nil {
		t.Fatal("Jira integration should not be nil")
	}

	if jira.GetType() != integrations.IntegrationTypeJira {
		t.Errorf("Expected type %s, got %s", integrations.IntegrationTypeJira, jira.GetType())
	}

	if jira.GetName() != "jira" {
		t.Errorf("Expected name 'jira', got '%s'", jira.GetName())
	}

	t.Log("✓ Jira integration created successfully")
}

func TestJiraConfiguration(t *testing.T) {
	jira := NewJiraIntegration("test-jira")

	config := &integrations.Config{
		Name:     "my-jira",
		BaseURL:  "https://example.atlassian.net",
		Username: "user@example.com",
		APIKey:   "test-api-key",
		Settings: map[string]interface{}{
			"project_key": "TEST",
			"issue_type":  "Task",
		},
	}

	err := jira.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure Jira: %v", err)
	}

	if jira.GetName() != "my-jira" {
		t.Errorf("Expected name 'my-jira', got '%s'", jira.GetName())
	}

	if jira.GetStatus() != integrations.StatusActive {
		t.Errorf("Expected status %s, got %s", integrations.StatusActive, jira.GetStatus())
	}

	if jira.projectKey != "TEST" {
		t.Errorf("Expected project key 'TEST', got '%s'", jira.projectKey)
	}

	if jira.issueType != "Task" {
		t.Errorf("Expected issue type 'Task', got '%s'", jira.issueType)
	}

	t.Log("✓ Jira configuration works")
}

func TestJiraValidation(t *testing.T) {
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
				Username: "user",
				APIKey:   "key",
				Settings: map[string]interface{}{
					"project_key": "TEST",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing credentials",
			config: &integrations.Config{
				BaseURL: "https://example.atlassian.net",
				Settings: map[string]interface{}{
					"project_key": "TEST",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing project key",
			config: &integrations.Config{
				BaseURL:  "https://example.atlassian.net",
				Username: "user",
				APIKey:   "key",
			},
			expectErr: true,
		},
		{
			name: "Valid configuration",
			config: &integrations.Config{
				BaseURL:  "https://example.atlassian.net",
				Username: "user",
				APIKey:   "key",
				Settings: map[string]interface{}{
					"project_key": "TEST",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jira := NewJiraIntegration("test")
			if tt.config != nil {
				jira.Configure(tt.config)
			}

			err := jira.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}

	t.Log("✓ Jira validation works")
}

func TestJiraPriorityMapping(t *testing.T) {
	jira := NewJiraIntegration("test")

	tests := []struct {
		priority integrations.Priority
		expected string
	}{
		{integrations.PriorityCritical, "Highest"},
		{integrations.PriorityHigh, "High"},
		{integrations.PriorityMedium, "Medium"},
		{integrations.PriorityLow, "Low"},
		{integrations.Priority("unknown"), "Medium"},
	}

	for _, tt := range tests {
		result := jira.mapPriority(tt.priority)
		if result != tt.expected {
			t.Errorf("For priority %s, expected %s, got %s", tt.priority, tt.expected, result)
		}
	}

	t.Log("✓ Jira priority mapping works")
}

func TestJiraMetrics(t *testing.T) {
	jira := NewJiraIntegration("test")

	metrics := jira.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Record some metrics
	metrics.RecordRequest(100 * time.Millisecond)
	metrics.RecordRequest(200 * time.Millisecond)
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

	successRate := metrics.GetSuccessRate()
	expectedRate := 66.66666666666666

	if successRate != expectedRate {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedRate, successRate)
	}

	t.Log("✓ Jira metrics tracking works")
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
