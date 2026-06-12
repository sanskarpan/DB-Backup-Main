package enhanced

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchProcessor handles intelligent notification batching and digests
type BatchProcessor struct {
	mu               sync.RWMutex
	engine           *Engine
	batchInterval    time.Duration
	batches          map[string]*Batch
	ticker           *time.Ticker
	stopChan         chan struct{}
	minBatchSize     int
	maxBatchSize     int
	maxBatchAge      time.Duration
	groupingStrategy GroupingStrategy
}

// Batch represents a batch of notifications
type Batch struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	Type          NotificationType       `json:"type,omitempty"`
	Category      string                 `json:"category,omitempty"`
	Priority      Priority               `json:"priority"`
	Notifications []*Notification        `json:"notifications"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ScheduledFor  time.Time              `json:"scheduled_for"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// GroupingStrategy defines how notifications should be grouped
type GroupingStrategy string

const (
	GroupByType     GroupingStrategy = "type"      // Group by notification type
	GroupByCategory GroupingStrategy = "category"  // Group by category
	GroupByPriority GroupingStrategy = "priority"  // Group by priority
	GroupBySmart    GroupingStrategy = "smart"     // AI-based smart grouping
	GroupByNone     GroupingStrategy = "none"      // No grouping
)

// BatchConfig represents batch processor configuration
type BatchConfig struct {
	Interval         time.Duration
	MinBatchSize     int
	MaxBatchSize     int
	MaxBatchAge      time.Duration
	GroupingStrategy GroupingStrategy
	EnableDigest     bool
	DigestSchedule   string // cron format
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(engine *Engine, config BatchConfig) *BatchProcessor {
	if config.Interval == 0 {
		config.Interval = 5 * time.Minute
	}
	if config.MinBatchSize == 0 {
		config.MinBatchSize = 3
	}
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 50
	}
	if config.MaxBatchAge == 0 {
		config.MaxBatchAge = 1 * time.Hour
	}
	if config.GroupingStrategy == "" {
		config.GroupingStrategy = GroupBySmart
	}

	return &BatchProcessor{
		engine:           engine,
		batchInterval:    config.Interval,
		batches:          make(map[string]*Batch),
		minBatchSize:     config.MinBatchSize,
		maxBatchSize:     config.MaxBatchSize,
		maxBatchAge:      config.MaxBatchAge,
		groupingStrategy: config.GroupingStrategy,
		stopChan:         make(chan struct{}),
	}
}

// Start starts the batch processor
func (bp *BatchProcessor) Start() {
	bp.ticker = time.NewTicker(bp.batchInterval)
	go bp.processBatches()
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() {
	if bp.ticker != nil {
		bp.ticker.Stop()
	}
	close(bp.stopChan)
}

// AddToBatch adds a notification to appropriate batch
func (bp *BatchProcessor) AddToBatch(notification *Notification) bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Don't batch urgent notifications
	if notification.Priority == PriorityUrgent {
		return false
	}

	// Check user preferences
	if bp.engine != nil {
		prefs := bp.engine.getPreferences(notification.UserID)
		if !prefs.BatchingEnabled {
			return false
		}
	}

	// Find or create appropriate batch
	batchKey := bp.getBatchKey(notification)
	batch, exists := bp.batches[batchKey]

	if !exists {
		batch = &Batch{
			ID:            fmt.Sprintf("batch_%s_%d", notification.UserID, time.Now().Unix()),
			UserID:        notification.UserID,
			Type:          notification.Type,
			Category:      notification.Category,
			Priority:      notification.Priority,
			Notifications: make([]*Notification, 0),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			ScheduledFor:  time.Now().Add(bp.batchInterval),
			Metadata:      make(map[string]interface{}),
		}
		bp.batches[batchKey] = batch
	}

	// Add notification to batch
	batch.Notifications = append(batch.Notifications, notification)
	batch.UpdatedAt = time.Now()

	// Update priority to highest in batch
	if notification.Priority > batch.Priority {
		batch.Priority = notification.Priority
	}

	// Check if batch should be sent immediately
	if len(batch.Notifications) >= bp.maxBatchSize {
		go bp.sendBatch(batch)
		delete(bp.batches, batchKey)
		return true
	}

	return true
}

// processBatches periodically processes batches
func (bp *BatchProcessor) processBatches() {
	for {
		select {
		case <-bp.ticker.C:
			bp.checkAndSendBatches()
		case <-bp.stopChan:
			return
		}
	}
}

