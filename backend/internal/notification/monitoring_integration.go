package notification

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// priorityHigh is the high-priority / high-severity level value.
const priorityHigh = "high"

// MonitoringIntegration integrates push notifications with the monitoring system.
type MonitoringIntegration struct {
	pushService *PushService
	mu          sync.RWMutex
	subscribers map[string][]string // metric -> user IDs
	thresholds  map[string]float64  // metric -> threshold value
}

// NewMonitoringIntegration creates a new monitoring integration.
func NewMonitoringIntegration(pushService *PushService) *MonitoringIntegration {
	return &MonitoringIntegration{
		pushService: pushService,
		subscribers: make(map[string][]string),
		thresholds:  make(map[string]float64),
	}
}

// SubscribeToMetric subscribes a user to notifications for a specific metric.
func (m *MonitoringIntegration) SubscribeToMetric(userID, metric string, threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscribers[metric]; !exists {
		m.subscribers[metric] = []string{}
	}

	// Add user if not already subscribed
	for _, id := range m.subscribers[metric] {
		if id == userID {
			return
		}
	}

	m.subscribers[metric] = append(m.subscribers[metric], userID)
	m.thresholds[metric] = threshold
}

// UnsubscribeFromMetric unsubscribes a user from a metric.
func (m *MonitoringIntegration) UnsubscribeFromMetric(userID, metric string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if subscribers, exists := m.subscribers[metric]; exists {
		newSubscribers := []string{}
		for _, id := range subscribers {
			if id != userID {
				newSubscribers = append(newSubscribers, id)
			}
		}
		m.subscribers[metric] = newSubscribers
	}
}

// NotifyBackupCompleted sends notification when backup completes.
func (m *MonitoringIntegration) NotifyBackupCompleted(ctx context.Context, userID, backupID, databaseName string, success bool, duration time.Duration, size int64) error {
	var notification *PushNotification

	if success {
		notification = &PushNotification{
			Title:    "Backup Completed Successfully",
			Body:     fmt.Sprintf("Database '%s' backed up in %s (%.2f MB)", databaseName, duration.Round(time.Second), float64(size)/(1024*1024)),
			Icon:     "/icons/success-icon.png",
			Badge:    "/icons/success-badge.png",
			Tag:      fmt.Sprintf("backup-%s", backupID),
			Type:     PushNotificationBackupSuccess,
			Priority: "normal",
			Data: map[string]interface{}{
				"type":      "backup_success",
				"backup_id": backupID,
				"database":  databaseName,
				"duration":  duration.Seconds(),
				"size":      size,
				"timestamp": time.Now().Unix(),
			},
			Actions: []NotificationAction{
				{
					Action: "view",
					Title:  "View Backup",
					Icon:   "/icons/view.png",
				},
			},
			Vibrate: []int{200, 100, 200},
		}
	} else {
		notification = &PushNotification{
			Title:              "Backup Failed",
			Body:               fmt.Sprintf("Failed to backup database '%s'. Please check the logs.", databaseName),
			Icon:               "/icons/error-icon.png",
			Badge:              "/icons/error-badge.png",
			Tag:                fmt.Sprintf("backup-%s", backupID),
			Type:               PushNotificationBackupFailed,
			Priority:           priorityHigh,
			RequireInteraction: true,
			Data: map[string]interface{}{
				"type":      "backup_failed",
				"backup_id": backupID,
				"database":  databaseName,
				"timestamp": time.Now().Unix(),
			},
			Actions: []NotificationAction{
				{
					Action: "view",
					Title:  "View Details",
					Icon:   "/icons/view.png",
				},
				{
					Action: "retry",
					Title:  "Retry",
					Icon:   "/icons/retry.png",
				},
			},
			Vibrate: []int{300, 100, 300, 100, 300},
		}
	}

	return m.pushService.SendToUser(ctx, userID, notification)
}

// NotifyMetricThresholdExceeded sends notification when a metric exceeds threshold.
func (m *MonitoringIntegration) NotifyMetricThresholdExceeded(ctx context.Context, metric string, value, threshold float64) error {
	m.mu.RLock()
	userIDs := m.subscribers[metric]
	m.mu.RUnlock()

	notification := &PushNotification{
		Title:              "Monitoring Alert",
		Body:               fmt.Sprintf("Metric '%s' exceeded threshold: %.2f > %.2f", metric, value, threshold),
		Icon:               "/icons/alert-icon.png",
		Badge:              "/icons/alert-badge.png",
		Tag:                fmt.Sprintf("metric-%s", metric),
		Type:               PushNotificationMonitoringAlert,
		Priority:           priorityHigh,
		RequireInteraction: true,
		Data: map[string]interface{}{
			"type":      "monitoring_alert",
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
			"timestamp": time.Now().Unix(),
		},
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "Check Status",
				Icon:   "/icons/view.png",
			},
			{
				Action: "acknowledge",
				Title:  "Acknowledge",
				Icon:   "/icons/check.png",
			},
		},
		Vibrate: []int{300, 100, 300},
	}

	var lastErr error
	for _, userID := range userIDs {
		if err := m.pushService.SendToUser(ctx, userID, notification); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// NotifyComplianceViolation sends notification for compliance violations.
func (m *MonitoringIntegration) NotifyComplianceViolation(ctx context.Context, userID, violationType, description, severity string) error {
	priority := "normal"
	requireInteraction := false

	if severity == priorityHigh || severity == "critical" {
		priority = priorityHigh
		requireInteraction = true
	}

	notification := &PushNotification{
		Title:              "Compliance Violation Detected",
		Body:               fmt.Sprintf("%s: %s", violationType, description),
		Icon:               "/icons/warning-icon.png",
		Badge:              "/icons/warning-badge.png",
		Tag:                fmt.Sprintf("compliance-%s", violationType),
		Type:               PushNotificationComplianceAlert,
		Priority:           priority,
		RequireInteraction: requireInteraction,
		Data: map[string]interface{}{
			"type":           "compliance_alert",
			"violation_type": violationType,
			"description":    description,
			"severity":       severity,
			"timestamp":      time.Now().Unix(),
		},
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "View Report",
				Icon:   "/icons/view.png",
			},
		},
		Vibrate: []int{200, 100, 200, 100, 200},
	}

	return m.pushService.SendToUser(ctx, userID, notification)
}

