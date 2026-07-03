package ai

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// ==================== Smart Features Manager ====================

// SmartFeaturesManager coordinates all AI-powered intelligent features.
type SmartFeaturesManager struct {
	predictiveScheduler   *PredictiveScheduler
	qualityScorer         *BackupQualityScorer
	sizeEstimator         *BackupSizeEstimator
	autoCategorizer       *AutoCategorizer
	duplicateDetector     *DuplicateDetector
	nlpSearchEngine       *NLPSearchEngine
	retentionPolicyGen    *SmartRetentionPolicyGenerator
	aiAdvisor             *AIAdvisor
	troubleshootingEngine *AutomatedTroubleshootingEngine
	failurePredictorAPI   *FailurePredictionAPI

	mu sync.RWMutex
}

// NewSmartFeaturesManager creates a new smart features manager.
func NewSmartFeaturesManager() *SmartFeaturesManager {
	return &SmartFeaturesManager{
		predictiveScheduler:   NewPredictiveScheduler(),
		qualityScorer:         NewBackupQualityScorer(),
		sizeEstimator:         NewBackupSizeEstimator(),
		autoCategorizer:       NewAutoCategorizer(),
		duplicateDetector:     NewDuplicateDetector(),
		nlpSearchEngine:       NewNLPSearchEngine(),
		retentionPolicyGen:    NewSmartRetentionPolicyGenerator(),
		aiAdvisor:             NewAIAdvisor(),
		troubleshootingEngine: NewAutomatedTroubleshootingEngine(),
		failurePredictorAPI:   NewFailurePredictionAPI(),
	}
}

// GetPredictiveScheduler returns the predictive scheduler.
func (sfm *SmartFeaturesManager) GetPredictiveScheduler() *PredictiveScheduler {
	return sfm.predictiveScheduler
}

// GetQualityScorer returns the quality scorer.
func (sfm *SmartFeaturesManager) GetQualityScorer() *BackupQualityScorer {
	return sfm.qualityScorer
}

// GetSizeEstimator returns the size estimator.
func (sfm *SmartFeaturesManager) GetSizeEstimator() *BackupSizeEstimator {
	return sfm.sizeEstimator
}

// GetAutoCategorizer returns the auto categorizer.
func (sfm *SmartFeaturesManager) GetAutoCategorizer() *AutoCategorizer {
	return sfm.autoCategorizer
}

// GetDuplicateDetector returns the duplicate detector.
func (sfm *SmartFeaturesManager) GetDuplicateDetector() *DuplicateDetector {
	return sfm.duplicateDetector
}

// GetNLPSearchEngine returns the NLP search engine.
func (sfm *SmartFeaturesManager) GetNLPSearchEngine() *NLPSearchEngine {
	return sfm.nlpSearchEngine
}

// GetRetentionPolicyGenerator returns the retention policy generator.
func (sfm *SmartFeaturesManager) GetRetentionPolicyGenerator() *SmartRetentionPolicyGenerator {
	return sfm.retentionPolicyGen
}

// GetAIAdvisor returns the AI advisor.
func (sfm *SmartFeaturesManager) GetAIAdvisor() *AIAdvisor {
	return sfm.aiAdvisor
}

// GetTroubleshootingEngine returns the troubleshooting engine.
func (sfm *SmartFeaturesManager) GetTroubleshootingEngine() *AutomatedTroubleshootingEngine {
	return sfm.troubleshootingEngine
}

// GetFailurePredictorAPI returns the failure predictor API.
func (sfm *SmartFeaturesManager) GetFailurePredictorAPI() *FailurePredictionAPI {
	return sfm.failurePredictorAPI
}

// ==================== 1. ML-Based Predictive Scheduling ====================

// PredictiveScheduler uses machine learning to predict optimal backup times.
type PredictiveScheduler struct {
	historicalData map[string][]*ScheduleDataPoint     // database -> data points
	models         map[string]*SchedulePredictionModel // database -> model
	mu             sync.RWMutex
}

// ScheduleDataPoint represents a historical backup execution.
type ScheduleDataPoint struct {
	Timestamp   time.Time
	Hour        int
	DayOfWeek   int
	Duration    time.Duration
	Size        int64
	Success     bool
	CPUUsage    float64
	MemoryUsage float64
	SystemLoad  float64
}

