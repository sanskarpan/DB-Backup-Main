package enhanced

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Engine is the enhanced notification engine.
type Engine struct {
	mu                  sync.RWMutex
	config              *Config
	channels            map[DeliveryChannel]Channel
	templates           map[string]*NotificationTemplate
	preferences         map[string]*NotificationPreferences
	notifications       map[string]*Notification
	groups              map[string]*NotificationGroup
	queue               chan *Notification
	analytics           *AnalyticsTracker
	aiEngine            *AIEngine
	batchProcessor      *BatchProcessor
	escalationProcessor *EscalationProcessor
	dataDir             string
	running             bool
	stopChan            chan struct{}
}

// Config represents engine configuration.
type Config struct {
	DataDir          string
	QueueSize        int
	WorkerCount      int
	MaxRetries       int
	RetryDelay       time.Duration
	BatchInterval    time.Duration
	GroupingEnabled  bool
	AIEnabled        bool
	AnalyticsEnabled bool
	AuditEnabled     bool
	RetentionDays    int
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		DataDir:          "~/.db-backup/notifications",
		QueueSize:        1000,
		WorkerCount:      5,
		MaxRetries:       3,
		RetryDelay:       time.Minute,
		BatchInterval:    5 * time.Minute,
		GroupingEnabled:  true,
		AIEnabled:        true,
		AnalyticsEnabled: true,
		AuditEnabled:     true,
		RetentionDays:    90,
	}
}

// NewEngine creates a new notification engine.
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Expand home directory
	if len(config.DataDir) >= 2 && config.DataDir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve home directory: %w", err)
		}
		config.DataDir = filepath.Join(home, config.DataDir[2:])
	}

	// Create data directory
	if err := os.MkdirAll(config.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	engine := &Engine{
		config:        config,
		channels:      make(map[DeliveryChannel]Channel),
		templates:     make(map[string]*NotificationTemplate),
		preferences:   make(map[string]*NotificationPreferences),
		notifications: make(map[string]*Notification),
		groups:        make(map[string]*NotificationGroup),
		queue:         make(chan *Notification, config.QueueSize),
		dataDir:       config.DataDir,
		stopChan:      make(chan struct{}),
	}

	// Initialize components
	if config.AnalyticsEnabled {
		engine.analytics = NewAnalyticsTracker(filepath.Join(config.DataDir, "analytics.json"), 10000)
		if err := engine.analytics.Load(); err != nil {
			return nil, fmt.Errorf("failed to load analytics: %w", err)
		}
	}

	if config.AIEnabled {
		engine.aiEngine = NewAIEngine(filepath.Join(config.DataDir, "ai_model.json"))
		if err := engine.aiEngine.Load(); err != nil {
			return nil, fmt.Errorf("failed to load AI model: %w", err)
		}
	}

	batchConfig := BatchConfig{
		Interval:         config.BatchInterval,
		MinBatchSize:     3,
		MaxBatchSize:     50,
		MaxBatchAge:      1 * time.Hour,
		GroupingStrategy: GroupBySmart,
	}
	engine.batchProcessor = NewBatchProcessor(engine, batchConfig)
	engine.escalationProcessor = NewEscalationProcessor(engine, time.Minute)

	// Load persisted data
	if err := engine.load(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return engine, nil
}

// RegisterChannel registers a delivery channel.
func (e *Engine) RegisterChannel(channelType DeliveryChannel, channel Channel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channels[channelType] = channel
}

// RegisterTemplate registers a notification template.
func (e *Engine) RegisterTemplate(template *NotificationTemplate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if template.ID == "" {
		template.ID = uuid.New().String()
	}
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	e.templates[template.ID] = template

	return e.saveTemplates()
}

// Send sends a notification.
func (e *Engine) Send(ctx context.Context, notification *Notification) error {
	// Generate ID if not provided
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}

	// Set timestamps
	notification.CreatedAt = time.Now()
	notification.Status = StatusPending

	// Get user preferences
	prefs := e.getPreferences(notification.UserID)

	// Check Do Not Disturb
	if e.isInDNDPeriod(prefs) && notification.Priority < PriorityUrgent {
		notification.Status = StatusQueued
		notification.ScheduledFor = e.getNextAvailableTime()
	}

	// Check quiet hours
	if e.isInQuietHours(prefs) && notification.Priority < PriorityHigh {
		notification.Status = StatusQueued
		notification.ScheduledFor = e.getNextAvailableTime()
	}

	// AI relevance scoring
	if e.config.AIEnabled && e.aiEngine != nil {
		notification.RelevanceScore = e.aiEngine.CalculateRelevance(notification, prefs)
	}

	// Apply channel preferences
	notification.Channels = e.filterChannelsByPreferences(notification.Channels, prefs, notification.Priority)

	// Check if should batch
	if prefs.BatchingEnabled && notification.Priority < PriorityHigh {
		e.batchProcessor.AddToBatch(notification)
		return nil
	}

	// Check if should group
	if e.config.GroupingEnabled && prefs.GroupingEnabled {
		if groupID := e.findOrCreateGroup(notification); groupID != "" {
			notification.GroupID = groupID
		}
	}

	// Add audit entry
	if e.config.AuditEnabled {
		notification.AuditTrail = append(notification.AuditTrail, AuditEntry{
			Timestamp: time.Now(),
			Action:    "created",
			UserID:    notification.UserID,
		})
	}

	// Store notification
	e.mu.Lock()
	e.notifications[notification.ID] = notification
	e.mu.Unlock()

	// Set status before queuing to avoid a race with the worker goroutine
	notification.Status = StatusQueued

	// Queue for delivery
	select {
	case e.queue <- notification:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("notification queue is full")
	}

	// Save
	go func() {
		if err := e.saveNotifications(); err != nil {
			fmt.Printf("Error saving notifications: %v\n", err)
		}
	}()

	return nil
}

