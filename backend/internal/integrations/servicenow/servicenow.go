package servicenow

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

// ServiceNowIntegration implements integration with ServiceNow for incident management.
//
//nolint:revive // keeps public name stable across packages
type ServiceNowIntegration struct {
	*integrations.BaseIntegration
	client          *http.Client
	assignmentGroup string
	category        string
	subcategory     string
}

// ServiceNow incident states.
const (
	StateNew        = "1"
	StateInProgress = "2"
	StateOnHold     = "3"
	StateResolved   = "6"
	StateClosed     = "7"
	StateCanceled   = "8"
)

// ServiceNow incident response.
type incidentResponse struct {
	Result struct {
		SysID         string `json:"sys_id"`
		Number        string `json:"number"`
		State         string `json:"state"`
		ShortDesc     string `json:"short_description"`
		Description   string `json:"description"`
		Priority      string `json:"priority"`
		Urgency       string `json:"urgency"`
		Impact        string `json:"impact"`
		AssignedTo    string `json:"assigned_to"`
		AssignmentGrp string `json:"assignment_group"`
		OpenedAt      string `json:"opened_at"`
		ResolvedAt    string `json:"resolved_at"`
		ClosedAt      string `json:"closed_at"`
		CloseNotes    string `json:"close_notes"`
	} `json:"result"`
}

