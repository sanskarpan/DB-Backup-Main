package webhooks

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := &ManagerConfig{
		Workers:   5,
		QueueSize: 100,
	}

	manager := NewManager(config)
	defer manager.Stop()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.workers != 5 {
		t.Errorf("Expected 5 workers, got %d", manager.workers)
	}

	if len(manager.subscriptions) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(manager.subscriptions))
	}

	t.Log("✓ Manager created successfully")
}

func TestSubscribe(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	url := helper.CreateMockServer("test")

	sub := &Subscription{
		Name:   "test-subscription",
		URL:    url,
		Events: []EventType{EventBackupCompleted},
	}

	err := helper.GetManager().Subscribe(sub)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if sub.ID == "" {
		t.Error("Subscription ID should be generated")
	}

	if !sub.Enabled {
		t.Error("Subscription should be enabled by default")
	}

	if sub.RetryConfig == nil {
		t.Error("Retry config should be set to default")
	}

	t.Log("✓ Subscription created successfully")
}

func TestSubscribeValidation(t *testing.T) {
	manager := NewManager(&ManagerConfig{})
	defer manager.Stop()

	tests := []struct {
		name        string
		sub         *Subscription
		expectError bool
	}{
		{
			name: "Missing name",
			sub: &Subscription{
				URL:    "http://example.com",
				Events: []EventType{EventBackupCompleted},
			},
			expectError: true,
		},
		{
			name: "Missing URL",
			sub: &Subscription{
				Name:   "test",
				Events: []EventType{EventBackupCompleted},
			},
			expectError: true,
		},
		{
			name: "Missing events",
			sub: &Subscription{
				Name: "test",
				URL:  "http://example.com",
			},
			expectError: true,
		},
		{
			name: "Valid subscription",
			sub: &Subscription{
				Name:   "test",
				URL:    "http://example.com",
				Events: []EventType{EventBackupCompleted},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Subscribe(tt.sub)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}

	t.Log("✓ Subscription validation works correctly")
}

func TestPublishEvent(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	serverURL := helper.CreateMockServer("test")

	sub := &Subscription{
		Name:   "test-subscription",
		URL:    serverURL,
		Events: []EventType{EventBackupCompleted},
		Secret: "test-secret",
	}

	err := helper.GetManager().Subscribe(sub)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish event
	event := &Event{
		Type:   EventBackupCompleted,
		Source: "test",
		Data: map[string]interface{}{
			"backup_id": "backup-123",
			"database":  "test-db",
			"status":    "completed",
		},
	}

	err = helper.GetManager().Publish(event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for delivery
	err = helper.WaitForDelivery("test", 1, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to deliver webhook: %v", err)
	}

	// Verify request
	server := helper.GetMockServer("test")
	requests := server.GetRequests()

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	req := requests[0]
	if req.Signature == "" {
		t.Error("Expected HMAC signature in request")
	}

	t.Log("✓ Event published and delivered successfully")
}

func TestEventFiltering(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	serverURL := helper.CreateMockServer("test")

	// Subscribe with filter
	sub := &Subscription{
		Name:   "test-subscription",
		URL:    serverURL,
		Events: []EventType{EventBackupCompleted},
		Filters: []EventFilter{
			{
				Field:    "database",
				Operator: "eq",
				Value:    "prod-db",
			},
		},
	}

	err := helper.GetManager().Subscribe(sub)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish matching event
	event1 := &Event{
		Type:   EventBackupCompleted,
		Source: "test",
		Data: map[string]interface{}{
			"database": "prod-db",
		},
	}
	helper.GetManager().Publish(event1)

	// Publish non-matching event
	event2 := &Event{
		Type:   EventBackupCompleted,
		Source: "test",
		Data: map[string]interface{}{
			"database": "dev-db",
		},
	}
	helper.GetManager().Publish(event2)

	// Wait and check
	time.Sleep(1 * time.Second)

	server := helper.GetMockServer("test")
	count := server.GetRequestCount()

	if count != 1 {
		t.Errorf("Expected 1 delivered webhook, got %d", count)
	}

	t.Log("✓ Event filtering works correctly")
}

func TestUnsubscribe(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	helper.CreateMockServer("test")

	sub, err := helper.CreateSubscription("test", []EventType{EventBackupCompleted}, "test")
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Unsubscribe
	err = helper.GetManager().Unsubscribe(sub.ID)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Verify subscription is removed
	_, err = helper.GetManager().GetSubscription(sub.ID)
	if err == nil {
		t.Error("Expected error when getting removed subscription")
	}

	t.Log("✓ Unsubscribe works correctly")
}

func TestEnableDisableSubscription(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	helper.CreateMockServer("test")

	sub, err := helper.CreateSubscription("test", []EventType{EventBackupCompleted}, "test")
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Disable subscription
	err = helper.GetManager().DisableSubscription(sub.ID)
	if err != nil {
		t.Fatalf("Failed to disable subscription: %v", err)
	}

	// Verify disabled
	updatedSub, err := helper.GetManager().GetSubscription(sub.ID)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if updatedSub.Enabled {
		t.Error("Subscription should be disabled")
	}

	// Publish event (should not be delivered)
	event := &Event{
		Type:   EventBackupCompleted,
		Source: "test",
		Data:   map[string]interface{}{},
	}
	helper.GetManager().Publish(event)

	time.Sleep(1 * time.Second)

	server := helper.GetMockServer("test")
	if server.GetRequestCount() != 0 {
		t.Error("No webhooks should be delivered to disabled subscription")
	}

	// Enable subscription
	err = helper.GetManager().EnableSubscription(sub.ID)
	if err != nil {
		t.Fatalf("Failed to enable subscription: %v", err)
	}

	// Verify enabled
	updatedSub, err = helper.GetManager().GetSubscription(sub.ID)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if !updatedSub.Enabled {
		t.Error("Subscription should be enabled")
	}

	t.Log("✓ Enable/disable subscription works correctly")
}

func TestListSubscriptions(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	// Create multiple subscriptions
	for i := 0; i < 3; i++ {
		serverName := "test"
		if i == 0 {
			helper.CreateMockServer(serverName)
		}
		helper.CreateSubscription("sub-"+string(rune(i)), []EventType{EventBackupCompleted}, serverName)
	}

	// List subscriptions
	subs := helper.GetManager().ListSubscriptions()

	if len(subs) != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", len(subs))
	}

	t.Log("✓ List subscriptions works correctly")
}

func TestRetryLogic(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	serverURL := helper.CreateMockServer("test")
	server := helper.GetMockServer("test")

	// Configure to fail first 2 attempts
	server.SetFailCount(2)

	// Create subscription with fast retry
	sub := &Subscription{
		Name:   "test-subscription",
		URL:    serverURL,
		Events: []EventType{EventBackupCompleted},
		RetryConfig: &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Timeout:      5 * time.Second,
		},
	}

	err := helper.GetManager().Subscribe(sub)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish event
	event := &Event{
		Type:   EventBackupCompleted,
		Source: "test",
		Data:   map[string]interface{}{},
	}

	helper.GetManager().Publish(event)

	// Wait for retries to complete
	time.Sleep(3 * time.Second)

	// Should have 3 attempts (1 initial + 2 retries)
	count := server.GetRequestCount()
	if count < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", count)
	}

	t.Log("✓ Retry logic works correctly")
}

