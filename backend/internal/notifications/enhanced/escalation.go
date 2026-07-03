package enhanced

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EscalationProcessor handles notification escalation logic.
type EscalationProcessor struct {
	mu              sync.RWMutex
	engine          *Engine
	rules           []*EscalationRule
	escalations     map[string]*EscalationState
	ticker          *time.Ticker
	stopChan        chan struct{}
	checkInterval   time.Duration
	escalationChain map[string][]string // userID -> [escalation targets]
}

// EscalationRule defines when and how to escalate a notification.
type EscalationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Priority    Priority               `json:"priority"`       // Apply to notifications with this priority
	Type        NotificationType       `json:"type,omitempty"` // Apply to specific types
	Category    string                 `json:"category,omitempty"`
	Levels      []EscalationLevel      `json:"levels"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// EscalationLevel represents a single escalation level.
type EscalationLevel struct {
	Level           int               `json:"level"`            // 1, 2, 3, etc.
	DelayAfter      time.Duration     `json:"delay_after"`      // Time to wait before this level
	Channels        []DeliveryChannel `json:"channels"`         // Channels to use
	Targets         []string          `json:"targets"`          // User IDs to escalate to
	Message         string            `json:"message"`          // Custom escalation message
	IncludeOriginal bool              `json:"include_original"` // Include original notification details
	StopOnRead      bool              `json:"stop_on_read"`     // Stop if original is read
}

// EscalationState tracks the escalation state of a notification.
type EscalationState struct {
	NotificationID  string                   `json:"notification_id"`
	OriginalUserID  string                   `json:"original_user_id"`
	CurrentLevel    int                      `json:"current_level"`
	LastEscalatedAt time.Time                `json:"last_escalated_at"`
	EscalationsSent int                      `json:"escalations_sent"`
	Status          string                   `json:"status"` // active, stopped, completed
	Rule            *EscalationRule          `json:"rule"`
	History         []EscalationHistoryEntry `json:"history"`
	Metadata        map[string]interface{}   `json:"metadata"`
}

// EscalationHistoryEntry represents a single escalation event.
type EscalationHistoryEntry struct {
	Level     int               `json:"level"`
	Timestamp time.Time         `json:"timestamp"`
	Targets   []string          `json:"targets"`
	Channels  []DeliveryChannel `json:"channels"`
	Success   bool              `json:"success"`
	Error     string            `json:"error,omitempty"`
}

// SLAConfig represents SLA configuration for notifications.
type SLAConfig struct {
	Priority       Priority      `json:"priority"`
	ResponseTime   time.Duration `json:"response_time"`   // Expected time to read
	ResolutionTime time.Duration `json:"resolution_time"` // Expected time to action
	EscalateAfter  time.Duration `json:"escalate_after"`
}

// NewEscalationProcessor creates a new escalation processor.
func NewEscalationProcessor(engine *Engine, checkInterval time.Duration) *EscalationProcessor {
	if checkInterval == 0 {
		checkInterval = 1 * time.Minute
	}

	ep := &EscalationProcessor{
		engine:          engine,
		rules:           make([]*EscalationRule, 0),
		escalations:     make(map[string]*EscalationState),
		checkInterval:   checkInterval,
		stopChan:        make(chan struct{}),
		escalationChain: make(map[string][]string),
	}

	// Add default escalation rules
	ep.addDefaultRules()

	return ep
}

// Start starts the escalation processor.
func (ep *EscalationProcessor) Start() {
	ep.ticker = time.NewTicker(ep.checkInterval)
	go ep.processEscalations()
}

// Stop stops the escalation processor.
func (ep *EscalationProcessor) Stop() {
	if ep.ticker != nil {
		ep.ticker.Stop()
	}
	close(ep.stopChan)
}

// TrackNotification starts tracking a notification for escalation.
func (ep *EscalationProcessor) TrackNotification(notification *Notification) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// Find applicable rule
	rule := ep.findApplicableRule(notification)
	if rule == nil || !rule.Enabled {
		return
	}

	// Don't track if already tracked
	if _, exists := ep.escalations[notification.ID]; exists {
		return
	}

	// Create escalation state
	state := &EscalationState{
		NotificationID:  notification.ID,
		OriginalUserID:  notification.UserID,
		CurrentLevel:    0,
		LastEscalatedAt: time.Now(),
		Status:          "active",
		Rule:            rule,
		History:         make([]EscalationHistoryEntry, 0),
		Metadata:        make(map[string]interface{}),
	}

	ep.escalations[notification.ID] = state
}

// StopEscalation stops escalation for a notification.
func (ep *EscalationProcessor) StopEscalation(notificationID string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if state, exists := ep.escalations[notificationID]; exists {
		state.Status = "stopped"
	}
}

// AddRule adds a custom escalation rule.
func (ep *EscalationProcessor) AddRule(rule *EscalationRule) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.rules = append(ep.rules, rule)
}

// SetEscalationChain sets the escalation chain for a user.
func (ep *EscalationProcessor) SetEscalationChain(userID string, targets []string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.escalationChain[userID] = targets
}

// processEscalations periodically checks and processes escalations.
func (ep *EscalationProcessor) processEscalations() {
	for {
		select {
		case <-ep.ticker.C:
			ep.checkAndEscalate()
		case <-ep.stopChan:
			return
		}
	}
}

// checkAndEscalate checks all tracked notifications and escalates if needed.
func (ep *EscalationProcessor) checkAndEscalate() {
	ep.mu.Lock()

	toEscalate := make([]*EscalationState, 0)
	now := time.Now()

	for _, state := range ep.escalations {
		if state.Status != "active" {
			continue
		}

		// Check if notification was read (would need to query engine)
		// For now, simplified check

		// Check if it's time to escalate to next level
		if state.CurrentLevel < len(state.Rule.Levels) {
			level := state.Rule.Levels[state.CurrentLevel]

			if now.Sub(state.LastEscalatedAt) >= level.DelayAfter {
				toEscalate = append(toEscalate, state)
			}
		} else {
			// Reached max escalation level
			state.Status = "completed"
		}
	}

	ep.mu.Unlock()

	// Perform escalations outside of lock
	for _, state := range toEscalate {
		ep.performEscalation(state)
	}
}

// performEscalation performs the actual escalation.
func (ep *EscalationProcessor) performEscalation(state *EscalationState) {
	ep.mu.Lock()
	level := state.Rule.Levels[state.CurrentLevel]
	ep.mu.Unlock()

	ctx := context.Background()

	// Determine escalation targets
	targets := level.Targets
	if len(targets) == 0 {
		// Use escalation chain
		if chain, exists := ep.escalationChain[state.OriginalUserID]; exists && state.CurrentLevel < len(chain) {
			targets = []string{chain[state.CurrentLevel]}
		}
	}

	if len(targets) == 0 {
		// No targets, skip
		return
	}

	// Create escalation notification for each target
	historyEntry := EscalationHistoryEntry{
		Level:     level.Level,
		Timestamp: time.Now(),
		Targets:   targets,
		Channels:  level.Channels,
		Success:   true,
	}

	for _, targetUserID := range targets {
		escalationNotif := ep.createEscalationNotification(state, level, targetUserID)

		if ep.engine != nil {
			if err := ep.engine.Send(ctx, escalationNotif); err != nil {
				historyEntry.Success = false
				historyEntry.Error = err.Error()
			}
		}
	}

	// Update state
	ep.mu.Lock()
	state.CurrentLevel++
	state.LastEscalatedAt = time.Now()
	state.EscalationsSent++
	state.History = append(state.History, historyEntry)
	ep.mu.Unlock()
}

// createEscalationNotification creates an escalation notification.
func (ep *EscalationProcessor) createEscalationNotification(state *EscalationState, level EscalationLevel, targetUserID string) *Notification {
	title := fmt.Sprintf("🔔 Escalation: Notification requires attention (Level %d)", level.Level)
	message := level.Message

	if message == "" {
		message = fmt.Sprintf("A %s notification from %s has not been read and requires immediate attention.",
			state.Rule.Priority, state.OriginalUserID)
	}

	notification := &Notification{
		ID:       fmt.Sprintf("%s_escalation_L%d_%s", state.NotificationID, level.Level, targetUserID),
		Type:     TypeAlertTriggered,
		Priority: PriorityHigh, // Escalations are always high priority
		Title:    title,
		Message:  message,
		Category: "escalation",
		Tags:     []string{"escalation", fmt.Sprintf("level_%d", level.Level)},
		Metadata: map[string]interface{}{
			"original_notification_id": state.NotificationID,
			"original_user_id":         state.OriginalUserID,
			"escalation_level":         level.Level,
			"escalation_rule_id":       state.Rule.ID,
			"escalations_sent":         state.EscalationsSent,
		},
		Channels:  level.Channels,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UserID:    targetUserID,
	}

	// Add actions
	notification.Actions = []NotificationAction{
		{
			ID:     "view_original",
			Label:  "View Original",
			Type:   "button",
			Action: "view",
			URL:    fmt.Sprintf("/notifications/%s", state.NotificationID),
			Style:  "primary",
		},
		{
			ID:     "acknowledge",
			Label:  "Acknowledge",
			Type:   "button",
			Action: "acknowledge",
			Style:  "secondary",
			Metadata: map[string]interface{}{
				"escalation_id": state.NotificationID,
			},
		},
	}

	return notification
}

// findApplicableRule finds the first matching escalation rule.
func (ep *EscalationProcessor) findApplicableRule(notification *Notification) *EscalationRule {
	for _, rule := range ep.rules {
		if !rule.Enabled {
			continue
		}

		// Check priority
		if rule.Priority != notification.Priority {
			continue
		}

		// Check type if specified
		if rule.Type != "" && rule.Type != notification.Type {
			continue
		}

		// Check category if specified
		if rule.Category != "" && rule.Category != notification.Category {
			continue
		}

		return rule
	}

	return nil
}

// addDefaultRules adds default escalation rules.
func (ep *EscalationProcessor) addDefaultRules() {
	// Urgent notifications - escalate quickly
	ep.rules = append(ep.rules, &EscalationRule{
		ID:          "urgent_default",
		Name:        "Urgent Notification Escalation",
		Description: "Escalate urgent notifications if not read within 5 minutes",
		Priority:    PriorityUrgent,
		Enabled:     true,
		Levels: []EscalationLevel{
			{
				Level:      1,
				DelayAfter: 5 * time.Minute,
				Channels:   []DeliveryChannel{ChannelEmail, ChannelSMS},
				Message:    "URGENT: A critical notification requires immediate attention.",
				StopOnRead: true,
			},
			{
				Level:      2,
				DelayAfter: 10 * time.Minute,
				Channels:   []DeliveryChannel{ChannelSMS, ChannelPush},
				Message:    "CRITICAL: Escalating to Level 2 - notification still unread.",
				StopOnRead: true,
			},
			{
				Level:      3,
				DelayAfter: 15 * time.Minute,
				Channels:   []DeliveryChannel{ChannelSMS, ChannelPush, ChannelSlack},
				Message:    "CRITICAL ALERT: Level 3 escalation - requires immediate action.",
				StopOnRead: false,
			},
		},
	})

	// High priority notifications
	ep.rules = append(ep.rules, &EscalationRule{
		ID:          "high_default",
		Name:        "High Priority Escalation",
		Description: "Escalate high priority notifications if not read within 30 minutes",
		Priority:    PriorityHigh,
		Enabled:     true,
		Levels: []EscalationLevel{
			{
				Level:      1,
				DelayAfter: 30 * time.Minute,
				Channels:   []DeliveryChannel{ChannelEmail},
				Message:    "A high-priority notification requires your attention.",
				StopOnRead: true,
			},
			{
				Level:      2,
				DelayAfter: 1 * time.Hour,
				Channels:   []DeliveryChannel{ChannelEmail, ChannelPush},
				Message:    "Escalating: High-priority notification still pending.",
				StopOnRead: true,
			},
		},
	})

	// Backup failure notifications - special handling
	ep.rules = append(ep.rules, &EscalationRule{
		ID:          "backup_failed",
		Name:        "Backup Failure Escalation",
		Description: "Escalate backup failures immediately",
		Priority:    PriorityHigh,
		Type:        TypeBackupFailed,
		Enabled:     true,
		Levels: []EscalationLevel{
			{
				Level:      1,
				DelayAfter: 15 * time.Minute,
				Channels:   []DeliveryChannel{ChannelEmail, ChannelSlack},
				Message:    "BACKUP FAILED: Immediate attention required to prevent data loss.",
				StopOnRead: true,
			},
			{
				Level:      2,
				DelayAfter: 30 * time.Minute,
				Channels:   []DeliveryChannel{ChannelSMS, ChannelSlack},
				Message:    "CRITICAL: Backup failure escalation - data protection at risk.",
				StopOnRead: false,
			},
		},
	})
}

// GetEscalationState returns the escalation state for a notification.
func (ep *EscalationProcessor) GetEscalationState(notificationID string) *EscalationState {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	if state, exists := ep.escalations[notificationID]; exists {
		return state
	}
	return nil
}

// GetActiveEscalations returns all active escalations.
func (ep *EscalationProcessor) GetActiveEscalations() []*EscalationState {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	active := make([]*EscalationState, 0)
	for _, state := range ep.escalations {
		if state.Status == "active" {
			active = append(active, state)
		}
	}
	return active
}

// SLATracker tracks SLA compliance for notifications.
type SLATracker struct {
	mu         sync.RWMutex
	configs    map[Priority]SLAConfig
	violations map[string]*SLAViolation
}

// SLAViolation represents an SLA violation.
type SLAViolation struct {
	NotificationID string        `json:"notification_id"`
	Priority       Priority      `json:"priority"`
	ViolationType  string        `json:"violation_type"` // response, resolution
	ExpectedTime   time.Duration `json:"expected_time"`
	ActualTime     time.Duration `json:"actual_time"`
	Timestamp      time.Time     `json:"timestamp"`
}

// NewSLATracker creates a new SLA tracker.
func NewSLATracker() *SLATracker {
	st := &SLATracker{
		configs:    make(map[Priority]SLAConfig),
		violations: make(map[string]*SLAViolation),
	}

	// Add default SLA configs
	st.configs[PriorityUrgent] = SLAConfig{
		Priority:       PriorityUrgent,
		ResponseTime:   5 * time.Minute,
		ResolutionTime: 15 * time.Minute,
		EscalateAfter:  5 * time.Minute,
	}
	st.configs[PriorityHigh] = SLAConfig{
		Priority:       PriorityHigh,
		ResponseTime:   30 * time.Minute,
		ResolutionTime: 2 * time.Hour,
		EscalateAfter:  30 * time.Minute,
	}
	st.configs[PriorityNormal] = SLAConfig{
		Priority:       PriorityNormal,
		ResponseTime:   4 * time.Hour,
		ResolutionTime: 24 * time.Hour,
		EscalateAfter:  8 * time.Hour,
	}
	st.configs[PriorityLow] = SLAConfig{
		Priority:       PriorityLow,
		ResponseTime:   24 * time.Hour,
		ResolutionTime: 7 * 24 * time.Hour,
		EscalateAfter:  48 * time.Hour,
	}

	return st
}

// CheckSLA checks if a notification violates SLA.
func (st *SLATracker) CheckSLA(notification *Notification, actualResponseTime time.Duration) *SLAViolation {
	st.mu.RLock()
	defer st.mu.RUnlock()

	config, exists := st.configs[notification.Priority]
	if !exists {
		return nil
	}

	if actualResponseTime > config.ResponseTime {
		violation := &SLAViolation{
			NotificationID: notification.ID,
			Priority:       notification.Priority,
			ViolationType:  "response",
			ExpectedTime:   config.ResponseTime,
			ActualTime:     actualResponseTime,
			Timestamp:      time.Now(),
		}

		st.violations[notification.ID] = violation
		return violation
	}

	return nil
}

// GetViolations returns all SLA violations.
func (st *SLATracker) GetViolations() []*SLAViolation {
	st.mu.RLock()
	defer st.mu.RUnlock()

	violations := make([]*SLAViolation, 0, len(st.violations))
	for _, v := range st.violations {
		violations = append(violations, v)
	}
	return violations
}
