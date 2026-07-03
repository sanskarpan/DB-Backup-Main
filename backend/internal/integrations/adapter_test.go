package integrations

import (
	"context"
	"testing"

	"github.com/sanskarpan/db-backup/internal/notification"
)

func TestIntegrationNotifierAdapter_FiresOnlyOnError(t *testing.T) {
	target := newCounting("jira", nil)
	adapter := NewNotifierAdapter(NewIncidentDispatcher(target))

	if adapter.GetType() != ProviderTypeIncident {
		t.Fatalf("GetType = %q", adapter.GetType())
	}
	if err := adapter.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}

	// Non-error levels must not open an incident.
	for _, level := range []notification.NotificationLevel{
		notification.LevelInfo, notification.LevelSuccess, notification.LevelWarning,
	} {
		if err := adapter.Send(context.Background(), &notification.Notification{Level: level}); err != nil {
			t.Fatalf("Send(%s): %v", level, err)
		}
	}
	if target.calls != 0 {
		t.Fatalf("non-error notifications should not create incidents, got %d calls", target.calls)
	}

	// An error notification opens exactly one incident on the integration.
	err := adapter.Send(context.Background(), &notification.Notification{
		Title:   "Backup failed",
		Message: "disk full",
		Level:   notification.LevelError,
		Tags:    []string{"backup", "failure"},
	})
	if err != nil {
		t.Fatalf("Send(error): %v", err)
	}
	if target.calls != 1 {
		t.Fatalf("error notification should create one incident, got %d", target.calls)
	}
}

func TestIntegrationNotifierAdapter_SwallowsDispatchError(t *testing.T) {
	target := newCounting("jira", context.DeadlineExceeded)
	adapter := NewNotifierAdapter(NewIncidentDispatcher(target))

	// Dispatch fails, but Send must never surface an error to the notification
	// router (a backup is never failed by an incident-creation problem).
	err := adapter.Send(context.Background(), &notification.Notification{
		Title: "Backup failed",
		Level: notification.LevelError,
	})
	if err != nil {
		t.Fatalf("Send must swallow dispatch errors, got %v", err)
	}
	if target.calls != 1 {
		t.Fatalf("expected dispatch attempt, got %d calls", target.calls)
	}
}

func TestIntegrationNotifierAdapter_NoDispatcherIsNoOp(t *testing.T) {
	adapter := NewNotifierAdapter(NewIncidentDispatcher())
	if err := adapter.Send(context.Background(), &notification.Notification{Level: notification.LevelError}); err != nil {
		t.Fatalf("Send with empty dispatcher: %v", err)
	}
}

func TestIncidentFromNotification(t *testing.T) {
	n := &notification.Notification{
		Title:    "Backup failed",
		Message:  "disk full",
		Level:    notification.LevelError,
		Tags:     []string{"backup", "failure"},
		Metadata: map[string]interface{}{"database": "orders"},
	}
	inc := incidentFromNotification(n)
	if inc.Title != n.Title || inc.Description != n.Message {
		t.Fatalf("unexpected incident title/description: %#v", inc)
	}
	if inc.Priority != PriorityHigh || inc.Severity != SeverityHigh {
		t.Fatalf("unexpected priority/severity: %v/%v", inc.Priority, inc.Severity)
	}
	if inc.Timestamp.IsZero() {
		t.Fatal("timestamp should be defaulted when notification has none")
	}
	if inc.CustomFields["database"] != "orders" {
		t.Fatalf("metadata not carried into incident: %#v", inc.CustomFields)
	}
}