// SendFromTemplate sends a notification from a template.
func (e *Engine) SendFromTemplate(ctx context.Context, templateID string, variables map[string]interface{}, userID string) error {
	e.mu.RLock()
	template, exists := e.templates[templateID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Render template
	notification := &Notification{
		Type:     template.Type,
		Title:    e.renderTemplate(template.Title, variables),
		Message:  e.renderTemplate(template.Message, variables),
		Priority: template.Priority,
		Channels: template.Channels,
		Actions:  template.Actions,
		UserID:   userID,
		Metadata: template.Metadata,
	}

	return e.Send(ctx, notification)
}

// Start starts the notification engine.
func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine is already running")
	}
	e.running = true
	e.mu.Unlock()

	// Start workers
	for i := 0; i < e.config.WorkerCount; i++ {
		go e.worker()
	}

	// Start batch processor
	if e.batchProcessor != nil {
		go e.batchProcessor.Start()
	}

	// Start escalation processor
	if e.escalationProcessor != nil {
		go e.escalationProcessor.Start()
	}

	// Start scheduler for queued notifications
	go e.scheduler()

	return nil
}

// Stop stops the notification engine.
func (e *Engine) Stop() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine is not running")
	}
	e.running = false
	e.mu.Unlock()

	// Signal stop
	close(e.stopChan)

	// Save all data
	if err := e.save(); err != nil {
		return fmt.Errorf("failed to save on stop: %w", err)
	}

	return nil
}

// worker processes notifications from the queue.
func (e *Engine) worker() {
	for {
		select {
		case notification := <-e.queue:
			e.processNotification(notification)
		case <-e.stopChan:
			return
		}
	}
}

// processNotification processes and delivers a notification.
func (e *Engine) processNotification(notification *Notification) {
	ctx := context.Background()

	notification.Status = StatusSending

	// Deliver to all channels
	results := make([]DeliveryResult, 0, len(notification.Channels))

	for _, channelType := range notification.Channels {
		result := e.deliverToChannel(ctx, notification, channelType)
		results = append(results, result)

		// Track analytics
		if e.analytics != nil {
			e.analytics.TrackDelivery(notification, result)
		}
	}

	// Update status based on results
	allFailed := true
	anySuccess := false

	for _, result := range results {
		if result.Success {
			anySuccess = true
			allFailed = false
		}
	}

	if anySuccess {
		notification.Status = StatusDelivered
		now := time.Now()
		notification.DeliveredAt = &now

		// Add audit entry
		if e.config.AuditEnabled {
			notification.AuditTrail = append(notification.AuditTrail, AuditEntry{
				Timestamp: time.Now(),
				Action:    "delivered",
				Metadata: map[string]interface{}{
					"channels": len(results),
					"results":  results,
				},
			})
		}
	} else if allFailed {
		notification.Status = StatusFailed
		notification.DeliveryAttempts++

		// Retry logic
		if notification.DeliveryAttempts < e.config.MaxRetries {
			time.AfterFunc(e.config.RetryDelay, func() {
				e.queue <- notification
			})
		}
	}

	// Update notification
	e.mu.Lock()
	e.notifications[notification.ID] = notification
	e.mu.Unlock()

	// Learn from delivery
	if e.config.AIEnabled && e.aiEngine != nil {
		go e.aiEngine.LearnFromDelivery(notification, results)
	}
}