// SchedulePredictionModel contains the ML model for schedule prediction.
type SchedulePredictionModel struct {
	DatabaseName      string
	OptimalHours      []int                 // Hours of day sorted by success rate
	OptimalDaysOfWeek []int                 // Days of week sorted by success rate
	HourlySuccessRate map[int]float64       // hour -> success rate
	DailySuccessRate  map[int]float64       // day of week -> success rate
	AvgDuration       map[int]time.Duration // hour -> avg duration
	AvgSystemLoad     map[int]float64       // hour -> avg system load
	TrainedAt         time.Time
	SampleCount       int
	Confidence        float64 // 0-100
}

// ScheduleRecommendation represents a recommended backup schedule.
type ScheduleRecommendation struct {
	DatabaseName        string
	RecommendedTime     time.Time
	CronExpression      string
	Confidence          float64 // 0-100
	ExpectedDuration    time.Duration
	ExpectedSuccessRate float64
	Reasoning           []string
	AlternativeTimes    []time.Time
	OptimalWindow       TimeWindow
}

// TimeWindow represents a time range.
type TimeWindow struct {
	Start time.Time
	End   time.Time
	Score float64
}

// NewPredictiveScheduler creates a new predictive scheduler.
func NewPredictiveScheduler() *PredictiveScheduler {
	return &PredictiveScheduler{
		historicalData: make(map[string][]*ScheduleDataPoint),
		models:         make(map[string]*SchedulePredictionModel),
	}
}

// AddDataPoint adds a historical backup execution data point.
func (ps *PredictiveScheduler) AddDataPoint(database string, point *ScheduleDataPoint) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.historicalData[database]; !exists {
		ps.historicalData[database] = make([]*ScheduleDataPoint, 0)
	}

	ps.historicalData[database] = append(ps.historicalData[database], point)

	// Keep only last 1000 data points per database
	if len(ps.historicalData[database]) > 1000 {
		ps.historicalData[database] = ps.historicalData[database][len(ps.historicalData[database])-1000:]
	}
}

// TrainModel trains the prediction model for a database.
func (ps *PredictiveScheduler) TrainModel(ctx context.Context, database string) (*SchedulePredictionModel, error) {
	ps.mu.RLock()
	data, exists := ps.historicalData[database]
	ps.mu.RUnlock()

	if !exists || len(data) < 10 {
		return nil, fmt.Errorf("insufficient data for training (need at least 10 data points, have %d)", len(data))
	}

	model := &SchedulePredictionModel{
		DatabaseName:      database,
		HourlySuccessRate: make(map[int]float64),
		DailySuccessRate:  make(map[int]float64),
		AvgDuration:       make(map[int]time.Duration),
		AvgSystemLoad:     make(map[int]float64),
		TrainedAt:         time.Now(),
		SampleCount:       len(data),
	}

	// Calculate hourly statistics
	hourlyStats := make(map[int]*hourStats)
	dailyStats := make(map[int]*dayStats)

	for _, point := range data {
		// Hourly stats
		if _, exists := hourlyStats[point.Hour]; !exists {
			hourlyStats[point.Hour] = &hourStats{}
		}
		hourlyStats[point.Hour].totalCount++
		if point.Success {
			hourlyStats[point.Hour].successCount++
		}
		hourlyStats[point.Hour].totalDuration += point.Duration
		hourlyStats[point.Hour].totalLoad += point.SystemLoad

		// Daily stats
		if _, exists := dailyStats[point.DayOfWeek]; !exists {
			dailyStats[point.DayOfWeek] = &dayStats{}
		}
		dailyStats[point.DayOfWeek].totalCount++
		if point.Success {
			dailyStats[point.DayOfWeek].successCount++
		}
	}

	// Calculate success rates and averages
	for hour, stats := range hourlyStats {
		model.HourlySuccessRate[hour] = float64(stats.successCount) / float64(stats.totalCount)
		model.AvgDuration[hour] = time.Duration(int64(stats.totalDuration) / int64(stats.totalCount))
		model.AvgSystemLoad[hour] = stats.totalLoad / float64(stats.totalCount)
	}

	for day, stats := range dailyStats {
		model.DailySuccessRate[day] = float64(stats.successCount) / float64(stats.totalCount)
	}

	// Find optimal hours (top 3)
	type hourScore struct {
		hour  int
		score float64
	}
	hourScores := make([]hourScore, 0)
	for hour, successRate := range model.HourlySuccessRate {
		// Score combines success rate (70%) and low system load (30%)
		score := successRate*0.7 + (1.0-model.AvgSystemLoad[hour])*0.3
		hourScores = append(hourScores, hourScore{hour: hour, score: score})
	}
	sort.Slice(hourScores, func(i, j int) bool {
		return hourScores[i].score > hourScores[j].score
	})

	model.OptimalHours = make([]int, 0)
	for i := 0; i < len(hourScores) && i < 3; i++ {
		model.OptimalHours = append(model.OptimalHours, hourScores[i].hour)
	}

	// Find optimal days (top 3)
	type dayScore struct {
		day   int
		score float64
	}
	dayScores := make([]dayScore, 0)
	for day, successRate := range model.DailySuccessRate {
		dayScores = append(dayScores, dayScore{day: day, score: successRate})
	}
	sort.Slice(dayScores, func(i, j int) bool {
		return dayScores[i].score > dayScores[j].score
	})

	model.OptimalDaysOfWeek = make([]int, 0)
	for i := 0; i < len(dayScores) && i < 3; i++ {
		model.OptimalDaysOfWeek = append(model.OptimalDaysOfWeek, dayScores[i].day)
	}

	// Calculate confidence based on sample size and variance
	model.Confidence = ps.calculateConfidence(len(data), model.HourlySuccessRate)

	// Save model
	ps.mu.Lock()
	ps.models[database] = model
	ps.mu.Unlock()

	return model, nil
}

