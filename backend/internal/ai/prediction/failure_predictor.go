// Package prediction provides predictive analytics for backup failures
package prediction

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// FailurePredictor predicts backup failures using historical analysis.
type FailurePredictor struct {
	mu             sync.RWMutex
	historicalData map[string][]BackupEvent
	patterns       map[string]*FailurePattern
	config         *PredictorConfig
	lastAnalysis   time.Time
}

// BackupEvent represents a historical backup event.
type BackupEvent struct {
	ID             string
	DatabaseName   string
	Timestamp      time.Time
	Success        bool
	Duration       time.Duration
	Size           int64
	ErrorType      string
	ErrorMessage   string
	DayOfWeek      time.Weekday
	HourOfDay      int
	SystemLoad     float64
	DiskUsage      float64
	NetworkLatency time.Duration
}

// FailurePattern represents identified failure patterns.
type FailurePattern struct {
	DatabaseName     string
	PatternType      PatternType
	Confidence       float64 // 0-100
	Frequency        float64 // Failures per day
	CommonErrors     map[string]int
	RiskFactors      []RiskFactor
	TimePatterns     *TimePattern
	ResourcePatterns *ResourcePattern
	LastOccurrence   time.Time
}

// PatternType represents the type of failure pattern.
type PatternType string

const (
	PatternTypeRecurring     PatternType = "recurring"      // Recurring at specific times
	PatternTypeTrend         PatternType = "trend"          // Increasing failure trend
	PatternTypeResourceBased PatternType = "resource_based" // Resource constraint failures
	PatternTypeTimeWindow    PatternType = "time_window"    // Specific time window failures
	PatternTypeErrorCascade  PatternType = "error_cascade"  // Cascading failures
)

// RiskFactor represents a risk factor for failure.
type RiskFactor struct {
	Factor      string
	Impact      float64 // 0-100
	Description string
}

// TimePattern represents time-based failure patterns.
type TimePattern struct {
	HighRiskDays  []time.Weekday
	HighRiskHours []int
	AvoidWindows  []TimeWindow
}

// TimeWindow represents a time window.
type TimeWindow struct {
	Start  time.Time
	End    time.Time
	Reason string
}

// ResourcePattern represents resource-based failure patterns.
type ResourcePattern struct {
	DiskThreshold    float64 // Percentage
	LoadThreshold    float64
	NetworkThreshold time.Duration
	MemoryThreshold  float64
}

// FailurePrediction represents a prediction of backup failure.
type FailurePrediction struct {
	ID                   string
	DatabaseName         string
	PredictionTime       time.Time
	PredictedFailureTime time.Time
	Confidence           float64 // 0-100
	Severity             PredictionSeverity
	RiskScore            float64 // 0-100
	Reasons              []string
	PreventiveActions    []PreventiveAction
	RelatedPatterns      []*FailurePattern
}

// PredictionSeverity represents prediction severity.
//
//nolint:revive // keeps public name stable; PredictionSeverity is part of the package API
type PredictionSeverity string

const (
	PredictionSeverityLow      PredictionSeverity = "LOW"
	PredictionSeverityMedium   PredictionSeverity = "MEDIUM"
	PredictionSeverityHigh     PredictionSeverity = "HIGH"
	PredictionSeverityCritical PredictionSeverity = "CRITICAL"
)

// PreventiveAction represents a suggested preventive action.
type PreventiveAction struct {
	Action      string
	Priority    int // 1-5, 1 being highest
	Impact      string
	Effort      string // LOW/MEDIUM/HIGH
	Description string
}

// PredictorConfig configures the failure predictor.
type PredictorConfig struct {
	MinHistoryDays      int
	PredictionWindow    time.Duration // How far ahead to predict
	ConfidenceThreshold float64       // Minimum confidence to report
	AnalysisInterval    time.Duration
}

// NewFailurePredictor creates a new failure predictor.
func NewFailurePredictor(config *PredictorConfig) *FailurePredictor {
	if config == nil {
		config = DefaultPredictorConfig()
	}

	return &FailurePredictor{
		historicalData: make(map[string][]BackupEvent),
		patterns:       make(map[string]*FailurePattern),
		config:         config,
	}
}

