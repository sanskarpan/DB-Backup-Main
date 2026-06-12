package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/notification"
)

// PushHandler handles push notification related requests
type PushHandler struct {
	pushService *notification.PushService
}

// NewPushHandler creates a new push notification handler
func NewPushHandler(pushService *notification.PushService) *PushHandler {
	return &PushHandler{
		pushService: pushService,
	}
}

// GetPublicKey returns the VAPID public key for client-side subscription
// GET /api/push/public-key
func (h *PushHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]string{
		"publicKey": h.pushService.GetPublicKey(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Subscribe handles push notification subscription
// POST /api/push/subscribe
func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID     string                             `json:"user_id"`
		Endpoint   string                             `json:"endpoint"`
		Keys       notification.SubscriptionKeys      `json:"keys"`
		Preferences notification.NotificationPreferences `json:"preferences,omitempty"`
		DeviceInfo map[string]string                  `json:"device_info,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Generate subscription ID
	id := generateID()

	sub := &notification.Subscription{
		ID:          id,
		UserID:      req.UserID,
		Endpoint:    req.Endpoint,
		Keys:        req.Keys,
		Preferences: req.Preferences,
		DeviceInfo:  req.DeviceInfo,
	}

	if err := h.pushService.Subscribe(r.Context(), sub); err != nil {
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"subscription": map[string]string{
			"id":       sub.ID,
			"endpoint": sub.Endpoint,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Unsubscribe handles push notification unsubscription
// POST /api/push/unsubscribe
func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "Missing endpoint", http.StatusBadRequest)
		return
	}

	if err := h.pushService.Unsubscribe(r.Context(), req.Endpoint); err != nil {
		http.Error(w, "Failed to unsubscribe", http.StatusInternalServerError)
		return
	}

	response := map[string]bool{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdatePreferences updates notification preferences for a subscription
// POST /api/push/preferences
func (h *PushHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Endpoint    string                             `json:"endpoint"`
		Preferences notification.NotificationPreferences `json:"preferences"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "Missing endpoint", http.StatusBadRequest)
		return
	}

	if err := h.pushService.UpdatePreferences(req.Endpoint, req.Preferences); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]bool{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SendTestNotification sends a test push notification
// POST /api/push/test
func (h *PushHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" {
		http.Error(w, "Missing endpoint", http.StatusBadRequest)
		return
	}

	// Create test notification
	testNotification := &notification.PushNotification{
		Title:    "DB Backup Manager - Test Notification",
		Body:     "This is a test notification. Your push notifications are working correctly!",
		Icon:     "/icons/icon-192x192.png",
		Badge:    "/icons/badge-72x72.png",
		Tag:      "test-notification",
		Type:     notification.PushNotificationBackupSuccess,
		Priority: "normal",
		Data: map[string]interface{}{
			"test":      true,
			"timestamp": time.Now().Unix(),
		},
		Actions: []notification.NotificationAction{
			{
				Action: "view",
				Title:  "View Dashboard",
			},
		},
		Vibrate: []int{200, 100, 200},
	}

	if err := h.pushService.SendNotification(r.Context(), req.Endpoint, testNotification); err != nil {
		http.Error(w, "Failed to send notification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]bool{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSubscriptions retrieves all subscriptions for a user
// GET /api/push/subscriptions?user_id=xxx
func (h *PushHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	subscriptions := h.pushService.GetUserSubscriptions(userID)

	response := map[string]interface{}{
		"subscriptions": subscriptions,
		"count":         len(subscriptions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats returns statistics about push subscriptions
// GET /api/push/stats
func (h *PushHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.pushService.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper function to generate unique IDs
func generateID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