type hourStats struct {
	totalCount    int
	successCount  int
	totalDuration time.Duration
	totalLoad     float64
}

type dayStats struct {
	totalCount   int
	successCount int
}

// calculateConfidence calculates model confidence based on data quality.
func (ps *PredictiveScheduler) calculateConfidence(sampleCount int, successRates map[int]float64) float64 {
	// Base confidence from sample size (logarithmic scale)
	sampleConfidence := math.Min(100.0, math.Log10(float64(sampleCount))*30)

	// Variance penalty (lower variance = higher confidence)
	if len(successRates) == 0 {
		return sampleConfidence
	}

	var mean, variance float64
	for _, rate := range successRates {
		mean += rate
	}
	mean /= float64(len(successRates))

	for _, rate := range successRates {
		variance += math.Pow(rate-mean, 2)
	}
	variance /= float64(len(successRates))

	// Low variance means consistent performance -> higher confidence
	varianceConfidence := math.Max(0, 100.0-variance*100)

	// Weighted average
	return sampleConfidence*0.6 + varianceConfidence*0.4
}

// GetRecommendation generates a schedule recommendation for a database.
func (ps *PredictiveScheduler) GetRecommendation(ctx context.Context, database string) (*ScheduleRecommendation, error) {
	ps.mu.RLock()
	model, exists := ps.models[database]
	ps.mu.RUnlock()

	if !exists {
		// Try to train model first
		var err error
		model, err = ps.TrainModel(ctx, database)
		if err != nil {
			return nil, fmt.Errorf("no model available and training failed: %w", err)
		}
	}

	if len(model.OptimalHours) == 0 || len(model.OptimalDaysOfWeek) == 0 {
		return nil, fmt.Errorf("insufficient data to make recommendation")
	}

	// Get best hour and day
	bestHour := model.OptimalHours[0]
	bestDay := model.OptimalDaysOfWeek[0]

	// Calculate next occurrence
	now := time.Now()
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), bestHour, 0, 0, 0, now.Location())

	// Adjust to next occurrence of best day
	for nextTime.Weekday() != time.Weekday(bestDay) || nextTime.Before(now) {
		nextTime = nextTime.Add(24 * time.Hour)
	}

	// Generate cron expression (minute hour * * dayofweek)
	cronExpr := fmt.Sprintf("0 %d * * %d", bestHour, bestDay)

	// Build reasoning
	reasoning := []string{
		fmt.Sprintf("Historical success rate at %02d:00 is %.1f%%", bestHour, model.HourlySuccessRate[bestHour]*100),
		fmt.Sprintf("%s has %.1f%% success rate", time.Weekday(bestDay), model.DailySuccessRate[bestDay]*100),
		fmt.Sprintf("Average duration: %s", model.AvgDuration[bestHour]),
		fmt.Sprintf("Average system load: %.1f%%", model.AvgSystemLoad[bestHour]*100),
	}

	// Generate alternatives
	alternatives := make([]time.Time, 0)
	for i := 1; i < len(model.OptimalHours) && i < 3; i++ {
		altTime := time.Date(now.Year(), now.Month(), now.Day(), model.OptimalHours[i], 0, 0, 0, now.Location())
		for altTime.Weekday() != time.Weekday(model.OptimalDaysOfWeek[0]) || altTime.Before(now) {
			altTime = altTime.Add(24 * time.Hour)
		}
		alternatives = append(alternatives, altTime)
	}

	recommendation := &ScheduleRecommendation{
		DatabaseName:        database,
		RecommendedTime:     nextTime,
		CronExpression:      cronExpr,
		Confidence:          model.Confidence,
		ExpectedDuration:    model.AvgDuration[bestHour],
		ExpectedSuccessRate: model.HourlySuccessRate[bestHour],
		Reasoning:           reasoning,
		AlternativeTimes:    alternatives,
		OptimalWindow: TimeWindow{
			Start: nextTime,
			End:   nextTime.Add(model.AvgDuration[bestHour]),
			Score: model.HourlySuccessRate[bestHour],
		},
	}

	return recommendation, nil
}