// DefaultPredictorConfig returns default configuration.
func DefaultPredictorConfig() *PredictorConfig {
	return &PredictorConfig{
		MinHistoryDays:      30,
		PredictionWindow:    48 * time.Hour,
		ConfidenceThreshold: 60.0,
		AnalysisInterval:    6 * time.Hour,
	}
}

// AddBackupEvent adds a historical backup event.
//
//nolint:gocritic // hugeParam: BackupEvent is passed by value to keep the public API stable
func (fp *FailurePredictor) AddBackupEvent(event BackupEvent) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.historicalData[event.DatabaseName] == nil {
		fp.historicalData[event.DatabaseName] = make([]BackupEvent, 0)
	}

	fp.historicalData[event.DatabaseName] = append(fp.historicalData[event.DatabaseName], event)

	// Keep only recent history (last 90 days)
	cutoff := time.Now().AddDate(0, 0, -90)
	events := fp.historicalData[event.DatabaseName]
	filtered := make([]BackupEvent, 0)
	for i := range events {
		if events[i].Timestamp.After(cutoff) {
			filtered = append(filtered, events[i])
		}
	}
	fp.historicalData[event.DatabaseName] = filtered
}

// AnalyzePatterns analyzes historical data to identify failure patterns.
func (fp *FailurePredictor) AnalyzePatterns(ctx context.Context) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	for dbName, events := range fp.historicalData {
		if len(events) < fp.config.MinHistoryDays {
			continue
		}

		// Analyze different pattern types
		pattern := &FailurePattern{
			DatabaseName: dbName,
			CommonErrors: make(map[string]int),
			RiskFactors:  make([]RiskFactor, 0),
		}

		// Analyze time patterns
		pattern.TimePatterns = fp.analyzeTimePatterns(events)

		// Analyze resource patterns
		pattern.ResourcePatterns = fp.analyzeResourcePatterns(events)

		// Analyze error patterns
		fp.analyzeErrorPatterns(events, pattern)

		// Calculate failure frequency
		failures := 0
		for i := range events {
			if !events[i].Success {
				failures++
				pattern.LastOccurrence = events[i].Timestamp
			}
		}

		daysCovered := time.Since(events[0].Timestamp).Hours() / 24
		pattern.Frequency = float64(failures) / daysCovered

		// Determine pattern type and confidence
		pattern.PatternType, pattern.Confidence = fp.determinePatternType(events, pattern)

		// Calculate risk factors
		pattern.RiskFactors = fp.calculateRiskFactors(pattern)

		fp.patterns[dbName] = pattern
	}

	fp.lastAnalysis = time.Now()
	return nil
}

// analyzeTimePatterns analyzes time-based failure patterns.
func (fp *FailurePredictor) analyzeTimePatterns(events []BackupEvent) *TimePattern {
	dayFailures := make(map[time.Weekday]int)
	hourFailures := make(map[int]int)

	for i := range events {
		if !events[i].Success {
			dayFailures[events[i].DayOfWeek]++
			hourFailures[events[i].HourOfDay]++
		}
	}

	// Find high-risk days (above average failures)
	avgDayFailures := 0.0
	for _, count := range dayFailures {
		avgDayFailures += float64(count)
	}
	avgDayFailures /= 7.0

	highRiskDays := make([]time.Weekday, 0)
	for day, count := range dayFailures {
		if float64(count) > avgDayFailures*1.5 {
			highRiskDays = append(highRiskDays, day)
		}
	}

	// Find high-risk hours
	avgHourFailures := 0.0
	for _, count := range hourFailures {
		avgHourFailures += float64(count)
	}
	avgHourFailures /= 24.0

	highRiskHours := make([]int, 0)
	for hour, count := range hourFailures {
		if float64(count) > avgHourFailures*2.0 {
			highRiskHours = append(highRiskHours, hour)
		}
	}

	return &TimePattern{
		HighRiskDays:  highRiskDays,
		HighRiskHours: highRiskHours,
		AvoidWindows:  make([]TimeWindow, 0),
	}
}

