package teams

import (
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewTeamsIntegration(t *testing.T) {
	teams := NewTeamsIntegration("test-teams")

	if teams == nil {
		t.Fatal("Teams integration should not be nil")
	}

	if teams.GetType() != integrations.IntegrationTypeTeams {
		t.Errorf("Expected type %s, got %s", integrations.IntegrationTypeTeams, teams.GetType())
	}

	if teams.GetName() != "teams" {
		t.Errorf("Expected name 'teams', got '%s'", teams.GetName())
	}

	t.Log("✓ Teams integration created successfully")
}

func TestTeamsConfiguration(t *testing.T) {
	teams := NewTeamsIntegration("test-teams")

	config := &integrations.Config{
		Name: "my-teams",
		Settings: map[string]interface{}{
			"webhook_url": "https://outlook.office.com/webhook/test",
		},
	}

	err := teams.Configure(config)
	if err != nil {
		t.Fatalf("Failed to configure Teams: %v", err)
	}

	if teams.GetName() != "my-teams" {
		t.Errorf("Expected name 'my-teams', got '%s'", teams.GetName())
	}

	if teams.GetStatus() != integrations.StatusActive {
		t.Errorf("Expected status %s, got %s", integrations.StatusActive, teams.GetStatus())
	}

	if teams.webhookURL != "https://outlook.office.com/webhook/test" {
		t.Errorf("Expected webhook URL 'https://outlook.office.com/webhook/test', got '%s'", teams.webhookURL)
	}

	t.Log("✓ Teams configuration works")
}

func TestTeamsValidation(t *testing.T) {
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
			name: "Missing webhook URL",
			config: &integrations.Config{
				Settings: map[string]interface{}{},
			},
			expectErr: true,
		},
		{
			name: "Valid configuration",
			config: &integrations.Config{
				Settings: map[string]interface{}{
					"webhook_url": "https://outlook.office.com/webhook/test",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teams := NewTeamsIntegration("test")
			if tt.config != nil {
				teams.Configure(tt.config)
			}

			err := teams.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}

	t.Log("✓ Teams validation works")
}

func TestTeamsPriorityColorMapping(t *testing.T) {
	teams := NewTeamsIntegration("test")

	tests := []struct {
		priority integrations.Priority
		expected string
	}{
		{integrations.PriorityCritical, "dc3545"},
		{integrations.PriorityHigh, "fd7e14"},
		{integrations.PriorityMedium, "ffc107"},
		{integrations.PriorityLow, "0078D4"},
		{integrations.Priority("unknown"), "6c757d"},
	}

	for _, tt := range tests {
		result := teams.mapPriorityColor(tt.priority)
		if result != tt.expected {
			t.Errorf("For priority %s, expected %s, got %s", tt.priority, tt.expected, result)
		}
	}

	t.Log("✓ Teams priority color mapping works")
}

func TestTeamsMetrics(t *testing.T) {
	teams := NewTeamsIntegration("test")

	metrics := teams.GetMetrics()
	if metrics == nil {
		t.Fatal("Metrics should not be nil")
	}

	// Record some metrics
	metrics.RecordRequest(120 * time.Millisecond)
	metrics.RecordRequest(180 * time.Millisecond)
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

	t.Log("✓ Teams metrics tracking works")
}

// Helper types

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