// ==================== 2. Backup Quality Scoring ====================

// BackupQualityScorer calculates a comprehensive quality score (0-100) for backups.
type BackupQualityScorer struct {
	mu sync.RWMutex
}

// QualityScore represents a backup's quality assessment.
type QualityScore struct {
	OverallScore    float64            // 0-100
	ComponentScores map[string]float64 // Individual component scores
	Grade           string             // A+, A, B+, B, C, D, F
	Issues          []string           // List of quality issues
	Strengths       []string           // List of quality strengths
	Recommendations []string           // Improvement recommendations
	Metadata        map[string]interface{}
}

// BackupMetrics represents metrics for quality scoring.
type BackupMetrics struct {
	DatabaseName     string
	Size             int64
	CompressedSize   int64
	Duration         time.Duration
	Success          bool
	ErrorCount       int
	ChecksumValid    bool
	Encrypted        bool
	CompressionRatio float64
	Age              time.Duration
	RedundancyLevel  int // Number of copies/replicas
	LastValidated    time.Time
	RecoveryTested   bool
	IncrementalChain int // Length of incremental chain
	Metadata         map[string]interface{}
}

// NewBackupQualityScorer creates a new quality scorer.
func NewBackupQualityScorer() *BackupQualityScorer {
	return &BackupQualityScorer{}
}

// CalculateScore calculates a comprehensive quality score for a backup.
func (bqs *BackupQualityScorer) CalculateScore(ctx context.Context, metrics *BackupMetrics) (*QualityScore, error) {
	score := &QualityScore{
		ComponentScores: make(map[string]float64),
		Issues:          make([]string, 0),
		Strengths:       make([]string, 0),
		Recommendations: make([]string, 0),
		Metadata:        make(map[string]interface{}),
	}

	// 1. Completeness Score (25 points)
	completenessScore := bqs.scoreCompleteness(metrics, score)
	score.ComponentScores["completeness"] = completenessScore

	// 2. Reliability Score (25 points)
	reliabilityScore := bqs.scoreReliability(metrics, score)
	score.ComponentScores["reliability"] = reliabilityScore

	// 3. Security Score (20 points)
	securityScore := bqs.scoreSecurity(metrics, score)
	score.ComponentScores["security"] = securityScore

	// 4. Efficiency Score (15 points)
	efficiencyScore := bqs.scoreEfficiency(metrics, score)
	score.ComponentScores["efficiency"] = efficiencyScore

	// 5. Freshness Score (10 points)
	freshnessScore := bqs.scoreFreshness(metrics, score)
	score.ComponentScores["freshness"] = freshnessScore

	// 6. Redundancy Score (5 points)
	redundancyScore := bqs.scoreRedundancy(metrics, score)
	score.ComponentScores["redundancy"] = redundancyScore

	// Calculate overall score
	score.OverallScore = completenessScore + reliabilityScore + securityScore +
		efficiencyScore + freshnessScore + redundancyScore

	// Assign grade
	score.Grade = bqs.assignGrade(score.OverallScore)

	// Add metadata
	score.Metadata["database"] = metrics.DatabaseName
	score.Metadata["size_mb"] = float64(metrics.Size) / (1024 * 1024)
	score.Metadata["compression_ratio"] = metrics.CompressionRatio
	score.Metadata["age_hours"] = metrics.Age.Hours()

	return score, nil
}