// analyzeResourcePatterns analyzes resource-based failure patterns.
func (fp *FailurePredictor) analyzeResourcePatterns(events []BackupEvent) *ResourcePattern {
	var failedDisk, failedLoad, failedNetwork []float64

	for i := range events {
		if !events[i].Success {
			failedDisk = append(failedDisk, events[i].DiskUsage)
			failedLoad = append(failedLoad, events[i].SystemLoad)
			if events[i].NetworkLatency > 0 {
				failedNetwork = append(failedNetwork, float64(events[i].NetworkLatency.Milliseconds()))
			}
		}
	}

	// Calculate thresholds (minimum values that led to failures)
	diskThreshold := 0.0
	if len(failedDisk) > 0 {
		sort.Float64s(failedDisk)
		diskThreshold = failedDisk[len(failedDisk)/2] // Median
	}

	loadThreshold := 0.0
	if len(failedLoad) > 0 {
		sort.Float64s(failedLoad)
		loadThreshold = failedLoad[len(failedLoad)/2]
	}

	networkThreshold := time.Duration(0)
	if len(failedNetwork) > 0 {
		sort.Float64s(failedNetwork)
		networkThreshold = time.Duration(failedNetwork[len(failedNetwork)/2]) * time.Millisecond
	}

	return &ResourcePattern{
		DiskThreshold:    diskThreshold,
		LoadThreshold:    loadThreshold,
		NetworkThreshold: networkThreshold,
		MemoryThreshold:  80.0, // Default 80%
	}
}

// analyzeErrorPatterns analyzes common error patterns.
func (fp *FailurePredictor) analyzeErrorPatterns(events []BackupEvent, pattern *FailurePattern) {
	for i := range events {
		if !events[i].Success && events[i].ErrorType != "" {
			pattern.CommonErrors[events[i].ErrorType]++
		}
	}
}

// determinePatternType determines the type of failure pattern.
func (fp *FailurePredictor) determinePatternType(events []BackupEvent, pattern *FailurePattern) (patternType PatternType, patternConfidence float64) {
	// Calculate recent failure rate vs historical
	recentEvents := 0
	recentFailures := 0
	cutoff := time.Now().AddDate(0, 0, -7)

	totalEvents := len(events)
	totalFailures := 0

	for i := range events {
		if events[i].Timestamp.After(cutoff) {
			recentEvents++
			if !events[i].Success {
				recentFailures++
			}
		}
		if !events[i].Success {
			totalFailures++
		}
	}

	recentFailureRate := 0.0
	if recentEvents > 0 {
		recentFailureRate = float64(recentFailures) / float64(recentEvents)
	}

	totalFailureRate := 0.0
	if totalEvents > 0 {
		totalFailureRate = float64(totalFailures) / float64(totalEvents)
	}

	// Determine pattern type
	if recentFailureRate > totalFailureRate*1.5 {
		// Increasing trend
		confidence := math.Min(100.0, recentFailureRate*100*1.5)
		return PatternTypeTrend, confidence
	}

	if len(pattern.TimePatterns.HighRiskHours) > 0 || len(pattern.TimePatterns.HighRiskDays) > 0 {
		// Time-based pattern
		confidence := math.Min(100.0, float64(len(pattern.TimePatterns.HighRiskHours)+len(pattern.TimePatterns.HighRiskDays))*10)
		return PatternTypeTimeWindow, confidence
	}

	if pattern.ResourcePatterns.DiskThreshold > 80.0 || pattern.ResourcePatterns.LoadThreshold > 0.8 {
		// Resource-based pattern
		confidence := 70.0
		return PatternTypeResourceBased, confidence
	}

	if pattern.Frequency > 0.5 {
		// Recurring pattern
		confidence := math.Min(100.0, pattern.Frequency*100)
		return PatternTypeRecurring, confidence
	}

	return PatternTypeRecurring, 50.0
}

