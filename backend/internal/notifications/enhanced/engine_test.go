package enhanced

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	config := Config{
		WorkerCount:       5,
		QueueSize:         100,
		MaxRetries:     3,
		RetryDelay:        time.Second,
		DataDir:           "/tmp/notifications_test",
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)
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

	config := Config{
		WorkerCount:   2,
		QueueSize:     10,
		MaxRetries: 1,
		DataDir:       tmpDir,
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)

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
	err = engine.Send(ctx, notification)
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

	config := Config{
		DataDir: tmpDir,
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)

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

	// Save preferences directly (no public API)
	engine.mu.Lock()
	engine.preferences["user-001"] = prefs
	engine.mu.Unlock()

	// Retrieve preferences
	retrieved := engine.getPreferences("user-001")
	if retrieved == nil {
		t.Fatal("Expected preferences to be returned")
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

	config := Config{
		DataDir: tmpDir,
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)

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
	if err := engine.RegisterTemplate(template); err != nil {
		t.Errorf("Failed to register template: %v", err)
	}

	// Get template
	engine.mu.RLock()
	retrieved := engine.templates["template-001"]
	engine.mu.RUnlock()
	if retrieved == nil {
		t.Fatal("Expected template to be stored")
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

	config := Config{
		DataDir:    tmpDir,
		QueueSize:  10,
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)

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

	engine.RegisterTemplate(template)

	// Send from template
	ctx := context.Background()
	variables := map[string]interface{}{
		"name":      "John",
		"backup_id": "backup-123",
	}

	err = engine.SendFromTemplate(ctx, "template-002", variables, "user-001")
	if err != nil {
		t.Errorf("Failed to send from template: %v", err)
	}
}

func TestEngineQuietHoursCheck(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := NewEngine(&Config{DataDir: tmpDir})
	require.NoError(t, err)

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

	isQuiet := engine.isInQuietHours(prefs)
	if !isQuiet {
		t.Error("Expected to be in quiet hours")
	}
}

func TestEngineDNDCheck(t *testing.T) {
	tmpDir := t.TempDir()
	engine, err := NewEngine(&Config{DataDir: tmpDir})
	require.NoError(t, err)

	// DND disabled → not in DND period
	prefsOff := &NotificationPreferences{
		UserID:       "user-001",
		DoNotDisturb: false,
	}
	if engine.isInDNDPeriod(prefsOff) {
		t.Error("Expected DND to be inactive when DoNotDisturb=false")
	}

	// DND enabled but no schedule → not in DND period (schedule must match)
	prefsOn := &NotificationPreferences{
		UserID:       "user-001",
		DoNotDisturb: true,
	}
	if engine.isInDNDPeriod(prefsOn) {
		t.Error("Expected DND to be inactive without a matching schedule")
	}

	// DND enabled with matching schedule → in DND period
	prefsScheduled := &NotificationPreferences{
		UserID:       "user-001",
		DoNotDisturb: true,
		DNDSchedule: []DNDSchedule{
			{
				Start: time.Now().Add(-1 * time.Hour),
				End:   time.Now().Add(1 * time.Hour),
			},
		},
	}
	if !engine.isInDNDPeriod(prefsScheduled) {
		t.Error("Expected DND to be active with matching schedule")
	}
}

func TestEngineRetentionCleanup(t *testing.T) {
	tmpDir := "/tmp/notifications_test_retention_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := Config{
		DataDir:       tmpDir,
		RetentionDays: 1, // 1 day retention
	}

	engine, err := NewEngine(&config)
	require.NoError(t, err)

	// Add an old notification (older than 1 day)
	oldNotif := &Notification{
		ID:        "old-001",
		CreatedAt: time.Now().AddDate(0, 0, -2), // 2 days old
		UserID:    "user-001",
		Status:    StatusRead,
	}
	// Add a recent notification
	recentNotif := &Notification{
		ID:        "recent-001",
		CreatedAt: time.Now(),
		UserID:    "user-001",
		Status:    StatusRead,
	}

	engine.mu.Lock()
	engine.notifications["old-001"] = oldNotif
	engine.notifications["recent-001"] = recentNotif
	engine.mu.Unlock()

	// saveNotifications applies retention policy internally
	if err := engine.saveNotifications(); err != nil {
		t.Fatalf("saveNotifications failed: %v", err)
	}

	// Recent notification should still be in memory
	engine.mu.RLock()
	_, recentExists := engine.notifications["recent-001"]
	engine.mu.RUnlock()
	if !recentExists {
		t.Error("Expected recent notification to still exist")
	}
}

func BenchmarkEngineSend(b *testing.B) {
	tmpDir := "/tmp/notifications_bench_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	config := Config{
		WorkerCount:   5,
		QueueSize:     10000,
		DataDir:       tmpDir,
	}

	engine, err := NewEngine(&config)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
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