// scoreCompleteness evaluates backup completeness (25 points max).
func (bqs *BackupQualityScorer) scoreCompleteness(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 25.0

	if !metrics.Success {
		points -= 15.0
		score.Issues = append(score.Issues, "Backup did not complete successfully")
		score.Recommendations = append(score.Recommendations, "Investigate and resolve backup failure causes")
	} else {
		score.Strengths = append(score.Strengths, "Backup completed successfully")
	}

	if metrics.ErrorCount > 0 {
		penalty := math.Min(5.0, float64(metrics.ErrorCount)*0.5)
		points -= penalty
		score.Issues = append(score.Issues, fmt.Sprintf("%d errors occurred during backup", metrics.ErrorCount))
	}

	if metrics.Size == 0 {
		points -= 5.0
		score.Issues = append(score.Issues, "Backup size is zero - possible incomplete backup")
	}

	return math.Max(0, points)
}

// scoreReliability evaluates backup reliability (25 points max).
func (bqs *BackupQualityScorer) scoreReliability(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 25.0

	if !metrics.ChecksumValid {
		points -= 10.0
		score.Issues = append(score.Issues, "Checksum validation failed")
		score.Recommendations = append(score.Recommendations, "Re-run backup to ensure data integrity")
	} else {
		score.Strengths = append(score.Strengths, "Checksum validation passed")
	}

	if !metrics.RecoveryTested {
		points -= 8.0
		score.Issues = append(score.Issues, "Backup has not been recovery tested")
		score.Recommendations = append(score.Recommendations, "Perform recovery test to validate backup")
	} else {
		score.Strengths = append(score.Strengths, "Recovery tested successfully")
	}

	timeSinceValidation := time.Since(metrics.LastValidated)
	if timeSinceValidation > 30*24*time.Hour {
		penalty := math.Min(7.0, timeSinceValidation.Hours()/(30*24))
		points -= penalty
		score.Issues = append(score.Issues, "Backup validation is outdated")
		score.Recommendations = append(score.Recommendations, "Re-validate backup integrity")
	}

	return math.Max(0, points)
}

// scoreSecurity evaluates backup security (20 points max).
func (bqs *BackupQualityScorer) scoreSecurity(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 20.0

	if !metrics.Encrypted {
		points -= 15.0
		score.Issues = append(score.Issues, "Backup is not encrypted")
		score.Recommendations = append(score.Recommendations, "Enable encryption for backup security")
	} else {
		score.Strengths = append(score.Strengths, "Backup is encrypted")
	}

	return math.Max(0, points)
}

// scoreEfficiency evaluates backup efficiency (15 points max).
func (bqs *BackupQualityScorer) scoreEfficiency(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 15.0

	// Compression efficiency
	if metrics.CompressionRatio > 0.9 {
		points -= 5.0
		score.Issues = append(score.Issues, fmt.Sprintf("Poor compression ratio: %.1f%%", metrics.CompressionRatio*100))
		score.Recommendations = append(score.Recommendations, "Consider using better compression algorithm")
	} else if metrics.CompressionRatio < 0.5 {
		score.Strengths = append(score.Strengths, fmt.Sprintf("Excellent compression ratio: %.1f%%", metrics.CompressionRatio*100))
	}

	// Duration efficiency (penalize if > 2 hours)
	if metrics.Duration > 2*time.Hour {
		penalty := math.Min(5.0, metrics.Duration.Hours()/2.0)
		points -= penalty
		score.Issues = append(score.Issues, fmt.Sprintf("Backup took %s - consider optimization", metrics.Duration))
		score.Recommendations = append(score.Recommendations, "Optimize backup performance (parallelism, incremental)")
	}

	// Incremental chain length
	if metrics.IncrementalChain > 10 {
		points -= 5.0
		score.Issues = append(score.Issues, fmt.Sprintf("Long incremental chain (%d backups)", metrics.IncrementalChain))
		score.Recommendations = append(score.Recommendations, "Perform full backup to reset incremental chain")
	}

	return math.Max(0, points)
}