func TestUpdateSubscription(t *testing.T) {
	helper := NewTestHelper()
	defer helper.Cleanup()

	helper.CreateMockServer("test")

	sub, err := helper.CreateSubscription("test", []EventType{EventBackupCompleted}, "test")
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Update subscription
	updates := &Subscription{
		Name:     "updated-name",
		Events:   []EventType{EventBackupCompleted, EventBackupFailed},
		Template: "slack",
	}

	err = helper.GetManager().UpdateSubscription(sub.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update subscription: %v", err)
	}

	// Verify updates
	updated, err := helper.GetManager().GetSubscription(sub.ID)
	if err != nil {
		t.Fatalf("Failed to get subscription: %v", err)
	}

	if updated.Name != "updated-name" {
		t.Errorf("Expected name 'updated-name', got '%s'", updated.Name)
	}

	if len(updated.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(updated.Events))
	}

	if updated.Template != "slack" {
		t.Errorf("Expected template 'slack', got '%s'", updated.Template)
	}

	t.Log("✓ Update subscription works correctly")
}

func TestEventBuilder(t *testing.T) {
	event := NewEventBuilder(EventBackupCompleted).
		WithSource("test-source").
		WithData("backup_id", "backup-123").
		WithData("database", "test-db").
		WithMetadata("region", "us-west").
		Build()

	if event.Type != EventBackupCompleted {
		t.Errorf("Expected type %s, got %s", EventBackupCompleted, event.Type)
	}

	if event.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got '%s'", event.Source)
	}

	if event.Data["backup_id"] != "backup-123" {
		t.Error("Expected backup_id in data")
	}

	if event.Metadata["region"] != "us-west" {
		t.Error("Expected region in metadata")
	}

	t.Log("✓ Event builder works correctly")
}

func TestSubscriptionBuilder(t *testing.T) {
	sub := NewSubscriptionBuilder("test-sub", "http://example.com").
		WithEvents(EventBackupCompleted, EventBackupFailed).
		WithFilter("database", "eq", "prod-db").
		WithTemplate("slack").
		WithHeader("X-Custom", "value").
		WithSecret("secret-key").
		Build()

	if sub.Name != "test-sub" {
		t.Errorf("Expected name 'test-sub', got '%s'", sub.Name)
	}

	if len(sub.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(sub.Events))
	}

	if len(sub.Filters) != 1 {
		t.Errorf("Expected 1 filter, got %d", len(sub.Filters))
	}

	if sub.Template != "slack" {
		t.Errorf("Expected template 'slack', got '%s'", sub.Template)
	}

	if sub.Headers["X-Custom"] != "value" {
		t.Error("Expected custom header")
	}

	if sub.Secret != "secret-key" {
		t.Error("Expected secret to be set")
	}

	t.Log("✓ Subscription builder works correctly")
}
