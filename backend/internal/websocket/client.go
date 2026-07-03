package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait).
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a WebSocket client connection.
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// Hub that manages this client
	hub *Hub

	// Buffered channel of outbound messages
	send chan *Message

	// Client metadata
	ID       string
	UserID   string
	Metadata map[string]interface{}

	// Subscribed topics
	topics map[string]bool
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		conn:     conn,
		hub:      hub,
		send:     make(chan *Message, 256),
		UserID:   userID,
		Metadata: make(map[string]interface{}),
		topics:   make(map[string]bool),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle the message
		c.handleMessage(messageData)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				// The hub closed the channel
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Printf("Failed to write close message: %v", err)
				}
				return
			}

			// Marshal message to JSON
			data, err := message.Marshal()
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			// Send the message
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client.
func (c *Client) handleMessage(data []byte) {
	message, err := UnmarshalMessage(data)
	if err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		c.sendError("Invalid message format")
		return
	}

	switch message.Type {
	case MessageTypeSubscribe:
		c.handleSubscribe(message)
	case MessageTypeUnsubscribe:
		c.handleUnsubscribe(message)
	case MessageTypePing:
		c.sendPong()
	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

// handleSubscribe handles subscription requests.
func (c *Client) handleSubscribe(message *Message) {
	if message.Topic == "" {
		c.sendError("Topic is required for subscription")
		return
	}

	c.hub.Subscribe(c, message.Topic)
	c.topics[message.Topic] = true

	// Send confirmation
	c.send <- &Message{
		Type:      MessageTypeNotification,
		Data:      map[string]interface{}{"message": "Subscribed to " + message.Topic},
		Timestamp: time.Now(),
	}
}

// handleUnsubscribe handles unsubscription requests.
func (c *Client) handleUnsubscribe(message *Message) {
	if message.Topic == "" {
		c.sendError("Topic is required for unsubscription")
		return
	}

	c.hub.Unsubscribe(c, message.Topic)
	delete(c.topics, message.Topic)

	// Send confirmation
	c.send <- &Message{
		Type:      MessageTypeNotification,
		Data:      map[string]interface{}{"message": "Unsubscribed from " + message.Topic},
		Timestamp: time.Now(),
	}
}

// sendPong sends a pong response.
func (c *Client) sendPong() {
	c.send <- &Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(errorMsg string) {
	c.send <- &Message{
		Type:      MessageTypeError,
		Data:      map[string]interface{}{"error": errorMsg},
		Timestamp: time.Now(),
	}
}

// SendMessage sends a message to the client.
func (c *Client) SendMessage(message *Message) {
	select {
	case c.send <- message:
	default:
		log.Printf("Client send buffer full, message dropped")
	}
}

// SendJSON sends arbitrary JSON data to the client.
func (c *Client) SendJSON(messageType MessageType, data interface{}) error {
	message := &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case c.send <- message:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

// GetSubscriptions returns the list of topics the client is subscribed to.
func (c *Client) GetSubscriptions() []string {
	topics := make([]string, 0, len(c.topics))
	for topic := range c.topics {
		topics = append(topics, topic)
	}
	return topics
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// IsConnected checks if the client is still connected.
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// MarshalClientInfo returns client information as JSON.
func (c *Client) MarshalClientInfo() ([]byte, error) {
	info := map[string]interface{}{
		"id":            c.ID,
		"user_id":       c.UserID,
		"subscriptions": c.GetSubscriptions(),
		"metadata":      c.Metadata,
	}
	return json.Marshal(info)
}
