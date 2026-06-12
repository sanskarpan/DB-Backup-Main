package prediction

import (
	"context"
	"testing"
	"time"
)

func TestNewFailurePredictor(t *testing.T) {
	fp := NewFailurePredictor(nil)
	if fp == nil {
		t.Fatal("Expected non-nil failure predictor")
	}

	if fp.config == nil {
		t.Fatal("Expected default config")
	}

	if fp.config.PredictionWindow != 48*time.Hour {
		t.Errorf("Expected 48h prediction window, got %v", fp.config.PredictionWindow)
	}
}

func TestAddBackupEvent(t *testing.T) {
	fp := NewFailurePredictor(nil)

	event := BackupEvent{
		ID:           "event-1",
		DatabaseName: "testdb",
		Timestamp:    time.Now(),
		Success:      true,
		Duration:     30 * time.Minute,
		Size:         1024 * 1024 * 1024,
		DayOfWeek:    time.Monday,
		HourOfDay:    14,
		SystemLoad:   0.5,
		DiskUsage:    70.0,
	}

	fp.AddBackupEvent(event)

	fp.mu.RLock()
	events := fp.historicalData["testdb"]
	fp.mu.RUnlock()

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].ID != "event-1" {
		t.Errorf("Expected event-1, got %s", events[0].ID)
	}
}

func TestAddBackupEvent_RetentionLimit(t *testing.T) {
	fp := NewFailurePredictor(nil)

	// Add old events (>90 days)
	oldEvent := BackupEvent{
		ID:           "old-event",
		DatabaseName: "testdb",
		Timestamp:    time.Now().AddDate(0, 0, -100),
		Success:      true,
	}
	fp.AddBackupEvent(oldEvent)

	// Add recent event
	recentEvent := BackupEvent{
		ID:           "recent-event",
		DatabaseName: "testdb",
		Timestamp:    time.Now(),
		Success:      true,
	}
	fp.AddBackupEvent(recentEvent)

	fp.mu.RLock()
	events := fp.historicalData["testdb"]
	fp.mu.RUnlock()

	// Old event should be filtered out
	if len(events) != 1 {
		t.Fatalf("Expected 1 event after retention, got %d", len(events))
	}

	if events[0].ID != "recent-event" {
		t.Errorf("Expected recent-event, got %s", events[0].ID)
	}
}

func TestAnalyzePatterns_InsufficientData(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Add only a few events (less than MinHistoryDays)
	for i := 0; i < 5; i++ {
		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      true,
		}
		fp.AddBackupEvent(event)
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	// Should not create patterns with insufficient data
	patterns := fp.GetPatterns()
	if len(patterns) > 0 {
		t.Errorf("Expected no patterns with insufficient data, got %d", len(patterns))
	}
}

func TestAnalyzePatterns_TimePatterns(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Create events with failures on Mondays
	for i := 0; i < 40; i++ {
		day := time.Now().AddDate(0, 0, -i)
		success := day.Weekday() != time.Monday

		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    day,
			Success:      success,
			DayOfWeek:    day.Weekday(),
			HourOfDay:    14,
			Duration:     30 * time.Minute,
		}
		fp.AddBackupEvent(event)
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	patterns := fp.GetPatterns()
	if len(patterns) == 0 {
		t.Fatal("Expected patterns to be identified")
	}

	pattern := patterns["testdb"]
	if pattern == nil {
		t.Fatal("Expected pattern for testdb")
	}

	// Check if Monday is identified as high-risk
	hasMonday := false
	for _, day := range pattern.TimePatterns.HighRiskDays {
		if day == time.Monday {
			hasMonday = true
			break
		}
	}

	if !hasMonday {
		t.Error("Expected Monday to be identified as high-risk day")
	}
}

func TestAnalyzePatterns_ResourcePatterns(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Create events that fail when disk usage is high
	for i := 0; i < 40; i++ {
		diskUsage := 70.0 + float64(i%20)
		success := diskUsage < 85.0

		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      success,
			DiskUsage:    diskUsage,
			SystemLoad:   0.5,
			DayOfWeek:    time.Monday,
			HourOfDay:    14,
		}
		fp.AddBackupEvent(event)
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	patterns := fp.GetPatterns()
	pattern := patterns["testdb"]
	if pattern == nil {
		t.Fatal("Expected pattern for testdb")
	}

	// Check resource patterns
	if pattern.ResourcePatterns.DiskThreshold < 80.0 {
		t.Errorf("Expected disk threshold around 85, got %.1f", pattern.ResourcePatterns.DiskThreshold)
	}
}

