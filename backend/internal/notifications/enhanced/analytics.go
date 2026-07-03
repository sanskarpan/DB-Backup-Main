package enhanced

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AnalyticsTracker tracks notification metrics and analytics.
type AnalyticsTracker struct {
	mu            sync.RWMutex
	dataFile      string
	metrics       *NotificationStats
	events        []AnalyticsEvent
	maxEvents     int
	aggregations  map[string]*TimeSeriesData
	realtimeStats *RealtimeStats
}

// AnalyticsEvent represents a single analytics event.
type AnalyticsEvent struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	EventType      string                 `json:"event_type"` // sent, delivered, failed, read, dismissed, action
	UserID         string                 `json:"user_id"`
	NotificationID string                 `json:"notification_id"`
	Type           NotificationType       `json:"type"`
	Priority       Priority               `json:"priority"`
	Channel        DeliveryChannel        `json:"channel,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Duration       float64                `json:"duration_ms,omitempty"`
	Success        bool                   `json:"success"`
	Error          string                 `json:"error,omitempty"`
}

// TimeSeriesData represents time-series metrics.
type TimeSeriesData struct {
	Interval   string                `json:"interval"` // hourly, daily, weekly, monthly
	DataPoints []TimeSeriesDataPoint `json:"data_points"`
	Aggregates map[string]float64    `json:"aggregates"`
}

// TimeSeriesDataPoint represents a single data point in time series.
type TimeSeriesDataPoint struct {
	Timestamp    time.Time              `json:"timestamp"`
	Count        int                    `json:"count"`
	DeliveryRate float64                `json:"delivery_rate"`
	ReadRate     float64                `json:"read_rate"`
	AvgTime      float64                `json:"avg_time_ms"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RealtimeStats represents real-time statistics.
type RealtimeStats struct {
	LastUpdated         time.Time                         `json:"last_updated"`
	ActiveNotifications int                               `json:"active_notifications"`
	QueueSize           int                               `json:"queue_size"`
	ProcessingRate      float64                           `json:"processing_rate"` // notifications/sec
	AvgLatency          float64                           `json:"avg_latency_ms"`
	ErrorRate           float64                           `json:"error_rate"`
	ChannelHealth       map[DeliveryChannel]ChannelHealth `json:"channel_health"`
}

// ChannelHealth represents health status of a delivery channel.
type ChannelHealth struct {
	Status      string    `json:"status"` // healthy, degraded, down
	SuccessRate float64   `json:"success_rate"`
	AvgLatency  float64   `json:"avg_latency_ms"`
	ErrorCount  int       `json:"error_count"`
	LastError   string    `json:"last_error,omitempty"`
	LastSuccess time.Time `json:"last_success"`
}

// UserEngagementMetrics represents user engagement analytics.
type UserEngagementMetrics struct {
	UserID                 string                   `json:"user_id"`
	TotalNotifications     int                      `json:"total_notifications"`
	ReadNotifications      int                      `json:"read_notifications"`
	ActionedNotifications  int                      `json:"actioned_notifications"`
	DismissedNotifications int                      `json:"dismissed_notifications"`
	ReadRate               float64                  `json:"read_rate"`
	ActionRate             float64                  `json:"action_rate"`
	DismissRate            float64                  `json:"dismiss_rate"`
	AvgTimeToRead          float64                  `json:"avg_time_to_read_seconds"`
	AvgTimeToAction        float64                  `json:"avg_time_to_action_seconds"`
	PreferredChannels      map[DeliveryChannel]int  `json:"preferred_channels"`
	PreferredTypes         map[NotificationType]int `json:"preferred_types"`
	MostActiveHours        []int                    `json:"most_active_hours"`
	EngagementScore        float64                  `json:"engagement_score"` // 0-100
}

