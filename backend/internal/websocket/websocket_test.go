package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("Hub should not be nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map should be initialized")
	}

	if hub.subscriptions == nil {
		t.Error("Hub subscriptions map should be initialized")
	}
}

func TestHubRegisterClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a mock WebSocket connection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		client := NewClient(hub, conn, "test-user")
		hub.register <- client

		// Wait a bit for registration
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	// Connect as a client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	defer ws.Close()

	time.Sleep(200 * time.Millisecond)

	stats := hub.GetStats()
	totalClients, ok := stats["total_clients"].(int)
	if !ok {
		t.Fatalf("total_clients is not an int: %T", stats["total_clients"])
	}
	if totalClients != 1 {
		t.Errorf("Expected 1 client, got %d", totalClients)
	}
}

func TestMessageMarshaling(t *testing.T) {
	msg := &Message{
		Type:      MessageTypeBackupProgress,
		Topic:     "backup:123",
		Data:      map[string]interface{}{"progress": 50},
		Timestamp: time.Now(),
	}

	data, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	decoded, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Expected type %s, got %s", msg.Type, decoded.Type)
	}

	if decoded.Topic != msg.Topic {
		t.Errorf("Expected topic %s, got %s", msg.Topic, decoded.Topic)
	}
}

func TestBackupProgressTracker(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tracker := NewBackupProgressTracker(hub)

	t.Run("start backup", func(t *testing.T) {
		tracker.StartBackup("backup-123", "db-456")
		// Give time for broadcast
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("update progress", func(t *testing.T) {
		progress := &BackupProgress{
			BackupID:       "backup-123",
			DatabaseID:     "db-456",
			Status:         "running",
			Progress:       50.0,
			BytesProcessed: 1024 * 1024,
			BytesTotal:     2048 * 1024,
			CurrentStep:    "Backing up tables",
		}
		tracker.UpdateProgress(progress)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("complete backup", func(t *testing.T) {
		tracker.CompleteBackup("backup-123", "db-456", 2048*1024, 30000)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("fail backup", func(t *testing.T) {
		tracker.FailBackup("backup-456", "db-789", "Connection timeout")
		time.Sleep(50 * time.Millisecond)
	})
}

func TestLogStreamer(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	streamer := NewLogStreamer(hub)

	tests := []struct {
		name   string
		level  string
		action func()
	}{
		{
			name:  "debug log",
			level: "debug",
			action: func() {
				streamer.Debug("test-source", "Debug message", map[string]interface{}{"key": "value"})
			},
		},
		{
			name:  "info log",
			level: "info",
			action: func() {
				streamer.Info("test-source", "Info message", nil)
			},
		},
		{
			name:  "warning log",
			level: "warn",
			action: func() {
				streamer.Warn("test-source", "Warning message", nil)
			},
		},
		{
			name:  "error log",
			level: "error",
			action: func() {
				streamer.Error("test-source", "Error message", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.action()
			time.Sleep(50 * time.Millisecond)
		})
	}
}

func TestNotificationBroadcaster(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	broadcaster := NewNotificationBroadcaster(hub)

	t.Run("info notification", func(t *testing.T) {
		broadcaster.Info("Test Info", "This is an info notification")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("success notification", func(t *testing.T) {
		broadcaster.Success("Test Success", "Operation completed successfully")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("warning notification", func(t *testing.T) {
		broadcaster.Warning("Test Warning", "This is a warning")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("error notification", func(t *testing.T) {
		broadcaster.Error("Test Error", "An error occurred")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("backup notification", func(t *testing.T) {
		broadcaster.BackupNotification("backup-123", "completed", "Backup completed successfully")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("restore notification", func(t *testing.T) {
		broadcaster.RestoreNotification("restore-456", "started", "Restore operation started")
		time.Sleep(50 * time.Millisecond)
	})
}

func TestRestoreProgressTracker(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	tracker := NewRestoreProgressTracker(hub)

	t.Run("update progress", func(t *testing.T) {
		tracker.UpdateProgress("restore-123", "db-456", "running", 75.0)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("complete restore", func(t *testing.T) {
		tracker.CompleteRestore("restore-123", "db-456")
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("fail restore", func(t *testing.T) {
		tracker.FailRestore("restore-456", "db-789", "Database not found")
		time.Sleep(50 * time.Millisecond)
	})
}

func TestClientSubscriptions(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := NewClient(hub, conn, "test-user")
		client.ID = "test-client-1"
		hub.register <- client

		// Start pumps
		go client.WritePump()
		go client.ReadPump()

		// Keep connection alive
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	// Connect as a client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	defer ws.Close()

	time.Sleep(200 * time.Millisecond)

	// Send subscription message
	subMsg := &Message{
		Type:  MessageTypeSubscribe,
		Topic: "backup:123",
	}
	data, _ := json.Marshal(subMsg)
	err = ws.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("Failed to send subscribe message: %v", err)
	}

	// Wait for subscription to process
	time.Sleep(200 * time.Millisecond)

	stats := hub.GetStats()
	totalSubs, ok := stats["total_subscriptions"].(int)
	if !ok {
		t.Fatalf("total_subscriptions is not an int: %T", stats["total_subscriptions"])
	}
	if totalSubs == 0 {
		t.Error("Expected at least one subscription")
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := NewClient(hub, conn, "test-user")
		hub.register <- client

		go client.WritePump()
		go client.ReadPump()

		// Keep connection alive
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	// Connect as a client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	defer ws.Close()

	time.Sleep(200 * time.Millisecond)

	// Start reading messages in a goroutine
	messageReceived := make(chan bool, 1)
	go func() {
		_, _, err := ws.ReadMessage()
		if err == nil {
			messageReceived <- true
		}
	}()

	// Broadcast a message
	hub.Broadcast(&Message{
		Type: MessageTypeNotification,
		Data: map[string]interface{}{"message": "Test broadcast"},
	})

	// Wait for message
	select {
	case <-messageReceived:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Message not received within timeout")
	}
}

func TestGetStats(t *testing.T) {
	hub := NewHub()

	stats := hub.GetStats()

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if _, ok := stats["total_clients"]; !ok {
		t.Error("Stats should include total_clients")
	}

	if _, ok := stats["total_subscriptions"]; !ok {
		t.Error("Stats should include total_subscriptions")
	}

	if _, ok := stats["topic_stats"]; !ok {
		t.Error("Stats should include topic_stats")
	}
}

func TestClientInfo(t *testing.T) {
	hub := NewHub()
	conn := &websocket.Conn{} // Mock connection

	client := NewClient(hub, conn, "user-123")
	client.ID = "client-456"
	client.topics["backup:123"] = true
	client.topics["logs"] = true

	data, err := client.MarshalClientInfo()
	if err != nil {
		t.Fatalf("Failed to marshal client info: %v", err)
	}

	var info map[string]interface{}
	err = json.Unmarshal(data, &info)
	if err != nil {
		t.Fatalf("Failed to unmarshal client info: %v", err)
	}

	if info["id"] != "client-456" {
		t.Errorf("Expected client ID 'client-456', got '%s'", info["id"])
	}

	if info["user_id"] != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", info["user_id"])
	}

	subscriptions, ok := info["subscriptions"].([]interface{})
	if !ok {
		t.Fatalf("subscriptions is not a slice: %T", info["subscriptions"])
	}
	if len(subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subscriptions))
	}
}
