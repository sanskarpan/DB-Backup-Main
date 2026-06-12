package pagerduty

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

// PagerDutyIntegration implements the Integration interface for PagerDuty
type PagerDutyIntegration struct {
	*integrations.BaseIntegration
	client         *http.Client
	integrationKey string
	serviceID      string
}

// NewPagerDutyIntegration creates a new PagerDuty integration
func NewPagerDutyIntegration(name string) *PagerDutyIntegration {
	return &PagerDutyIntegration{
		BaseIntegration: integrations.NewBaseIntegration(),
		client:          &http.Client{Timeout: 30 * time.Second},
	}
}

// GetType returns the integration type
func (p *PagerDutyIntegration) GetType() integrations.IntegrationType {
	return integrations.IntegrationTypePagerDuty
}

// GetName returns the integration name
func (p *PagerDutyIntegration) GetName() string {
	config := p.GetConfig()
	if config != nil {
		return config.Name
	}
	return "pagerduty"
}

// Configure configures the PagerDuty integration
func (p *PagerDutyIntegration) Configure(config *integrations.Config) error {
	if err := p.BaseIntegration.Configure(config); err != nil {
		return err
	}

	// Extract PagerDuty-specific settings
	if settings := config.Settings; settings != nil {
		if integrationKey, ok := settings["integration_key"].(string); ok {
			p.integrationKey = integrationKey
		}
		if serviceID, ok := settings["service_id"].(string); ok {
			p.serviceID = serviceID
		}
	}

	// Set timeout if configured
	if config.Timeout > 0 {
		p.client.Timeout = config.Timeout
	}

	p.SetStatus(integrations.StatusActive)

	log.Info().
		Str("name", p.GetName()).
		Str("service_id", p.serviceID).
		Msg("PagerDuty integration configured")

	return nil
}

// Validate validates the PagerDuty configuration
func (p *PagerDutyIntegration) Validate() error {
	config := p.GetConfig()
	if config == nil {
		return fmt.Errorf("integration not configured")
	}

	if config.Token == "" && p.integrationKey == "" {
		return fmt.Errorf("token or integration_key is required for PagerDuty")
	}

	return nil
}