// checkAndSendBatches checks batches and sends ready ones
func (bp *BatchProcessor) checkAndSendBatches() {
	bp.mu.Lock()
	toSend := make([]*Batch, 0)
	toDelete := make([]string, 0)

	now := time.Now()

	for key, batch := range bp.batches {
		// Send if:
		// 1. Reached scheduled time
		// 2. Reached min batch size and max age
		// 3. Reached max batch size (already handled in AddToBatch)

		shouldSend := false

		if now.After(batch.ScheduledFor) {
			shouldSend = true
		} else if len(batch.Notifications) >= bp.minBatchSize && now.Sub(batch.CreatedAt) >= bp.maxBatchAge {
			shouldSend = true
		}

		if shouldSend {
			toSend = append(toSend, batch)
			toDelete = append(toDelete, key)
		}
	}

	// Delete batches that will be sent
	for _, key := range toDelete {
		delete(bp.batches, key)
	}

	bp.mu.Unlock()

	// Send batches outside of lock
	for _, batch := range toSend {
		go bp.sendBatch(batch)
	}
}

// sendBatch sends a batch as a grouped notification
func (bp *BatchProcessor) sendBatch(batch *Batch) {
	if len(batch.Notifications) == 0 {
		return
	}

	ctx := context.Background()

	// Create a grouped notification
	grouped := bp.createGroupedNotification(batch)

	// Send through engine
	if bp.engine != nil {
		if err := bp.engine.Send(ctx, grouped); err != nil {
			// Log error - in production would use proper logging
			fmt.Printf("Error sending batch %s: %v\n", batch.ID, err)
		}
	}
}

// createGroupedNotification creates a single notification from a batch
func (bp *BatchProcessor) createGroupedNotification(batch *Batch) *Notification {
	firstNotif := batch.Notifications[0]

	// Create summary
	summary := bp.generateBatchSummary(batch)

	// Determine channels from user preferences
	channels := firstNotif.Channels
	if bp.engine != nil {
		prefs := bp.engine.getPreferences(batch.UserID)
		channels = prefs.EnabledChannels
	}

	notification := &Notification{
		ID:       batch.ID,
		Type:     TypeCustom,
		Priority: batch.Priority,
		Title:    summary.Title,
		Message:  summary.Message,
		Category: batch.Category,
		Tags:     []string{"batch", "digest"},
		Metadata: map[string]interface{}{
			"batch_id":          batch.ID,
			"notification_count": len(batch.Notifications),
			"notification_ids":   bp.getNotificationIDs(batch),
			"types":             bp.getUniqueTypes(batch),
			"time_range": map[string]interface{}{
				"start": batch.CreatedAt,
				"end":   batch.UpdatedAt,
			},
		},
		Channels:  channels,
		Status:    StatusPending,
		Actions:   bp.generateBatchActions(batch),
		CreatedAt: time.Now(),
		UserID:    batch.UserID,
		GroupID:   batch.ID,
	}

	// Add chart data for visualization
	notification.ChartData = bp.generateBatchChartData(batch)

	return notification
}

// generateBatchSummary generates a summary for the batch
func (bp *BatchProcessor) generateBatchSummary(batch *Batch) struct {
	Title   string
	Message string
} {
	count := len(batch.Notifications)

	var title, message string

	if batch.Type != "" {
		// Type-specific summary
		title = fmt.Sprintf("%d %s notifications", count, batch.Type)
		message = fmt.Sprintf("You have %d %s notifications from the last %s",
			count, batch.Type, bp.formatDuration(time.Since(batch.CreatedAt)))
	} else if batch.Category != "" {
		// Category-specific summary
		title = fmt.Sprintf("%d %s updates", count, batch.Category)
		message = fmt.Sprintf("You have %d updates in %s category",
			count, batch.Category)
	} else {
		// Generic summary
		title = fmt.Sprintf("%d notifications", count)
		message = fmt.Sprintf("You have %d notifications waiting for you", count)
	}

	// Add breakdown by type
	typeCounts := bp.getTypeCounts(batch)
	if len(typeCounts) > 1 {
		message += "\n\nBreakdown:\n"
		for notifType, typeCount := range typeCounts {
			message += fmt.Sprintf("• %d %s\n", typeCount, notifType)
		}
	}

	return struct {
		Title   string
		Message string
	}{title, message}
}

// generateBatchActions generates action buttons for batch
func (bp *BatchProcessor) generateBatchActions(batch *Batch) []NotificationAction {
	return []NotificationAction{
		{
			ID:     "view_all",
			Label:  "View All",
			Type:   "button",
			Action: "view",
			URL:    fmt.Sprintf("/notifications?batch=%s", batch.ID),
			Style:  "primary",
		},
		{
			ID:     "mark_read",
			Label:  "Mark All Read",
			Type:   "button",
			Action: "mark_read",
			Style:  "secondary",
			Metadata: map[string]interface{}{
				"batch_id": batch.ID,
			},
		},
		{
			ID:     "dismiss_all",
			Label:  "Dismiss All",
			Type:   "button",
			Action: "dismiss",
			Style:  "secondary",
		},
	}
}

