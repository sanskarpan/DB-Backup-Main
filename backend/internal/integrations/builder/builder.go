// Package builder constructs incident integrations from configuration. It lives
// in its own package (rather than internal/integrations) so it can import the
// concrete provider packages without creating an import cycle: the provider
// packages import internal/integrations, and internal/integrations must not
// import them.
package builder

import (
	"errors"
	"fmt"

	"github.com/sanskarpan/db-backup/internal/integrations"
	"github.com/sanskarpan/db-backup/internal/integrations/jira"
	"github.com/sanskarpan/db-backup/internal/integrations/opsgenie"
	"github.com/sanskarpan/db-backup/internal/integrations/pagerduty"
	"github.com/sanskarpan/db-backup/internal/integrations/servicenow"
	"github.com/sanskarpan/db-backup/internal/integrations/teams"
)

// Config aggregates the per-provider settings used to construct the set of
// enabled incident integrations.
type Config struct {
	Jira       JiraConfig
	Opsgenie   OpsgenieConfig
	ServiceNow ServiceNowConfig
	PagerDuty  PagerDutyConfig
	Teams      TeamsConfig
}

// JiraConfig holds the settings needed to open Jira issues.
type JiraConfig struct {
	Enabled    bool
	BaseURL    string
	Username   string
	APIToken   string
	ProjectKey string
	IssueType  string
}

// OpsgenieConfig holds the settings needed to open Opsgenie alerts.
type OpsgenieConfig struct {
	Enabled bool
	APIKey  string
	APIURL  string
}

// ServiceNowConfig holds the settings needed to open ServiceNow incidents.
type ServiceNowConfig struct {
	Enabled  bool
	BaseURL  string
	Username string
	Password string
}

// PagerDutyConfig holds the settings needed to trigger PagerDuty incidents.
type PagerDutyConfig struct {
	Enabled    bool
	RoutingKey string
	Token      string
}

// TeamsConfig holds the settings needed to post to Microsoft Teams.
type TeamsConfig struct {
	Enabled    bool
	WebhookURL string
}

// Build constructs and configures the set of enabled incident integrations.
// Disabled providers are skipped; an enabled provider whose configuration is
// invalid is skipped and reported via the returned joined error so the other
// providers still load.
func Build(cfg *Config) ([]integrations.Integration, error) {
	var (
		built []integrations.Integration
		errs  []error
	)

	add := func(integration integrations.Integration, ic *integrations.Config) {
		configured, err := configure(integration, ic)
		if err != nil {
			errs = append(errs, err)
			return
		}
		built = append(built, configured)
	}

	if cfg.Jira.Enabled {
		add(jira.NewJiraIntegration("jira"), jiraConfig(&cfg.Jira))
	}
	if cfg.Opsgenie.Enabled {
		add(opsgenie.NewOpsgenieIntegration("opsgenie"), opsgenieConfig(&cfg.Opsgenie))
	}
	if cfg.ServiceNow.Enabled {
		add(servicenow.NewServiceNowIntegration("servicenow"), servicenowConfig(&cfg.ServiceNow))
	}
	if cfg.PagerDuty.Enabled {
		add(pagerduty.NewPagerDutyIntegration("pagerduty"), pagerdutyConfig(&cfg.PagerDuty))
	}
	if cfg.Teams.Enabled {
		add(teams.NewTeamsIntegration("teams"), teamsConfig(&cfg.Teams))
	}

	return built, errors.Join(errs...)
}

// BuildDispatcher builds the enabled integrations and wraps them in an
// IncidentDispatcher. The dispatcher is always non-nil (empty when nothing is
// enabled); any per-provider construction error is returned alongside it.
func BuildDispatcher(cfg *Config) (*integrations.IncidentDispatcher, error) {
	built, err := Build(cfg)
	return integrations.NewIncidentDispatcher(built...), err
}

// configure configures and validates an integration, returning it only when it
// is usable.
func configure(integration integrations.Integration, ic *integrations.Config) (integrations.Integration, error) {
	if err := integration.Configure(ic); err != nil {
		return nil, fmt.Errorf("configure %s: %w", ic.Name, err)
	}
	if err := integration.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", ic.Name, err)
	}
	return integration, nil
}

func jiraConfig(c *JiraConfig) *integrations.Config {
	return &integrations.Config{
		Type:     integrations.IntegrationTypeJira,
		Name:     "jira",
		Enabled:  true,
		BaseURL:  c.BaseURL,
		Username: c.Username,
		APIKey:   c.APIToken,
		Settings: map[string]interface{}{
			"project_key": c.ProjectKey,
			"issue_type":  c.IssueType,
		},
	}
}

func opsgenieConfig(c *OpsgenieConfig) *integrations.Config {
	return &integrations.Config{
		Type:     integrations.IntegrationTypeOpsgenie,
		Name:     "opsgenie",
		Enabled:  true,
		APIKey:   c.APIKey,
		Settings: map[string]interface{}{"api_url": c.APIURL},
	}
}

func servicenowConfig(c *ServiceNowConfig) *integrations.Config {
	return &integrations.Config{
		Type:     integrations.IntegrationTypeServiceNow,
		Name:     "servicenow",
		Enabled:  true,
		BaseURL:  c.BaseURL,
		Username: c.Username,
		Password: c.Password,
	}
}

func pagerdutyConfig(c *PagerDutyConfig) *integrations.Config {
	return &integrations.Config{
		Type:     integrations.IntegrationTypePagerDuty,
		Name:     "pagerduty",
		Enabled:  true,
		Token:    c.Token,
		Settings: map[string]interface{}{"integration_key": c.RoutingKey},
	}
}

func teamsConfig(c *TeamsConfig) *integrations.Config {
	return &integrations.Config{
		Type:     integrations.IntegrationTypeTeams,
		Name:     "teams",
		Enabled:  true,
		Settings: map[string]interface{}{"webhook_url": c.WebhookURL},
	}
}