// HealthCheck performs a health check on the PagerDuty integration
func (p *PagerDutyIntegration) HealthCheck(ctx context.Context) error {
	if err := p.Validate(); err != nil {
		return err
	}

	config := p.GetConfig()

	// Use Events API v2 for health check
	url := "https://events.pagerduty.com/v2/enqueue"

	// Send a test event
	payload := map[string]interface{}{
		"routing_key":  p.integrationKey,
		"event_action": "trigger",
		"dedup_key":    fmt.Sprintf("health-check-%d", time.Now().Unix()),
		"payload": map[string]interface{}{
			"summary":  "Health Check",
			"severity": "info",
			"source":   "db-backup-health-check",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", config.Token))
	}

	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		p.GetMetrics().RecordError(err)
		p.SetStatus(integrations.StatusError)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusAccepted {
		err := fmt.Errorf("health check failed with status: %d, body: %s", resp.StatusCode, string(body))
		p.GetMetrics().RecordError(err)
		p.SetStatus(integrations.StatusError)
		return err
	}

	p.GetMetrics().RecordRequest(duration)
	p.SetStatus(integrations.StatusActive)

	log.Debug().Str("integration", p.GetName()).Msg("PagerDuty health check passed")

	return nil
}

// CreateIncident creates a PagerDuty incident
func (p *PagerDutyIntegration) CreateIncident(ctx context.Context, incident *integrations.Incident) (*integrations.IncidentResponse, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	url := "https://events.pagerduty.com/v2/enqueue"

	// Map severity
	severity := p.mapSeverity(incident.Severity)

	// Generate unique dedup key
	dedupKey := fmt.Sprintf("db-backup-%s-%d", incident.Source, incident.Timestamp.Unix())

	payload := map[string]interface{}{
		"routing_key":  p.integrationKey,
		"event_action": "trigger",
		"dedup_key":    dedupKey,
		"payload": map[string]interface{}{
			"summary":   incident.Title,
			"severity":  severity,
			"source":    incident.Source,
			"timestamp": incident.Timestamp.Format(time.RFC3339),
		},
	}

	// Add custom details
	if incident.Description != "" || len(incident.Tags) > 0 || len(incident.CustomFields) > 0 {
		customDetails := make(map[string]interface{})

		if incident.Description != "" {
			customDetails["description"] = incident.Description
		}

		if len(incident.Tags) > 0 {
			customDetails["tags"] = incident.Tags
		}

		// Merge custom fields
		for k, v := range incident.CustomFields {
			customDetails[k] = v
		}

		payload["payload"].(map[string]interface{})["custom_details"] = customDetails
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		err := fmt.Errorf("failed to create incident, status: %d, body: %s", resp.StatusCode, string(body))
		p.GetMetrics().RecordError(err)
		return nil, err
	}

	var pdResponse struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		DedupKey string `json:"dedup_key"`
	}

	if err := json.Unmarshal(body, &pdResponse); err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	p.GetMetrics().RecordRequest(duration)

	response := &integrations.IncidentResponse{
		ID:        pdResponse.DedupKey,
		Key:       pdResponse.DedupKey,
		URL:       fmt.Sprintf("https://app.pagerduty.com/incidents/%s", pdResponse.DedupKey),
		Status:    "triggered",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info().
		Str("integration", p.GetName()).
		Str("dedup_key", pdResponse.DedupKey).
		Str("status", pdResponse.Status).
		Msg("PagerDuty incident created")

	return response, nil
}

// UpdateIncident updates a PagerDuty incident
func (p *PagerDutyIntegration) UpdateIncident(ctx context.Context, dedupKey string, update *integrations.IncidentUpdate) error {
	// PagerDuty doesn't support direct updates via Events API
	// We can trigger a new event with the same dedup_key
	if err := p.Validate(); err != nil {
		return err
	}

	url := "https://events.pagerduty.com/v2/enqueue"

	payload := map[string]interface{}{
		"routing_key":  p.integrationKey,
		"event_action": "trigger",
		"dedup_key":    dedupKey,
		"payload": map[string]interface{}{
			"summary":  update.Title,
			"severity": string(update.Priority),
			"source":   "db-backup-update",
		},
	}

	if update.Comment != "" {
		payload["payload"].(map[string]interface{})["custom_details"] = map[string]string{
			"update": update.Comment,
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

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		p.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to update incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to update incident, status: %d, body: %s", resp.StatusCode, string(body))
		p.GetMetrics().RecordError(err)
		return err
	}

	p.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", p.GetName()).
		Str("dedup_key", dedupKey).
		Msg("PagerDuty incident updated")

	return nil
}

// GetIncident retrieves a PagerDuty incident
func (p *PagerDutyIntegration) GetIncident(ctx context.Context, incidentID string) (*integrations.IncidentResponse, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	config := p.GetConfig()
	if config.Token == "" {
		return nil, fmt.Errorf("API token required to get incident details")
	}

	url := fmt.Sprintf("https://api.pagerduty.com/incidents/%s", incidentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
	req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", config.Token))

	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to get incident, status: %d, body: %s", resp.StatusCode, string(body))
		p.GetMetrics().RecordError(err)
		return nil, err
	}

	var pdIncident struct {
		Incident struct {
			ID          string    `json:"id"`
			IncidentKey string    `json:"incident_key"`
			Status      string    `json:"status"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
			HTMLURL     string    `json:"html_url"`
		} `json:"incident"`
	}

	if err := json.Unmarshal(body, &pdIncident); err != nil {
		p.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	p.GetMetrics().RecordRequest(duration)

	response := &integrations.IncidentResponse{
		ID:        pdIncident.Incident.ID,
		Key:       pdIncident.Incident.IncidentKey,
		URL:       pdIncident.Incident.HTMLURL,
		Status:    pdIncident.Incident.Status,
		CreatedAt: pdIncident.Incident.CreatedAt,
		UpdatedAt: pdIncident.Incident.UpdatedAt,
	}

	return response, nil
}

// CloseIncident closes/resolves a PagerDuty incident
func (p *PagerDutyIntegration) CloseIncident(ctx context.Context, dedupKey string, resolution string) error {
	if err := p.Validate(); err != nil {
		return err
	}

	url := "https://events.pagerduty.com/v2/enqueue"

	payload := map[string]interface{}{
		"routing_key":  p.integrationKey,
		"event_action": "resolve",
		"dedup_key":    dedupKey,
	}

	if resolution != "" {
		payload["payload"] = map[string]interface{}{
			"summary": resolution,
			"source":  "db-backup",
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

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		p.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to resolve incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to resolve incident, status: %d, body: %s", resp.StatusCode, string(body))
		p.GetMetrics().RecordError(err)
		return err
	}

	p.GetMetrics().RecordRequest(duration)

	log.Info().
		Str("integration", p.GetName()).
		Str("dedup_key", dedupKey).
		Msg("PagerDuty incident resolved")

	return nil
}

// SendNotification sends a notification via PagerDuty (creates a low-priority incident)
func (p *PagerDutyIntegration) SendNotification(ctx context.Context, notification *integrations.Notification) error {
	incident := &integrations.Incident{
		Title:       notification.Title,
		Description: notification.Message,
		Priority:    notification.Priority,
		Severity:    integrations.SeverityLow,
		Tags:        notification.Tags,
		Source:      "db-backup-notification",
		Timestamp:   notification.Timestamp,
		CustomFields: notification.CustomData,
	}

	_, err := p.CreateIncident(ctx, incident)
	return err
}

// Helper methods

func (p *PagerDutyIntegration) mapSeverity(severity integrations.Severity) string {
	switch severity {
	case integrations.SeverityCritical:
		return "critical"
	case integrations.SeverityHigh:
		return "error"
	case integrations.SeverityMedium:
		return "warning"
	case integrations.SeverityLow:
		return "info"
	default:
		return "info"
	}
}
