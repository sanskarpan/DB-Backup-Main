package enhanced

import (
	"os"
	"testing"
	"time"
)

func TestNewAnalyticsTracker(t *testing.T) {
	dataFile := "/tmp/analytics_test.json"
	defer os.Remove(dataFile)

	tracker := NewAnalyticsTracker(dataFile, 1000)
	if tracker == nil {
		t.Fatal("Expected analytics tracker to be created")
	}

	if tracker.maxEvents != 1000 {
		t.Errorf("Expected maxEvents to be 1000, got %d", tracker.maxEvents)
	}
}

func TestAnalyticsTrackerLoadSave(t *testing.T) {
	dataFile := "/tmp/analytics_test_save.json"
	defer os.Remove(dataFile)

	tracker := NewAnalyticsTracker(dataFile, 100)

	// Add some events
	event := AnalyticsEvent{
		ID:             "event-001",
		EventType:      "sent",
		UserID:         "user-001",
		NotificationID: "notif-001",
		Type:           TypeBackupCompleted,
		Priority:       PriorityNormal,
		Success:        true,
		Timestamp:      time.Now(),
	}

	tracker.TrackEvent(event)

	// Save
	if err := tracker.Save(); err != nil {
		t.Fatalf("Failed to save analytics data: %v", err)
	}

	// Load in new tracker
	tracker2 := NewAnalyticsTracker(dataFile, 100)
	if err := tracker2.Load(); err != nil {
		t.Fatalf("Failed to load analytics data: %v", err)
	}

	if len(tracker2.events) != 1 {
		t.Errorf("Expected 1 event to be loaded, got %d", len(tracker2.events))
	}
}

func TestAnalyticsTrackerTrackNotificationSent(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_sent.json", 100)

	notification := &Notification{
		ID:       "notif-001",
		UserID:   "user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityHigh,
	}

	tracker.TrackNotificationSent(notification)

	tracker.mu.RLock()
	eventCount := len(tracker.events)
	totalSent := tracker.metrics.TotalSent
	tracker.mu.RUnlock()

	if eventCount != 1 {
		t.Errorf("Expected 1 event, got %d", eventCount)
	}

	if totalSent != 1 {
		t.Errorf("Expected TotalSent to be 1, got %d", totalSent)
	}
}

func TestAnalyticsTrackerTrackDelivery(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_delivery.json", 100)

	notification := &Notification{
		ID:       "notif-002",
		UserID:   "user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityNormal,
	}

	result := DeliveryResult{
		Channel:      ChannelEmail,
		Success:      true,
		DeliveredAt:  time.Now(),
		ResponseTime: 150.5,
		MessageID:    "msg-001",
	}

	tracker.TrackDelivery(notification, result)

	tracker.mu.RLock()
	totalDelivered := tracker.metrics.TotalDelivered
	channelStats := tracker.metrics.ByChannel[ChannelEmail]
	tracker.mu.RUnlock()

	if totalDelivered != 1 {
		t.Errorf("Expected TotalDelivered to be 1, got %d", totalDelivered)
	}

	if channelStats.Delivered != 1 {
		t.Errorf("Expected channel delivered count to be 1, got %d", channelStats.Delivered)
	}

	if channelStats.AvgTime != 150.5 {
		t.Errorf("Expected AvgTime to be 150.5, got %f", channelStats.AvgTime)
	}
}

func TestAnalyticsTrackerTrackRead(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_read.json", 100)

	notification := &Notification{
		ID:       "notif-003",
		UserID:   "user-001",
		Type:     TypeBackupFailed,
		Priority: PriorityHigh,
	}

	timeToRead := 30 * time.Second

	tracker.TrackRead(notification, timeToRead)

	tracker.mu.RLock()
	totalRead := tracker.metrics.TotalRead
	typeStats := tracker.metrics.ByType[TypeBackupFailed]
	tracker.mu.RUnlock()

	if totalRead != 1 {
		t.Errorf("Expected TotalRead to be 1, got %d", totalRead)
	}

	if typeStats.Read != 1 {
		t.Errorf("Expected type read count to be 1, got %d", typeStats.Read)
	}

	expectedReadTime := 30.0 // seconds
	if typeStats.AvgReadTime != expectedReadTime {
		t.Errorf("Expected AvgReadTime to be %f, got %f", expectedReadTime, typeStats.AvgReadTime)
	}
}

func TestAnalyticsTrackerGetUserEngagement(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_engagement.json", 1000)

	// Track multiple events for a user
	userID := "user-001"

	// Sent events
	for i := 0; i < 10; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-sent-" + string(rune(i)),
			EventType:      "sent",
			UserID:         userID,
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Channel:        ChannelEmail,
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	// Read events
	for i := 0; i < 7; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-read-" + string(rune(i)),
			EventType:      "read",
			UserID:         userID,
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Duration:       25000, // 25 seconds in ms
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	// Action events
	for i := 0; i < 5; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-action-" + string(rune(i)),
			EventType:      "action",
			UserID:         userID,
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Duration:       60000, // 60 seconds in ms
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	// Dismissed events
	for i := 0; i < 2; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-dismiss-" + string(rune(i)),
			EventType:      "dismissed",
			UserID:         userID,
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	// Get engagement metrics
	metrics := tracker.GetUserEngagement(userID)

	if metrics.TotalNotifications != 10 {
		t.Errorf("Expected 10 total notifications, got %d", metrics.TotalNotifications)
	}

	if metrics.ReadNotifications != 7 {
		t.Errorf("Expected 7 read notifications, got %d", metrics.ReadNotifications)
	}

	if metrics.ActionedNotifications != 5 {
		t.Errorf("Expected 5 actioned notifications, got %d", metrics.ActionedNotifications)
	}

	if metrics.DismissedNotifications != 2 {
		t.Errorf("Expected 2 dismissed notifications, got %d", metrics.DismissedNotifications)
	}

	expectedReadRate := 0.7 // 7/10
	if metrics.ReadRate != expectedReadRate {
		t.Errorf("Expected read rate to be %f, got %f", expectedReadRate, metrics.ReadRate)
	}

	expectedActionRate := 0.5 // 5/10
	if metrics.ActionRate != expectedActionRate {
		t.Errorf("Expected action rate to be %f, got %f", expectedActionRate, metrics.ActionRate)
	}

	// Engagement score should be high (good read and action rates)
	if metrics.EngagementScore < 50 {
		t.Errorf("Expected high engagement score, got %f", metrics.EngagementScore)
	}
}

