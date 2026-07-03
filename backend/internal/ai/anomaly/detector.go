// Package anomaly provides AI-powered anomaly detection for backup operations
package anomaly

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// AnomalyType represents the type of anomaly detected.
type AnomalyType string

const (
	// AnomalyTypeSizeIncrease indicates backup size increased significantly.
	AnomalyTypeSizeIncrease AnomalyType = "SIZE_INCREASE"
	// AnomalyTypeSizeDecrease indicates backup size decreased significantly.
	AnomalyTypeSizeDecrease AnomalyType = "SIZE_DECREASE"
	// AnomalyTypeDurationIncrease indicates backup took longer than expected.
	AnomalyTypeDurationIncrease AnomalyType = "DURATION_INCREASE"
	// AnomalyTypeFailureCluster indicates multiple failures in short period.
	AnomalyTypeFailureCluster AnomalyType = "FAILURE_CLUSTER"
	// AnomalyTypeUnusualTiming indicates backup at unusual time.
	AnomalyTypeUnusualTiming AnomalyType = "UNUSUAL_TIMING"
	// AnomalyTypeSuccessRateDrop indicates success rate dropped.
	AnomalyTypeSuccessRateDrop AnomalyType = "SUCCESS_RATE_DROP"
)

// AnomalySeverity represents the severity of an anomaly.
type AnomalySeverity string

const (
	// AnomalySeverityLow indicates minor deviation.
	AnomalySeverityLow AnomalySeverity = "LOW"
	// AnomalySeverityMedium indicates moderate deviation.
	AnomalySeverityMedium AnomalySeverity = "MEDIUM"
	// AnomalySeverityHigh indicates significant deviation.
	AnomalySeverityHigh AnomalySeverity = "HIGH"
	// AnomalySeverityCritical indicates critical deviation requiring immediate attention.
	AnomalySeverityCritical AnomalySeverity = "CRITICAL"
)

// AnomalyReport represents a detected anomaly.
type AnomalyReport struct {
	ID             string
	DatabaseName   string
	BackupID       string
	Type           AnomalyType
	Severity       AnomalySeverity
	Score          float64 // Anomaly score 0-100
	DetectedAt     time.Time
	Metric         BackupMetric
	Baseline       *BaselineModel
	Deviations     map[string]float64 // Key: metric name, Value: standard deviations from mean
	Description    string
	Recommendation string
}

// DetectorConfig represents configuration for anomaly detection.
type DetectorConfig struct {
	// SizeDeviationThreshold in standard deviations (default: 3.0)
	SizeDeviationThreshold float64

	// DurationDeviationThreshold in standard deviations (default: 3.0)
	DurationDeviationThreshold float64

	// FailureClusterWindow time window to check for failure clusters
	FailureClusterWindow time.Duration

	// FailureClusterThreshold number of failures to trigger alert
	FailureClusterThreshold int

	// SuccessRateThreshold minimum success rate (default: 0.95)
	SuccessRateThreshold float64

	// MinSamplesForDetection minimum samples before detection is enabled
	MinSamplesForDetection int
}

// DefaultDetectorConfig returns default configuration.
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		SizeDeviationThreshold:     3.0,
		DurationDeviationThreshold: 3.0,
		FailureClusterWindow:       24 * time.Hour,
		FailureClusterThreshold:    3,
		SuccessRateThreshold:       0.95,
		MinSamplesForDetection:     10,
	}
}

// Detector detects anomalies in backup operations.
type Detector struct {
	config  *DetectorConfig
	trainer *BaselineTrainer

	mu             sync.RWMutex
	recentFailures []BackupMetric
	anomalyHistory []AnomalyReport
	alertCallbacks []func(AnomalyReport)
}

// NewDetector creates a new anomaly detector.
func NewDetector(config *DetectorConfig, trainer *BaselineTrainer) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}

	return &Detector{
		config:         config,
		trainer:        trainer,
		recentFailures: make([]BackupMetric, 0),
		anomalyHistory: make([]AnomalyReport, 0),
		alertCallbacks: make([]func(AnomalyReport), 0),
	}
}

// RegisterAlertCallback registers a callback for anomaly alerts.
func (d *Detector) RegisterAlertCallback(callback func(AnomalyReport)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.alertCallbacks = append(d.alertCallbacks, callback)
}

