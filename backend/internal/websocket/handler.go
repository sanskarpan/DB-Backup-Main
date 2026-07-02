package websocket

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// AllowedOrigins should be configured via environment or config file
var AllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8080",
	"https://localhost:3000",
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Allow requests with no origin (e.g., mobile apps, curl)
			return true
		}
		
		// Check against whitelist
		for _, allowed := range AllowedOrigins {
			if strings.HasPrefix(origin, allowed) {
				return true
			}
		}
		
		log.Printf("WebSocket: Rejected origin: %s", origin)
		return false
	},
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// ServeHTTP handles the WebSocket upgrade
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by authentication middleware)
	userID := getUserIDFromRequest(r)

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create new client
	client := NewClient(h.hub, conn, userID)
	client.ID = generateClientID()

	// Register client with hub
	h.hub.register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}

// ServeWithUserID upgrades the HTTP connection to a WebSocket and registers the
// resulting client with the hub using an already-authenticated user ID. Callers
// that authenticate out-of-band (for example, validating a JWT passed as a query
// parameter before the handshake) should use this instead of ServeHTTP.
func (h *Handler) ServeWithUserID(w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(h.hub, conn, userID)
	client.ID = generateClientID()

	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// getUserIDFromRequest extracts user ID from the request context
func getUserIDFromRequest(r *http.Request) string {
	// Try to get user ID from context (set by auth middleware)
	if userID := r.Context().Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}

	// Try to get from query parameter (for testing)
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		return userID
	}

	// Default to "anonymous"
	return "anonymous"
}

// generateClientID generates a cryptographically secure unique client ID
func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		log.Printf("Warning: Failed to generate secure client ID: %v", err)
		return "client_insecure_" + hex.EncodeToString(b)
	}
	return "client_" + hex.EncodeToString(b)
}
