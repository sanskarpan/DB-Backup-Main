package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
	"github.com/rs/zerolog/log"
)

// JiraIntegration implements the Integration interface for Jira
type JiraIntegration struct {
	*integrations.BaseIntegration
	client     *http.Client
	projectKey string
	issueType  string
}

// NewJiraIntegration creates a new Jira integration
func NewJiraIntegration(name string) *JiraIntegration {
	return &JiraIntegration{
		BaseIntegration: integrations.NewBaseIntegration(),
		client:          &http.Client{Timeout: 30 * time.Second},
		issueType:       "Bug", // Default issue type
	}
}

// GetType returns the integration type
func (j *JiraIntegration) GetType() integrations.IntegrationType {
	return integrations.IntegrationTypeJira
}

// GetName returns the integration name
func (j *JiraIntegration) GetName() string {
	config := j.GetConfig()
	if config != nil {
		return config.Name
	}
	return "jira"
}

// Configure configures the Jira integration
func (j *JiraIntegration) Configure(config *integrations.Config) error {
	if err := j.BaseIntegration.Configure(config); err != nil {
		return err
	}

	// Extract Jira-specific settings
	if settings := config.Settings; settings != nil {
		if projectKey, ok := settings["project_key"].(string); ok {
			j.projectKey = projectKey
		}
		if issueType, ok := settings["issue_type"].(string); ok {
			j.issueType = issueType
		}
	}

	// Set timeout if configured
	if config.Timeout > 0 {
		j.client.Timeout = config.Timeout
	}

	j.SetStatus(integrations.StatusActive)

	log.Info().
		Str("name", j.GetName()).
		Str("base_url", config.BaseURL).
		Str("project_key", j.projectKey).
		Msg("Jira integration configured")

	return nil
}

// Validate validates the Jira configuration
func (j *JiraIntegration) Validate() error {
	config := j.GetConfig()
	if config == nil {
		return fmt.Errorf("integration not configured")
	}

	if config.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if config.Username == "" || config.APIKey == "" {
		return fmt.Errorf("username and api_key are required for Jira")
	}

	if j.projectKey == "" {
		return fmt.Errorf("project_key is required")
	}

	return nil
}

// HealthCheck performs a health check on the Jira integration
func (j *JiraIntegration) HealthCheck(ctx context.Context) error {
	if err := j.Validate(); err != nil {
		return err
	}

	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/myself", config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := j.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		j.GetMetrics().RecordError(err)
		j.SetStatus(integrations.StatusError)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("health check failed with status: %d", resp.StatusCode)
		j.GetMetrics().RecordError(err)
		j.SetStatus(integrations.StatusError)
		return err
	}

	j.GetMetrics().RecordRequest(duration)
	j.SetStatus(integrations.StatusActive)

	return nil
}

// CreateIncident creates a Jira issue
func (j *JiraIntegration) CreateIncident(ctx context.Context, incident *integrations.Incident) (*integrations.IncidentResponse, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}

	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue", config.BaseURL)

	// Map priority
	priority := j.mapPriority(incident.Priority)

	// Build Jira issue payload
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": j.projectKey,
			},
			"summary":     incident.Title,
			"description": incident.Description,
			"issuetype": map[string]string{
				"name": j.issueType,
			},
			"priority": map[string]string{
				"name": priority,
			},
		},
	}

	// Add labels if provided
	if len(incident.Tags) > 0 {
		payload["fields"].(map[string]interface{})["labels"] = incident.Tags
	}

	// Add custom fields
	if incident.CustomFields != nil {
		fields := payload["fields"].(map[string]interface{})
		for k, v := range incident.CustomFields {
			fields[k] = v
		}
	}

	// Add assignee if provided
	if incident.Assignee != "" {
		payload["fields"].(map[string]interface{})["assignee"] = map[string]string{
			"name": incident.Assignee,
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := j.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		err := fmt.Errorf("failed to create issue, status: %d, body: %s", resp.StatusCode, string(body))
		j.GetMetrics().RecordError(err)
		return nil, err
	}

	var jiraResponse struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}

	if err := json.Unmarshal(body, &jiraResponse); err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	j.GetMetrics().RecordRequest(duration)

	response := &integrations.IncidentResponse{
		ID:        jiraResponse.ID,
		Key:       jiraResponse.Key,
		URL:       jiraResponse.Self,
		Status:    "Open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info().
		Str("integration", j.GetName()).
		Str("issue_key", jiraResponse.Key).
		Str("issue_id", jiraResponse.ID).
		Msg("Jira issue created")

	return response, nil
}