// Detect detects anomalies in a backup metric.
func (d *Detector) Detect(ctx context.Context, metric BackupMetric) ([]AnomalyReport, error) {
	// Get baseline model for this database
	baseline, err := d.trainer.GetModel(metric.DatabaseName)
	if err != nil {
		// No baseline yet - not an error, just can't detect anomalies
		return nil, nil
	}

	if baseline.SampleCount < d.config.MinSamplesForDetection {
		// Insufficient training data
		return nil, nil
	}

	var anomalies []AnomalyReport

	// Track failures for cluster detection
	if !metric.Success {
		d.trackFailure(metric)
	}

	// Check for size anomalies
	if sizeAnomaly := d.detectSizeAnomaly(metric, baseline); sizeAnomaly != nil {
		anomalies = append(anomalies, *sizeAnomaly)
	}

	// Check for duration anomalies
	if durationAnomaly := d.detectDurationAnomaly(metric, baseline); durationAnomaly != nil {
		anomalies = append(anomalies, *durationAnomaly)
	}

	// Check for failure clusters
	if failureCluster := d.detectFailureCluster(metric, baseline); failureCluster != nil {
		anomalies = append(anomalies, *failureCluster)
	}

	// Check for unusual timing
	if timingAnomaly := d.detectUnusualTiming(metric, baseline); timingAnomaly != nil {
		anomalies = append(anomalies, *timingAnomaly)
	}

	// Store anomalies and trigger callbacks
	for _, anomaly := range anomalies {
		d.recordAnomaly(anomaly)
		d.triggerCallbacks(anomaly)
	}

	return anomalies, nil
}

// detectSizeAnomaly detects size-related anomalies.
func (d *Detector) detectSizeAnomaly(metric BackupMetric, baseline *BaselineModel) *AnomalyReport {
	if baseline.SizeStdDev == 0 {
		return nil // No variance in historical data
	}

	size := float64(metric.Size)
	deviation := (size - baseline.SizeMean) / baseline.SizeStdDev

	if math.Abs(deviation) < d.config.SizeDeviationThreshold {
		return nil // Within normal range
	}

	var anomalyType AnomalyType
	var description string

	if deviation > 0 {
		anomalyType = AnomalyTypeSizeIncrease
		description = fmt.Sprintf("Backup size increased by %.1f%% (%.2f standard deviations above mean)",
			((size-baseline.SizeMean)/baseline.SizeMean)*100, deviation)
	} else {
		anomalyType = AnomalyTypeSizeDecrease
		description = fmt.Sprintf("Backup size decreased by %.1f%% (%.2f standard deviations below mean)",
			((baseline.SizeMean-size)/baseline.SizeMean)*100, math.Abs(deviation))
	}

	severity := d.calculateSeverity(math.Abs(deviation))
	score := d.calculateScore(math.Abs(deviation))

	return &AnomalyReport{
		ID:           generateAnomalyID(),
		DatabaseName: metric.DatabaseName,
		BackupID:     metric.BackupID,
		Type:         anomalyType,
		Severity:     severity,
		Score:        score,
		DetectedAt:   time.Now(),
		Metric:       metric,
		Baseline:     baseline,
		Deviations: map[string]float64{
			"size": deviation,
		},
		Description:    description,
		Recommendation: "Investigate data growth/shrinkage. Check for schema changes, data deletions, or database corruption.",
	}
}

// detectDurationAnomaly detects duration-related anomalies.
func (d *Detector) detectDurationAnomaly(metric BackupMetric, baseline *BaselineModel) *AnomalyReport {
	if baseline.DurationStdDev == 0 {
		return nil
	}

	deviation := (metric.Duration - baseline.DurationMean) / baseline.DurationStdDev

	if deviation < d.config.DurationDeviationThreshold {
		return nil // Within normal range or faster (not a problem)
	}

	description := fmt.Sprintf("Backup duration increased by %.1f%% (%.2f standard deviations above mean)",
		((metric.Duration-baseline.DurationMean)/baseline.DurationMean)*100, deviation)

	severity := d.calculateSeverity(deviation)
	score := d.calculateScore(deviation)

	return &AnomalyReport{
		ID:           generateAnomalyID(),
		DatabaseName: metric.DatabaseName,
		BackupID:     metric.BackupID,
		Type:         AnomalyTypeDurationIncrease,
		Severity:     severity,
		Score:        score,
		DetectedAt:   time.Now(),
		Metric:       metric,
		Baseline:     baseline,
		Deviations: map[string]float64{
			"duration": deviation,
		},
		Description:    description,
		Recommendation: "Check database performance, network bandwidth, and storage I/O. Consider database optimization.",
	}
}

// detectFailureCluster detects clusters of backup failures.
func (d *Detector) detectFailureCluster(metric BackupMetric, baseline *BaselineModel) *AnomalyReport {
	if metric.Success {
		return nil // Only check for failed backups
	}

	d.mu.RLock()
	failureCount := len(d.recentFailures)
	d.mu.RUnlock()

	if failureCount < d.config.FailureClusterThreshold {
		return nil
	}

	description := fmt.Sprintf("%d backup failures detected in the last %v",
		failureCount, d.config.FailureClusterWindow)

	return &AnomalyReport{
		ID:           generateAnomalyID(),
		DatabaseName: metric.DatabaseName,
		BackupID:     metric.BackupID,
		Type:         AnomalyTypeFailureCluster,
		Severity:     AnomalySeverityCritical,
		Score:        90.0,
		DetectedAt:   time.Now(),
		Metric:       metric,
		Baseline:     baseline,
		Deviations: map[string]float64{
			"failure_count": float64(failureCount),
		},
		Description:    description,
		Recommendation: "CRITICAL: Multiple backup failures detected. Investigate backup infrastructure, database health, and error logs immediately.",
	}
}