// deliverToChannel delivers notification to a specific channel.
func (e *Engine) deliverToChannel(ctx context.Context, notification *Notification, channelType DeliveryChannel) DeliveryResult {
	e.mu.RLock()
	channel, exists := e.channels[channelType]
	e.mu.RUnlock()

	if !exists {
		return DeliveryResult{
			Channel: channelType,
			Success: false,
			Error:   "channel not registered",
		}
	}

	startTime := time.Now()
	err := channel.Send(ctx, notification)
	responseTime := time.Since(startTime).Milliseconds()

	return DeliveryResult{
		Channel: channelType,
		Success: err == nil,
		Error: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		DeliveredAt:  time.Now(),
		ResponseTime: float64(responseTime),
	}
}

// scheduler handles scheduled/queued notifications.
func (e *Engine) scheduler() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.processScheduled()
		case <-e.stopChan:
			return
		}
	}
}

// processScheduled processes scheduled notifications.
func (e *Engine) processScheduled() {
	e.mu.RLock()
	notifications := make([]*Notification, 0)
	for _, n := range e.notifications {
		if n.Status == StatusQueued && n.ScheduledFor != nil && time.Now().After(*n.ScheduledFor) {
			notifications = append(notifications, n)
		}
	}
	e.mu.RUnlock()

	for _, n := range notifications {
		e.queue <- n
	}
}

// Helper methods

func (e *Engine) getPreferences(userID string) *NotificationPreferences {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if prefs, exists := e.preferences[userID]; exists {
		return prefs
	}

	// Return defaults
	return &NotificationPreferences{
		UserID:          userID,
		EnabledChannels: []DeliveryChannel{ChannelEmail, ChannelInApp},
		MinimumPriority: PriorityNormal,
		GroupingEnabled: true,
		BatchingEnabled: false,
	}
}

func (e *Engine) isInDNDPeriod(prefs *NotificationPreferences) bool {
	if !prefs.DoNotDisturb {
		return false
	}

	now := time.Now()
	for _, schedule := range prefs.DNDSchedule {
		if now.After(schedule.Start) && now.Before(schedule.End) {
			return true
		}
	}

	return false
}

func (e *Engine) isInQuietHours(prefs *NotificationPreferences) bool {
	now := time.Now()
	currentDay := now.Weekday().String()[:3] // Mon, Tue, etc.

	for _, period := range prefs.QuietHours {
		// Check if today is in the days list
		inDays := false
		for _, day := range period.Days {
			if day == currentDay {
				inDays = true
				break
			}
		}

		if !inDays {
			continue
		}

		// Parse times
		// Simplified - should handle timezone properly
		startTime := period.Start
		endTime := period.End

		currentTime := now.Format("15:04")

		if currentTime >= startTime && currentTime <= endTime {
			return true
		}
	}

	return false
}

func (e *Engine) getNextAvailableTime() *time.Time {
	// Simplified implementation
	next := time.Now().Add(time.Hour)
	return &next
}

func (e *Engine) filterChannelsByPreferences(channels []DeliveryChannel, prefs *NotificationPreferences, priority Priority) []DeliveryChannel {
	if len(channels) == 0 {
		channels = prefs.EnabledChannels
	}

	filtered := make([]DeliveryChannel, 0)

	for _, channel := range channels {
		// Check if channel is enabled
		enabled := false
		for _, enabledChannel := range prefs.EnabledChannels {
			if channel == enabledChannel {
				enabled = true
				break
			}
		}

		if !enabled {
			continue
		}

		// Check channel-specific preferences
		if channelPref, exists := prefs.ChannelPreferences[channel]; exists {
			if !channelPref.Enabled {
				continue
			}
			if priority < channelPref.MinPriority {
				continue
			}
		}

		filtered = append(filtered, channel)
	}

	return filtered
}