// calculateRiskFactors calculates risk factors for failures.
func (fp *FailurePredictor) calculateRiskFactors(pattern *FailurePattern) []RiskFactor {
	factors := make([]RiskFactor, 0)

	// High failure frequency
	if pattern.Frequency > 0.3 {
		factors = append(factors, RiskFactor{
			Factor:      "High Failure Frequency",
			Impact:      math.Min(100.0, pattern.Frequency*100),
			Description: fmt.Sprintf("%.2f failures per day on average", pattern.Frequency),
		})
	}

	// Resource constraints
	if pattern.ResourcePatterns.DiskThreshold > 85.0 {
		factors = append(factors, RiskFactor{
			Factor:      "Disk Space Constraint",
			Impact:      90.0,
			Description: fmt.Sprintf("Failures occur when disk usage exceeds %.1f%%", pattern.ResourcePatterns.DiskThreshold),
		})
	}

	if pattern.ResourcePatterns.LoadThreshold > 0.8 {
		factors = append(factors, RiskFactor{
			Factor:      "High System Load",
			Impact:      80.0,
			Description: fmt.Sprintf("Failures occur when system load exceeds %.2f", pattern.ResourcePatterns.LoadThreshold),
		})
	}

	// Time-based risks
	if len(pattern.TimePatterns.HighRiskHours) > 0 {
		factors = append(factors, RiskFactor{
			Factor:      "Time Window Risk",
			Impact:      70.0,
			Description: fmt.Sprintf("Higher failure rate during hours: %v", pattern.TimePatterns.HighRiskHours),
		})
	}

	// Common errors
	if len(pattern.CommonErrors) > 0 {
		var mostCommonError string
		maxCount := 0
		for errType, count := range pattern.CommonErrors {
			if count > maxCount {
				maxCount = count
				mostCommonError = errType
			}
		}
		factors = append(factors, RiskFactor{
			Factor:      "Recurring Error Type",
			Impact:      60.0,
			Description: fmt.Sprintf("Most common error: %s (%d occurrences)", mostCommonError, maxCount),
		})
	}

	return factors
}

// PredictFailures predicts potential failures in the next 24-48 hours.
func (fp *FailurePredictor) PredictFailures(ctx context.Context, databaseName string) ([]*FailurePrediction, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	predictions := make([]*FailurePrediction, 0)

	// Get pattern for this database
	pattern, exists := fp.patterns[databaseName]
	if !exists {
		return predictions, nil
	}

	now := time.Now()
	predictionEnd := now.Add(fp.config.PredictionWindow)

	// Check time-based predictions
	for t := now; t.Before(predictionEnd); t = t.Add(1 * time.Hour) {
		prediction := fp.evaluateTimeWindow(t, pattern)
		if prediction != nil && prediction.Confidence >= fp.config.ConfidenceThreshold {
			predictions = append(predictions, prediction)
		}
	}

	// Add resource-based prediction if applicable
	resourcePrediction := fp.evaluateResourceRisk(pattern)
	if resourcePrediction != nil && resourcePrediction.Confidence >= fp.config.ConfidenceThreshold {
		predictions = append(predictions, resourcePrediction)
	}

	// Add trend-based prediction
	if pattern.PatternType == PatternTypeTrend {
		trendPrediction := fp.evaluateTrendRisk(pattern)
		if trendPrediction != nil && trendPrediction.Confidence >= fp.config.ConfidenceThreshold {
			predictions = append(predictions, trendPrediction)
		}
	}

	return predictions, nil
}

// evaluateTimeWindow evaluates failure risk for a specific time window.
func (fp *FailurePredictor) evaluateTimeWindow(t time.Time, pattern *FailurePattern) *FailurePrediction {
	dayOfWeek := t.Weekday()
	hourOfDay := t.Hour()

	isHighRiskDay := false
	for _, day := range pattern.TimePatterns.HighRiskDays {
		if day == dayOfWeek {
			isHighRiskDay = true
			break
		}
	}

	isHighRiskHour := false
	for _, hour := range pattern.TimePatterns.HighRiskHours {
		if hour == hourOfDay {
			isHighRiskHour = true
			break
		}
	}

	if !isHighRiskDay && !isHighRiskHour {
		return nil
	}

	confidence := 0.0
	reasons := make([]string, 0)

	if isHighRiskDay {
		confidence += 30.0
		reasons = append(reasons, fmt.Sprintf("High failure rate on %s", dayOfWeek))
	}

	if isHighRiskHour {
		confidence += 40.0
		reasons = append(reasons, fmt.Sprintf("High failure rate during hour %d:00", hourOfDay))
	}

	// Add pattern confidence
	confidence += pattern.Confidence * 0.3

	prediction := &FailurePrediction{
		ID:                   generatePredictionID(),
		DatabaseName:         pattern.DatabaseName,
		PredictionTime:       time.Now(),
		PredictedFailureTime: t,
		Confidence:           math.Min(100.0, confidence),
		Reasons:              reasons,
		RelatedPatterns:      []*FailurePattern{pattern},
	}

	prediction.RiskScore = prediction.Confidence
	prediction.Severity = fp.determineSeverity(prediction.RiskScore)
	prediction.PreventiveActions = fp.generatePreventiveActions(pattern)

	return prediction
}

