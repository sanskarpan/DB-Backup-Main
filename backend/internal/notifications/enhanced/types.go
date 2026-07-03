package enhanced

import (
	"time"
)

// Priority levels for notifications.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

func (p Priority) String() string {
	return [...]string{"low", "normal", "high", "urgent"}[p]
}

// NotificationType defines the type of notification.
type NotificationType string

const (
	TypeBackupStarted     NotificationType = "backup_started"
	TypeBackupCompleted   NotificationType = "backup_completed"
	TypeBackupFailed      NotificationType = "backup_failed"
	TypeRestoreStarted    NotificationType = "restore_started"
	TypeRestoreCompleted  NotificationType = "restore_completed"
	TypeRestoreFailed     NotificationType = "restore_failed"
	TypeScheduleTriggered NotificationType = "schedule_triggered"
	TypeAlertTriggered    NotificationType = "alert_triggered"
	TypeSystemWarning     NotificationType = "system_warning"
	TypeSystemError       NotificationType = "system_error"
	TypeComplianceScan    NotificationType = "compliance_scan"
	TypeCustom            NotificationType = "custom"
)

// DeliveryChannel represents a notification delivery method.
type DeliveryChannel string

const (
	ChannelEmail   DeliveryChannel = "email"
	ChannelSMS     DeliveryChannel = "sms"
	ChannelPush    DeliveryChannel = "push"
	ChannelSlack   DeliveryChannel = "slack"
	ChannelTeams   DeliveryChannel = "teams"
	ChannelDiscord DeliveryChannel = "discord"
	ChannelWebhook DeliveryChannel = "webhook"
	ChannelInApp   DeliveryChannel = "in_app"
)

// NotificationStatus represents the delivery status.
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusQueued    NotificationStatus = "queued"
	StatusSending   NotificationStatus = "sending"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
	StatusDismissed NotificationStatus = "dismissed"
	StatusSnoozed   NotificationStatus = "snoozed"
)

// Notification represents a single notification.
type Notification struct {
	ID       string                 `json:"id"`
	Type     NotificationType       `json:"type"`
	Priority Priority               `json:"priority"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Category string                 `json:"category"` // backup, restore, system, etc.
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`

	// Delivery
	Channels         []DeliveryChannel  `json:"channels"`
	Status           NotificationStatus `json:"status"`
	DeliveryAttempts int                `json:"delivery_attempts"`

	// Actions
	Actions []NotificationAction `json:"actions"`

	// Timing
	CreatedAt    time.Time  `json:"created_at"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	SnoozedUntil *time.Time `json:"snoozed_until,omitempty"`

	// Grouping
	GroupID  string `json:"group_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`

	// Targeting
	UserID  string   `json:"user_id"`
	RoleIDs []string `json:"role_ids,omitempty"`

	// Rich content
	ImageURL  string                 `json:"image_url,omitempty"`
	VideoURL  string                 `json:"video_url,omitempty"`
	ChartData map[string]interface{} `json:"chart_data,omitempty"`

	// Analytics
	RelevanceScore  float64 `json:"relevance_score"`
	EngagementScore float64 `json:"engagement_score"`

	// Compliance
	AuditTrail      []AuditEntry `json:"audit_trail,omitempty"`
	RetentionPolicy string       `json:"retention_policy,omitempty"`
}

// NotificationAction represents an action button in notification.
type NotificationAction struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`   // button, link, inline_form
	Action   string                 `json:"action"` // approve, reject, retry, view, etc.
	URL      string                 `json:"url,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Style    string                 `json:"style"` // primary, secondary, danger
}

// AuditEntry represents an audit log entry.
type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"` // created, delivered, read, dismissed
	UserID    string                 `json:"user_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationGroup represents a group of related notifications.
type NotificationGroup struct {
	ID            string                 `json:"id"`
	Type          NotificationType       `json:"type"`
	Category      string                 `json:"category"`
	Title         string                 `json:"title"`
	Summary       string                 `json:"summary"`
	Count         int                    `json:"count"`
	Priority      Priority               `json:"priority"` // Highest priority in group
	Notifications []*Notification        `json:"notifications"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NotificationTemplate represents a reusable notification template.
