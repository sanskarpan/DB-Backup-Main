package notification

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// PushNotificationType defines the type of push notification.
type PushNotificationType string

const (
	PushNotificationBackupSuccess   PushNotificationType = "backup_success"
	PushNotificationBackupFailed    PushNotificationType = "backup_failed"
	PushNotificationComplianceAlert PushNotificationType = "compliance_alert"
	PushNotificationMonitoringAlert PushNotificationType = "monitoring_alert"
	PushNotificationSystemCritical  PushNotificationType = "system_critical"
)

// Subscription represents a push notification subscription.
type Subscription struct {
	ID          string                  `json:"id"`
	UserID      string                  `json:"user_id"`
	Endpoint    string                  `json:"endpoint"`
	Keys        SubscriptionKeys        `json:"keys"`
	Preferences NotificationPreferences `json:"preferences"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	LastUsedAt  *time.Time              `json:"last_used_at,omitempty"`
	Active      bool                    `json:"active"`
	DeviceInfo  map[string]string       `json:"device_info,omitempty"`
}

// SubscriptionKeys contains the encryption keys for the subscription.
type SubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// NotificationPreferences defines user preferences for notifications.
type NotificationPreferences struct { //nolint:revive // keeps public name stable across packages
	BackupSuccess   bool `json:"backup_success"`
	BackupFailed    bool `json:"backup_failed"`
	ComplianceAlert bool `json:"compliance_alert"`
	MonitoringAlert bool `json:"monitoring_alert"`
	SystemCritical  bool `json:"system_critical"`
	QuietHoursStart int  `json:"quiet_hours_start"` // 24-hour format
	QuietHoursEnd   int  `json:"quiet_hours_end"`   // 24-hour format
}

// PushNotification represents a notification to be sent.
type PushNotification struct {
	Title              string                 `json:"title"`
	Body               string                 `json:"body"`
	Icon               string                 `json:"icon,omitempty"`
	Badge              string                 `json:"badge,omitempty"`
	Tag                string                 `json:"tag,omitempty"`
	Type               PushNotificationType   `json:"type"`
	Priority           string                 `json:"priority"` // low, normal, high
	Data               map[string]interface{} `json:"data,omitempty"`
	Actions            []NotificationAction   `json:"actions,omitempty"`
	RequireInteraction bool                   `json:"require_interaction,omitempty"`
	Silent             bool                   `json:"silent,omitempty"`
	Vibrate            []int                  `json:"vibrate,omitempty"`
	BadgeCount         int                    `json:"badge_count,omitempty"`
}

// NotificationAction represents an action button in the notification.
type NotificationAction struct { //nolint:revive // keeps public name stable across packages
	Action string `json:"action"`
	Title  string `json:"title"`
	Icon   string `json:"icon,omitempty"`
}

// PushService handles push notifications.
type PushService struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription // endpoint -> subscription
	vapidKeys     *VAPIDKeys
	publicKey     string
	privateKey    string
	subject       string // Email or URL for VAPID
}

// VAPIDKeys contains the VAPID key pair.
type VAPIDKeys struct {
	PublicKey  string
	PrivateKey string
}

// NewPushService creates a new push notification service.
func NewPushService(subject string) (*PushService, error) {
	// Generate VAPID keys
	privateKey, publicKey, err := generateVAPIDKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to generate VAPID keys: %w", err)
	}

	return &PushService{
		subscriptions: make(map[string]*Subscription),
		vapidKeys: &VAPIDKeys{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		},
		publicKey:  publicKey,
		privateKey: privateKey,
		subject:    subject,
	}, nil
}

// generateVAPIDKeys generates a new VAPID key pair.
func generateVAPIDKeys() (privateKey, publicKey string, err error) {
	// Generate ECDSA key pair on the P-256 curve
	curve := elliptic.P256()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return "", "", err
	}

	// Encode private key
	privateKeyBytes := key.D.Bytes()
	privateKey = base64.RawURLEncoding.EncodeToString(privateKeyBytes)

	// Encode public key (uncompressed point format)
	publicKeyBytes := elliptic.Marshal(curve, key.PublicKey.X, key.PublicKey.Y) //nolint:staticcheck // SA1019: uncompressed point encoding required for VAPID public key; crypto/ecdh migration deferred
	publicKey = base64.RawURLEncoding.EncodeToString(publicKeyBytes)

	return privateKey, publicKey, nil
}

// GetPublicKey returns the VAPID public key for client-side subscription.
func (s *PushService) GetPublicKey() string {
	return s.publicKey
}

// Subscribe adds a new push subscription.
func (s *PushService) Subscribe(ctx context.Context, sub *Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set default preferences if not provided
	if sub.Preferences == (NotificationPreferences{}) {
		sub.Preferences = NotificationPreferences{
			BackupSuccess:   true,
			BackupFailed:    true,
			ComplianceAlert: true,
			MonitoringAlert: true,
			SystemCritical:  true,
			QuietHoursStart: 22, // 10 PM
			QuietHoursEnd:   7,  // 7 AM
		}
	}

	sub.Active = true
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()

	s.subscriptions[sub.Endpoint] = sub

	return nil
}

// Unsubscribe removes a push subscription.
func (s *PushService) Unsubscribe(ctx context.Context, endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub, exists := s.subscriptions[endpoint]; exists {
		sub.Active = false
		sub.UpdatedAt = time.Now()
	}

	return nil
}

// GetSubscription retrieves a subscription by endpoint.
func (s *PushService) GetSubscription(endpoint string) (*Subscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, exists := s.subscriptions[endpoint]
	return sub, exists
}

// GetUserSubscriptions retrieves all subscriptions for a user.
func (s *PushService) GetUserSubscriptions(userID string) []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var subs []*Subscription
	for _, sub := range s.subscriptions {
		if sub.UserID == userID && sub.Active {
			subs = append(subs, sub)
		}
	}

	return subs
}

// UpdatePreferences updates notification preferences for a subscription.
func (s *PushService) UpdatePreferences(endpoint string, prefs NotificationPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[endpoint]
	if !exists {
		return fmt.Errorf("subscription not found")
	}

	sub.Preferences = prefs
	sub.UpdatedAt = time.Now()

	return nil
}

// SendNotification sends a push notification to a specific subscription.
func (s *PushService) SendNotification(ctx context.Context, endpoint string, notification *PushNotification) error {
	s.mu.RLock()
	sub, exists := s.subscriptions[endpoint]
	s.mu.RUnlock()

	if !exists || !sub.Active {
		return fmt.Errorf("subscription not found or inactive")
	}

	// Check if user wants this type of notification
	if !s.shouldSendNotification(sub, notification) {
		return nil // Silently skip
	}

	// Check quiet hours
	if s.isQuietHours(sub.Preferences) && notification.Priority != "high" {
		return nil // Silently skip during quiet hours for non-high priority
	}

	// Prepare the notification payload
	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create web push subscription
	webPushSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.Keys.P256dh,
			Auth:   sub.Keys.Auth,
		},
	}

	// Set TTL (time to live) based on priority
	ttl := 3600 // 1 hour default
	if notification.Priority == "high" {
		ttl = 86400 // 24 hours
	} else if notification.Priority == "low" {
		ttl = 1800 // 30 minutes
	}

	// Send the notification
	resp, err := webpush.SendNotification(payload, webPushSub, &webpush.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             ttl,
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	// Update last used timestamp
	s.mu.Lock()
	now := time.Now()
	sub.LastUsedAt = &now
	s.mu.Unlock()

	// Handle response status
	if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
		// Subscription is no longer valid, deactivate it
		if err := s.Unsubscribe(ctx, endpoint); err != nil {
			return fmt.Errorf("subscription expired and failed to unsubscribe: %w", err)
		}
		return fmt.Errorf("subscription expired or not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push service returned error: %d", resp.StatusCode)
	}

	return nil
}

// SendToUser sends a notification to all active subscriptions for a user.
func (s *PushService) SendToUser(ctx context.Context, userID string, notification *PushNotification) error {
	subscriptions := s.GetUserSubscriptions(userID)

	var lastErr error
	successCount := 0

	for _, sub := range subscriptions {
		err := s.SendNotification(ctx, sub.Endpoint, notification)
		if err != nil {
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send to any subscription: %w", lastErr)
	}

	return nil
}

// BroadcastNotification sends a notification to all active subscriptions.
func (s *PushService) BroadcastNotification(ctx context.Context, notification *PushNotification) error {
	s.mu.RLock()
	endpoints := make([]string, 0, len(s.subscriptions))
	for endpoint, sub := range s.subscriptions {
		if sub.Active {
			endpoints = append(endpoints, endpoint)
		}
	}
	s.mu.RUnlock()

	var lastErr error
	successCount := 0

	for _, endpoint := range endpoints {
		err := s.SendNotification(ctx, endpoint, notification)
		if err != nil {
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send to any subscription: %w", lastErr)
	}

	return nil
}

// shouldSendNotification checks if notification should be sent based on preferences.
func (s *PushService) shouldSendNotification(sub *Subscription, notification *PushNotification) bool {
	prefs := sub.Preferences

	switch notification.Type {
	case PushNotificationBackupSuccess:
		return prefs.BackupSuccess
	case PushNotificationBackupFailed:
		return prefs.BackupFailed
	case PushNotificationComplianceAlert:
		return prefs.ComplianceAlert
	case PushNotificationMonitoringAlert:
		return prefs.MonitoringAlert
	case PushNotificationSystemCritical:
		return prefs.SystemCritical
	default:
		return true
	}
}

// isQuietHours checks if current time is within quiet hours.
func (s *PushService) isQuietHours(prefs NotificationPreferences) bool {
	now := time.Now()
	currentHour := now.Hour()

	start := prefs.QuietHoursStart
	end := prefs.QuietHoursEnd

	if start < end {
		// Normal case: quiet hours don't cross midnight
		return currentHour >= start && currentHour < end
	} else if start > end {
		// Quiet hours cross midnight
		return currentHour >= start || currentHour < end
	}

	return false
}

// CleanupInactiveSubscriptions removes subscriptions that haven't been used recently.
func (s *PushService) CleanupInactiveSubscriptions(inactiveDays int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -inactiveDays)
	removed := 0

	for _, sub := range s.subscriptions {
		if !sub.Active {
			continue
		}

		// If never used, check creation date
		if sub.LastUsedAt == nil {
			if sub.CreatedAt.Before(cutoff) {
				sub.Active = false
				removed++
			}
		} else if sub.LastUsedAt.Before(cutoff) {
			sub.Active = false
			removed++
		}
	}

	return removed
}

// GetStats returns statistics about subscriptions.
func (s *PushService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.subscriptions)
	active := 0

	for _, sub := range s.subscriptions {
		if sub.Active {
			active++
		}
	}

	return map[string]interface{}{
		"total_subscriptions":    total,
		"active_subscriptions":   active,
		"inactive_subscriptions": total - active,
	}
}
