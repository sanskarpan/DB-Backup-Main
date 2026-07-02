package factory

import (
	"context"
	"testing"

	"github.com/sanskarpan/db-backup/internal/config"
	"github.com/sanskarpan/db-backup/internal/notification"
)

func TestNewRouterFromConfigEmpty(t *testing.T) {
	router, err := NewRouterFromConfig(&config.NotificationConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if router == nil {
		t.Fatal("expected non-nil router when no channels are enabled")
	}
	// A no-op router must accept a send without error.
	if err := router.Send(context.Background(), &notification.Notification{Title: "x"}); err != nil {
		t.Errorf("empty router Send returned error: %v", err)
	}
}

func TestNewRouterFromConfigEnabledButInvalid(t *testing.T) {
	// Slack enabled without a webhook URL must surface a construction error.
	cfg := config.NotificationConfig{
		Slack: config.SlackConfig{Enabled: true},
	}
	router, err := NewRouterFromConfig(&cfg)
	if err == nil {
		t.Fatal("expected error for enabled Slack channel without a webhook URL")
	}
	if router == nil {
		t.Fatal("expected router to be non-nil even on partial init failure")
	}
}

func TestEventFilterHonorsNotifyOn(t *testing.T) {
	rec := &recorder{}
	filtered := newEventFilter(rec, []string{"failure"})

	// A success event must be dropped when notifyOn is ["failure"].
	if err := filtered.Send(context.Background(), &notification.Notification{Tags: []string{"backup", "success"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.count != 0 {
		t.Errorf("expected success event to be filtered out, got %d sends", rec.count)
	}

	// A failure event must pass through.
	if err := filtered.Send(context.Background(), &notification.Notification{Tags: []string{"backup", "failure"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.count != 1 {
		t.Errorf("expected failure event to pass, got %d sends", rec.count)
	}
}

// recorder is a minimal notification.Notifier used to assert filtering.
type recorder struct {
	count int
}

func (r *recorder) Send(_ context.Context, _ *notification.Notification) error {
	r.count++
	return nil
}

func (r *recorder) GetType() notification.ProviderType { return notification.ProviderTypeWebhook }

func (r *recorder) ValidateConfig() error { return nil }
