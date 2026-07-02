package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
)

func TestNotifierAdapterImplementsNotifier(t *testing.T) {
	// Compile-time guarantee that the adapter satisfies the interface the
	// notification router expects.
	var _ notification.Notifier = NewNotifierAdapter(NewHub())
}

func TestNotifierAdapterForwardsToHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// A client subscribed to the "notifications" topic should receive anything
	// the adapter forwards. A nil connection is fine because we never start the
	// read/write pumps in this test.
	client := NewClient(hub, nil, "user-123")
	hub.register <- client
	hub.Subscribe(client, "notifications")

	adapter := NewNotifierAdapter(hub)
	if err := adapter.Send(context.Background(), &notification.Notification{
		Title:   "Backup complete",
		Message: "db-prod backup succeeded",
		Level:   notification.LevelSuccess,
	}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	select {
	case msg := <-client.send:
		if msg.Type != MessageTypeNotification {
			t.Fatalf("expected message type %q, got %q", MessageTypeNotification, msg.Type)
		}
		notif, ok := msg.Data.(*Notification)
		if !ok {
			t.Fatalf("expected data of type *Notification, got %T", msg.Data)
		}
		if notif.Title != "Backup complete" {
			t.Errorf("unexpected title: %q", notif.Title)
		}
		if notif.Type != "success" {
			t.Errorf("expected mapped type %q, got %q", "success", notif.Type)
		}
		if notif.Priority != 6 {
			t.Errorf("expected priority 6 for success level, got %d", notif.Priority)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the adapter to forward the notification")
	}
}

func TestNotifierAdapterNilNotification(t *testing.T) {
	adapter := NewNotifierAdapter(NewHub())
	if err := adapter.Send(context.Background(), nil); err != nil {
		t.Fatalf("Send(nil) should be a no-op, got error: %v", err)
	}
}

func TestNotifierAdapterGetType(t *testing.T) {
	adapter := NewNotifierAdapter(NewHub())
	if adapter.GetType() != ProviderTypeWebSocket {
		t.Errorf("expected provider type %q, got %q", ProviderTypeWebSocket, adapter.GetType())
	}
	if err := adapter.ValidateConfig(); err != nil {
		t.Errorf("ValidateConfig should not error: %v", err)
	}
}