// evaluateResourceRisk evaluates resource-based failure risk.
func (fp *FailurePredictor) evaluateResourceRisk(pattern *FailurePattern) *FailurePrediction {
	if pattern.PatternType != PatternTypeResourceBased {
		return nil
	}

	reasons := make([]string, 0)
	confidence := pattern.Confidence

	if pattern.ResourcePatterns.DiskThreshold > 85.0 {
		reasons = append(reasons, fmt.Sprintf("Disk usage approaching threshold (%.1f%%)", pattern.ResourcePatterns.DiskThreshold))
	}

	if pattern.ResourcePatterns.LoadThreshold > 0.8 {
		reasons = append(reasons, fmt.Sprintf("System load may exceed threshold (%.2f)", pattern.ResourcePatterns.LoadThreshold))
	}

	if len(reasons) == 0 {
		return nil
	}

	prediction := &FailurePrediction{
		ID:                   generatePredictionID(),
		DatabaseName:         pattern.DatabaseName,
		PredictionTime:       time.Now(),
		PredictedFailureTime: time.Now().Add(24 * time.Hour),
		Confidence:           confidence,
		Reasons:              reasons,
		RelatedPatterns:      []*FailurePattern{pattern},
	}

	prediction.RiskScore = confidence
	prediction.Severity = fp.determineSeverity(prediction.RiskScore)
	prediction.PreventiveActions = fp.generatePreventiveActions(pattern)

	return prediction
}

// evaluateTrendRisk evaluates trend-based failure risk.
func (fp *FailurePredictor) evaluateTrendRisk(pattern *FailurePattern) *FailurePrediction {
	prediction := &FailurePrediction{
		ID:                   generatePredictionID(),
		DatabaseName:         pattern.DatabaseName,
		PredictionTime:       time.Now(),
		PredictedFailureTime: time.Now().Add(24 * time.Hour),
		Confidence:           pattern.Confidence,
		Reasons: []string{
			"Increasing failure trend detected",
			fmt.Sprintf("Failure frequency: %.2f per day", pattern.Frequency),
		},
		RelatedPatterns: []*FailurePattern{pattern},
	}

	prediction.RiskScore = pattern.Confidence
	prediction.Severity = fp.determineSeverity(prediction.RiskScore)
	prediction.PreventiveActions = fp.generatePreventiveActions(pattern)

	return prediction
}

// determineSeverity determines prediction severity based on risk score.
func (fp *FailurePredictor) determineSeverity(riskScore float64) PredictionSeverity {
	switch {
	case riskScore >= 90.0:
		return PredictionSeverityCritical
	case riskScore >= 75.0:
		return PredictionSeverityHigh
	case riskScore >= 60.0:
		return PredictionSeverityMedium
	default:
		return PredictionSeverityLow
	}
}