// scoreFreshness evaluates backup freshness (10 points max).
func (bqs *BackupQualityScorer) scoreFreshness(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 10.0

	ageHours := metrics.Age.Hours()

	if ageHours < 24 {
		score.Strengths = append(score.Strengths, "Backup is fresh (< 24 hours old)")
	} else if ageHours < 7*24 {
		points -= 3.0
		score.Issues = append(score.Issues, "Backup is 1-7 days old")
	} else if ageHours < 30*24 {
		points -= 6.0
		score.Issues = append(score.Issues, "Backup is 1-4 weeks old")
		score.Recommendations = append(score.Recommendations, "Run new backup soon")
	} else {
		points -= 10.0
		score.Issues = append(score.Issues, "Backup is over 30 days old")
		score.Recommendations = append(score.Recommendations, "Urgently run new backup")
	}

	return math.Max(0, points)
}

// scoreRedundancy evaluates backup redundancy (5 points max).
func (bqs *BackupQualityScorer) scoreRedundancy(metrics *BackupMetrics, score *QualityScore) float64 {
	points := 5.0

	if metrics.RedundancyLevel == 0 {
		points = 0
		score.Issues = append(score.Issues, "No backup redundancy")
		score.Recommendations = append(score.Recommendations, "Enable backup replication for redundancy")
	} else if metrics.RedundancyLevel == 1 {
		points = 2.5
		score.Issues = append(score.Issues, "Single backup copy only")
		score.Recommendations = append(score.Recommendations, "Consider additional backup copies")
	} else if metrics.RedundancyLevel >= 3 {
		score.Strengths = append(score.Strengths, fmt.Sprintf("%d backup copies available", metrics.RedundancyLevel))
	}

	return points
}

// assignGrade assigns a letter grade based on overall score.
func (bqs *BackupQualityScorer) assignGrade(score float64) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "A-"
	case score >= 87:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 80:
		return "B-"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 70:
		return "C-"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// ==================== 3. Backup Size Estimation ====================

// BackupSizeEstimator predicts backup sizes using historical growth patterns.
type BackupSizeEstimator struct {
	historicalSizes map[string][]SizeDataPoint // database -> size history
	growthModels    map[string]*GrowthModel
	mu              sync.RWMutex
}

// SizeDataPoint represents a historical size measurement.
type SizeDataPoint struct {
	Timestamp time.Time
	Size      int64
	Type      string // "full", "incremental", "differential"
}

// GrowthModel represents a linear growth model.
type GrowthModel struct {
	DatabaseName     string
	GrowthRatePerDay float64 // bytes per day
	BaseSize         int64   // starting size
	BaseTime         time.Time
	R2               float64 // coefficient of determination (fit quality)
	Confidence       float64 // 0-100
	SampleCount      int
	TrainedAt        time.Time
}

// SizeEstimation represents an estimated backup size.
type SizeEstimation struct {
	DatabaseName     string
	EstimatedSize    int64
	EstimatedSizeStr string
	Confidence       float64
	EstimationMethod string
	GrowthRate       float64 // bytes/day
	DaysProjected    int
	LowerBound       int64
	UpperBound       int64
	Reasoning        []string
}

// NewBackupSizeEstimator creates a new size estimator.
func NewBackupSizeEstimator() *BackupSizeEstimator {
	return &BackupSizeEstimator{
		historicalSizes: make(map[string][]SizeDataPoint),
		growthModels:    make(map[string]*GrowthModel),
	}
}

// AddSizeDataPoint adds a historical size measurement.
func (bse *BackupSizeEstimator) AddSizeDataPoint(database string, point SizeDataPoint) {
	bse.mu.Lock()
	defer bse.mu.Unlock()

	if _, exists := bse.historicalSizes[database]; !exists {
		bse.historicalSizes[database] = make([]SizeDataPoint, 0)
	}

	bse.historicalSizes[database] = append(bse.historicalSizes[database], point)

	// Keep last 365 days of data
	if len(bse.historicalSizes[database]) > 365 {
		bse.historicalSizes[database] = bse.historicalSizes[database][len(bse.historicalSizes[database])-365:]
	}
}