func TestAnalyzePatterns_ErrorPatterns(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Create events with recurring errors
	for i := 0; i < 40; i++ {
		success := i%3 != 0
		errorType := ""
		if !success {
			errorType = "NETWORK_ERROR"
		}

		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      success,
			ErrorType:    errorType,
			DayOfWeek:    time.Monday,
			HourOfDay:    14,
		}
		fp.AddBackupEvent(event)
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	patterns := fp.GetPatterns()
	pattern := patterns["testdb"]
	if pattern == nil {
		t.Fatal("Expected pattern for testdb")
	}

	// Check error patterns
	if pattern.CommonErrors["NETWORK_ERROR"] == 0 {
		t.Error("Expected NETWORK_ERROR to be identified")
	}
}

func TestPredictFailures_TimeWindow(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Create pattern with Monday failures
	for i := 0; i < 40; i++ {
		day := time.Now().AddDate(0, 0, -i)
		success := day.Weekday() != time.Monday

		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    day,
			Success:      success,
			DayOfWeek:    day.Weekday(),
			HourOfDay:    14,
			Duration:     30 * time.Minute,
		}
		fp.AddBackupEvent(event)
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	predictions, err := fp.PredictFailures(ctx, "testdb")
	if err != nil {
		t.Fatalf("PredictFailures failed: %v", err)
	}

	// Should have predictions if next 48 hours includes Monday
	t.Logf("Got %d predictions", len(predictions))

	for _, pred := range predictions {
		if pred.Confidence < fp.config.ConfidenceThreshold {
			t.Errorf("Prediction confidence %.1f below threshold %.1f", pred.Confidence, fp.config.ConfidenceThreshold)
		}

		if len(pred.PreventiveActions) == 0 {
			t.Error("Expected preventive actions to be generated")
		}
	}
}

func TestPredictFailures_NoPattern(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	predictions, err := fp.PredictFailures(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("PredictFailures failed: %v", err)
	}

	if len(predictions) != 0 {
		t.Errorf("Expected no predictions for nonexistent database, got %d", len(predictions))
	}
}

func TestDeterminePatternType_Trend(t *testing.T) {
	fp := NewFailurePredictor(nil)

	// Create events with increasing failure rate
	events := make([]BackupEvent, 0)
	for i := 0; i < 40; i++ {
		// Recent events (last 7 days) have higher failure rate
		success := true
		if i < 7 {
			success = i%2 == 0 // 50% failure rate recently
		} else {
			success = i%10 != 0 // 10% failure rate historically
		}

		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      success,
			DayOfWeek:    time.Monday,
			HourOfDay:    14,
		}
		events = append(events, event)
	}

	pattern := &FailurePattern{
		CommonErrors: make(map[string]int),
		TimePatterns: &TimePattern{
			HighRiskDays:  []time.Weekday{},
			HighRiskHours: []int{},
		},
		ResourcePatterns: &ResourcePattern{},
	}

	patternType, confidence := fp.determinePatternType(events, pattern)

	if patternType != PatternTypeTrend {
		t.Errorf("Expected trend pattern, got %s", patternType)
	}

	if confidence < 50.0 {
		t.Errorf("Expected confidence >= 50, got %.1f", confidence)
	}
}

func TestGeneratePreventiveActions_TimeWindow(t *testing.T) {
	fp := NewFailurePredictor(nil)

	pattern := &FailurePattern{
		DatabaseName: "testdb",
		PatternType:  PatternTypeTimeWindow,
		TimePatterns: &TimePattern{
			HighRiskDays:  []time.Weekday{time.Monday},
			HighRiskHours: []int{23, 0, 1},
		},
		CommonErrors:     make(map[string]int),
		ResourcePatterns: &ResourcePattern{},
	}

	prediction := &FailurePrediction{
		DatabaseName: "testdb",
	}

	actions := fp.generatePreventiveActions(pattern, prediction)

	if len(actions) == 0 {
		t.Fatal("Expected preventive actions")
	}

	// Should recommend rescheduling
	hasReschedule := false
	for _, action := range actions {
		if action.Priority == 1 {
			hasReschedule = true
			break
		}
	}

	if !hasReschedule {
		t.Error("Expected high-priority reschedule action")
	}
}

func TestGeneratePreventiveActions_ResourceBased(t *testing.T) {
	fp := NewFailurePredictor(nil)

	pattern := &FailurePattern{
		DatabaseName: "testdb",
		PatternType:  PatternTypeResourceBased,
		TimePatterns: &TimePattern{
			HighRiskDays:  []time.Weekday{},
			HighRiskHours: []int{},
		},
		ResourcePatterns: &ResourcePattern{
			DiskThreshold: 90.0,
			LoadThreshold: 0.9,
		},
		CommonErrors: make(map[string]int),
	}

	prediction := &FailurePrediction{
		DatabaseName: "testdb",
	}

	actions := fp.generatePreventiveActions(pattern, prediction)

	if len(actions) == 0 {
		t.Fatal("Expected preventive actions")
	}

	// Should recommend disk cleanup
	hasDiskAction := false
	for _, action := range actions {
		if action.Action == "Free up disk space before backup" {
			hasDiskAction = true
			break
		}
	}

	if !hasDiskAction {
		t.Error("Expected disk cleanup action")
	}
}