// generateBatchChartData generates chart data for visualization
func (bp *BatchProcessor) generateBatchChartData(batch *Batch) map[string]interface{} {
	typeCounts := bp.getTypeCounts(batch)
	priorityCounts := bp.getPriorityCounts(batch)

	return map[string]interface{}{
		"type": "batch_summary",
		"data": map[string]interface{}{
			"by_type": typeCounts,
			"by_priority": priorityCounts,
			"timeline":    bp.getTimeline(batch),
		},
	}
}

// getBatchKey generates a unique key for batching
func (bp *BatchProcessor) getBatchKey(notification *Notification) string {
	switch bp.groupingStrategy {
	case GroupByType:
		return fmt.Sprintf("%s_type_%s", notification.UserID, notification.Type)
	case GroupByCategory:
		return fmt.Sprintf("%s_cat_%s", notification.UserID, notification.Category)
	case GroupByPriority:
		return fmt.Sprintf("%s_pri_%d", notification.UserID, notification.Priority)
	case GroupBySmart:
		// Smart grouping based on AI patterns
		if bp.engine != nil && bp.engine.aiEngine != nil {
			// Group similar types together
			return fmt.Sprintf("%s_smart_%s_%s", notification.UserID, notification.Category, notification.Type)
		}
		fallthrough
	default:
		return fmt.Sprintf("%s_default", notification.UserID)
	}
}

// Helper methods

func (bp *BatchProcessor) getNotificationIDs(batch *Batch) []string {
	ids := make([]string, len(batch.Notifications))
	for i, n := range batch.Notifications {
		ids[i] = n.ID
	}
	return ids
}

func (bp *BatchProcessor) getUniqueTypes(batch *Batch) []NotificationType {
	typeMap := make(map[NotificationType]bool)
	for _, n := range batch.Notifications {
		typeMap[n.Type] = true
	}

	types := make([]NotificationType, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}
	return types
}

func (bp *BatchProcessor) getTypeCounts(batch *Batch) map[NotificationType]int {
	counts := make(map[NotificationType]int)
	for _, n := range batch.Notifications {
		counts[n.Type]++
	}
	return counts
}

func (bp *BatchProcessor) getPriorityCounts(batch *Batch) map[Priority]int {
	counts := make(map[Priority]int)
	for _, n := range batch.Notifications {
		counts[n.Priority]++
	}
	return counts
}

func (bp *BatchProcessor) getTimeline(batch *Batch) []map[string]interface{} {
	timeline := make([]map[string]interface{}, 0)

	for _, n := range batch.Notifications {
		timeline = append(timeline, map[string]interface{}{
			"id":        n.ID,
			"type":      n.Type,
			"timestamp": n.CreatedAt,
			"priority":  n.Priority,
		})
	}

	return timeline
}

func (bp *BatchProcessor) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}

// DigestGenerator generates periodic notification digests
type DigestGenerator struct {
	mu           sync.RWMutex
	engine       *Engine
	frequency    string // daily, weekly, monthly
	scheduledFor time.Time
	ticker       *time.Ticker
	stopChan     chan struct{}
}

// DigestConfig represents digest configuration
type DigestConfig struct {
	Frequency    string // daily, weekly, monthly
	TimeOfDay    string // HH:MM format
	DayOfWeek    string // Mon, Tue, etc. (for weekly)
	DayOfMonth   int    // 1-31 (for monthly)
	IncludeStats bool
	IncludeChart bool
}

// NewDigestGenerator creates a new digest generator
func NewDigestGenerator(engine *Engine, config DigestConfig) *DigestGenerator {
	return &DigestGenerator{
		engine:    engine,
		frequency: config.Frequency,
		stopChan:  make(chan struct{}),
	}
}

// Start starts the digest generator
func (dg *DigestGenerator) Start() {
	// Calculate next digest time
	interval := dg.getInterval()
	dg.ticker = time.NewTicker(interval)

	go dg.generateDigests()
}

// Stop stops the digest generator
func (dg *DigestGenerator) Stop() {
	if dg.ticker != nil {
		dg.ticker.Stop()
	}
	close(dg.stopChan)
}

// generateDigests periodically generates digests
func (dg *DigestGenerator) generateDigests() {
	for {
		select {
		case <-dg.ticker.C:
			dg.generateAndSendDigests()
		case <-dg.stopChan:
			return
		}
	}
}

// generateAndSendDigests generates and sends digests for all users
func (dg *DigestGenerator) generateAndSendDigests() {
	// In a real implementation, would iterate through all users
	// For now, this is a placeholder

	// Get users with digest enabled
	// For each user:
	//   - Get notifications from last period
	//   - Generate digest notification
	//   - Send digest
}

// getInterval returns the ticker interval based on frequency
func (dg *DigestGenerator) getInterval() time.Duration {
	switch dg.frequency {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