// detectUnusualTiming detects backups at unusual times.
func (d *Detector) detectUnusualTiming(metric BackupMetric, baseline *BaselineModel) *AnomalyReport {
	hour := metric.Timestamp.Hour()
	dayOfWeek := int(metric.Timestamp.Weekday())

	// Calculate expected frequency for this time
	hourlyCount := baseline.HourlyDistribution[hour]
	dayCount := baseline.DayOfWeekDistribution[dayOfWeek]

	// If this time has very few historical backups, it's unusual
	avgHourlyCount := float64(baseline.TotalBackups) / 24.0
	avgDayCount := float64(baseline.TotalBackups) / 7.0

	if float64(hourlyCount) < avgHourlyCount*0.1 || float64(dayCount) < avgDayCount*0.3 {
		description := fmt.Sprintf("Backup scheduled at unusual time (hour %d, %s). Historically, only %d%% of backups occur at this hour.",
			hour, metric.Timestamp.Weekday(), int((float64(hourlyCount)/float64(baseline.TotalBackups))*100))

		return &AnomalyReport{
			ID:           generateAnomalyID(),
			DatabaseName: metric.DatabaseName,
			BackupID:     metric.BackupID,
			Type:         AnomalyTypeUnusualTiming,
			Severity:     AnomalySeverityLow,
			Score:        30.0,
			DetectedAt:   time.Now(),
			Metric:       metric,
			Baseline:     baseline,
			Deviations: map[string]float64{
				"hourly_deviation": avgHourlyCount - float64(hourlyCount),
			},
			Description:    description,
			Recommendation: "Verify backup schedule configuration. Unusual timing may indicate configuration drift or unauthorized changes.",
		}
	}

	return nil
}

// trackFailure tracks recent backup failures.
func (d *Detector) trackFailure(metric BackupMetric) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.recentFailures = append(d.recentFailures, metric)

	// Remove old failures outside window
	cutoff := time.Now().Add(-d.config.FailureClusterWindow)
	filtered := make([]BackupMetric, 0)

	for _, f := range d.recentFailures {
		if f.Timestamp.After(cutoff) {
			filtered = append(filtered, f)
		}
	}

	d.recentFailures = filtered
}

// recordAnomaly records an anomaly in history.
func (d *Detector) recordAnomaly(anomaly AnomalyReport) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.anomalyHistory = append(d.anomalyHistory, anomaly)

	// Keep last 1000 anomalies
	if len(d.anomalyHistory) > 1000 {
		d.anomalyHistory = d.anomalyHistory[len(d.anomalyHistory)-1000:]
	}
}

// triggerCallbacks triggers all registered alert callbacks.
func (d *Detector) triggerCallbacks(anomaly AnomalyReport) {
	d.mu.RLock()
	callbacks := make([]func(AnomalyReport), len(d.alertCallbacks))
	copy(callbacks, d.alertCallbacks)
	d.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(anomaly)
	}
}

// GetAnomalyHistory returns recent anomaly history.
func (d *Detector) GetAnomalyHistory(limit int) []AnomalyReport {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > len(d.anomalyHistory) {
		limit = len(d.anomalyHistory)
	}

	start := len(d.anomalyHistory) - limit
	history := make([]AnomalyReport, limit)
	copy(history, d.anomalyHistory[start:])

	return history
}

// calculateSeverity calculates severity based on deviation.
func (d *Detector) calculateSeverity(deviation float64) AnomalySeverity {
	absDeviation := math.Abs(deviation)

	if absDeviation >= 5.0 {
		return AnomalySeverityCritical
	} else if absDeviation >= 4.0 {
		return AnomalySeverityHigh
	} else if absDeviation >= 3.0 {
		return AnomalySeverityMedium
	}

	return AnomalySeverityLow
}

// calculateScore calculates anomaly score (0-100).
func (d *Detector) calculateScore(deviation float64) float64 {
	absDeviation := math.Abs(deviation)

	// Score ranges:
	// 0-3 sigma: 0-50
	// 3-4 sigma: 50-70
	// 4-5 sigma: 70-90
	// 5+ sigma: 90-100

	if absDeviation <= 3.0 {
		return (absDeviation / 3.0) * 50.0
	} else if absDeviation <= 4.0 {
		return 50.0 + ((absDeviation-3.0)/1.0)*20.0
	} else if absDeviation <= 5.0 {
		return 70.0 + ((absDeviation-4.0)/1.0)*20.0
	}

	return math.Min(90.0+((absDeviation-5.0)/2.0)*10.0, 100.0)
}

// generateAnomalyID generates a unique ID for an anomaly.
func generateAnomalyID() string {
	return uid.New("anomaly")
}