// CohortAnalysis represents cohort-based analytics.
type CohortAnalysis struct {
	CohortID           string                 `json:"cohort_id"`
	Name               string                 `json:"name"`
	UserCount          int                    `json:"user_count"`
	AvgEngagementScore float64                `json:"avg_engagement_score"`
	TopChannels        []DeliveryChannel      `json:"top_channels"`
	TopTypes           []NotificationType     `json:"top_types"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// NewAnalyticsTracker creates a new analytics tracker.
func NewAnalyticsTracker(dataFile string, maxEvents int) *AnalyticsTracker {
	return &AnalyticsTracker{
		dataFile:     dataFile,
		maxEvents:    maxEvents,
		events:       make([]AnalyticsEvent, 0),
		aggregations: make(map[string]*TimeSeriesData),
		metrics: &NotificationStats{
			ByChannel:  make(map[DeliveryChannel]ChannelStats),
			ByType:     make(map[NotificationType]TypeStats),
			ByPriority: make(map[Priority]PriorityStats),
		},
		realtimeStats: &RealtimeStats{
			ChannelHealth: make(map[DeliveryChannel]ChannelHealth),
		},
	}
}

// Load loads analytics data from file.
func (a *AnalyticsTracker) Load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := os.ReadFile(a.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type state struct {
		Metrics      *NotificationStats         `json:"metrics"`
		Events       []AnalyticsEvent           `json:"events"`
		Aggregations map[string]*TimeSeriesData `json:"aggregations"`
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	a.metrics = s.Metrics
	a.events = s.Events
	a.aggregations = s.Aggregations

	return nil
}

// Save saves analytics data to file.
func (a *AnalyticsTracker) Save() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	type state struct {
		Metrics      *NotificationStats         `json:"metrics"`
		Events       []AnalyticsEvent           `json:"events"`
		Aggregations map[string]*TimeSeriesData `json:"aggregations"`
	}

	s := state{
		Metrics:      a.metrics,
		Events:       a.events,
		Aggregations: a.aggregations,
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.dataFile, data, 0o600)
}

// TrackEvent tracks a single analytics event.
func (a *AnalyticsTracker) TrackEvent(event AnalyticsEvent) { //nolint:gocritic // hugeParam: exported API uses value semantics for AnalyticsEvent
	a.mu.Lock()
	defer a.mu.Unlock()

	// Only set timestamp if not already set (allow historical events)
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	a.events = append(a.events, event)

	// Trim events if exceeds max
	if len(a.events) > a.maxEvents {
		a.events = a.events[len(a.events)-a.maxEvents:]
	}

	// Update metrics based on event type
	a.updateMetrics(&event)

	// Update realtime stats
	a.updateRealtimeStats(&event)

	// Auto-save periodically
	if len(a.events)%100 == 0 {
		go func() {
			if err := a.Save(); err != nil {
				fmt.Printf("Error saving analytics: %v\n", err)
			}
		}()
	}
}

// TrackNotificationSent tracks when a notification is sent.
func (a *AnalyticsTracker) TrackNotificationSent(notification *Notification) {
	a.TrackEvent(AnalyticsEvent{
		ID:             notification.ID + "_sent",
		EventType:      "sent",
		UserID:         notification.UserID,
		NotificationID: notification.ID,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Success:        true,
	})
}

// TrackDelivery tracks notification delivery to a channel.
func (a *AnalyticsTracker) TrackDelivery(notification *Notification, result DeliveryResult) { //nolint:gocritic // hugeParam: exported API uses value semantics for DeliveryResult
	a.TrackEvent(AnalyticsEvent{
		ID:             notification.ID + "_delivered_" + string(result.Channel),
		EventType:      string(StatusDelivered),
		UserID:         notification.UserID,
		NotificationID: notification.ID,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Channel:        result.Channel,
		Duration:       result.ResponseTime,
		Success:        result.Success,
		Error:          result.Error,
		Metadata: map[string]interface{}{
			"message_id": result.MessageID,
		},
	})
}

// TrackRead tracks when a notification is read.
func (a *AnalyticsTracker) TrackRead(notification *Notification, timeToRead time.Duration) {
	a.TrackEvent(AnalyticsEvent{
		ID:             notification.ID + "_read",
		EventType:      string(StatusRead),
		UserID:         notification.UserID,
		NotificationID: notification.ID,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Duration:       timeToRead.Seconds() * 1000,
		Success:        true,
	})
}

// TrackAction tracks when user takes action on notification.
func (a *AnalyticsTracker) TrackAction(notification *Notification, actionID string, timeToAction time.Duration) {
	a.TrackEvent(AnalyticsEvent{
		ID:             notification.ID + "_action_" + actionID,
		EventType:      "action",
		UserID:         notification.UserID,
		NotificationID: notification.ID,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Duration:       timeToAction.Seconds() * 1000,
		Success:        true,
		Metadata: map[string]interface{}{
			"action_id": actionID,
		},
	})
}

// TrackDismiss tracks when notification is dismissed.
func (a *AnalyticsTracker) TrackDismiss(notification *Notification) {
	a.TrackEvent(AnalyticsEvent{
		ID:             notification.ID + "_dismissed",
		EventType:      "dismissed",
		UserID:         notification.UserID,
		NotificationID: notification.ID,
		Type:           notification.Type,
		Priority:       notification.Priority,
		Success:        true,
	})
}

// GetMetrics returns overall notification statistics.
func (a *AnalyticsTracker) GetMetrics() *NotificationStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a deep copy to avoid concurrent modification
	return a.copyMetrics(a.metrics)
}

// GetRealtimeStats returns real-time statistics.
func (a *AnalyticsTracker) GetRealtimeStats() *RealtimeStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy
	stats := *a.realtimeStats
	stats.ChannelHealth = make(map[DeliveryChannel]ChannelHealth)
	for k, v := range a.realtimeStats.ChannelHealth {
		stats.ChannelHealth[k] = v
	}

	return &stats
}

// GetUserEngagement returns engagement metrics for a user.
func (a *AnalyticsTracker) GetUserEngagement(userID string) *UserEngagementMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	metrics := &UserEngagementMetrics{
		UserID:            userID,
		PreferredChannels: make(map[DeliveryChannel]int),
		PreferredTypes:    make(map[NotificationType]int),
		MostActiveHours:   make([]int, 0),
	}

	var totalTimeToRead, totalTimeToAction float64
	var readCount, actionCount int
	hourActivity := make(map[int]int)

	for i := range a.events {
		event := &a.events[i]
		if event.UserID != userID {
			continue
		}

		switch event.EventType {
		case "sent":
			metrics.TotalNotifications++
			if event.Channel != "" {
				metrics.PreferredChannels[event.Channel]++
			}
			metrics.PreferredTypes[event.Type]++
			hourActivity[event.Timestamp.Hour()]++

		case string(StatusRead):
			metrics.ReadNotifications++
			totalTimeToRead += event.Duration / 1000 // Convert to seconds
			readCount++

		case "action":
			metrics.ActionedNotifications++
			totalTimeToAction += event.Duration / 1000
			actionCount++

		case "dismissed":
			metrics.DismissedNotifications++
		}
	}

	// Calculate rates
	if metrics.TotalNotifications > 0 {
		metrics.ReadRate = float64(metrics.ReadNotifications) / float64(metrics.TotalNotifications)
		metrics.ActionRate = float64(metrics.ActionedNotifications) / float64(metrics.TotalNotifications)
		metrics.DismissRate = float64(metrics.DismissedNotifications) / float64(metrics.TotalNotifications)
	}

	// Calculate averages
	if readCount > 0 {
		metrics.AvgTimeToRead = totalTimeToRead / float64(readCount)
	}
	if actionCount > 0 {
		metrics.AvgTimeToAction = totalTimeToAction / float64(actionCount)
	}

	// Find most active hours (top 5)
	type hourCount struct {
		hour  int
		count int
	}
	hours := make([]hourCount, 0)
	for hour, count := range hourActivity {
		hours = append(hours, hourCount{hour, count})
	}
	// Simple sort by count (descending)
	for i := 0; i < len(hours); i++ {
		for j := i + 1; j < len(hours); j++ {
			if hours[j].count > hours[i].count {
				hours[i], hours[j] = hours[j], hours[i]
			}
		}
	}
	for i := 0; i < len(hours) && i < 5; i++ {
		metrics.MostActiveHours = append(metrics.MostActiveHours, hours[i].hour)
	}

	// Calculate engagement score (0-100)
	metrics.EngagementScore = a.calculateEngagementScore(metrics)

	return metrics
}

// GetTimeSeries returns time-series data for a given interval.
func (a *AnalyticsTracker) GetTimeSeries(interval string, startTime, endTime time.Time) *TimeSeriesData {
	a.mu.RLock()
	defer a.mu.RUnlock()

	data := &TimeSeriesData{
		Interval:   interval,
		DataPoints: make([]TimeSeriesDataPoint, 0),
		Aggregates: make(map[string]float64),
	}

	// Group events by interval
	buckets := make(map[time.Time]*TimeSeriesDataPoint)

	for i := range a.events {
		event := &a.events[i]
		if event.Timestamp.Before(startTime) || event.Timestamp.After(endTime) {
			continue
		}

		bucketTime := a.getBucketTime(event.Timestamp, interval)
		if _, exists := buckets[bucketTime]; !exists {
			buckets[bucketTime] = &TimeSeriesDataPoint{
				Timestamp: bucketTime,
				Metadata:  make(map[string]interface{}),
			}
		}

		bucket := buckets[bucketTime]
		bucket.Count++

		// Track delivery metrics
		if event.EventType == string(StatusDelivered) && event.Success {
			if avgTime, ok := bucket.Metadata["total_time"].(float64); ok {
				bucket.Metadata["total_time"] = avgTime + event.Duration
			} else {
				bucket.Metadata["total_time"] = event.Duration
			}
		}
	}

	// Convert to slice and calculate rates
	for _, bucket := range buckets {
		if totalTime, ok := bucket.Metadata["total_time"].(float64); ok && bucket.Count > 0 {
			bucket.AvgTime = totalTime / float64(bucket.Count)
		}
		data.DataPoints = append(data.DataPoints, *bucket)
	}

	// Sort by timestamp
	for i := 0; i < len(data.DataPoints); i++ {
		for j := i + 1; j < len(data.DataPoints); j++ {
			if data.DataPoints[j].Timestamp.Before(data.DataPoints[i].Timestamp) {
				data.DataPoints[i], data.DataPoints[j] = data.DataPoints[j], data.DataPoints[i]
			}
		}
	}

	return data
}

// ExportEvents exports analytics events to JSON.
func (a *AnalyticsTracker) ExportEvents(startTime, endTime time.Time) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	filteredEvents := make([]AnalyticsEvent, 0)
	for i := range a.events {
		event := &a.events[i]
		if event.Timestamp.After(startTime) && event.Timestamp.Before(endTime) {
			filteredEvents = append(filteredEvents, *event)
		}
	}

	return json.MarshalIndent(filteredEvents, "", "  ")
}

// Helper methods

func (a *AnalyticsTracker) updateMetrics(event *AnalyticsEvent) {
	switch event.EventType {
	case "sent":
		a.metrics.TotalSent++

	case string(StatusDelivered):
		if event.Success {
			a.metrics.TotalDelivered++

			// Update channel stats
			channelStats := a.metrics.ByChannel[event.Channel]
			channelStats.Sent++
			channelStats.Delivered++
			channelStats.AvgTime = (channelStats.AvgTime*float64(channelStats.Delivered-1) + event.Duration) / float64(channelStats.Delivered)
			channelStats.DeliveryRate = float64(channelStats.Delivered) / float64(channelStats.Sent)
			a.metrics.ByChannel[event.Channel] = channelStats

			// Update avg delivery time
			if a.metrics.TotalDelivered > 0 {
				a.metrics.AvgDeliveryTime = (a.metrics.AvgDeliveryTime*float64(a.metrics.TotalDelivered-1) + event.Duration) / float64(a.metrics.TotalDelivered)
			}
		} else {
			a.metrics.TotalFailed++

			// Update channel stats
			channelStats := a.metrics.ByChannel[event.Channel]
			channelStats.Sent++
			channelStats.Failed++
			channelStats.DeliveryRate = float64(channelStats.Delivered) / float64(channelStats.Sent)
			a.metrics.ByChannel[event.Channel] = channelStats
		}

	case string(StatusRead):
		a.metrics.TotalRead++

		// Update type stats
		typeStats := a.metrics.ByType[event.Type]
		typeStats.Read++
		typeStats.AvgReadTime = (typeStats.AvgReadTime*float64(typeStats.Read-1) + event.Duration/1000) / float64(typeStats.Read)
		if typeStats.Sent > 0 {
			typeStats.ReadRate = float64(typeStats.Read) / float64(typeStats.Sent)
		}
		a.metrics.ByType[event.Type] = typeStats

		// Update priority stats
		priorityStats := a.metrics.ByPriority[event.Priority]
		priorityStats.Read++
		priorityStats.AvgReadTime = (priorityStats.AvgReadTime*float64(priorityStats.Read-1) + event.Duration/1000) / float64(priorityStats.Read)
		a.metrics.ByPriority[event.Priority] = priorityStats
	}

	// Update overall rates
	if a.metrics.TotalSent > 0 {
		a.metrics.DeliveryRate = float64(a.metrics.TotalDelivered) / float64(a.metrics.TotalSent)
		a.metrics.ReadRate = float64(a.metrics.TotalRead) / float64(a.metrics.TotalSent)
	}
}

func (a *AnalyticsTracker) updateRealtimeStats(event *AnalyticsEvent) {
	a.realtimeStats.LastUpdated = time.Now()

	if event.EventType == string(StatusDelivered) {
		health := a.realtimeStats.ChannelHealth[event.Channel]
		if event.Success {
			health.Status = "healthy"
			health.LastSuccess = time.Now()

			// Update success rate (exponential moving average)
			health.SuccessRate = health.SuccessRate*0.9 + 0.1

			// Update avg latency
			if health.AvgLatency == 0 {
				health.AvgLatency = event.Duration
			} else {
				health.AvgLatency = health.AvgLatency*0.8 + event.Duration*0.2
			}
		} else {
			health.ErrorCount++
			health.LastError = event.Error
			health.SuccessRate *= 0.9

			if health.SuccessRate < 0.5 {
				health.Status = "down"
			} else if health.SuccessRate < 0.8 {
				health.Status = "degraded"
			}
		}
		a.realtimeStats.ChannelHealth[event.Channel] = health
	}
}

func (a *AnalyticsTracker) calculateEngagementScore(metrics *UserEngagementMetrics) float64 {
	if metrics.TotalNotifications == 0 {
		return 0
	}

	score := 0.0

	// Read rate (40%)
	score += metrics.ReadRate * 40

	// Action rate (35%)
	score += metrics.ActionRate * 35

	// Low dismiss rate bonus (15%)
	score += (1.0 - metrics.DismissRate) * 15

	// Quick response time bonus (10%)
	if metrics.AvgTimeToRead > 0 && metrics.AvgTimeToRead < 60 {
		score += 10.0
	} else if metrics.AvgTimeToRead < 300 {
		score += 5.0
	}

	return score
}

func (a *AnalyticsTracker) copyMetrics(src *NotificationStats) *NotificationStats {
	dst := &NotificationStats{
		TotalSent:       src.TotalSent,
		TotalDelivered:  src.TotalDelivered,
		TotalFailed:     src.TotalFailed,
		TotalRead:       src.TotalRead,
		DeliveryRate:    src.DeliveryRate,
		ReadRate:        src.ReadRate,
		AvgDeliveryTime: src.AvgDeliveryTime,
		ByChannel:       make(map[DeliveryChannel]ChannelStats),
		ByType:          make(map[NotificationType]TypeStats),
		ByPriority:      make(map[Priority]PriorityStats),
	}

	for k, v := range src.ByChannel {
		dst.ByChannel[k] = v
	}
	for k, v := range src.ByType {
		dst.ByType[k] = v
	}
	for k, v := range src.ByPriority {
		dst.ByPriority[k] = v
	}

	return dst
}

func (a *AnalyticsTracker) getBucketTime(t time.Time, interval string) time.Time {
	switch interval {
	case "hourly":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case "daily":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "weekly":
		// Start of week (Monday)
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
	case "monthly":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}
