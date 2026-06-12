package enhanced

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	config := EngineConfig{
		WorkerCount:       5,
		QueueSize:         100,
		RetryAttempts:     3,
		RetryDelay:        time.Second,
		DataDir:           "/tmp/notifications_test",
		EnableBatching:    true,
		EnableEscalation:  true,
		EnableAnalytics:   true,
	}

	engine := NewEngine(config)
	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.config.WorkerCount != 5 {
		t.Errorf("Expected WorkerCount to be 5, got %d", engine.config.WorkerCount)
	}
}

func TestEngineSendNotification(t *testing.T) {
	// Create temp directory for test
	tmpDir := "/tmp/notifications_test_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		WorkerCount:   2,
		QueueSize:     10,
		RetryAttempts: 1,
		DataDir:       tmpDir,
	}

	engine := NewEngine(config)

	// Start engine
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Create test notification
	notification := &Notification{
		ID:       "test-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityNormal,
		Title:    "Test Notification",
		Message:  "This is a test notification",
		Category: "backup",
		Tags:     []string{"test"},
		Metadata: map[string]interface{}{
			"test": true,
		},
		Channels:  []DeliveryChannel{ChannelInApp},
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UserID:    "user-001",
	}

	// Send notification
	ctx := context.Background()
	err := engine.Send(ctx, notification)
	if err != nil {
		t.Errorf("Failed to send notification: %v", err)
	}

	// Wait a bit for processing
	time.Sleep(500 * time.Millisecond)

	// Verify notification was queued
	if len(engine.queue) > engine.config.QueueSize {
		t.Error("Queue size exceeded")
	}
}

func TestEngineUserPreferences(t *testing.T) {
	tmpDir := "/tmp/notifications_test_prefs_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		DataDir: tmpDir,
	}

	engine := NewEngine(config)

	// Create preferences
	prefs := &NotificationPreferences{
		UserID:          "user-001",
		EnabledChannels: []DeliveryChannel{ChannelEmail, ChannelPush},
		DisabledTypes:   []NotificationType{TypeBackupStarted},
		MinimumPriority: PriorityNormal,
		QuietHours: []QuietHoursPeriod{
			{
				Start:    "22:00",
				End:      "08:00",
				Days:     []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
				Timezone: "UTC",
			},
		},
		DoNotDisturb:    false,
		GroupingEnabled: true,
		BatchingEnabled: true,
		BatchInterval:   15,
		DigestEnabled:   false,
		UpdatedAt:       time.Now(),
	}

	// Save preferences
	if err := engine.SetUserPreferences(prefs); err != nil {
		t.Errorf("Failed to save preferences: %v", err)
	}

	// Retrieve preferences
	retrieved, err := engine.GetUserPreferences("user-001")
	if err != nil {
		t.Errorf("Failed to get preferences: %v", err)
	}

	if retrieved.UserID != "user-001" {
		t.Errorf("Expected UserID to be user-001, got %s", retrieved.UserID)
	}

	if !retrieved.GroupingEnabled {
		t.Error("Expected GroupingEnabled to be true")
	}
}

func TestEngineTemplate(t *testing.T) {
	tmpDir := "/tmp/notifications_test_templates_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		DataDir: tmpDir,
	}

	engine := NewEngine(config)

	// Create template
	template := &NotificationTemplate{
		ID:       "template-001",
		Name:     "Backup Success",
		Type:     TypeBackupCompleted,
		Title:    "Backup completed for ${database}",
		Message:  "The backup of ${database} completed successfully in ${duration}",
		Priority: PriorityNormal,
		Channels: []DeliveryChannel{ChannelEmail, ChannelSlack},
		Actions: []NotificationAction{
			{
				ID:     "view",
				Label:  "View Details",
				Type:   "button",
				Action: "view",
				URL:    "/backups/${backup_id}",
				Style:  "primary",
			},
		},
		Variables: []string{"database", "duration", "backup_id"},
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add template
	if err := engine.AddTemplate(template); err != nil {
		t.Errorf("Failed to add template: %v", err)
	}

	// Get template
	retrieved, err := engine.GetTemplate("template-001")
	if err != nil {
		t.Errorf("Failed to get template: %v", err)
	}

	if retrieved.Name != "Backup Success" {
		t.Errorf("Expected Name to be 'Backup Success', got %s", retrieved.Name)
	}

	if len(retrieved.Variables) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(retrieved.Variables))
	}
}

