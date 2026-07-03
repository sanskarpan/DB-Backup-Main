package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
)

func TestEventTypeFor(t *testing.T) {
	cases := []struct {
		name  string
		notif *notification.Notification
		want  EventType
	}{
		{
			name:  "backup success",
			notif: &notification.Notification{Level: notification.LevelSuccess, Tags: []string{"backup", "success"}},
			want:  EventBackupCompleted,
		},
		{
			name:  "backup failure",
			notif: &notification.Notification{Level: notification.LevelError, Tags: []string{"backup", "failure"}},
			want:  EventBackupFailed,
		},
		{
			name:  "restore success",
			notif: &notification.Notification{Level: notification.LevelSuccess, Tags: []string{"restore", "success"}},
			want:  EventRestoreCompleted,
		},
		{
			name:  "restore failure",
			notif: &notification.Notification{Level: notification.LevelError, Tags: []string{"restore", "failure"}},
			want:  EventRestoreFailed,
		},
		{
			name:  "failure derived from level when no outcome tag",
			notif: &notification.Notification{Level: notification.LevelError, Tags: []string{"backup"}},
			want:  EventBackupFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := eventTypeFor(tc.notif); got != tc.want {
				t.Fatalf("eventTypeFor = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildEventDataIncludesMetadata(t *testing.T) {
	n := &notification.Notification{
		Title:    "Backup failed",
		Message:  "disk full",
		Level:    notification.LevelError,
		Metadata: map[string]interface{}{"database": "orders"},
	}
	data := buildEventData(n)
	if data["title"] != "Backup failed" || data["message"] != "disk full" {
		t.Fatalf("unexpected title/message: %#v", data)
	}
	if data["level"] != string(notification.LevelError) {
		t.Fatalf("unexpected level: %v", data["level"])
	}
	if data["database"] != "orders" {
		t.Fatalf("metadata not merged: %#v", data)
	}
}

// TestNotifierAdapter_PublishInvoked wires a real Manager to an httptest
// receiver and asserts that Send publishes an event that reaches a matching
// subscriber with the derived event type.
func TestNotifierAdapter_PublishInvoked(t *testing.T) {
	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var evt Event
		if err := json.Unmarshal(body, &evt); err != nil {
			t.Errorf("unmarshal delivered event: %v", err)
		}
		received <- evt
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mgr := NewManager(&ManagerConfig{Workers: 1, QueueSize: 4})
	defer mgr.Stop()

	if err := mgr.Subscribe(&Subscription{
		Name:   "test",
		URL:    srv.URL,
		Events: []EventType{EventBackupFailed},
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	adapter := NewNotifierAdapter(mgr)
	if adapter.GetType() != ProviderTypeWebhooks {
		t.Fatalf("GetType = %q", adapter.GetType())
	}
	if err := adapter.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig: %v", err)
	}

	err := adapter.Send(context.Background(), &notification.Notification{
		Title:   "Backup failed",
		Message: "disk full",
		Level:   notification.LevelError,
		Tags:    []string{"backup", "failure"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case evt := <-received:
		if evt.Type != EventBackupFailed {
			t.Fatalf("delivered event type = %q, want %q", evt.Type, EventBackupFailed)
		}
		if evt.Data["message"] != "disk full" {
			t.Fatalf("delivered event data = %#v", evt.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}
}

// TestNotifierAdapter_NilSafe verifies the adapter tolerates a nil notification.
func TestNotifierAdapter_NilSafe(t *testing.T) {
	adapter := NewNotifierAdapter(nil)
	if err := adapter.Send(context.Background(), nil); err != nil {
		t.Fatalf("Send(nil) = %v", err)
	}
}
