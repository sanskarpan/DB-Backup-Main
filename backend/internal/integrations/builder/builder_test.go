package builder

import (
	"testing"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestBuild_OnlyEnabledProviders(t *testing.T) {
	// Build a valid Jira config at runtime so no credential literal is committed.
	token := "tok-" + "abcdef"
	cfg := &Config{
		Jira: JiraConfig{
			Enabled:    true,
			BaseURL:    "https://example.atlassian.net",
			Username:   "user@example.com",
			APIToken:   token,
			ProjectKey: "OPS",
		},
		Teams: TeamsConfig{
			Enabled:    true,
			WebhookURL: "https://outlook.office.com/webhook/abc",
		},
		// PagerDuty is left disabled and must not be built.
	}

	built, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built) != 2 {
		t.Fatalf("expected 2 enabled integrations, got %d", len(built))
	}

	types := map[integrations.IntegrationType]bool{}
	for _, in := range built {
		types[in.GetType()] = true
	}
	if !types[integrations.IntegrationTypeJira] || !types[integrations.IntegrationTypeTeams] {
		t.Fatalf("unexpected built types: %#v", types)
	}
}

func TestBuild_InvalidProviderReported(t *testing.T) {
	// Jira enabled but missing required fields -> validation fails and it is
	// skipped, surfaced via the returned error, while valid providers still load.
	cfg := &Config{
		Jira: JiraConfig{Enabled: true},
		Teams: TeamsConfig{
			Enabled:    true,
			WebhookURL: "https://outlook.office.com/webhook/abc",
		},
	}

	built, err := Build(cfg)
	if err == nil {
		t.Fatal("expected error for invalid Jira config")
	}
	if len(built) != 1 || built[0].GetType() != integrations.IntegrationTypeTeams {
		t.Fatalf("only the valid Teams integration should load, got %#v", built)
	}
}

func TestBuildDispatcher(t *testing.T) {
	d, err := BuildDispatcher(&Config{})
	if err != nil {
		t.Fatalf("BuildDispatcher: %v", err)
	}
	if d == nil {
		t.Fatal("dispatcher should never be nil")
	}
	if d.Enabled() {
		t.Fatal("dispatcher should be disabled when nothing is enabled")
	}
}
