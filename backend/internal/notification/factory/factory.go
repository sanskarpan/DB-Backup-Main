// Package factory builds a notification Router from application configuration.
package factory

import (
	"context"
	"errors"

	"github.com/sanskarpan/db-backup/internal/config"
	"github.com/sanskarpan/db-backup/internal/notification"
	"github.com/sanskarpan/db-backup/internal/notification/email"
	"github.com/sanskarpan/db-backup/internal/notification/slack"
	"github.com/sanskarpan/db-backup/internal/notification/webhook"
)

// NewRouterFromConfig builds a *notification.Router and registers the Slack,
// Email and Webhook notifiers that are enabled in cfg. Each channel's NotifyOn
// list (when present) restricts which backup events ("success"/"failure")
// reach that channel. The returned router is always non-nil; when no channels
// are enabled it is an empty, no-op router. A non-nil error reports channels
// that were enabled but could not be constructed - the router still contains
// every channel that built successfully.
func NewRouterFromConfig(cfg config.NotificationConfig) (*notification.Router, error) {
	router := notification.NewRouter()
	var errs []error

	if cfg.Slack.Enabled {
		notifier, err := slack.NewSlackNotifier(&slack.Config{
			WebhookURL: cfg.Slack.WebhookURL,
			Channel:    cfg.Slack.Channel,
		})
		if err != nil {
			errs = append(errs, err)
		} else {
			router.AddNotifier(newEventFilter(notifier, cfg.Slack.NotifyOn))
		}
	}

	if cfg.Email.Enabled {
		notifier, err := email.NewEmailNotifier(&email.Config{
			SMTPHost: cfg.Email.SMTPHost,
			SMTPPort: cfg.Email.SMTPPort,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
			To:       cfg.Email.To,
		})
		if err != nil {
			errs = append(errs, err)
		} else {
			router.AddNotifier(newEventFilter(notifier, nil))
		}
	}

	if cfg.Webhook.Enabled {
		notifier, err := webhook.NewWebhookNotifier(&webhook.Config{
			URL:     cfg.Webhook.URL,
			Method:  cfg.Webhook.Method,
			Headers: cfg.Webhook.Headers,
		})
		if err != nil {
			errs = append(errs, err)
		} else {
			router.AddNotifier(newEventFilter(notifier, nil))
		}
	}

	return router, errors.Join(errs...)
}

// eventFilter wraps a notifier so that only notifications tagged with one of the
// configured events are forwarded. It lets each channel honor its own NotifyOn
// list even though the Router applies filters globally.
type eventFilter struct {
	inner    notification.Notifier
	notifyOn map[string]struct{}
}

// newEventFilter returns inner unchanged when notifyOn is empty (notify on every
// event); otherwise it wraps inner so only matching events are delivered.
func newEventFilter(inner notification.Notifier, notifyOn []string) notification.Notifier {
	if len(notifyOn) == 0 {
		return inner
	}
	set := make(map[string]struct{}, len(notifyOn))
	for _, event := range notifyOn {
		set[event] = struct{}{}
	}
	return &eventFilter{inner: inner, notifyOn: set}
}

// Send forwards the notification only when one of its tags matches the filter.
func (f *eventFilter) Send(ctx context.Context, notif *notification.Notification) error {
	for _, tag := range notif.Tags {
		if _, ok := f.notifyOn[tag]; ok {
			return f.inner.Send(ctx, notif)
		}
	}
	return nil
}

// GetType returns the wrapped notifier's provider type.
func (f *eventFilter) GetType() notification.ProviderType {
	return f.inner.GetType()
}

// ValidateConfig validates the wrapped notifier's configuration.
func (f *eventFilter) ValidateConfig() error {
	return f.inner.ValidateConfig()
}
