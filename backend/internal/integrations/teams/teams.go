package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

// TeamsIntegration implements the Integration interface for Microsoft Teams.
//
//nolint:revive // keeps public name stable across integration packages
type TeamsIntegration struct {
	*integrations.BaseIntegration
	client     *http.Client
	webhookURL string
}

// NewTeamsIntegration creates a new Microsoft Teams integration.
func NewTeamsIntegration(name string) *TeamsIntegration {
	return &TeamsIntegration{
		BaseIntegration: integrations.NewBaseIntegration(),
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// GetType returns the integration type.
func (t *TeamsIntegration) GetType() integrations.IntegrationType {
	return integrations.IntegrationTypeTeams
}

// GetName returns the integration name.
func (t *TeamsIntegration) GetName() string {
	config := t.GetConfig()
	if config != nil {
		return config.Name
	}
	return "teams"
}

// Configure configures the Teams integration.
func (t *TeamsIntegration) Configure(config *integrations.Config) error {
	if err := t.BaseIntegration.Configure(config); err != nil {
		return err
	}

	// Extract Teams-specific settings
	if settings := config.Settings; settings != nil {
		if webhookURL, ok := settings["webhook_url"].(string); ok {
			t.webhookURL = webhookURL
		}
	}

	// Set timeout if configured
	if config.Timeout > 0 {
		t.client.Timeout = config.Timeout
	}

	t.SetStatus(integrations.StatusActive)

	log.Info().
		Str("name", t.GetName()).
		Msg("Microsoft Teams integration configured")

	return nil
}

// Validate validates the Teams configuration.
func (t *TeamsIntegration) Validate() error {
	config := t.GetConfig()
	if config == nil {
		return fmt.Errorf("integration not configured")
	}

	if t.webhookURL == "" {
		return fmt.Errorf("webhook_url is required for Teams")
	}

	return nil
}

// HealthCheck performs a health check on the Teams integration.
func (t *TeamsIntegration) HealthCheck(ctx context.Context) error {
	if err := t.Validate(); err != nil {
		return err
	}

	// Send a simple test message
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    "Health Check",
		"themeColor": "0078D4",
		"title":      "Health Check",
		"text":       "DB Backup Teams integration health check",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		t.GetMetrics().RecordError(err)
		t.SetStatus(integrations.StatusError)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.GetMetrics().RecordError(err)
		t.SetStatus(integrations.StatusError)
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("health check failed with status: %d, body: %s", resp.StatusCode, string(body))
		t.GetMetrics().RecordError(err)
		t.SetStatus(integrations.StatusError)
		return err
	}

	t.GetMetrics().RecordRequest(duration)
	t.SetStatus(integrations.StatusActive)

	log.Debug().Str("integration", t.GetName()).Msg("Teams health check passed")

	return nil
}

// CreateIncident creates an incident notification in Teams.
func (t *TeamsIntegration) CreateIncident(ctx context.Context, incident *integrations.Incident) (*integrations.IncidentResponse, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	// Map priority to color
	themeColor := t.mapPriorityColor(incident.Priority)

	// Build MessageCard
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    incident.Title,
		"themeColor": themeColor,
		"title":      incident.Title,
		"sections": []map[string]interface{}{
			{
				"activityTitle":    "New Incident",
				"activitySubtitle": fmt.Sprintf("Priority: %s | Source: %s", incident.Priority, incident.Source),
				"facts": []map[string]string{
					{"name": "Description", "value": incident.Description},
					{"name": "Priority", "value": string(incident.Priority)},
					{"name": "Severity", "value": string(incident.Severity)},
					{"name": "Source", "value": incident.Source},
					{"name": "Timestamp", "value": incident.Timestamp.Format(time.RFC3339)},
				},
			},
		},
	}

	// Add tags if present
	if len(incident.Tags) > 0 {
		sections, ok := payload["sections"].([]map[string]interface{})
		if ok && len(sections) > 0 {
			if facts, ok := sections[0]["facts"].([]map[string]string); ok {
				facts = append(facts, map[string]string{
					"name":  "Tags",
					"value": fmt.Sprintf("%v", incident.Tags),
				})
				sections[0]["facts"] = facts
				payload["sections"] = sections
			}
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		t.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to send message, status: %d, body: %s", resp.StatusCode, string(body))
		t.GetMetrics().RecordError(err)
		return nil, err
	}

	t.GetMetrics().RecordRequest(duration)

	// Teams webhook doesn't return incident ID, so we generate one
	incidentID := fmt.Sprintf("teams-%d", time.Now().Unix())

	response := &integrations.IncidentResponse{
		ID:        incidentID,
		Key:       incidentID,
		URL:       t.webhookURL,
		Status:    "sent",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info().
		Str("integration", t.GetName()).
		Str("title", incident.Title).
		Msg("Teams incident notification sent")

	return response, nil
}

// UpdateIncident updates an incident (sends an update message).
func (t *TeamsIntegration) UpdateIncident(ctx context.Context, incidentID string, update *integrations.IncidentUpdate) error {
	if err := t.Validate(); err != nil {
		return err
	}

	themeColor := t.mapPriorityColor(update.Priority)

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    "Incident Update",
		"themeColor": themeColor,
		"title":      "Incident Update",
		"sections": []map[string]interface{}{
			{
				"activityTitle":    update.Title,
				"activitySubtitle": "Incident Updated",
				"facts": []map[string]string{
					{"name": "Incident ID", "value": incidentID},
					{"name": "Description", "value": update.Description},
				},
			},
		},
	}

	if update.Comment != "" {
		sections, ok := payload["sections"].([]map[string]interface{})
		if ok && len(sections) > 0 {
			if facts, ok := sections[0]["facts"].([]map[string]string); ok {
				facts = append(facts, map[string]string{
					"name":  "Comment",
					"value": update.Comment,
				})
				sections[0]["facts"] = facts
				payload["sections"] = sections
			}
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		t.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to send update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.GetMetrics().RecordError(readErr)
			return fmt.Errorf("failed to read response: %w", readErr)
		}
		err := fmt.Errorf("failed to send update, status: %d, body: %s", resp.StatusCode, string(body))
		t.GetMetrics().RecordError(err)
		return err
	}

	t.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", t.GetName()).
		Str("incident_id", incidentID).
		Msg("Teams incident update sent")

	return nil
}

// GetIncident retrieves an incident (not supported for Teams webhooks).
func (t *TeamsIntegration) GetIncident(ctx context.Context, incidentID string) (*integrations.IncidentResponse, error) {
	return nil, fmt.Errorf("get incident not supported for Teams webhooks")
}

// CloseIncident closes an incident (sends a closure message).
func (t *TeamsIntegration) CloseIncident(ctx context.Context, incidentID, resolution string) error {
	if err := t.Validate(); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    "Incident Closed",
		"themeColor": "28a745", // Green
		"title":      "Incident Closed",
		"sections": []map[string]interface{}{
			{
				"activityTitle":    "Incident Resolved",
				"activitySubtitle": incidentID,
				"facts": []map[string]string{
					{"name": "Incident ID", "value": incidentID},
					{"name": "Resolution", "value": resolution},
					{"name": "Closed At", "value": time.Now().Format(time.RFC3339)},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		t.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to send closure message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.GetMetrics().RecordError(readErr)
			return fmt.Errorf("failed to read response: %w", readErr)
		}
		err := fmt.Errorf("failed to send closure message, status: %d, body: %s", resp.StatusCode, string(body))
		t.GetMetrics().RecordError(err)
		return err
	}

	t.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", t.GetName()).
		Str("incident_id", incidentID).
		Msg("Teams incident closure sent")

	return nil
}

// SendNotification sends a notification via Teams.
func (t *TeamsIntegration) SendNotification(ctx context.Context, notification *integrations.Notification) error {
	if err := t.Validate(); err != nil {
		return err
	}

	themeColor := t.mapPriorityColor(notification.Priority)

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    notification.Title,
		"themeColor": themeColor,
		"title":      notification.Title,
		"text":       notification.Message,
	}

	// Add sections if tags or custom data present
	if len(notification.Tags) > 0 || len(notification.CustomData) > 0 {
		facts := []map[string]string{}

		if len(notification.Tags) > 0 {
			facts = append(facts, map[string]string{
				"name":  "Tags",
				"value": fmt.Sprintf("%v", notification.Tags),
			})
		}

		if notification.CustomData != nil {
			for k, v := range notification.CustomData {
				facts = append(facts, map[string]string{
					"name":  k,
					"value": fmt.Sprintf("%v", v),
				})
			}
		}

		payload["sections"] = []map[string]interface{}{
			{
				"facts": facts,
			},
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		t.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			t.GetMetrics().RecordError(readErr)
			return fmt.Errorf("failed to read response: %w", readErr)
		}
		err := fmt.Errorf("failed to send notification, status: %d, body: %s", resp.StatusCode, string(body))
		t.GetMetrics().RecordError(err)
		return err
	}

	t.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", t.GetName()).
		Str("title", notification.Title).
		Msg("Teams notification sent")

	return nil
}

// Helper methods

func (t *TeamsIntegration) mapPriorityColor(priority integrations.Priority) string {
	switch priority {
	case integrations.PriorityCritical:
		return "dc3545" // Red
	case integrations.PriorityHigh:
		return "fd7e14" // Orange
	case integrations.PriorityMedium:
		return "ffc107" // Yellow
	case integrations.PriorityLow:
		return "0078D4" // Blue
	default:
		return "6c757d" // Gray
	}
}