func TestAnalyticsTrackerGetTimeSeries(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_timeseries.json", 1000)

	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	// Add events over time
	for i := 0; i < 24; i++ {
		timestamp := startTime.Add(time.Duration(i) * time.Hour)
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-" + string(rune(i)),
			EventType:      "sent",
			UserID:         "user-001",
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Success:        true,
			Timestamp:      timestamp,
		})
	}

	// Get hourly time series
	timeSeries := tracker.GetTimeSeries("hourly", startTime, endTime)

	if timeSeries.Interval != "hourly" {
		t.Errorf("Expected interval to be 'hourly', got %s", timeSeries.Interval)
	}

	if len(timeSeries.DataPoints) != 24 {
		t.Errorf("Expected 24 data points, got %d", len(timeSeries.DataPoints))
	}

	// Check if data points are sorted by timestamp
	for i := 1; i < len(timeSeries.DataPoints); i++ {
		if timeSeries.DataPoints[i].Timestamp.Before(timeSeries.DataPoints[i-1].Timestamp) {
			t.Error("Expected data points to be sorted by timestamp")
			break
		}
	}
}

func TestAnalyticsTrackerRealtimeStats(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_realtime.json", 100)

	notification := &Notification{
		ID:       "notif-realtime",
		UserID:   "user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityNormal,
	}

	// Track successful delivery
	successResult := DeliveryResult{
		Channel:      ChannelEmail,
		Success:      true,
		ResponseTime: 100.0,
	}
	tracker.TrackDelivery(notification, successResult)

	stats := tracker.GetRealtimeStats()

	if stats == nil {
		t.Fatal("Expected realtime stats to be returned")
	}

	emailHealth := stats.ChannelHealth[ChannelEmail]
	if emailHealth.Status != "healthy" {
		t.Errorf("Expected email channel to be healthy, got %s", emailHealth.Status)
	}

	// Track failed delivery
	failResult := DeliveryResult{
		Channel:      ChannelSMS,
		Success:      false,
		Error:        "Connection timeout",
		ResponseTime: 5000.0,
	}
	tracker.TrackDelivery(notification, failResult)

	stats = tracker.GetRealtimeStats()
	smsHealth := stats.ChannelHealth[ChannelSMS]

	if smsHealth.ErrorCount != 1 {
		t.Errorf("Expected 1 error for SMS channel, got %d", smsHealth.ErrorCount)
	}

	if smsHealth.LastError != "Connection timeout" {
		t.Errorf("Expected last error to be 'Connection timeout', got %s", smsHealth.LastError)
	}
}

func TestAnalyticsTrackerExportEvents(t *testing.T) {
	tracker := NewAnalyticsTracker("/tmp/analytics_export.json", 100)

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	// Add events
	for i := 0; i < 10; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "export-" + string(rune(i)),
			EventType:      "sent",
			UserID:         "user-001",
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Success:        true,
			Timestamp:      time.Now().Add(-30 * time.Minute),
		})
	}

	// Export events
	data, err := tracker.ExportEvents(startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to export events: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected exported data to be non-empty")
	}
}

func TestAnalyticsTrackerMaxEvents(t *testing.T) {
	maxEvents := 10
	tracker := NewAnalyticsTracker("/tmp/analytics_max.json", maxEvents)

	// Add more events than max
	for i := 0; i < 20; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-" + string(rune(i)),
			EventType:      "sent",
			UserID:         "user-001",
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	tracker.mu.RLock()
	eventCount := len(tracker.events)
	tracker.mu.RUnlock()

	if eventCount > maxEvents {
		t.Errorf("Expected event count to be <= %d, got %d", maxEvents, eventCount)
	}
}

func BenchmarkTrackEvent(b *testing.B) {
	tracker := NewAnalyticsTracker("/tmp/analytics_bench.json", 100000)

	event := AnalyticsEvent{
		EventType:      "sent",
		UserID:         "user-bench",
		NotificationID: "notif-bench",
		Type:           TypeBackupCompleted,
		Priority:       PriorityNormal,
		Success:        true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.ID = "event-" + string(rune(i))
		tracker.TrackEvent(event)
	}
}

func BenchmarkGetUserEngagement(b *testing.B) {
	tracker := NewAnalyticsTracker("/tmp/analytics_engagement_bench.json", 10000)

	// Add events
	for i := 0; i < 1000; i++ {
		tracker.TrackEvent(AnalyticsEvent{
			ID:             "event-" + string(rune(i)),
			EventType:      "sent",
			UserID:         "user-bench",
			NotificationID: "notif-" + string(rune(i)),
			Type:           TypeBackupCompleted,
			Priority:       PriorityNormal,
			Success:        true,
			Timestamp:      time.Now(),
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker.GetUserEngagement("user-bench")
	}
}
