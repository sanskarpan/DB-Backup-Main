package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// Alert represents a monitoring alert
type Alert struct {
	ID          string            `json:"id"`
	Severity    AlertSeverity     `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	Timestamp   time.Time         `json:"timestamp"`
	Labels      map[string]string `json:"labels"`
	Resolved    bool              `json:"resolved"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
}

// AlertManager manages monitoring alerts
type AlertManager struct {
	mu           sync.RWMutex
	config       MonitorConfig
	alerts       map[string]*Alert
	alertHistory []*Alert
	maxHistory   int
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config MonitorConfig) *AlertManager {
	return &AlertManager{
		config:       config,
		alerts:       make(map[string]*Alert),
		alertHistory: make([]*Alert, 0),
		maxHistory:   1000,
	}
}

// TriggerAlert triggers a new alert
func (am *AlertManager) TriggerAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("%s-%d", alert.Source, time.Now().UnixNano())
	}

	// Store active alert
	am.alerts[alert.ID] = alert

	// Add to history
	am.alertHistory = append(am.alertHistory, alert)
	if len(am.alertHistory) > am.maxHistory {
		am.alertHistory = am.alertHistory[1:]
	}

	// Update Prometheus metric
	activeAlertsGauge.Set(float64(len(am.alerts)))

	// Send notifications
	go am.sendNotifications(alert)
}

// ResolveAlert resolves an active alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	delete(am.alerts, alertID)
	activeAlertsGauge.Set(float64(len(am.alerts)))

	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetActiveAlertCount returns the count of active alerts
func (am *AlertManager) GetActiveAlertCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.alerts)
}

// GetAlertHistory returns the alert history
func (am *AlertManager) GetAlertHistory(limit int) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	start := len(am.alertHistory) - limit
	return am.alertHistory[start:]
}

// sendNotifications sends alert notifications
func (am *AlertManager) sendNotifications(alert *Alert) {
	// Send to Slack if configured
	if am.config.SlackWebhookURL != "" {
		_ = am.sendSlackNotification(alert)
	}

	// Send to Alertmanager if configured
	if am.config.AlertManagerURL != "" {
		_ = am.sendAlertmanagerNotification(alert)
	}

	// Send email if configured
	if len(am.config.EmailRecipients) > 0 {
		// Email sending would be implemented here
	}
}

// sendSlackNotification sends an alert to Slack
func (am *AlertManager) sendSlackNotification(alert *Alert) error {
	color := "good"
	switch alert.Severity {
	case SeverityCritical:
		color = "danger"
	case SeverityHigh:
		color = "warning"
	case SeverityWarning:
		color = "warning"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"fallback": fmt.Sprintf("[%s] %s", alert.Severity, alert.Title),
				"color":    color,
				"title":    alert.Title,
				"text":     alert.Description,
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
					{
						"title": "Source",
						"value": alert.Source,
						"short": true,
					},
					{
						"title": "Timestamp",
						"value": alert.Timestamp.Format(time.RFC3339),
						"short": false,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(am.config.SlackWebhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// sendAlertmanagerNotification sends an alert to Alertmanager
func (am *AlertManager) sendAlertmanagerNotification(alert *Alert) error {
	// Alertmanager v2 API format
	payload := []map[string]interface{}{
		{
			"labels": map[string]string{
				"alertname": alert.Title,
				"severity":  string(alert.Severity),
				"source":    alert.Source,
			},
			"annotations": map[string]string{
				"summary":     alert.Title,
				"description": alert.Description,
			},
			"startsAt": alert.Timestamp.Format(time.RFC3339),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2/alerts", am.config.AlertManagerURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("alertmanager returned status %d", resp.StatusCode)
	}

	return nil
}

// ClearResolvedAlerts clears old resolved alerts from history
func (am *AlertManager) ClearResolvedAlerts(olderThan time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	newHistory := make([]*Alert, 0)

	for _, alert := range am.alertHistory {
		if !alert.Resolved || (alert.ResolvedAt != nil && alert.ResolvedAt.After(cutoff)) {
			newHistory = append(newHistory, alert)
		}
	}

	cleared := len(am.alertHistory) - len(newHistory)
	am.alertHistory = newHistory

	return cleared
}