// NotifySystemCritical sends critical system notifications.
func (m *MonitoringIntegration) NotifySystemCritical(ctx context.Context, title, message string, data map[string]interface{}) error {
	notification := &PushNotification{
		Title:              title,
		Body:               message,
		Icon:               "/icons/error-icon.png",
		Badge:              "/icons/error-badge.png",
		Tag:                "system-critical",
		Type:               PushNotificationSystemCritical,
		Priority:           priorityHigh,
		RequireInteraction: true,
		Data:               data,
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "View Details",
				Icon:   "/icons/view.png",
			},
		},
		Vibrate: []int{500, 200, 500, 200, 500},
	}

	return m.pushService.BroadcastNotification(ctx, notification)
}

// NotifyScheduledBackup sends notification for scheduled backup.
func (m *MonitoringIntegration) NotifyScheduledBackup(ctx context.Context, userID, scheduleName, databaseName string, scheduledTime time.Time) error {
	notification := &PushNotification{
		Title:    "Scheduled Backup Starting",
		Body:     fmt.Sprintf("Backup for '%s' is starting as scheduled at %s", databaseName, scheduledTime.Format("15:04")),
		Icon:     "/icons/icon-192x192.png",
		Badge:    "/icons/badge-72x72.png",
		Tag:      fmt.Sprintf("schedule-%s", scheduleName),
		Type:     PushNotificationBackupSuccess,
		Priority: "low",
		Silent:   true,
		Data: map[string]interface{}{
			"type":           "scheduled_backup",
			"schedule_name":  scheduleName,
			"database":       databaseName,
			"scheduled_time": scheduledTime.Unix(),
			"timestamp":      time.Now().Unix(),
		},
	}

	return m.pushService.SendToUser(ctx, userID, notification)
}

// NotifyStorageWarning sends notification when storage is running low.
func (m *MonitoringIntegration) NotifyStorageWarning(ctx context.Context, userID, provider string, usedPercent float64, available int64) error {
	priority := "normal"
	requireInteraction := false

	if usedPercent >= 90 {
		priority = priorityHigh
		requireInteraction = true
	}

	notification := &PushNotification{
		Title:              "Storage Warning",
		Body:               fmt.Sprintf("Storage '%s' is %.1f%% full. %.2f GB available.", provider, usedPercent, float64(available)/(1024*1024*1024)),
		Icon:               "/icons/warning-icon.png",
		Badge:              "/icons/warning-badge.png",
		Tag:                fmt.Sprintf("storage-%s", provider),
		Type:               PushNotificationMonitoringAlert,
		Priority:           priority,
		RequireInteraction: requireInteraction,
		Data: map[string]interface{}{
			"type":         "storage_warning",
			"provider":     provider,
			"used_percent": usedPercent,
			"available":    available,
			"timestamp":    time.Now().Unix(),
		},
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "View Storage",
				Icon:   "/icons/view.png",
			},
		},
		Vibrate: []int{200, 100, 200},
	}

	return m.pushService.SendToUser(ctx, userID, notification)
}

// NotifyDatabaseConnectionFailed sends notification when database connection fails.
func (m *MonitoringIntegration) NotifyDatabaseConnectionFailed(ctx context.Context, userID, databaseName, errorMsg string, retrying bool) error {
	body := fmt.Sprintf("Failed to connect to database '%s': %s", databaseName, errorMsg)
	if retrying {
		body += ". Retrying..."
	}

	notification := &PushNotification{
		Title:              "Database Connection Failed",
		Body:               body,
		Icon:               "/icons/error-icon.png",
		Badge:              "/icons/error-badge.png",
		Tag:                fmt.Sprintf("db-connection-%s", databaseName),
		Type:               PushNotificationMonitoringAlert,
		Priority:           priorityHigh,
		RequireInteraction: !retrying,
		Data: map[string]interface{}{
			"type":      "connection_failed",
			"database":  databaseName,
			"error":     errorMsg,
			"retrying":  retrying,
			"timestamp": time.Now().Unix(),
		},
		Actions: []NotificationAction{
			{
				Action: "view",
				Title:  "View Details",
				Icon:   "/icons/view.png",
			},
		},
		Vibrate: []int{300, 100, 300},
	}

	return m.pushService.SendToUser(ctx, userID, notification)
}

// GetSubscriberCount returns the number of subscribers for a metric.
func (m *MonitoringIntegration) GetSubscriberCount(metric string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.subscribers[metric])
}

// GetAllMetrics returns all metrics being monitored.
func (m *MonitoringIntegration) GetAllMetrics() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make([]string, 0, len(m.subscribers))
	for metric := range m.subscribers {
		metrics = append(metrics, metric)
	}

	return metrics
}
