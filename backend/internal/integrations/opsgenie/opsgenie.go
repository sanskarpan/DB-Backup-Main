package opsgenie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

// OpsgenieIntegration implements integration with Opsgenie for alert/incident management.
//
//nolint:revive // OpsgenieIntegration keeps the public type name stable across the integrations API.
type OpsgenieIntegration struct {
	*integrations.BaseIntegration
	client     *http.Client
	apiURL     string
	responders []map[string]string
	tags       []string
}

// Opsgenie priority levels.
const (
	PriorityP1 = "P1" // Critical
	PriorityP2 = "P2" // High
	PriorityP3 = "P3" // Moderate
	PriorityP4 = "P4" // Low
	PriorityP5 = "P5" // Informational
)

// Alert response structures.
type createAlertResponse struct {
	Result struct {
		AlertID string `json:"alertId"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type getAlertResponse struct {
	Data struct {
		ID             string   `json:"id"`
		TinyID         string   `json:"tinyId"`
		Alias          string   `json:"alias"`
		Message        string   `json:"message"`
		Status         string   `json:"status"`
		Priority       string   `json:"priority"`
		Source         string   `json:"source"`
		Tags           []string `json:"tags"`
		CreatedAt      string   `json:"createdAt"`
		UpdatedAt      string   `json:"updatedAt"`
		AcknowledgedAt string   `json:"acknowledgedAt,omitempty"`
		ClosedAt       string   `json:"closedAt,omitempty"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

// NewOpsgenieIntegration creates a new Opsgenie integration.
func NewOpsgenieIntegration(name string) *OpsgenieIntegration {
	return &OpsgenieIntegration{
		BaseIntegration: integrations.NewBaseIntegration(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiURL: "https://api.opsgenie.com/v2",
	}
}

// GetType returns the integration type.
func (o *OpsgenieIntegration) GetType() integrations.IntegrationType {
	return integrations.IntegrationTypeOpsgenie
}

// GetName returns the integration name.
func (o *OpsgenieIntegration) GetName() string {
	config := o.GetConfig()
	if config != nil && config.Name != "" {
		return config.Name
	}
	return "opsgenie"
}

// Configure configures the Opsgenie integration.
func (o *OpsgenieIntegration) Configure(config *integrations.Config) error {
	if err := o.BaseIntegration.Configure(config); err != nil {
		return err
	}

	// Extract Opsgenie-specific settings
	if settings := config.Settings; settings != nil {
		if apiURL, ok := settings["api_url"].(string); ok && apiURL != "" {
			o.apiURL = apiURL
		}

		if responders, ok := settings["responders"].([]interface{}); ok {
			o.responders = parseResponders(responders)
		}

		if tags, ok := settings["tags"].([]interface{}); ok {
			o.tags = parseTags(tags)
		}
	}

	// Configure timeout if specified
	if config.Timeout > 0 {
		o.client.Timeout = config.Timeout
	}

	o.SetStatus(integrations.StatusActive)
	return nil
}

// parseResponders converts a raw settings slice into responder maps.
func parseResponders(responders []interface{}) []map[string]string {
	result := make([]map[string]string, 0, len(responders))
	for _, r := range responders {
		responder, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		resp := make(map[string]string)
		if id, ok := responder["id"].(string); ok {
			resp["id"] = id
		}
		if typ, ok := responder["type"].(string); ok {
			resp["type"] = typ
		}
		if len(resp) > 0 {
			result = append(result, resp)
		}
	}
	return result
}

// parseTags converts a raw settings slice into a slice of tag strings.
func parseTags(tags []interface{}) []string {
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		if tag, ok := t.(string); ok {
			result = append(result, tag)
		}
	}
	return result
}

// Validate validates the Opsgenie configuration.
func (o *OpsgenieIntegration) Validate() error {
	config := o.GetConfig()
	if config == nil {
		return fmt.Errorf("no configuration provided")
	}

	if config.APIKey == "" {
		return fmt.Errorf("opsgenie API key is required")
	}

	return nil
}

// HealthCheck performs a health check on the Opsgenie integration.
func (o *OpsgenieIntegration) HealthCheck(ctx context.Context) error {
	if err := o.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Test by getting heartbeat info or current user
	url := fmt.Sprintf("%s/account", o.apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	o.GetMetrics().RecordRequest(time.Since(start))

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = nil
		}
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateIncident creates a new alert in Opsgenie.
func (o *OpsgenieIntegration) CreateIncident(ctx context.Context, incident *integrations.Incident) (*integrations.IncidentResponse, error) {
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/alerts", o.apiURL)

	priority := o.mapPriority(incident.Priority)

	// Generate alias for deduplication
	alias := fmt.Sprintf("db-backup-%s-%d", incident.Source, incident.Timestamp.Unix())

	payload := map[string]interface{}{
		"message":     incident.Title,
		"description": incident.Description,
		"priority":    priority,
		"source":      incident.Source,
		"alias":       alias,
	}

	// Add responders if configured
	if len(o.responders) > 0 {
		payload["responders"] = o.responders
	}

	// Add tags
	tags := o.tags
	if len(tags) == 0 {
		tags = []string{"db-backup", "automated"}
	}
	payload["tags"] = tags

	// Add custom fields as details
	if len(incident.CustomFields) > 0 {
		details := make(map[string]string)
		for k, v := range incident.CustomFields {
			details[k] = fmt.Sprintf("%v", v)
		}
		payload["details"] = details
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		o.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return nil, fmt.Errorf("failed to create alert, status %d: %s", resp.StatusCode, string(body))
	}

	o.GetMetrics().RecordRequest(time.Since(start))

	var result createAlertResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &integrations.IncidentResponse{
		ID:        result.Result.AlertID,
		Status:    "open",
		CreatedAt: time.Now(),
	}, nil
}

// UpdateIncident updates an existing alert in Opsgenie.
func (o *OpsgenieIntegration) UpdateIncident(ctx context.Context, incidentID string, update *integrations.IncidentUpdate) error {
	if err := o.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Handle status updates
	if update.Status != "" {
		switch update.Status {
		case "acknowledged", "acked":
			return o.acknowledgeAlert(ctx, incidentID, update.Comment)
		case "closed", "resolved":
			return o.CloseIncident(ctx, incidentID, update.Comment)
		}
	}

	// Add note if comment provided
	if update.Comment != "" {
		return o.addNote(ctx, incidentID, update.Comment)
	}

	return nil
}

// GetIncident retrieves an alert from Opsgenie.
func (o *OpsgenieIntegration) GetIncident(ctx context.Context, incidentID string) (*integrations.IncidentResponse, error) {
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/alerts/%s", o.apiURL, incidentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		o.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return nil, fmt.Errorf("failed to get alert, status %d: %s", resp.StatusCode, string(body))
	}

	o.GetMetrics().RecordRequest(time.Since(start))

	var result getAlertResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &integrations.IncidentResponse{
		ID:     result.Data.ID,
		Key:    result.Data.TinyID,
		Status: result.Data.Status,
	}, nil
}

// CloseIncident closes an alert in Opsgenie.
func (o *OpsgenieIntegration) CloseIncident(ctx context.Context, incidentID, resolution string) error {
	if err := o.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/alerts/%s/close", o.apiURL, incidentID)

	payload := map[string]interface{}{}
	if resolution != "" {
		payload["note"] = resolution
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to close alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = nil
		}
		o.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return fmt.Errorf("failed to close alert, status %d: %s", resp.StatusCode, string(body))
	}

	o.GetMetrics().RecordRequest(time.Since(start))
	return nil
}

// SendNotification sends a notification via Opsgenie (creates a low-priority alert).
func (o *OpsgenieIntegration) SendNotification(ctx context.Context, notification *integrations.Notification) error {
	// Create a low-priority alert for notifications
	customData := notification.CustomData
	if customData == nil {
		customData = make(map[string]interface{})
	}
	customData["notification_channel"] = notification.Channel

	incident := &integrations.Incident{
		Title:        notification.Title,
		Description:  notification.Message,
		Priority:     notification.Priority,
		Tags:         notification.Tags,
		Source:       "notification",
		Timestamp:    notification.Timestamp,
		CustomFields: customData,
	}

	_, err := o.CreateIncident(ctx, incident)
	return err
}

// Helper methods

// setAuthHeaders sets authentication headers on the request.
func (o *OpsgenieIntegration) setAuthHeaders(req *http.Request) {
	config := o.GetConfig()
	req.Header.Set("Authorization", fmt.Sprintf("GenieKey %s", config.APIKey))
}

// mapPriority maps our priority to Opsgenie priority.
func (o *OpsgenieIntegration) mapPriority(priority integrations.Priority) string {
	switch priority {
	case integrations.PriorityCritical:
		return PriorityP1
	case integrations.PriorityHigh:
		return PriorityP2
	case integrations.PriorityMedium:
		return PriorityP3
	case integrations.PriorityLow:
		return PriorityP4
	default:
		return PriorityP3
	}
}

// acknowledgeAlert acknowledges an alert.
func (o *OpsgenieIntegration) acknowledgeAlert(ctx context.Context, alertID, note string) error {
	url := fmt.Sprintf("%s/alerts/%s/acknowledge", o.apiURL, alertID)

	payload := map[string]interface{}{}
	if note != "" {
		payload["note"] = note
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = nil
		}
		o.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return fmt.Errorf("failed to acknowledge alert, status %d: %s", resp.StatusCode, string(body))
	}

	o.GetMetrics().RecordRequest(time.Since(start))
	return nil
}

// addNote adds a note to an alert.
func (o *OpsgenieIntegration) addNote(ctx context.Context, alertID, note string) error {
	url := fmt.Sprintf("%s/alerts/%s/notes", o.apiURL, alertID)

	payload := map[string]interface{}{
		"note": note,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	o.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.client.Do(req)
	if err != nil {
		o.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to add note: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = nil
		}
		o.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return fmt.Errorf("failed to add note, status %d: %s", resp.StatusCode, string(body))
	}

	o.GetMetrics().RecordRequest(time.Since(start))
	return nil
}
