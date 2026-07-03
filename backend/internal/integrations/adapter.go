package integrations

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sanskarpan/db-backup/internal/notification"
)

// ProviderTypeIncident identifies the incident-integration notifier so it can be
// registered alongside the config-driven notification channels.
const ProviderTypeIncident notification.ProviderType = "incident"

// incidentSource labels incidents opened by the adapter.
const incidentSource = "db-backup"

// NotifierAdapter adapts the IncidentDispatcher to the notification.Notifier
// interface. Registering it with the notification router causes every
// ERROR-level notification (a backup failure) to open an incident on each
// enabled integration. Non-error notifications are ignored.
type NotifierAdapter struct {
	dispatcher *IncidentDispatcher
}

// NewNotifierAdapter creates a Notifier that opens incidents on backup failures.
func NewNotifierAdapter(dispatcher *IncidentDispatcher) *NotifierAdapter {
	return &NotifierAdapter{dispatcher: dispatcher}
}

// Send opens an incident on every enabled integration when the notification is
// an ERROR (a backup failure). Any dispatch error is logged and swallowed so
// the backup is never failed by an incident-creation problem.
func (a *NotifierAdapter) Send(ctx context.Context, n *notification.Notification) error {
	if n == nil || a.dispatcher == nil || !a.dispatcher.Enabled() {
		return nil
	}
	if n.Level != notification.LevelError {
		return nil
	}

	incident := incidentFromNotification(n)
	if err := a.dispatcher.Dispatch(ctx, incident); err != nil {
		log.Warn().
			Err(err).
			Str("title", n.Title).
			Msg("Failed to open incident(s) for backup failure")
	}

	return nil
}

// GetType returns the provider type.
func (a *NotifierAdapter) GetType() notification.ProviderType {
	return ProviderTypeIncident
}

// ValidateConfig validates the provider configuration. The adapter only needs a
// dispatcher, so there is nothing to validate.
func (a *NotifierAdapter) ValidateConfig() error {
	return nil
}

// incidentFromNotification builds an incident describing a backup failure.
func incidentFromNotification(n *notification.Notification) *Incident {
	timestamp := n.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	return &Incident{
		Title:        n.Title,
		Description:  n.Message,
		Priority:     PriorityHigh,
		Severity:     SeverityHigh,
		Source:       incidentSource,
		Tags:         n.Tags,
		Timestamp:    timestamp,
		CustomFields: n.Metadata,
	}
}
