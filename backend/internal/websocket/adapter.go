package websocket

import (
	"context"

	"github.com/sanskarpan/db-backup/internal/notification"
)

// ProviderTypeWebSocket identifies the WebSocket-backed notifier so it can be
// registered alongside the config-driven channels (slack/email/webhook).
const ProviderTypeWebSocket notification.ProviderType = "websocket"

// NotifierAdapter adapts the WebSocket hub to the notification.Notifier
// interface. Registering it with the notification router causes every
// notification (backup success/failure, etc.) to also be broadcast to
// connected WebSocket clients in real time.
type NotifierAdapter struct {
	broadcaster *NotificationBroadcaster
}

// NewNotifierAdapter creates a Notifier that forwards notifications to the hub.
func NewNotifierAdapter(hub *Hub) *NotifierAdapter {
	return &NotifierAdapter{
		broadcaster: NewNotificationBroadcaster(hub),
	}
}

// Send forwards a notification to all connected WebSocket clients.
func (a *NotifierAdapter) Send(_ context.Context, n *notification.Notification) error {
	if n == nil {
		return nil
	}

	a.broadcaster.Broadcast(&Notification{
		Type:     mapLevel(n.Level),
		Title:    n.Title,
		Message:  n.Message,
		Priority: priorityForLevel(n.Level),
	})

	return nil
}

// GetType returns the provider type.
func (a *NotifierAdapter) GetType() notification.ProviderType {
	return ProviderTypeWebSocket
}

// ValidateConfig validates the provider configuration. The adapter only depends
// on a live hub, so there is nothing to validate.
func (a *NotifierAdapter) ValidateConfig() error {
	return nil
}

// Notification payload type strings understood by the WebSocket client.
const (
	notifTypeWarning = "warning"
	notifTypeError   = "error"
	notifTypeSuccess = "success"
	notifTypeInfo    = "info"
)

// mapLevel maps a notification severity level to the string type understood by
// the WebSocket Notification payload.
func mapLevel(level notification.NotificationLevel) string {
	switch level {
	case notification.LevelWarning:
		return notifTypeWarning
	case notification.LevelError:
		return notifTypeError
	case notification.LevelSuccess:
		return notifTypeSuccess
	case notification.LevelInfo:
		return notifTypeInfo
	default:
		return notifTypeInfo
	}
}

// priorityForLevel derives a UI priority (0-10) from the severity level.
func priorityForLevel(level notification.NotificationLevel) int {
	switch level {
	case notification.LevelError:
		return 10
	case notification.LevelWarning:
		return 7
	case notification.LevelSuccess:
		return 6
	case notification.LevelInfo:
		return 5
	default:
		return 5
	}
}
