package webhooks

import (
	"context"

	"github.com/sanskarpan/db-backup/internal/notification"
)

// ProviderTypeWebhooks identifies the webhook-subscription notifier so it can be
// registered alongside the config-driven channels (slack/email/webhook).
const ProviderTypeWebhooks notification.ProviderType = "webhooks"

// notificationSource labels webhook events published by the adapter.
const notificationSource = "db-backup"

// Notification tag values understood by the adapter when deriving an event type.
const (
	tagBackup  = "backup"
	tagRestore = "restore"
	tagFailure = "failure"
)

// NotifierAdapter adapts the webhook Manager to the notification.Notifier
// interface. Registering it with the notification router fans every backup and
// restore notification out to all matching webhook subscribers.
type NotifierAdapter struct {
	manager *Manager
}

// NewNotifierAdapter creates a Notifier that publishes notifications as webhook
// events on the given Manager.
func NewNotifierAdapter(manager *Manager) *NotifierAdapter {
	return &NotifierAdapter{manager: manager}
}

// Send builds a webhook Event from the notification and publishes it to every
// matching subscriber. A publish failure is returned to the caller but never
// aborts the originating backup: the router only logs notifier errors.
func (a *NotifierAdapter) Send(_ context.Context, n *notification.Notification) error {
	if n == nil || a.manager == nil {
		return nil
	}

	event := &Event{
		Type:      eventTypeFor(n),
		Timestamp: n.Timestamp,
		Source:    notificationSource,
		Data:      buildEventData(n),
		Metadata:  n.Metadata,
	}

	return a.manager.Publish(event)
}

// GetType returns the provider type.
func (a *NotifierAdapter) GetType() notification.ProviderType {
	return ProviderTypeWebhooks
}

// ValidateConfig validates the provider configuration. The adapter only needs a
// live Manager, so there is nothing to validate.
func (a *NotifierAdapter) ValidateConfig() error {
	return nil
}

// eventTypeFor derives a webhook EventType from a notification's tags and level.
// The backup engine tags outcomes as {"backup"|"restore", "success"|"failure"};
// the level (error) is used as a fallback when the outcome tag is absent.
func eventTypeFor(n *notification.Notification) EventType {
	kind, failed := classifyNotification(n)
	switch {
	case kind == tagRestore && failed:
		return EventRestoreFailed
	case kind == tagRestore:
		return EventRestoreCompleted
	case failed:
		return EventBackupFailed
	default:
		return EventBackupCompleted
	}
}

// classifyNotification reports the subject (backup/restore) and whether the
// notification represents a failure.
func classifyNotification(n *notification.Notification) (kind string, failed bool) {
	kind = tagBackup
	failed = n.Level == notification.LevelError
	for _, tag := range n.Tags {
		switch tag {
		case tagRestore:
			kind = tagRestore
		case tagBackup:
			kind = tagBackup
		case tagFailure:
			failed = true
		}
	}
	return kind, failed
}

// buildEventData renders the notification into the webhook event payload.
func buildEventData(n *notification.Notification) map[string]interface{} {
	data := map[string]interface{}{
		"title":   n.Title,
		"message": n.Message,
		"level":   string(n.Level),
	}
	for k, v := range n.Metadata {
		data[k] = v
	}
	return data
}