func TestGeneratePreventiveActions_ErrorSpecific(t *testing.T) {
	fp := NewFailurePredictor(nil)

	pattern := &FailurePattern{
		DatabaseName: "testdb",
		PatternType:  PatternTypeRecurring,
		TimePatterns: &TimePattern{
			HighRiskDays:  []time.Weekday{},
			HighRiskHours: []int{},
		},
		ResourcePatterns: &ResourcePattern{},
		CommonErrors: map[string]int{
			"NETWORK_ERROR": 15,
			"DISK_FULL":     5,
		},
	}

	prediction := &FailurePrediction{
		DatabaseName: "testdb",
	}

	actions := fp.generatePreventiveActions(pattern, prediction)

	// Should address the most common error (NETWORK_ERROR)
	hasNetworkAction := false
	for _, action := range actions {
		if action.Action == "Address recurring error: NETWORK_ERROR" {
			hasNetworkAction = true
			if action.Description == "" {
				t.Error("Expected description for network error action")
			}
			break
		}
	}

	if !hasNetworkAction {
		t.Error("Expected action for most common error")
	}
}

func TestGetStatistics(t *testing.T) {
	fp := NewFailurePredictor(nil)
	ctx := context.Background()

	// Add events for multiple databases
	for db := 0; db < 3; db++ {
		dbName := "testdb-" + string(rune(db))
		for i := 0; i < 40; i++ {
			event := BackupEvent{
				ID:           "event-" + string(rune(i)),
				DatabaseName: dbName,
				Timestamp:    time.Now().AddDate(0, 0, -i),
				Success:      i%2 == 0,
				DayOfWeek:    time.Monday,
				HourOfDay:    14,
			}
			fp.AddBackupEvent(event)
		}
	}

	err := fp.AnalyzePatterns(ctx)
	if err != nil {
		t.Fatalf("AnalyzePatterns failed: %v", err)
	}

	stats := fp.GetStatistics()

	totalDatabases, ok := stats["total_databases"].(int)
	if !ok || totalDatabases != 3 {
		t.Errorf("Expected 3 total databases, got %v", totalDatabases)
	}

	patternsFound, ok := stats["patterns_found"].(int)
	if !ok || patternsFound != 3 {
		t.Errorf("Expected 3 patterns found, got %v", patternsFound)
	}
}

func TestCalculateRiskFactors(t *testing.T) {
	fp := NewFailurePredictor(nil)

	events := make([]BackupEvent, 0)
	for i := 0; i < 40; i++ {
		event := BackupEvent{
			ID:           "event-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      i%3 != 0, // 33% failure rate
			DiskUsage:    90.0,
			SystemLoad:   0.9,
			ErrorType:    "TIMEOUT",
			DayOfWeek:    time.Monday,
			HourOfDay:    23,
		}
		events = append(events, event)
	}

	pattern := &FailurePattern{
		DatabaseName: "testdb",
		Frequency:    0.5, // High frequency
		TimePatterns: &TimePattern{
			HighRiskHours: []int{23},
		},
		ResourcePatterns: &ResourcePattern{
			DiskThreshold: 90.0,
			LoadThreshold: 0.9,
		},
		CommonErrors: map[string]int{
			"TIMEOUT": 10,
		},
	}

	riskFactors := fp.calculateRiskFactors(events, pattern)

	if len(riskFactors) == 0 {
		t.Fatal("Expected risk factors to be identified")
	}

	// Should identify disk constraint
	hasDiskRisk := false
	hasLoadRisk := false
	hasTimeRisk := false
	hasErrorRisk := false

	for _, rf := range riskFactors {
		if rf.Factor == "Disk Space Constraint" {
			hasDiskRisk = true
		}
		if rf.Factor == "High System Load" {
			hasLoadRisk = true
		}
		if rf.Factor == "Time Window Risk" {
			hasTimeRisk = true
		}
		if rf.Factor == "Recurring Error Type" {
			hasErrorRisk = true
		}
	}

	if !hasDiskRisk {
		t.Error("Expected disk space risk factor")
	}
	if !hasLoadRisk {
		t.Error("Expected system load risk factor")
	}
	if !hasTimeRisk {
		t.Error("Expected time window risk factor")
	}
	if !hasErrorRisk {
		t.Error("Expected recurring error risk factor")
	}
}

func TestDetermineSeverity(t *testing.T) {
	fp := NewFailurePredictor(nil)

	tests := []struct {
		riskScore float64
		expected  PredictionSeverity
	}{
		{95.0, PredictionSeverityCritical},
		{85.0, PredictionSeverityHigh},
		{70.0, PredictionSeverityMedium},
		{50.0, PredictionSeverityLow},
	}

	for _, tt := range tests {
		result := fp.determineSeverity(tt.riskScore)
		if result != tt.expected {
			t.Errorf("riskScore %.1f: expected %s, got %s", tt.riskScore, tt.expected, result)
		}
	}
}