// generatePreventiveActions generates suggested preventive actions.
func (fp *FailurePredictor) generatePreventiveActions(pattern *FailurePattern) []PreventiveAction {
	actions := make([]PreventiveAction, 0)

	// Time-based actions
	if pattern.PatternType == PatternTypeTimeWindow || pattern.PatternType == PatternTypeRecurring {
		actions = append(actions, PreventiveAction{
			Action:      "Reschedule backup to avoid high-risk time window",
			Priority:    1,
			Impact:      "Reduces failure probability by 60-80%",
			Effort:      "LOW",
			Description: fmt.Sprintf("Avoid running backups during: %v (hours) and %v (days)", pattern.TimePatterns.HighRiskHours, pattern.TimePatterns.HighRiskDays),
		})
	}

	// Resource-based actions
	if pattern.PatternType == PatternTypeResourceBased {
		if pattern.ResourcePatterns.DiskThreshold > 85.0 {
			actions = append(actions, PreventiveAction{
				Action:      "Free up disk space before backup",
				Priority:    1,
				Impact:      "Prevents disk-full failures",
				Effort:      "MEDIUM",
				Description: fmt.Sprintf("Ensure disk usage stays below %.1f%% before backup runs", pattern.ResourcePatterns.DiskThreshold-5.0),
			})
		}

		if pattern.ResourcePatterns.LoadThreshold > 0.8 {
			actions = append(actions, PreventiveAction{
				Action:      "Reduce system load before backup",
				Priority:    2,
				Impact:      "Improves backup success rate",
				Effort:      "MEDIUM",
				Description: "Schedule backups during low-load periods or reduce concurrent operations",
			})
		}
	}

	// Error-specific actions
	if len(pattern.CommonErrors) > 0 {
		var mostCommonError string
		maxCount := 0
		for errType, count := range pattern.CommonErrors {
			if count > maxCount {
				maxCount = count
				mostCommonError = errType
			}
		}

		action := PreventiveAction{
			Action:      fmt.Sprintf("Address recurring error: %s", mostCommonError),
			Priority:    1,
			Impact:      "Resolves root cause of failures",
			Effort:      "HIGH",
			Description: fmt.Sprintf("This error accounts for %d of recent failures", maxCount),
		}

		// Add specific recommendations based on error type
		switch mostCommonError {
		case "NETWORK_ERROR":
			action.Description += ". Check network connectivity and firewall rules."
		case "DISK_FULL":
			action.Description += ". Implement automatic cleanup of old backups."
		case "TIMEOUT":
			action.Description += ". Increase timeout thresholds or optimize backup process."
		case "PERMISSION_DENIED":
			action.Description += ". Verify backup user has necessary permissions."
		}

		actions = append(actions, action)
	}

	// General preventive actions
	actions = append(actions, PreventiveAction{
		Action:      "Enable proactive monitoring and alerting",
		Priority:    3,
		Impact:      "Early detection of issues",
		Effort:      "LOW",
		Description: "Set up alerts for resource thresholds and backup failures",
	})

	if pattern.Frequency > 0.3 {
		actions = append(actions, PreventiveAction{
			Action:      "Increase backup frequency for critical databases",
			Priority:    2,
			Impact:      "Reduces data loss window",
			Effort:      "LOW",
			Description: "More frequent backups provide better RPO in case of failures",
		})
	}

	return actions
}

// GetPatterns returns all identified patterns.
func (fp *FailurePredictor) GetPatterns() map[string]*FailurePattern {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	patterns := make(map[string]*FailurePattern)
	for k, v := range fp.patterns {
		patterns[k] = v
	}
	return patterns
}

// GetStatistics returns statistics about failure prediction.
func (fp *FailurePredictor) GetStatistics() map[string]interface{} {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	stats := map[string]interface{}{
		"total_databases": len(fp.historicalData),
		"patterns_found":  len(fp.patterns),
		"last_analysis":   fp.lastAnalysis,
		"by_pattern_type": make(map[PatternType]int),
		"avg_frequency":   0.0,
		"high_risk_dbs":   0,
	}

	byType := make(map[PatternType]int)
	totalFreq := 0.0
	highRisk := 0

	for _, pattern := range fp.patterns {
		byType[pattern.PatternType]++
		totalFreq += pattern.Frequency
		if pattern.Confidence >= 80.0 {
			highRisk++
		}
	}

	stats["by_pattern_type"] = byType
	if len(fp.patterns) > 0 {
		stats["avg_frequency"] = totalFreq / float64(len(fp.patterns))
	}
	stats["high_risk_dbs"] = highRisk

	return stats
}

// Helper functions

func generatePredictionID() string {
	return uid.New("prediction")
}