func (e *Engine) findOrCreateGroup(notification *Notification) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Look for existing group
	for _, group := range e.groups {
		if group.Type != notification.Type || group.Category != notification.Category {
			continue
		}
		// Add to existing group
		group.Notifications = append(group.Notifications, notification)
		group.Count = len(group.Notifications)
		group.UpdatedAt = time.Now()
		if notification.Priority > group.Priority {
			group.Priority = notification.Priority
		}
		return group.ID
	}

	// Create new group
	groupID := uuid.New().String()
	group := &NotificationGroup{
		ID:            groupID,
		Type:          notification.Type,
		Category:      notification.Category,
		Title:         fmt.Sprintf("%s notifications", notification.Type),
		Count:         1,
		Priority:      notification.Priority,
		Notifications: []*Notification{notification},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	e.groups[groupID] = group
	return groupID
}

func (e *Engine) renderTemplate(template string, variables map[string]interface{}) string {
	// Simple template rendering - should use proper template engine
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		result = replaceString(result, placeholder, replacement)
	}
	return result
}

// replaceString replaces all occurrences of old with replacement in s.
func replaceString(s, old, replacement string) string {
	result := ""
	for {
		idx := findString(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + replacement
		s = s[idx+len(old):]
	}
	return result
}

// findString returns the index of the first occurrence of substr in s, or -1 if not found.
func findString(s, substr string) int {
	if substr == "" {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Data persistence

func (e *Engine) load() error {
	// Load templates
	if err := e.loadTemplates(); err != nil {
		return err
	}

	// Load preferences
	if err := e.loadPreferences(); err != nil {
		return err
	}

	// Load notifications
	if err := e.loadNotifications(); err != nil {
		return err
	}

	return nil
}

func (e *Engine) save() error {
	if err := e.saveTemplates(); err != nil {
		return err
	}

	if err := e.savePreferences(); err != nil {
		return err
	}

	if err := e.saveNotifications(); err != nil {
		return err
	}

	return nil
}

func (e *Engine) loadTemplates() error {
	path := filepath.Join(e.dataDir, "templates.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var templates []*NotificationTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return err
	}

	for _, template := range templates {
		e.templates[template.ID] = template
	}

	return nil
}

func (e *Engine) saveTemplates() error {
	templates := make([]*NotificationTemplate, 0, len(e.templates))
	for _, template := range e.templates {
		templates = append(templates, template)
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(e.dataDir, "templates.json")
	return os.WriteFile(path, data, 0o600)
}

func (e *Engine) loadPreferences() error {
	path := filepath.Join(e.dataDir, "preferences.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var prefs []*NotificationPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return err
	}

	for _, pref := range prefs {
		e.preferences[pref.UserID] = pref
	}

	return nil
}

func (e *Engine) savePreferences() error {
	prefs := make([]*NotificationPreferences, 0, len(e.preferences))
	for _, pref := range e.preferences {
		prefs = append(prefs, pref)
	}

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(e.dataDir, "preferences.json")
	return os.WriteFile(path, data, 0o600)
}

func (e *Engine) loadNotifications() error {
	path := filepath.Join(e.dataDir, "notifications.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var notifications []*Notification
	if err := json.Unmarshal(data, &notifications); err != nil {
		return err
	}

	for _, notification := range notifications {
		e.notifications[notification.ID] = notification
	}

	return nil
}

func (e *Engine) saveNotifications() error {
	// Only save recent notifications (retention policy)
	cutoff := time.Now().AddDate(0, 0, -e.config.RetentionDays)

	e.mu.RLock()
	notifications := make([]*Notification, 0, len(e.notifications))
	for _, notification := range e.notifications {
		if notification.CreatedAt.After(cutoff) {
			notifications = append(notifications, notification)
		}
	}
	e.mu.RUnlock()

	data, err := json.MarshalIndent(notifications, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(e.dataDir, "notifications.json")
	return os.WriteFile(path, data, 0o600)
}