type NotificationTemplate struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      NotificationType       `json:"type"`
	Title     string                 `json:"title"`   // Template with variables
	Message   string                 `json:"message"` // Template with variables
	Priority  Priority               `json:"priority"`
	Channels  []DeliveryChannel      `json:"channels"`
	Actions   []NotificationAction   `json:"actions"`
	Variables []string               `json:"variables"` // Required variables
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NotificationPreferences represents user notification preferences.
type NotificationPreferences struct {
	UserID              string                                `json:"user_id"`
	EnabledChannels     []DeliveryChannel                     `json:"enabled_channels"`
	DisabledTypes       []NotificationType                    `json:"disabled_types"`
	MinimumPriority     Priority                              `json:"minimum_priority"`
	QuietHours          []QuietHoursPeriod                    `json:"quiet_hours"`
	DoNotDisturb        bool                                  `json:"do_not_disturb"`
	DNDSchedule         []DNDSchedule                         `json:"dnd_schedule"`
	GroupingEnabled     bool                                  `json:"grouping_enabled"`
	BatchingEnabled     bool                                  `json:"batching_enabled"`
	BatchInterval       int                                   `json:"batch_interval"` // minutes
	DigestEnabled       bool                                  `json:"digest_enabled"`
	DigestFrequency     string                                `json:"digest_frequency"` // daily, weekly
	ChannelPreferences  map[DeliveryChannel]ChannelPreference `json:"channel_preferences"`
	CategoryPreferences map[string]CategoryPreference         `json:"category_preferences"`
	UpdatedAt           time.Time                             `json:"updated_at"`
}

// QuietHoursPeriod represents a quiet hours period.
type QuietHoursPeriod struct {
	Start    string   `json:"start"` // HH:MM format
	End      string   `json:"end"`   // HH:MM format
	Days     []string `json:"days"`  // Mon, Tue, Wed, etc.
	Timezone string   `json:"timezone"`
}

// DNDSchedule represents a Do Not Disturb schedule.
type DNDSchedule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Recurring bool      `json:"recurring"`
	Pattern   string    `json:"pattern"` // daily, weekly, custom
}

// ChannelPreference represents preferences for a specific channel.
type ChannelPreference struct {
	Enabled     bool     `json:"enabled"`
	MinPriority Priority `json:"min_priority"`
	MaxPerHour  int      `json:"max_per_hour"`
	Sound       string   `json:"sound,omitempty"`
	Vibrate     bool     `json:"vibrate"`
}

// CategoryPreference represents preferences for a notification category.
type CategoryPreference struct {
	Enabled     bool              `json:"enabled"`
	Channels    []DeliveryChannel `json:"channels"`
	MinPriority Priority          `json:"min_priority"`
}

// DeliveryResult represents the result of a notification delivery attempt.
type DeliveryResult struct {
	Channel      DeliveryChannel        `json:"channel"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	DeliveredAt  time.Time              `json:"delivered_at"`
	ResponseTime float64                `json:"response_time_ms"`
	MessageID    string                 `json:"message_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationStats represents notification statistics.
type NotificationStats struct {
	TotalSent       int                              `json:"total_sent"`
	TotalDelivered  int                              `json:"total_delivered"`
	TotalFailed     int                              `json:"total_failed"`
	TotalRead       int                              `json:"total_read"`
	DeliveryRate    float64                          `json:"delivery_rate"`
	ReadRate        float64                          `json:"read_rate"`
	AvgDeliveryTime float64                          `json:"avg_delivery_time_ms"`
	ByChannel       map[DeliveryChannel]ChannelStats `json:"by_channel"`
	ByType          map[NotificationType]TypeStats   `json:"by_type"`
	ByPriority      map[Priority]PriorityStats       `json:"by_priority"`
}

// ChannelStats represents statistics for a delivery channel.
type ChannelStats struct {
	Sent         int     `json:"sent"`
	Delivered    int     `json:"delivered"`
	Failed       int     `json:"failed"`
	DeliveryRate float64 `json:"delivery_rate"`
	AvgTime      float64 `json:"avg_time_ms"`
}

// TypeStats represents statistics for a notification type.
type TypeStats struct {
	Sent        int     `json:"sent"`
	Read        int     `json:"read"`
	ReadRate    float64 `json:"read_rate"`
	AvgReadTime float64 `json:"avg_read_time_seconds"`
}

// PriorityStats represents statistics for a priority level.
type PriorityStats struct {
	Sent        int     `json:"sent"`
	Delivered   int     `json:"delivered"`
	Read        int     `json:"read"`
	AvgReadTime float64 `json:"avg_read_time_seconds"`
}
