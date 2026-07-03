// Package notification provides notification delivery interfaces
package notification

import (
	"context"
	"time"
)

// Notifier defines the interface for notification providers.
type Notifier interface {
	// Send sends a notification
	Send(ctx context.Context, notification *Notification) error

	// GetType returns the provider type
	GetType() ProviderType

	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// ProviderType represents the type of notification provider.
type ProviderType string

const (
	ProviderTypeSlack   ProviderType = "slack"
	ProviderTypeEmail   ProviderType = "email"
	ProviderTypeWebhook ProviderType = "webhook"
)

// Notification represents a notification message.
type Notification struct {
	Title       string
	Message     string
	Level       NotificationLevel
	Timestamp   time.Time
	Metadata    map[string]interface{}
	Tags        []string
	Attachments []*Attachment
}

// NotificationLevel represents the severity level.
type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "info"
	LevelWarning NotificationLevel = "warning"
	LevelError   NotificationLevel = "error"
	LevelSuccess NotificationLevel = "success"
)

// Attachment represents a file or data attachment.
type Attachment struct {
	Title    string
	Text     string
	Color    string
	Fields   []*Field
	ImageURL string
	FileURL  string
}

// Field represents a key-value field in an attachment.
type Field struct {
	Title string
	Value string
	Short bool
}

// Router manages multiple notifiers and routes notifications.
type Router struct {
	notifiers []Notifier
	filters   []NotificationFilter
}

// NotificationFilter filters which notifications should be sent.
type NotificationFilter interface {
	ShouldSend(notification *Notification) bool
}

// NewRouter creates a new notification router.
func NewRouter() *Router {
	return &Router{
		notifiers: make([]Notifier, 0),
		filters:   make([]NotificationFilter, 0),
	}
}

// AddNotifier adds a notifier to the router.
func (r *Router) AddNotifier(notifier Notifier) {
	r.notifiers = append(r.notifiers, notifier)
}

// AddFilter adds a notification filter.
func (r *Router) AddFilter(filter NotificationFilter) {
	r.filters = append(r.filters, filter)
}

// Send sends a notification to all registered notifiers.
func (r *Router) Send(ctx context.Context, notification *Notification) error {
	// Apply filters
	for _, filter := range r.filters {
		if !filter.ShouldSend(notification) {
			return nil
		}
	}

	// Set timestamp if not set
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	// Send to all notifiers
	var lastErr error
	for _, notifier := range r.notifiers {
		if err := notifier.Send(ctx, notification); err != nil {
			lastErr = err
			// Continue sending to other notifiers even if one fails
		}
	}

	return lastErr
}

// LevelFilter filters notifications by level.
type LevelFilter struct {
	MinLevel NotificationLevel
}

func (f *LevelFilter) ShouldSend(notification *Notification) bool {
	levels := map[NotificationLevel]int{
		LevelInfo:    0,
		LevelSuccess: 1,
		LevelWarning: 2,
		LevelError:   3,
	}

	minLevel, ok1 := levels[f.MinLevel]
	notifLevel, ok2 := levels[notification.Level]

	if !ok1 || !ok2 {
		return true
	}

	return notifLevel >= minLevel
}

// TagFilter filters notifications by tags.
type TagFilter struct {
	RequiredTags []string
}

func (f *TagFilter) ShouldSend(notification *Notification) bool {
	if len(f.RequiredTags) == 0 {
		return true
	}

	tagMap := make(map[string]bool)
	for _, tag := range notification.Tags {
		tagMap[tag] = true
	}

	for _, required := range f.RequiredTags {
		if !tagMap[required] {
			return false
		}
	}

	return true
}