// TrainGrowthModel trains a linear growth model for size prediction.
func (bse *BackupSizeEstimator) TrainGrowthModel(ctx context.Context, database string) (*GrowthModel, error) {
	bse.mu.RLock()
	data, exists := bse.historicalSizes[database]
	bse.mu.RUnlock()

	if !exists || len(data) < 7 {
		return nil, fmt.Errorf("insufficient data for training (need at least 7 data points, have %d)", len(data))
	}

	// Filter full backups only for growth calculation
	fullBackups := make([]SizeDataPoint, 0)
	for _, point := range data {
		if point.Type == "full" || point.Type == "" {
			fullBackups = append(fullBackups, point)
		}
	}

	if len(fullBackups) < 3 {
		return nil, fmt.Errorf("insufficient full backup data points (need at least 3, have %d)", len(fullBackups))
	}

	// Linear regression: size = growthRate * days + baseSize
	baseTime := fullBackups[0].Timestamp
	n := len(fullBackups)

	var sumX, sumY, sumXY, sumX2 float64
	for _, point := range fullBackups {
		x := point.Timestamp.Sub(baseTime).Hours() / 24.0 // days since base
		y := float64(point.Size)

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope (growth rate per day) and intercept (base size)
	growthRate := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)
	baseSize := (sumY - growthRate*sumX) / float64(n)

	// Calculate R² (goodness of fit)
	meanY := sumY / float64(n)
	var ssTotal, ssResidual float64
	for _, point := range fullBackups {
		x := point.Timestamp.Sub(baseTime).Hours() / 24.0
		y := float64(point.Size)
		predicted := baseSize + growthRate*x

		ssTotal += math.Pow(y-meanY, 2)
		ssResidual += math.Pow(y-predicted, 2)
	}

	r2 := 1 - (ssResidual / ssTotal)

	// Calculate confidence based on R² and sample size
	sampleConfidence := math.Min(100.0, math.Log10(float64(n))*30)
	fitConfidence := math.Max(0, r2*100)
	confidence := sampleConfidence*0.4 + fitConfidence*0.6

	model := &GrowthModel{
		DatabaseName:     database,
		GrowthRatePerDay: growthRate,
		BaseSize:         int64(baseSize),
		BaseTime:         baseTime,
		R2:               r2,
		Confidence:       confidence,
		SampleCount:      len(fullBackups),
		TrainedAt:        time.Now(),
	}

	bse.mu.Lock()
	bse.growthModels[database] = model
	bse.mu.Unlock()

	return model, nil
}

// EstimateSize estimates the backup size for a future date.
func (bse *BackupSizeEstimator) EstimateSize(ctx context.Context, database string, targetDate time.Time) (*SizeEstimation, error) {
	bse.mu.RLock()
	model, exists := bse.growthModels[database]
	bse.mu.RUnlock()

	if !exists {
		// Try to train model
		var err error
		model, err = bse.TrainGrowthModel(ctx, database)
		if err != nil {
			return nil, fmt.Errorf("no model available and training failed: %w", err)
		}
	}

	// Calculate days from base time
	daysProjected := int(targetDate.Sub(model.BaseTime).Hours() / 24.0)

	// Estimate size
	estimatedSize := model.BaseSize + int64(model.GrowthRatePerDay*float64(daysProjected))

	// Calculate confidence intervals (±20% for low confidence, ±10% for high confidence)
	marginPercent := 0.20 - (model.Confidence/100.0)*0.10
	margin := int64(float64(estimatedSize) * marginPercent)

	lowerBound := estimatedSize - margin
	upperBound := estimatedSize + margin

	// Build reasoning
	reasoning := []string{
		fmt.Sprintf("Growth rate: %s/day", formatBytes(int64(model.GrowthRatePerDay))),
		fmt.Sprintf("Model trained on %d data points", model.SampleCount),
		fmt.Sprintf("Model fit quality (R²): %.3f", model.R2),
		fmt.Sprintf("Projecting %d days from base", daysProjected),
	}

	estimation := &SizeEstimation{
		DatabaseName:     database,
		EstimatedSize:    estimatedSize,
		EstimatedSizeStr: formatBytes(estimatedSize),
		Confidence:       model.Confidence,
		EstimationMethod: "Linear Regression",
		GrowthRate:       model.GrowthRatePerDay,
		DaysProjected:    daysProjected,
		LowerBound:       lowerBound,
		UpperBound:       upperBound,
		Reasoning:        reasoning,
	}

	return estimation, nil
}

// formatBytes formats bytes to human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ==================== Helper Functions ====================

// Continue in next part due to size constraints...
