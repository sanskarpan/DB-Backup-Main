// Package websocket provides WebSocket support for real-time communications
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub manages WebSocket connections and message broadcasting
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Client subscriptions (topic -> clients)
	subscriptions map[string]map[*Client]bool

	mu sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType            `json:"type"`
	Topic     string                 `json:"topic,omitempty"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Message types
	MessageTypeBackupProgress    MessageType = "backup_progress"
	MessageTypeBackupComplete    MessageType = "backup_complete"
	MessageTypeBackupFailed      MessageType = "backup_failed"
	MessageTypeRestoreProgress   MessageType = "restore_progress"
	MessageTypeRestoreComplete   MessageType = "restore_complete"
	MessageTypeRestoreFailed     MessageType = "restore_failed"
	MessageTypeLog               MessageType = "log"
	MessageTypeNotification      MessageType = "notification"
	MessageTypeSubscribe         MessageType = "subscribe"
	MessageTypeUnsubscribe       MessageType = "unsubscribe"
	MessageTypePing              MessageType = "ping"
	MessageTypePong              MessageType = "pong"
	MessageTypeError             MessageType = "error"
)

// BackupProgress represents backup progress data
type BackupProgress struct {
	BackupID         string  `json:"backup_id"`
	DatabaseID       string  `json:"database_id"`
	Status           string  `json:"status"`
	Progress         float64 `json:"progress"` // 0-100
	BytesProcessed   int64   `json:"bytes_processed"`
	BytesTotal       int64   `json:"bytes_total"`
	ElapsedTime      int64   `json:"elapsed_time"` // milliseconds
	EstimatedTimeLeft int64  `json:"estimated_time_left"` // milliseconds
	CurrentStep      string  `json:"current_step"`
}

// LogMessage represents a log message
type LogMessage struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Notification represents a user notification
type Notification struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // info, warning, error, success
	Title    string    `json:"title"`
	Message  string    `json:"message"`
	Priority int       `json:"priority"` // 0-10
	Action   string    `json:"action,omitempty"`
	Time     time.Time `json:"time"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		broadcast:     make(chan *Message, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		// Remove from subscriptions
		for topic := range h.subscriptions {
			delete(h.subscriptions[topic], client)
			if len(h.subscriptions[topic]) == 0 {
				delete(h.subscriptions, topic)
			}
		}

		delete(h.clients, client)
		close(client.send)
		log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))
	}
}

// broadcastMessage broadcasts a message to subscribed clients
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// If message has a topic, send only to subscribers
	if message.Topic != "" {
		if subscribers, ok := h.subscriptions[message.Topic]; ok {
			for client := range subscribers {
				select {
				case client.send <- message:
				default:
					// Client's send buffer is full, skip
					log.Printf("Skipping client - send buffer full")
				}
			}
		}
		return
	}

	// Broadcast to all clients
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// Client's send buffer is full, skip
			log.Printf("Skipping client - send buffer full")
		}
	}
}

// Subscribe subscribes a client to a topic
func (h *Hub) Subscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.subscriptions[topic]; !ok {
		h.subscriptions[topic] = make(map[*Client]bool)
	}
	h.subscriptions[topic][client] = true

	log.Printf("Client subscribed to topic: %s. Subscribers: %d", topic, len(h.subscriptions[topic]))
}

// Unsubscribe unsubscribes a client from a topic
func (h *Hub) Unsubscribe(client *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, ok := h.subscriptions[topic]; ok {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(h.subscriptions, topic)
		}
		log.Printf("Client unsubscribed from topic: %s", topic)
	}
}

// Broadcast sends a message to all clients or topic subscribers
func (h *Hub) Broadcast(message *Message) {
	message.Timestamp = time.Now()
	h.broadcast <- message
}

// BroadcastBackupProgress broadcasts backup progress update
func (h *Hub) BroadcastBackupProgress(progress *BackupProgress) {
	h.Broadcast(&Message{
		Type:  MessageTypeBackupProgress,
		Topic: "backup:" + progress.BackupID,
		Data:  progress,
	})
}

// BroadcastLog broadcasts a log message
func (h *Hub) BroadcastLog(logMsg *LogMessage) {
	h.Broadcast(&Message{
		Type:  MessageTypeLog,
		Topic: "logs",
		Data:  logMsg,
	})
}

// BroadcastNotification broadcasts a notification
func (h *Hub) BroadcastNotification(notification *Notification) {
	h.Broadcast(&Message{
		Type:  MessageTypeNotification,
		Topic: "notifications",
		Data:  notification,
	})
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	topicStats := make(map[string]int)
	for topic, subscribers := range h.subscriptions {
		topicStats[topic] = len(subscribers)
	}

	return map[string]interface{}{
		"total_clients":      len(h.clients),
		"total_subscriptions": len(h.subscriptions),
		"topic_stats":        topicStats,
	}
}

// Marshal converts a message to JSON
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal converts JSON to a message
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// Shutdown gracefully stops the hub
func (h *Hub) Shutdown(ctx context.Context) error {
	log.Println("WebSocket hub shutting down...")
	
	// Close all client connections
	h.mu.Lock()
	for client := range h.clients {
		close(client.send)
		if err := client.conn.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
	}
	h.mu.Unlock()
	
	log.Println("WebSocket hub shutdown complete")
	return nil
}