// UpdateIncident updates a Jira issue
func (j *JiraIntegration) UpdateIncident(ctx context.Context, issueKey string, update *integrations.IncidentUpdate) error {
	if err := j.Validate(); err != nil {
		return err
	}

	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", config.BaseURL, issueKey)

	fields := make(map[string]interface{})

	if update.Title != "" {
		fields["summary"] = update.Title
	}

	if update.Description != "" {
		fields["description"] = update.Description
	}

	if update.Priority != "" {
		fields["priority"] = map[string]string{
			"name": j.mapPriority(update.Priority),
		}
	}

	if update.Assignee != "" {
		fields["assignee"] = map[string]string{
			"name": update.Assignee,
		}
	}

	if len(update.Tags) > 0 {
		fields["labels"] = update.Tags
	}

	// Add custom fields
	if update.CustomFields != nil {
		for k, v := range update.CustomFields {
			fields[k] = v
		}
	}

	payload := map[string]interface{}{
		"fields": fields,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := j.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		j.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to update issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to update issue, status: %d, body: %s", resp.StatusCode, string(body))
		j.GetMetrics().RecordError(err)
		return err
	}

	j.GetMetrics().RecordRequest(duration)

	// Add comment if provided
	if update.Comment != "" {
		if err := j.addComment(ctx, issueKey, update.Comment); err != nil {
			log.Warn().Err(err).Str("issue_key", issueKey).Msg("Failed to add comment")
		}
	}

	log.Info().
		Str("integration", j.GetName()).
		Str("issue_key", issueKey).
		Msg("Jira issue updated")

	return nil
}

// GetIncident retrieves a Jira issue
func (j *JiraIntegration) GetIncident(ctx context.Context, issueKey string) (*integrations.IncidentResponse, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}

	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue/%s", config.BaseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := j.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to get issue, status: %d, body: %s", resp.StatusCode, string(body))
		j.GetMetrics().RecordError(err)
		return nil, err
	}

	var jiraIssue struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Self   string `json:"self"`
		Fields struct {
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Created string `json:"created"`
			Updated string `json:"updated"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(body, &jiraIssue); err != nil {
		j.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	j.GetMetrics().RecordRequest(duration)

	created, _ := time.Parse(time.RFC3339, jiraIssue.Fields.Created)
	updated, _ := time.Parse(time.RFC3339, jiraIssue.Fields.Updated)

	response := &integrations.IncidentResponse{
		ID:        jiraIssue.ID,
		Key:       jiraIssue.Key,
		URL:       jiraIssue.Self,
		Status:    jiraIssue.Fields.Status.Name,
		CreatedAt: created,
		UpdatedAt: updated,
	}

	return response, nil
}

// CloseIncident closes a Jira issue
func (j *JiraIntegration) CloseIncident(ctx context.Context, issueKey string, resolution string) error {
	if err := j.Validate(); err != nil {
		return err
	}

	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/transitions", config.BaseURL, issueKey)

	// Get available transitions
	transitions, err := j.getTransitions(ctx, issueKey)
	if err != nil {
		return err
	}

	// Find "Done" or "Closed" transition
	var transitionID string
	for _, transition := range transitions {
		if transition.Name == "Done" || transition.Name == "Closed" || transition.Name == "Close" {
			transitionID = transition.ID
			break
		}
	}

	if transitionID == "" {
		return fmt.Errorf("no transition found to close issue")
	}

	payload := map[string]interface{}{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	if resolution != "" {
		payload["fields"] = map[string]interface{}{
			"resolution": map[string]string{
				"name": resolution,
			},
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := j.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		j.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to close issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to close issue, status: %d, body: %s", resp.StatusCode, string(body))
		j.GetMetrics().RecordError(err)
		return err
	}

	j.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", j.GetName()).
		Str("issue_key", issueKey).
		Str("resolution", resolution).
		Msg("Jira issue closed")

	return nil
}

// SendNotification sends a notification (as a comment on an issue)
func (j *JiraIntegration) SendNotification(ctx context.Context, notification *integrations.Notification) error {
	// For Jira, we can't send arbitrary notifications, but we can add comments to issues
	// The issue key should be in CustomData
	if notification.CustomData == nil {
		return fmt.Errorf("issue_key required in custom_data for Jira notifications")
	}

	issueKey, ok := notification.CustomData["issue_key"].(string)
	if !ok || issueKey == "" {
		return fmt.Errorf("issue_key required in custom_data")
	}

	message := fmt.Sprintf("*%s*\n\n%s", notification.Title, notification.Message)
	return j.addComment(ctx, issueKey, message)
}

// Helper methods

func (j *JiraIntegration) mapPriority(priority integrations.Priority) string {
	switch priority {
	case integrations.PriorityCritical:
		return "Highest"
	case integrations.PriorityHigh:
		return "High"
	case integrations.PriorityMedium:
		return "Medium"
	case integrations.PriorityLow:
		return "Low"
	default:
		return "Medium"
	}
}

func (j *JiraIntegration) addComment(ctx context.Context, issueKey string, comment string) error {
	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/comment", config.BaseURL, issueKey)

	payload := map[string]interface{}{
		"body": comment,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add comment, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

type jiraTransition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (j *JiraIntegration) getTransitions(ctx context.Context, issueKey string) ([]jiraTransition, error) {
	config := j.GetConfig()
	url := fmt.Sprintf("%s/rest/api/2/issue/%s/transitions", config.BaseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get transitions, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Transitions []jiraTransition `json:"transitions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Transitions, nil
}
