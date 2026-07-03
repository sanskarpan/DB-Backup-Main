package enhanced

import (
	"os"
	"testing"
	"time"
)

func TestAIEngineNew(t *testing.T) {
	modelFile := "/tmp/ai_model_test.json"
	defer os.Remove(modelFile)

	ai := NewAIEngine(modelFile)
	if ai == nil {
		t.Fatal("Expected AI engine to be created")
	}

	if ai.modelFile != modelFile {
		t.Errorf("Expected modelFile to be %s, got %s", modelFile, ai.modelFile)
	}
}

func TestAIEngineLoadSave(t *testing.T) {
	modelFile := "/tmp/ai_model_test_save.json"
	defer os.Remove(modelFile)

	ai := NewAIEngine(modelFile)

	// Add some user patterns
	pattern := &UserPattern{
		UserID: "user-001",
		PreferredChannels: map[DeliveryChannel]float64{
			ChannelEmail: 0.8,
			ChannelPush:  0.6,
		},
		PreferredTimes: []TimeWindow{
			{StartHour: 9, EndHour: 17, Weight: 0.9},
		},
		TypeEngagement: map[NotificationType]float64{
			TypeBackupCompleted: 0.95,
			TypeBackupFailed:    1.0,
		},
		PriorityResponse: map[Priority]float64{
			PriorityUrgent: 1.0,
			PriorityHigh:   0.9,
		},
		ReadTimeByType: map[NotificationType]float64{
			TypeBackupCompleted: 30.0,
		},
		ActionRate:         0.75,
		DismissRate:        0.1,
		TotalNotifications: 100,
		LastUpdated:        time.Now(),
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	// Save
	if err := ai.Save(); err != nil {
		t.Fatalf("Failed to save AI model: %v", err)
	}

	// Create new engine and load
	ai2 := NewAIEngine(modelFile)
	if err := ai2.Load(); err != nil {
		t.Fatalf("Failed to load AI model: %v", err)
	}

	// Verify loaded data
	ai2.mu.RLock()
	loadedPattern, exists := ai2.userPatterns["user-001"]
	ai2.mu.RUnlock()

	if !exists {
		t.Fatal("Expected user pattern to be loaded")
	}

	if loadedPattern.ActionRate != 0.75 {
		t.Errorf("Expected ActionRate to be 0.75, got %f", loadedPattern.ActionRate)
	}
}

func TestAIEngineCalculateRelevance(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test.json")

	// Add user pattern
	pattern := &UserPattern{
		UserID: "user-001",
		PreferredChannels: map[DeliveryChannel]float64{
			ChannelEmail: 0.8,
		},
		PreferredTimes: []TimeWindow{
			{StartHour: 9, EndHour: 17, Weight: 0.9},
		},
		TypeEngagement: map[NotificationType]float64{
			TypeBackupCompleted: 0.95,
		},
		PriorityResponse: map[Priority]float64{
			PriorityHigh: 0.9,
		},
		DismissRate: 0.1,
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	notification := &Notification{
		UserID:   "user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityHigh,
		Channels: []DeliveryChannel{ChannelEmail},
	}

	prefs := &NotificationPreferences{
		UserID: "user-001",
	}

	score := ai.CalculateRelevance(notification, prefs)

	if score < 0 || score > 1 {
		t.Errorf("Expected relevance score between 0 and 1, got %f", score)
	}

	// Score should be high for this well-matched notification
	if score < 0.7 {
		t.Errorf("Expected high relevance score, got %f", score)
	}
}

func TestAIEngineCalculateRelevanceNewUser(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_new_user.json")

	// Set up global stats
	ai.globalStats.TypePopularity = map[NotificationType]int{
		TypeBackupCompleted: 100,
		TypeBackupFailed:    50,
	}

	notification := &Notification{
		UserID:   "new-user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityUrgent, // Urgent should get high score
		Channels: []DeliveryChannel{ChannelEmail},
	}

	prefs := &NotificationPreferences{
		UserID: "new-user-001",
	}

	score := ai.CalculateRelevance(notification, prefs)

	// Urgent notifications should get high relevance for new users
	if score < 0.8 {
		t.Errorf("Expected high relevance for urgent notification, got %f", score)
	}
}

func TestAIEnginePredictOptimalDeliveryTime(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_predict.json")

	// Add user pattern with preferred time
	pattern := &UserPattern{
		UserID: "user-001",
		PreferredTimes: []TimeWindow{
			{StartHour: 9, EndHour: 11, Weight: 0.9},
			{StartHour: 14, EndHour: 16, Weight: 0.7},
		},
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	optimalTime := ai.PredictOptimalDeliveryTime("user-001")

	if optimalTime == nil {
		t.Fatal("Expected optimal delivery time to be predicted")
	}

	// Should predict 9 AM (highest weight)
	if optimalTime.Hour() != 9 {
		t.Errorf("Expected hour to be 9, got %d", optimalTime.Hour())
	}
}

func TestAIEngineLearnFromDelivery(t *testing.T) {
	modelFile := "/tmp/ai_test_learn.json"
	defer os.Remove(modelFile)

	ai := NewAIEngine(modelFile)

	notification := &Notification{
		UserID: "user-001",
		Type:   TypeBackupCompleted,
	}

	results := []DeliveryResult{
		{
			Channel: ChannelEmail,
			Success: true,
		},
		{
			Channel: ChannelPush,
			Success: false,
		},
	}

	// Learn from delivery
	ai.LearnFromDelivery(notification, results)

	// Check if pattern was created
	ai.mu.RLock()
	pattern, exists := ai.userPatterns["user-001"]
	ai.mu.RUnlock()

	if !exists {
		t.Fatal("Expected user pattern to be created")
	}

	// Email should have higher preference than push
	emailPref := pattern.PreferredChannels[ChannelEmail]
	pushPref := pattern.PreferredChannels[ChannelPush]

	if emailPref <= pushPref {
		t.Errorf("Expected email preference (%f) to be higher than push (%f)", emailPref, pushPref)
	}
}

func TestAIEngineLearnFromEngagement(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_engage.json")

	// Create initial pattern
	pattern := &UserPattern{
		UserID:           "user-001",
		TypeEngagement:   make(map[NotificationType]float64),
		PriorityResponse: make(map[Priority]float64),
		ActionRate:       0.5,
		DismissRate:      0.3,
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	notification := &Notification{
		UserID:    "user-001",
		Type:      TypeBackupCompleted,
		Priority:  PriorityHigh,
		CreatedAt: time.Now(),
	}

	// Learn from action (positive signal)
	ai.LearnFromEngagement("user-001", notification, "action", 30*time.Second)

	ai.mu.RLock()
	updatedPattern := ai.userPatterns["user-001"]
	ai.mu.RUnlock()

	// Action rate should increase
	if updatedPattern.ActionRate <= 0.5 {
		t.Errorf("Expected ActionRate to increase from 0.5, got %f", updatedPattern.ActionRate)
	}

	// Type engagement should increase (lerp from 0 to 1.0 with alpha 0.25 = 0.25)
	typeEngagement := updatedPattern.TypeEngagement[TypeBackupCompleted]
	if typeEngagement < 0.2 {
		t.Errorf("Expected type engagement >= 0.2, got %f", typeEngagement)
	}
}

func TestAIEngineLearnFromDismiss(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_dismiss.json")

	pattern := &UserPattern{
		UserID:         "user-001",
		TypeEngagement: make(map[NotificationType]float64),
		DismissRate:    0.2,
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	notification := &Notification{
		UserID:    "user-001",
		Type:      TypeBackupStarted,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
	}

	// Learn from dismiss (negative signal)
	ai.LearnFromEngagement("user-001", notification, "dismiss", 5*time.Second)

	ai.mu.RLock()
	updatedPattern := ai.userPatterns["user-001"]
	ai.mu.RUnlock()

	// Dismiss rate should increase
	if updatedPattern.DismissRate <= 0.2 {
		t.Errorf("Expected DismissRate to increase from 0.2, got %f", updatedPattern.DismissRate)
	}

	// Type engagement should decrease
	typeEngagement := updatedPattern.TypeEngagement[TypeBackupStarted]
	if typeEngagement > 0.5 {
		t.Errorf("Expected low type engagement, got %f", typeEngagement)
	}
}

func TestAIEngineSuggestQuietHours(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_quiet.json")

	pattern := &UserPattern{
		UserID: "user-001",
		PreferredTimes: []TimeWindow{
			{StartHour: 9, EndHour: 17, Weight: 0.9},
		},
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	quietHours := ai.SuggestQuietHours("user-001")

	if len(quietHours) == 0 {
		t.Fatal("Expected quiet hours to be suggested")
	}

	// Should suggest quiet hours outside of 9-17 (e.g., 22-08)
	firstPeriod := quietHours[0]
	if firstPeriod.Start == "" || firstPeriod.End == "" {
		t.Error("Expected quiet hours to have start and end times")
	}
}

func TestAIEngineLerpFunction(t *testing.T) {
	ai := NewAIEngine("/tmp/ai_test_lerp.json")

	// Test linear interpolation
	result := ai.lerp(0.5, 1.0, 0.5)
	expected := 0.75 // 0.5 + 0.5 * (1.0 - 0.5) = 0.75

	if result != expected {
		t.Errorf("Expected lerp result to be %f, got %f", expected, result)
	}

	// Test with zero alpha (no change)
	result = ai.lerp(0.5, 1.0, 0.0)
	if result != 0.5 {
		t.Errorf("Expected lerp with alpha=0 to return current value, got %f", result)
	}

	// Test with alpha=1 (full change)
	result = ai.lerp(0.5, 1.0, 1.0)
	if result != 1.0 {
		t.Errorf("Expected lerp with alpha=1 to return target value, got %f", result)
	}
}

func BenchmarkCalculateRelevance(b *testing.B) {
	ai := NewAIEngine("/tmp/ai_bench.json")

	pattern := &UserPattern{
		UserID: "user-001",
		PreferredChannels: map[DeliveryChannel]float64{
			ChannelEmail: 0.8,
			ChannelPush:  0.6,
		},
		PreferredTimes: []TimeWindow{
			{StartHour: 9, EndHour: 17, Weight: 0.9},
		},
		TypeEngagement: map[NotificationType]float64{
			TypeBackupCompleted: 0.95,
		},
		PriorityResponse: map[Priority]float64{
			PriorityHigh: 0.9,
		},
		DismissRate: 0.1,
	}

	ai.mu.Lock()
	ai.userPatterns["user-001"] = pattern
	ai.mu.Unlock()

	notification := &Notification{
		UserID:   "user-001",
		Type:     TypeBackupCompleted,
		Priority: PriorityHigh,
		Channels: []DeliveryChannel{ChannelEmail},
	}

	prefs := &NotificationPreferences{
		UserID: "user-001",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ai.CalculateRelevance(notification, prefs)
	}
}