// NewServiceNowIntegration creates a new ServiceNow integration.
func NewServiceNowIntegration(name string) *ServiceNowIntegration {
	return &ServiceNowIntegration{
		BaseIntegration: integrations.NewBaseIntegration(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetType returns the integration type.
func (s *ServiceNowIntegration) GetType() integrations.IntegrationType {
	return integrations.IntegrationTypeServiceNow
}

// GetName returns the integration name.
func (s *ServiceNowIntegration) GetName() string {
	config := s.GetConfig()
	if config != nil && config.Name != "" {
		return config.Name
	}
	return "servicenow"
}

// Configure configures the ServiceNow integration.
func (s *ServiceNowIntegration) Configure(config *integrations.Config) error {
	if err := s.BaseIntegration.Configure(config); err != nil {
		return err
	}

	// Extract ServiceNow-specific settings
	if settings := config.Settings; settings != nil {
		if ag, ok := settings["assignment_group"].(string); ok {
			s.assignmentGroup = ag
		}
		if cat, ok := settings["category"].(string); ok {
			s.category = cat
		}
		if subcat, ok := settings["subcategory"].(string); ok {
			s.subcategory = subcat
		}
	}

	// Configure timeout if specified
	if config.Timeout > 0 {
		s.client.Timeout = config.Timeout
	}

	s.SetStatus(integrations.StatusActive)
	return nil
}

// Validate validates the ServiceNow configuration.
func (s *ServiceNowIntegration) Validate() error {
	config := s.GetConfig()
	if config == nil {
		return fmt.Errorf("no configuration provided")
	}

	if config.BaseURL == "" {
		return fmt.Errorf("ServiceNow instance URL is required")
	}

	// ServiceNow requires either username/password or OAuth token
	hasBasicAuth := config.Username != "" && config.Password != ""
	hasOAuth := config.OAuth != nil && config.OAuth.AccessToken != ""

	if !hasBasicAuth && !hasOAuth {
		return fmt.Errorf("authentication required: provide username/password or OAuth token")
	}

	return nil
}

// HealthCheck performs a health check on the ServiceNow integration.
func (s *ServiceNowIntegration) HealthCheck(ctx context.Context) error {
	if err := s.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	config := s.GetConfig()

	// Test connection by getting system properties
	url := fmt.Sprintf("%s/api/now/table/sys_properties?sysparm_limit=1", config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	s.GetMetrics().RecordRequest(time.Since(start))

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("health check failed with status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateIncident creates a new incident in ServiceNow.
func (s *ServiceNowIntegration) CreateIncident(ctx context.Context, incident *integrations.Incident) (*integrations.IncidentResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	config := s.GetConfig()
	url := fmt.Sprintf("%s/api/now/table/incident", config.BaseURL)

	priority, urgency, impact := s.mapPriority(incident.Priority)

	payload := map[string]interface{}{
		"short_description": incident.Title,
		"description":       incident.Description,
		"priority":          priority,
		"urgency":           urgency,
		"impact":            impact,
		"state":             StateNew,
	}

	// Add optional fields
	if s.assignmentGroup != "" {
		payload["assignment_group"] = s.assignmentGroup
	}
	if s.category != "" {
		payload["category"] = s.category
	}
	if s.subcategory != "" {
		payload["subcategory"] = s.subcategory
	}
	if incident.Source != "" {
		payload["u_source"] = incident.Source
	}

	// Add custom fields as additional comments
	if len(incident.CustomFields) > 0 {
		customFieldsJSON, err := json.MarshalIndent(incident.CustomFields, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal custom fields: %w", err)
		}
		payload["work_notes"] = fmt.Sprintf("Additional Context:\n%s", string(customFieldsJSON))
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		s.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return nil, fmt.Errorf("failed to create incident, status %d: %s", resp.StatusCode, string(body))
	}

	s.GetMetrics().RecordRequest(time.Since(start))

	var result incidentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &integrations.IncidentResponse{
		ID:        result.Result.SysID,
		Key:       result.Result.Number,
		Status:    s.mapStateToStatus(result.Result.State),
		URL:       fmt.Sprintf("%s/nav_to.do?uri=incident.do?sys_id=%s", config.BaseURL, result.Result.SysID),
		CreatedAt: time.Now(),
	}, nil
}

// UpdateIncident updates an existing incident in ServiceNow.
func (s *ServiceNowIntegration) UpdateIncident(ctx context.Context, incidentID string, update *integrations.IncidentUpdate) error {
	if err := s.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	config := s.GetConfig()
	url := fmt.Sprintf("%s/api/now/table/incident/%s", config.BaseURL, incidentID)

	payload := make(map[string]interface{})

	if update.Status != "" {
		state := s.mapStatusToState(update.Status)
		payload["state"] = state
	}

	if update.Comment != "" {
		payload["work_notes"] = update.Comment
	}

	if update.Assignee != "" {
		payload["assigned_to"] = update.Assignee
	}

	if len(payload) == 0 {
		return fmt.Errorf("no fields to update")
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to update incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.GetMetrics().RecordError(err)
			return fmt.Errorf("failed to update incident, status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		s.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return fmt.Errorf("failed to update incident, status %d: %s", resp.StatusCode, string(body))
	}

	s.GetMetrics().RecordRequest(time.Since(start))
	return nil
}

// GetIncident retrieves an incident from ServiceNow.
func (s *ServiceNowIntegration) GetIncident(ctx context.Context, incidentID string) (*integrations.IncidentResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	config := s.GetConfig()
	url := fmt.Sprintf("%s/api/now/table/incident/%s", config.BaseURL, incidentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return nil, fmt.Errorf("failed to get incident, status %d: %s", resp.StatusCode, string(body))
	}

	s.GetMetrics().RecordRequest(time.Since(start))

	var result incidentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &integrations.IncidentResponse{
		ID:     result.Result.SysID,
		Key:    result.Result.Number,
		Status: s.mapStateToStatus(result.Result.State),
		URL:    fmt.Sprintf("%s/nav_to.do?uri=incident.do?sys_id=%s", config.BaseURL, result.Result.SysID),
	}, nil
}

// CloseIncident closes an incident in ServiceNow.
func (s *ServiceNowIntegration) CloseIncident(ctx context.Context, incidentID, resolution string) error {
	if err := s.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	config := s.GetConfig()
	url := fmt.Sprintf("%s/api/now/table/incident/%s", config.BaseURL, incidentID)

	payload := map[string]interface{}{
		"state":       StateClosed,
		"close_notes": resolution,
		"close_code":  "Solved (Permanently)",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	s.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.GetMetrics().RecordError(err)
		return fmt.Errorf("failed to close incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.GetMetrics().RecordError(err)
			return fmt.Errorf("failed to close incident, status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		s.GetMetrics().RecordError(fmt.Errorf("status code: %d", resp.StatusCode))
		return fmt.Errorf("failed to close incident, status %d: %s", resp.StatusCode, string(body))
	}

	s.GetMetrics().RecordRequest(time.Since(start))
	return nil
}

// SendNotification sends a notification via ServiceNow (creates an event).
func (s *ServiceNowIntegration) SendNotification(ctx context.Context, notification *integrations.Notification) error {
	// ServiceNow doesn't have a direct notification API
	// We create a low-priority incident for notifications
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

	_, err := s.CreateIncident(ctx, incident)
	return err
}

// Helper methods

// setAuthHeaders sets authentication headers on the request.
func (s *ServiceNowIntegration) setAuthHeaders(req *http.Request) {
	config := s.GetConfig()

	// Try OAuth first
	if config.OAuth != nil && config.OAuth.AccessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.OAuth.AccessToken))
		return
	}

	// Fall back to basic auth
	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	}
}

// mapPriority maps our priority to ServiceNow priority, urgency, and impact
// ServiceNow uses a matrix: Priority = function(Impact, Urgency).
func (s *ServiceNowIntegration) mapPriority(priority integrations.Priority) (priorityVal, urgency, impact string) {
	switch priority {
	case integrations.PriorityCritical:
		return "1", "1", "1" // Critical - High urgency, High impact
	case integrations.PriorityHigh:
		return "2", "2", "2" // High - Medium urgency, Medium impact
	case integrations.PriorityMedium:
		return "3", "2", "3" // Medium - Medium urgency, Low impact
	case integrations.PriorityLow:
		return "4", "3", "3" // Low - Low urgency, Low impact
	default:
		return "3", "2", "3" // Default to Medium
	}
}

// mapStateToStatus maps ServiceNow state to our status.
func (s *ServiceNowIntegration) mapStateToStatus(state string) string {
	switch state {
	case StateNew:
		return "new"
	case StateInProgress:
		return "in_progress"
	case StateOnHold:
		return "on_hold"
	case StateResolved:
		return "resolved"
	case StateClosed:
		return "closed"
	case StateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// mapStatusToState maps our status to ServiceNow state.
func (s *ServiceNowIntegration) mapStatusToState(status string) string {
	switch status {
	case "new":
		return StateNew
	case "in_progress":
		return StateInProgress
	case "on_hold":
		return StateOnHold
	case "resolved":
		return StateResolved
	case "closed":
		return StateClosed
	case "canceled":
		return StateCanceled
	default:
		return StateInProgress
	}
}