func TestEngineSendFromTemplate(t *testing.T) {
	tmpDir := "/tmp/notifications_test_template_send_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		DataDir:    tmpDir,
		QueueSize:  10,
	}

	engine := NewEngine(config)

	// Add template
	template := &NotificationTemplate{
		ID:       "template-002",
		Name:     "Test Template",
		Type:     TypeBackupCompleted,
		Title:    "Hello ${name}",
		Message:  "Your backup ${backup_id} is complete",
		Priority: PriorityNormal,
		Channels: []DeliveryChannel{ChannelInApp},
		Variables: []string{"name", "backup_id"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	engine.AddTemplate(template)

	// Send from template
	ctx := context.Background()
	variables := map[string]interface{}{
		"name":      "John",
		"backup_id": "backup-123",
	}

	err := engine.SendFromTemplate(ctx, "template-002", variables, "user-001")
	if err != nil {
		t.Errorf("Failed to send from template: %v", err)
	}
}

func TestEngineQuietHoursCheck(t *testing.T) {
	engine := NewEngine(EngineConfig{})

	// Set up quiet hours (current time)
	now := time.Now()
	currentHour := now.Format("15:04")
	endHour := now.Add(2 * time.Hour).Format("15:04")

	prefs := &NotificationPreferences{
		UserID: "user-001",
		QuietHours: []QuietHoursPeriod{
			{
				Start:    currentHour,
				End:      endHour,
				Days:     []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
				Timezone: "UTC",
			},
		},
	}

	notification := &Notification{
		UserID:   "user-001",
		Priority: PriorityNormal,
	}

	isQuiet := engine.isQuietHours(notification, prefs)
	if !isQuiet {
		t.Error("Expected to be in quiet hours")
	}
}

func TestEngineDNDCheck(t *testing.T) {
	engine := NewEngine(EngineConfig{})

	prefs := &NotificationPreferences{
		UserID:       "user-001",
		DoNotDisturb: true,
	}

	notification := &Notification{
		UserID:   "user-001",
		Priority: PriorityNormal,
	}

	isDND := engine.isDoNotDisturb(notification, prefs)
	if !isDND {
		t.Error("Expected DND to be active")
	}

	// Urgent notifications should bypass DND
	notification.Priority = PriorityUrgent
	isDND = engine.isDoNotDisturb(notification, prefs)
	if isDND {
		t.Error("Expected urgent notification to bypass DND")
	}
}

func TestEngineRetentionCleanup(t *testing.T) {
	tmpDir := "/tmp/notifications_test_retention_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		DataDir:         tmpDir,
		RetentionPeriod: 1 * time.Second, // Very short for testing
	}

	engine := NewEngine(config)

	// Add old notification
	oldNotif := &Notification{
		ID:        "old-001",
		CreatedAt: time.Now().Add(-2 * time.Second),
		UserID:    "user-001",
		Status:    StatusRead,
	}

	engine.mu.Lock()
	engine.notifications["old-001"] = oldNotif
	engine.mu.Unlock()

	// Run cleanup
	engine.cleanupOldNotifications()

	// Check if old notification was removed
	engine.mu.RLock()
	_, exists := engine.notifications["old-001"]
	engine.mu.RUnlock()

	if exists {
		t.Error("Expected old notification to be cleaned up")
	}
}

func BenchmarkEngineSend(b *testing.B) {
	tmpDir := "/tmp/notifications_bench_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := EngineConfig{
		WorkerCount:   5,
		QueueSize:     10000,
		DataDir:       tmpDir,
	}

	engine := NewEngine(config)
	engine.Start()
	defer engine.Stop()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		notification := &Notification{
			ID:        time.Now().Format("20060102150405.000000"),
			Type:      TypeBackupCompleted,
			Priority:  PriorityNormal,
			Title:     "Benchmark Notification",
			Message:   "This is a benchmark",
			Channels:  []DeliveryChannel{ChannelInApp},
			Status:    StatusPending,
			CreatedAt: time.Now(),
			UserID:    "user-bench",
		}

		engine.Send(ctx, notification)
	}
}
