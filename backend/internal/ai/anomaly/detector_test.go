package anomaly

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDetector(t *testing.T) (*Detector, *BaselineTrainer) {
	trainer := NewBaselineTrainer(30)

	// Create baseline with some variance
	now := time.Now()
	for i := 0; i < 30; i++ {
		// Add variance: sizes between 950k and 1050k, durations between 55 and 65
		variance := float64(i%10 - 5) // -5 to +5
		trainer.AddMetric(BackupMetric{
			BackupID:     "baseline-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Size:         1000000 + int64(variance*10000), // ±50k variance
			Duration:     60.0 + variance,                 // ±5 seconds variance
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	config := DefaultDetectorConfig()
	detector := NewDetector(config, trainer)

	return detector, trainer
}

func TestNewDetector(t *testing.T) {
	trainer := NewBaselineTrainer(30)
	config := DefaultDetectorConfig()
	detector := NewDetector(config, trainer)

	assert.NotNil(t, detector)
	assert.NotNil(t, detector.config)
	assert.NotNil(t, detector.trainer)
	assert.NotNil(t, detector.recentFailures)
	assert.NotNil(t, detector.anomalyHistory)
}

func TestDetector_DetectSizeAnomaly(t *testing.T) {
	detector, _ := setupTestDetector(t)

	// Create metric with significantly larger size (5x normal)
	metric := BackupMetric{
		BackupID:     "test-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         5000000, // 5x normal size
		Duration:     60.0,
		Success:      true,
	}

	ctx := context.Background()
	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	// Should detect size increase
	foundSizeAnomaly := false
	for _, a := range anomalies {
		if a.Type == AnomalyTypeSizeIncrease {
			foundSizeAnomaly = true
			assert.Greater(t, a.Score, 50.0)
			assert.NotEmpty(t, a.Description)
			assert.NotEmpty(t, a.Recommendation)
		}
	}

	assert.True(t, foundSizeAnomaly, "Should detect size anomaly")
}

func TestDetector_DetectDurationAnomaly(t *testing.T) {
	detector, _ := setupTestDetector(t)

	// Create metric with significantly longer duration (10x normal)
	metric := BackupMetric{
		BackupID:     "test-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         1000000,
		Duration:     600.0, // 10x normal duration
		Success:      true,
	}

	ctx := context.Background()
	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	// Should detect duration increase
	foundDurationAnomaly := false
	for _, a := range anomalies {
		if a.Type == AnomalyTypeDurationIncrease {
			foundDurationAnomaly = true
			assert.Greater(t, a.Score, 50.0)
		}
	}

	assert.True(t, foundDurationAnomaly, "Should detect duration anomaly")
}

func TestDetector_DetectFailureCluster(t *testing.T) {
	detector, _ := setupTestDetector(t)

	ctx := context.Background()

	// Create multiple failures in short period
	for i := 0; i < 5; i++ {
		metric := BackupMetric{
			BackupID:     "fail-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Minute),
			Size:         1000000,
			Duration:     60.0,
			Success:      false,
			ErrorMessage: "Test failure",
		}

		anomalies, err := detector.Detect(ctx, metric)
		assert.NoError(t, err)

		// Should detect cluster after threshold
		if i >= detector.config.FailureClusterThreshold-1 {
			foundCluster := false
			for _, a := range anomalies {
				if a.Type == AnomalyTypeFailureCluster {
					foundCluster = true
					assert.Equal(t, AnomalySeverityCritical, a.Severity)
					assert.Greater(t, a.Score, 80.0)
				}
			}
			assert.True(t, foundCluster, "Should detect failure cluster")
		}
	}
}

func TestDetector_NoAnomalyForNormalMetric(t *testing.T) {
	detector, _ := setupTestDetector(t)

	// Create normal metric matching baseline
	metric := BackupMetric{
		BackupID:     "normal-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         1000000, // Same as baseline
		Duration:     60.0,    // Same as baseline
		Success:      true,
	}

	ctx := context.Background()
	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)
	assert.Empty(t, anomalies, "Should not detect anomalies for normal metrics")
}

func TestDetector_NoBaselineNoDetection(t *testing.T) {
	// Create detector without training
	trainer := NewBaselineTrainer(30)
	config := DefaultDetectorConfig()
	detector := NewDetector(config, trainer)

	metric := BackupMetric{
		BackupID:     "test-1",
		DatabaseName: "unknown-db",
		Timestamp:    time.Now(),
		Size:         5000000,
		Duration:     600.0,
		Success:      true,
	}

	ctx := context.Background()
	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)
	assert.Nil(t, anomalies, "Should not detect anomalies without baseline")
}

func TestDetector_RegisterAlertCallback(t *testing.T) {
	detector, _ := setupTestDetector(t)

	var mu sync.Mutex
	callbackCalled := false
	var receivedAnomaly AnomalyReport

	detector.RegisterAlertCallback(func(anomaly AnomalyReport) {
		mu.Lock()
		callbackCalled = true
		receivedAnomaly = anomaly
		mu.Unlock()
	})

	// Trigger anomaly
	metric := BackupMetric{
		BackupID:     "test-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         5000000,
		Duration:     60.0,
		Success:      true,
	}

	ctx := context.Background()
	_, err := detector.Detect(ctx, metric)
	assert.NoError(t, err)

	// Wait for callback (it's async)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	anomaly := receivedAnomaly
	mu.Unlock()

	assert.True(t, called, "Callback should be called")
	assert.NotEmpty(t, anomaly.ID)
	assert.Equal(t, "test-db", anomaly.DatabaseName)
}

func TestDetector_GetAnomalyHistory(t *testing.T) {
	detector, _ := setupTestDetector(t)

	ctx := context.Background()

	// Generate several anomalies
	for i := 0; i < 5; i++ {
		metric := BackupMetric{
			BackupID:     "test-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    time.Now(),
			Size:         5000000,
			Duration:     60.0,
			Success:      true,
		}

		_, err := detector.Detect(ctx, metric)
		assert.NoError(t, err)
	}

	history := detector.GetAnomalyHistory(10)
	assert.NotEmpty(t, history)
	assert.LessOrEqual(t, len(history), 10)
}

func TestDetector_CalculateSeverity(t *testing.T) {
	detector, _ := setupTestDetector(t)

	tests := []struct {
		deviation float64
		expected  AnomalySeverity
	}{
		{2.5, AnomalySeverityLow},
		{3.5, AnomalySeverityMedium},
		{4.5, AnomalySeverityHigh},
		{5.5, AnomalySeverityCritical},
	}

	for _, tt := range tests {
		severity := detector.calculateSeverity(tt.deviation)
		assert.Equal(t, tt.expected, severity, "Deviation %.1f", tt.deviation)
	}
}

func TestDetector_CalculateScore(t *testing.T) {
	detector, _ := setupTestDetector(t)

	tests := []struct {
		deviation float64
		minScore  float64
		maxScore  float64
	}{
		{2.0, 0.0, 50.0},
		{3.5, 50.0, 70.0},
		{4.5, 70.0, 90.0},
		{6.0, 90.0, 100.0},
	}

	for _, tt := range tests {
		score := detector.calculateScore(tt.deviation)
		assert.GreaterOrEqual(t, score, tt.minScore, "Deviation %.1f", tt.deviation)
		assert.LessOrEqual(t, score, tt.maxScore, "Deviation %.1f", tt.deviation)
	}
}

func TestDetector_DetectUnusualTiming(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	// Create baseline with all backups at 2 AM
	now := time.Now()
	for i := 0; i < 30; i++ {
		timestamp := time.Date(now.Year(), now.Month(), now.Day()-i, 2, 0, 0, 0, time.UTC)
		trainer.AddMetric(BackupMetric{
			BackupID:     "baseline-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    timestamp,
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	config := DefaultDetectorConfig()
	detector := NewDetector(config, trainer)

	// Create backup at unusual time (2 PM instead of 2 AM)
	timestamp := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, time.UTC)
	metric := BackupMetric{
		BackupID:     "unusual-1",
		DatabaseName: "test-db",
		Timestamp:    timestamp,
		Size:         1000000,
		Duration:     60.0,
		Success:      true,
	}

	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)

	// Should detect unusual timing
	foundTimingAnomaly := false
	for _, a := range anomalies {
		if a.Type == AnomalyTypeUnusualTiming {
			foundTimingAnomaly = true
			assert.Equal(t, AnomalySeverityLow, a.Severity)
		}
	}

	assert.True(t, foundTimingAnomaly, "Should detect unusual timing")
}

func TestDetector_MultipleSimultaneousAnomalies(t *testing.T) {
	detector, _ := setupTestDetector(t)

	// Create metric with multiple anomalies
	metric := BackupMetric{
		BackupID:     "multi-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         5000000, // Anomalous size
		Duration:     600.0,   // Anomalous duration
		Success:      true,
	}

	ctx := context.Background()
	anomalies, err := detector.Detect(ctx, metric)

	assert.NoError(t, err)
	assert.NotEmpty(t, anomalies)

	// Should detect both size and duration anomalies
	types := make(map[AnomalyType]bool)
	for _, a := range anomalies {
		types[a.Type] = true
	}

	assert.True(t, types[AnomalyTypeSizeIncrease], "Should detect size anomaly")
	assert.True(t, types[AnomalyTypeDurationIncrease], "Should detect duration anomaly")
}

func TestDefaultDetectorConfig(t *testing.T) {
	config := DefaultDetectorConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 3.0, config.SizeDeviationThreshold)
	assert.Equal(t, 3.0, config.DurationDeviationThreshold)
	assert.Equal(t, 24*time.Hour, config.FailureClusterWindow)
	assert.Equal(t, 3, config.FailureClusterThreshold)
	assert.Equal(t, 0.95, config.SuccessRateThreshold)
	assert.Equal(t, 10, config.MinSamplesForDetection)
}

func TestAnomalyReport_Fields(t *testing.T) {
	report := AnomalyReport{
		ID:           "anomaly-123",
		DatabaseName: "test-db",
		BackupID:     "backup-123",
		Type:         AnomalyTypeSizeIncrease,
		Severity:     AnomalySeverityHigh,
		Score:        85.5,
		DetectedAt:   time.Now(),
		Deviations: map[string]float64{
			"size": 4.2,
		},
		Description:    "Size increased significantly",
		Recommendation: "Investigate data growth",
	}

	assert.Equal(t, "anomaly-123", report.ID)
	assert.Equal(t, "test-db", report.DatabaseName)
	assert.Equal(t, AnomalyTypeSizeIncrease, report.Type)
	assert.Equal(t, AnomalySeverityHigh, report.Severity)
	assert.Equal(t, 85.5, report.Score)
	assert.Contains(t, report.Deviations, "size")
	assert.Equal(t, 4.2, report.Deviations["size"])
}
