// Package optimization provides intelligent backup optimization
package optimization

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// BackupOptimizer provides intelligent backup optimization recommendations.
type BackupOptimizer struct {
	mu              sync.RWMutex
	historicalData  map[string][]OptimizationMetrics
	recommendations map[string]*OptimizationRecommendation
	config          *OptimizerConfig
	costCalculator  *CostCalculator
}

// OptimizationMetrics represents metrics for a backup operation.
type OptimizationMetrics struct {
	ID                string
	DatabaseName      string
	Timestamp         time.Time
	Duration          time.Duration
	OriginalSize      int64
	CompressedSize    int64
	CompressionAlgo   CompressionAlgorithm
	CompressionRatio  float64
	ParallelWorkers   int
	CPUUsage          float64
	MemoryUsage       float64
	DiskIOPS          float64
	NetworkThroughput float64
	Success           bool
	StartHour         int
	DayOfWeek         time.Weekday
	DataType          DataType
	StorageTier       StorageTier
	Cost              float64
}

// CompressionAlgorithm represents a compression algorithm.
type CompressionAlgorithm string

const (
	CompressionNone   CompressionAlgorithm = "none"
	CompressionGzip   CompressionAlgorithm = "gzip"
	CompressionZstd   CompressionAlgorithm = "zstd"
	CompressionLZ4    CompressionAlgorithm = "lz4"
	CompressionBzip2  CompressionAlgorithm = "bzip2"
	CompressionSnappy CompressionAlgorithm = "snappy"
)

// DataType represents the type of data being backed up.
type DataType string

const (
	DataTypeText   DataType = "text"
	DataTypeBinary DataType = "binary"
	DataTypeMixed  DataType = "mixed"
	DataTypeJSON   DataType = "json"
	DataTypeImages DataType = "images"
	DataTypeVideo  DataType = "video"
)

// StorageTier represents storage tier.
type StorageTier string

const (
	StorageTierHot     StorageTier = "hot"     // Frequent access
	StorageTierCool    StorageTier = "cool"    // Infrequent access
	StorageTierArchive StorageTier = "archive" // Rare access
	StorageTierGlacier StorageTier = "glacier" // Long-term archive
)

// OptimizationRecommendation represents optimization recommendations.
type OptimizationRecommendation struct {
	DatabaseName              string
	GeneratedAt               time.Time
	CompressionRecommendation *CompressionRecommendation
	WindowRecommendation      *WindowRecommendation
	ParallelismRecommendation *ParallelismRecommendation
	StorageRecommendation     *StorageRecommendation
	OverallScore              float64 // 0-100
	EstimatedSavings          CostSavings
}

// CompressionRecommendation recommends optimal compression.
type CompressionRecommendation struct {
	Algorithm        CompressionAlgorithm
	Confidence       float64 // 0-100
	ExpectedRatio    float64
	ExpectedDuration time.Duration
	CPUCost          float64 // 0-100
	Reasoning        []string
	Alternatives     []CompressionAlternative
}

// CompressionAlternative represents an alternative compression option.
type CompressionAlternative struct {
	Algorithm        CompressionAlgorithm
	Score            float64
	ExpectedRatio    float64
	ExpectedDuration time.Duration
	TradeOff         string
}

// WindowRecommendation recommends optimal backup windows.
type WindowRecommendation struct {
	OptimalWindows  []BackupWindow
	WorstWindows    []BackupWindow
	Confidence      float64
	AverageDuration time.Duration
	Reasoning       []string
}

// BackupWindow represents a backup time window.
type BackupWindow struct {
	DayOfWeek   time.Weekday
	StartHour   int
	EndHour     int
	Score       float64 // 0-100
	AvgDuration time.Duration
	SuccessRate float64
	AvgLoad     float64
}

// ParallelismRecommendation recommends optimal parallelism.
type ParallelismRecommendation struct {
	OptimalWorkers      int
	MinWorkers          int
	MaxWorkers          int
	Confidence          float64
	ExpectedSpeedup     float64
	ResourceUtilization map[string]float64
	Reasoning           []string
}

// StorageRecommendation recommends storage optimization.
type StorageRecommendation struct {
	RecommendedTier       StorageTier
	CurrentCost           float64
	ProjectedCost         float64
	Savings               float64
	GrowthRate            float64 // GB per day
	RetentionOptimization *RetentionOptimization
	Reasoning             []string
}

// RetentionOptimization recommends retention policy optimization.
type RetentionOptimization struct {
	CurrentRetention int // days
	OptimalRetention int // days
	EstimatedSavings float64
	RiskAssessment   string
}

// CostSavings represents cost savings from optimization.
type CostSavings struct {
	TimesSaved       time.Duration
	StorageSaved     int64 // bytes
	MonthlyCostSaved float64
	AnnualCostSaved  float64
}

// OptimizerConfig configures the optimizer.
type OptimizerConfig struct {
	MinHistoryDays      int
	ConfidenceThreshold float64
	StorageCostPerGB    float64
	CPUCostPerHour      float64
	NetworkCostPerGB    float64
	TargetSuccessRate   float64
}

// CostCalculator calculates storage costs.
type CostCalculator struct {
	tierPricing map[StorageTier]float64
	config      *OptimizerConfig
}

// NewBackupOptimizer creates a new backup optimizer.
func NewBackupOptimizer(config *OptimizerConfig) *BackupOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	return &BackupOptimizer{
		historicalData:  make(map[string][]OptimizationMetrics),
		recommendations: make(map[string]*OptimizationRecommendation),
		config:          config,
		costCalculator:  NewCostCalculator(config),
	}
}

// DefaultOptimizerConfig returns default configuration.
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		MinHistoryDays:      7,
		ConfidenceThreshold: 70.0,
		StorageCostPerGB:    0.023, // S3 Standard pricing
		CPUCostPerHour:      0.05,
		NetworkCostPerGB:    0.09,
		TargetSuccessRate:   95.0,
	}
}

// NewCostCalculator creates a new cost calculator.
func NewCostCalculator(config *OptimizerConfig) *CostCalculator {
	return &CostCalculator{
		tierPricing: map[StorageTier]float64{
			StorageTierHot:     0.023,   // S3 Standard
			StorageTierCool:    0.0125,  // S3 Infrequent Access
			StorageTierArchive: 0.004,   // S3 Glacier Flexible
			StorageTierGlacier: 0.00099, // S3 Glacier Deep Archive
		},
		config: config,
	}
}

// AddMetrics adds optimization metrics.
func (bo *BackupOptimizer) AddMetrics(metrics OptimizationMetrics) {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	if bo.historicalData[metrics.DatabaseName] == nil {
		bo.historicalData[metrics.DatabaseName] = make([]OptimizationMetrics, 0)
	}

	bo.historicalData[metrics.DatabaseName] = append(bo.historicalData[metrics.DatabaseName], metrics)

	// Keep only recent history (last 90 days)
	cutoff := time.Now().AddDate(0, 0, -90)
	filtered := make([]OptimizationMetrics, 0)
	for _, m := range bo.historicalData[metrics.DatabaseName] {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	bo.historicalData[metrics.DatabaseName] = filtered
}

// GenerateRecommendations generates optimization recommendations.
func (bo *BackupOptimizer) GenerateRecommendations(ctx context.Context, databaseName string) (*OptimizationRecommendation, error) {
	bo.mu.Lock()
	defer bo.mu.Unlock()

	metrics, exists := bo.historicalData[databaseName]
	if !exists || len(metrics) < bo.config.MinHistoryDays {
		return nil, fmt.Errorf("insufficient historical data for %s", databaseName)
	}

	recommendation := &OptimizationRecommendation{
		DatabaseName: databaseName,
		GeneratedAt:  time.Now(),
	}

	// Generate compression recommendation
	recommendation.CompressionRecommendation = bo.recommendCompression(metrics)

	// Generate window recommendation
	recommendation.WindowRecommendation = bo.recommendBackupWindow(metrics)

	// Generate parallelism recommendation
	recommendation.ParallelismRecommendation = bo.recommendParallelism(metrics)

	// Generate storage recommendation
	recommendation.StorageRecommendation = bo.recommendStorage(metrics)

	// Calculate overall score
	recommendation.OverallScore = bo.calculateOverallScore(recommendation)

	// Calculate estimated savings
	recommendation.EstimatedSavings = bo.calculateSavings(metrics, recommendation)

	// Store recommendation
	bo.recommendations[databaseName] = recommendation

	return recommendation, nil
}

// recommendCompression recommends optimal compression algorithm.
func (bo *BackupOptimizer) recommendCompression(metrics []OptimizationMetrics) *CompressionRecommendation {
	// Analyze historical compression performance
	algoPerformance := make(map[CompressionAlgorithm]*CompressionStats)

	for _, m := range metrics {
		if !m.Success {
			continue
		}

		if algoPerformance[m.CompressionAlgo] == nil {
			algoPerformance[m.CompressionAlgo] = &CompressionStats{
				Algorithm: m.CompressionAlgo,
				Samples:   make([]CompressionSample, 0),
			}
		}

		algoPerformance[m.CompressionAlgo].Samples = append(
			algoPerformance[m.CompressionAlgo].Samples,
			CompressionSample{
				Ratio:    m.CompressionRatio,
				Duration: m.Duration,
				CPUUsage: m.CPUUsage,
			},
		)
	}

	// Calculate statistics for each algorithm
	for _, stats := range algoPerformance {
		stats.Calculate()
	}

	// Determine data type from most recent metrics
	dataType := DataTypeMixed
	if len(metrics) > 0 {
		dataType = metrics[len(metrics)-1].DataType
	}

	// Score algorithms based on data type and performance
	bestAlgo := CompressionGzip
	bestScore := 0.0
	alternatives := make([]CompressionAlternative, 0)

	for algo, stats := range algoPerformance {
		score := bo.scoreCompressionAlgorithm(algo, stats, dataType)

		alt := CompressionAlternative{
			Algorithm:        algo,
			Score:            score,
			ExpectedRatio:    stats.AvgRatio,
			ExpectedDuration: stats.AvgDuration,
			TradeOff:         bo.getCompressionTradeOff(algo),
		}
		alternatives = append(alternatives, alt)

		if score > bestScore {
			bestScore = score
			bestAlgo = algo
		}
	}

	// Sort alternatives by score
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	bestStats := algoPerformance[bestAlgo]
	reasoning := bo.getCompressionReasoning(bestAlgo, bestStats, dataType)

	return &CompressionRecommendation{
		Algorithm:        bestAlgo,
		Confidence:       bestScore,
		ExpectedRatio:    bestStats.AvgRatio,
		ExpectedDuration: bestStats.AvgDuration,
		CPUCost:          bestStats.AvgCPUUsage,
		Reasoning:        reasoning,
		Alternatives:     alternatives,
	}
}

// scoreCompressionAlgorithm scores a compression algorithm.
func (bo *BackupOptimizer) scoreCompressionAlgorithm(algo CompressionAlgorithm, stats *CompressionStats, dataType DataType) float64 {
	score := 0.0

	// Base score from compression ratio (40%)
	score += stats.AvgRatio * 40.0

	// Speed score (30%)
	speedScore := 100.0 / (1.0 + stats.AvgDuration.Seconds()/60.0)
	score += speedScore * 0.3

	// CPU efficiency score (20%)
	cpuScore := 100.0 - stats.AvgCPUUsage
	score += cpuScore * 0.2

	// Data type bonus (10%)
	dataTypeBonus := bo.getDataTypeBonus(algo, dataType)
	score += dataTypeBonus * 0.1

	return math.Min(100.0, score)
}

// getDataTypeBonus returns bonus score based on data type.
func (bo *BackupOptimizer) getDataTypeBonus(algo CompressionAlgorithm, dataType DataType) float64 {
	bonuses := map[DataType]map[CompressionAlgorithm]float64{
		DataTypeText: {
			CompressionGzip:   90.0,
			CompressionZstd:   95.0,
			CompressionBzip2:  85.0,
			CompressionLZ4:    70.0,
			CompressionSnappy: 65.0,
		},
		DataTypeBinary: {
			CompressionLZ4:    95.0,
			CompressionSnappy: 90.0,
			CompressionZstd:   85.0,
			CompressionGzip:   70.0,
		},
		DataTypeJSON: {
			CompressionZstd:  100.0,
			CompressionGzip:  95.0,
			CompressionBzip2: 90.0,
		},
		DataTypeImages: {
			CompressionNone:   100.0, // Already compressed
			CompressionLZ4:    80.0,
			CompressionSnappy: 75.0,
		},
	}

	if bonus, exists := bonuses[dataType][algo]; exists {
		return bonus
	}
	return 50.0
}

// getCompressionTradeOff returns tradeoff description.
func (bo *BackupOptimizer) getCompressionTradeOff(algo CompressionAlgorithm) string {
	tradeoffs := map[CompressionAlgorithm]string{
		CompressionNone:   "Fastest, no compression",
		CompressionLZ4:    "Very fast, moderate compression",
		CompressionSnappy: "Fast, good for already compressed data",
		CompressionGzip:   "Balanced speed and compression",
		CompressionZstd:   "Best compression, moderate speed",
		CompressionBzip2:  "High compression, slower",
	}
	return tradeoffs[algo]
}

// getCompressionReasoning returns reasoning for compression choice.
func (bo *BackupOptimizer) getCompressionReasoning(algo CompressionAlgorithm, stats *CompressionStats, dataType DataType) []string {
	reasoning := make([]string, 0)

	reasoning = append(reasoning, fmt.Sprintf("Algorithm %s achieves %.1f%% compression ratio", algo, stats.AvgRatio*100))
	reasoning = append(reasoning, fmt.Sprintf("Average compression time: %s", stats.AvgDuration.Round(time.Second)))
	reasoning = append(reasoning, fmt.Sprintf("CPU usage: %.1f%%", stats.AvgCPUUsage))
	reasoning = append(reasoning, fmt.Sprintf("Optimal for %s data type", dataType))

	if stats.AvgRatio > 0.7 {
		reasoning = append(reasoning, "Excellent compression efficiency")
	}

	if stats.AvgDuration < 5*time.Minute {
		reasoning = append(reasoning, "Fast compression speed")
	}

	return reasoning
}

// recommendBackupWindow recommends optimal backup windows.
func (bo *BackupOptimizer) recommendBackupWindow(metrics []OptimizationMetrics) *WindowRecommendation {
	// Analyze performance by time window
	windowStats := make(map[string]*WindowStats)

	for _, m := range metrics {
		if !m.Success {
			continue
		}

		key := fmt.Sprintf("%s-%d", m.DayOfWeek, m.StartHour)
		if windowStats[key] == nil {
			windowStats[key] = &WindowStats{
				DayOfWeek: m.DayOfWeek,
				Hour:      m.StartHour,
				Samples:   make([]WindowSample, 0),
			}
		}

		windowStats[key].Samples = append(windowStats[key].Samples, WindowSample{
			Duration: m.Duration,
			CPUUsage: m.CPUUsage,
			DiskIOPS: m.DiskIOPS,
			Success:  m.Success,
		})
	}

	// Calculate statistics
	windows := make([]BackupWindow, 0)
	for _, stats := range windowStats {
		stats.Calculate()

		window := BackupWindow{
			DayOfWeek:   stats.DayOfWeek,
			StartHour:   stats.Hour,
			EndHour:     (stats.Hour + 2) % 24,
			Score:       bo.scoreBackupWindow(stats),
			AvgDuration: stats.AvgDuration,
			SuccessRate: stats.SuccessRate,
			AvgLoad:     stats.AvgLoad,
		}
		windows = append(windows, window)
	}

	// Sort by score
	sort.Slice(windows, func(i, j int) bool {
		return windows[i].Score > windows[j].Score
	})

	// Get top 3 optimal windows
	optimalCount := 3
	if len(windows) < optimalCount {
		optimalCount = len(windows)
	}
	optimalWindows := windows[:optimalCount]

	// Get bottom 3 worst windows
	worstCount := 3
	if len(windows) < worstCount {
		worstCount = len(windows)
	}
	worstWindows := windows[len(windows)-worstCount:]

	// Calculate average duration
	totalDuration := time.Duration(0)
	for _, w := range windows {
		totalDuration += w.AvgDuration
	}
	avgDuration := totalDuration / time.Duration(len(windows))

	reasoning := bo.getWindowReasoning(optimalWindows, worstWindows)

	return &WindowRecommendation{
		OptimalWindows:  optimalWindows,
		WorstWindows:    worstWindows,
		Confidence:      85.0,
		AverageDuration: avgDuration,
		Reasoning:       reasoning,
	}
}

// scoreBackupWindow scores a backup window.
func (bo *BackupOptimizer) scoreBackupWindow(stats *WindowStats) float64 {
	score := 0.0

	// Success rate (40%)
	score += stats.SuccessRate * 0.4

	// Speed (30%)
	speedScore := 100.0 / (1.0 + stats.AvgDuration.Hours())
	score += speedScore * 0.3

	// Low load (20%)
	loadScore := 100.0 - stats.AvgLoad
	score += loadScore * 0.2

	// Off-peak bonus (10%)
	if stats.Hour >= 0 && stats.Hour <= 6 {
		score += 10.0 // Night time bonus
	}

	return math.Min(100.0, score)
}

// getWindowReasoning returns reasoning for window recommendations.
func (bo *BackupOptimizer) getWindowReasoning(optimal, worst []BackupWindow) []string {
	reasoning := make([]string, 0)

	if len(optimal) > 0 {
		best := optimal[0]
		reasoning = append(reasoning, fmt.Sprintf("Best window: %s at %02d:00 (score: %.1f)", best.DayOfWeek, best.StartHour, best.Score))
		reasoning = append(reasoning, fmt.Sprintf("Average duration in best window: %s", best.AvgDuration.Round(time.Second)))
	}

	if len(worst) > 0 {
		worst := worst[0]
		reasoning = append(reasoning, fmt.Sprintf("Avoid: %s at %02d:00 (score: %.1f)", worst.DayOfWeek, worst.StartHour, worst.Score))
	}

	reasoning = append(reasoning, "Recommendation based on historical success rates and system load")

	return reasoning
}

// recommendParallelism recommends optimal parallelism.
func (bo *BackupOptimizer) recommendParallelism(metrics []OptimizationMetrics) *ParallelismRecommendation {
	// Analyze performance by worker count
	workerPerformance := make(map[int]*ParallelismStats)

	for _, m := range metrics {
		if !m.Success {
			continue
		}

		if workerPerformance[m.ParallelWorkers] == nil {
			workerPerformance[m.ParallelWorkers] = &ParallelismStats{
				Workers: m.ParallelWorkers,
				Samples: make([]ParallelismSample, 0),
			}
		}

		workerPerformance[m.ParallelWorkers].Samples = append(
			workerPerformance[m.ParallelWorkers].Samples,
			ParallelismSample{
				Duration:    m.Duration,
				CPUUsage:    m.CPUUsage,
				MemoryUsage: m.MemoryUsage,
				DiskIOPS:    m.DiskIOPS,
			},
		)
	}

	// Calculate statistics
	for _, stats := range workerPerformance {
		stats.Calculate()
	}

	// Find optimal worker count (best duration/resource ratio)
	optimalWorkers := 4
	bestEfficiency := 0.0

	for workers, stats := range workerPerformance {
		efficiency := bo.calculateParallelismEfficiency(stats)
		if efficiency > bestEfficiency {
			bestEfficiency = efficiency
			optimalWorkers = workers
		}
	}

	optimalStats := workerPerformance[optimalWorkers]

	// Calculate expected speedup
	baselineWorkers := 1
	baselineStats := workerPerformance[baselineWorkers]
	expectedSpeedup := 1.0
	if baselineStats != nil && optimalStats != nil {
		expectedSpeedup = baselineStats.AvgDuration.Seconds() / optimalStats.AvgDuration.Seconds()
	}

	resourceUtil := map[string]float64{
		"cpu":    optimalStats.AvgCPUUsage,
		"memory": optimalStats.AvgMemoryUsage,
		"disk":   optimalStats.AvgDiskIOPS,
	}

	reasoning := bo.getParallelismReasoning(optimalWorkers, expectedSpeedup, resourceUtil)

	return &ParallelismRecommendation{
		OptimalWorkers:      optimalWorkers,
		MinWorkers:          max(1, optimalWorkers/2),
		MaxWorkers:          optimalWorkers * 2,
		Confidence:          80.0,
		ExpectedSpeedup:     expectedSpeedup,
		ResourceUtilization: resourceUtil,
		Reasoning:           reasoning,
	}
}

// calculateParallelismEfficiency calculates parallelism efficiency.
func (bo *BackupOptimizer) calculateParallelismEfficiency(stats *ParallelismStats) float64 {
	// Efficiency = Speed improvement / Resource cost
	speedScore := 100.0 / (1.0 + stats.AvgDuration.Minutes())
	resourceCost := (stats.AvgCPUUsage + stats.AvgMemoryUsage) / 2.0

	if resourceCost == 0 {
		return 0
	}

	return speedScore / resourceCost
}

// getParallelismReasoning returns reasoning for parallelism recommendation.
func (bo *BackupOptimizer) getParallelismReasoning(workers int, speedup float64, resources map[string]float64) []string {
	reasoning := make([]string, 0)

	reasoning = append(reasoning, fmt.Sprintf("Optimal parallelism: %d workers", workers))
	reasoning = append(reasoning, fmt.Sprintf("Expected speedup: %.2fx", speedup))
	reasoning = append(reasoning, fmt.Sprintf("CPU utilization: %.1f%%", resources["cpu"]))
	reasoning = append(reasoning, fmt.Sprintf("Memory utilization: %.1f%%", resources["memory"]))

	if workers > 4 {
		reasoning = append(reasoning, "High parallelism recommended for large databases")
	} else if workers <= 2 {
		reasoning = append(reasoning, "Low parallelism recommended to reduce resource contention")
	}

	return reasoning
}

// recommendStorage recommends storage optimization.
func (bo *BackupOptimizer) recommendStorage(metrics []OptimizationMetrics) *StorageRecommendation {
	// Calculate current storage usage and growth
	totalSize := int64(0)
	totalCost := 0.0
	var sizes []int64

	for _, m := range metrics {
		totalSize += m.CompressedSize
		totalCost += m.Cost
		sizes = append(sizes, m.CompressedSize)
	}

	// Calculate growth rate
	growthRate := bo.calculateGrowthRate(metrics)

	// Determine access patterns
	currentTier := StorageTierHot
	if len(metrics) > 0 {
		currentTier = metrics[len(metrics)-1].StorageTier
	}

	// Recommend tier based on access patterns
	recommendedTier := bo.recommendStorageTier(metrics)

	// Calculate costs
	avgSize := float64(totalSize) / float64(len(metrics))
	currentCost := bo.costCalculator.CalculateMonthlyCost(avgSize, currentTier)
	projectedCost := bo.costCalculator.CalculateMonthlyCost(avgSize, recommendedTier)
	savings := currentCost - projectedCost

	// Retention optimization
	retentionOpt := bo.optimizeRetention(metrics)

	reasoning := bo.getStorageReasoning(currentTier, recommendedTier, growthRate, savings)

	return &StorageRecommendation{
		RecommendedTier:       recommendedTier,
		CurrentCost:           currentCost,
		ProjectedCost:         projectedCost,
		Savings:               savings,
		GrowthRate:            growthRate,
		RetentionOptimization: retentionOpt,
		Reasoning:             reasoning,
	}
}

// calculateGrowthRate calculates data growth rate in GB per day.
func (bo *BackupOptimizer) calculateGrowthRate(metrics []OptimizationMetrics) float64 {
	if len(metrics) < 2 {
		return 0.0
	}

	// Sort by timestamp
	sorted := make([]OptimizationMetrics, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	first := sorted[0]
	last := sorted[len(sorted)-1]

	sizeGrowth := float64(last.CompressedSize-first.CompressedSize) / (1024 * 1024 * 1024) // GB
	daysElapsed := last.Timestamp.Sub(first.Timestamp).Hours() / 24.0

	if daysElapsed == 0 {
		return 0.0
	}

	return sizeGrowth / daysElapsed
}

// recommendStorageTier recommends optimal storage tier.
func (bo *BackupOptimizer) recommendStorageTier(metrics []OptimizationMetrics) StorageTier {
	// Analyze access patterns (simplified for now)
	avgAge := bo.calculateAverageAge(metrics)

	if avgAge < 7*24*time.Hour {
		return StorageTierHot
	} else if avgAge < 30*24*time.Hour {
		return StorageTierCool
	} else if avgAge < 90*24*time.Hour {
		return StorageTierArchive
	}
	return StorageTierGlacier
}

// calculateAverageAge calculates average age of backups.
func (bo *BackupOptimizer) calculateAverageAge(metrics []OptimizationMetrics) time.Duration {
	if len(metrics) == 0 {
		return 0
	}

	totalAge := time.Duration(0)
	now := time.Now()

	for _, m := range metrics {
		totalAge += now.Sub(m.Timestamp)
	}

	return totalAge / time.Duration(len(metrics))
}

// optimizeRetention optimizes retention policy.
func (bo *BackupOptimizer) optimizeRetention(metrics []OptimizationMetrics) *RetentionOptimization {
	// Current retention (assume 30 days default)
	currentRetention := 30

	// Calculate optimal retention based on access patterns
	optimalRetention := bo.calculateOptimalRetention(metrics)

	// Calculate savings
	savingsPercent := float64(currentRetention-optimalRetention) / float64(currentRetention) * 100
	estimatedSavings := savingsPercent * bo.config.StorageCostPerGB * 1000 // Estimate

	riskAssessment := "LOW"
	if optimalRetention < 7 {
		riskAssessment = "HIGH"
	} else if optimalRetention < 14 {
		riskAssessment = "MEDIUM"
	}

	return &RetentionOptimization{
		CurrentRetention: currentRetention,
		OptimalRetention: optimalRetention,
		EstimatedSavings: estimatedSavings,
		RiskAssessment:   riskAssessment,
	}
}

// calculateOptimalRetention calculates optimal retention period.
func (bo *BackupOptimizer) calculateOptimalRetention(metrics []OptimizationMetrics) int {
	// Simplified: recommend 7 days for frequent changes, 30 for stable
	if len(metrics) < 7 {
		return 30
	}

	// Check data change rate
	changeRate := bo.calculateChangeRate(metrics)

	if changeRate > 0.5 {
		return 7 // High change rate, shorter retention
	} else if changeRate > 0.2 {
		return 14
	}
	return 30
}

// calculateChangeRate calculates data change rate.
func (bo *BackupOptimizer) calculateChangeRate(metrics []OptimizationMetrics) float64 {
	if len(metrics) < 2 {
		return 0.0
	}

	changes := 0
	for i := 1; i < len(metrics); i++ {
		if metrics[i].OriginalSize != metrics[i-1].OriginalSize {
			changes++
		}
	}

	return float64(changes) / float64(len(metrics)-1)
}

// getStorageReasoning returns reasoning for storage recommendation.
func (bo *BackupOptimizer) getStorageReasoning(current, recommended StorageTier, growth, savings float64) []string {
	reasoning := make([]string, 0)

	if current != recommended {
		reasoning = append(reasoning, fmt.Sprintf("Switch from %s to %s tier for cost optimization", current, recommended))
	} else {
		reasoning = append(reasoning, fmt.Sprintf("Current %s tier is optimal", current))
	}

	reasoning = append(reasoning, fmt.Sprintf("Data growth rate: %.2f GB/day", growth))

	if savings > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Estimated monthly savings: $%.2f", savings))
	}

	if growth > 10 {
		reasoning = append(reasoning, "High growth rate detected, consider cleanup policies")
	}

	return reasoning
}

// calculateOverallScore calculates overall optimization score.
func (bo *BackupOptimizer) calculateOverallScore(rec *OptimizationRecommendation) float64 {
	score := 0.0

	// Compression score (25%)
	score += rec.CompressionRecommendation.Confidence * 0.25

	// Window score (25%)
	score += rec.WindowRecommendation.Confidence * 0.25

	// Parallelism score (25%)
	score += rec.ParallelismRecommendation.Confidence * 0.25

	// Storage optimization score (25%)
	storageScore := 100.0
	if rec.StorageRecommendation.Savings > 0 {
		storageScore = math.Min(100.0, rec.StorageRecommendation.Savings*10)
	}
	score += storageScore * 0.25

	return score
}

// calculateSavings calculates estimated cost savings.
func (bo *BackupOptimizer) calculateSavings(metrics []OptimizationMetrics, rec *OptimizationRecommendation) CostSavings {
	// Calculate time savings from optimal compression and parallelism
	avgDuration := time.Duration(0)
	for _, m := range metrics {
		avgDuration += m.Duration
	}
	avgDuration /= time.Duration(len(metrics))

	expectedDuration := rec.CompressionRecommendation.ExpectedDuration
	timeSaved := avgDuration - expectedDuration
	if timeSaved < 0 {
		timeSaved = 0
	}

	// Apply parallelism speedup
	timeSaved = time.Duration(float64(timeSaved) * rec.ParallelismRecommendation.ExpectedSpeedup)

	// Storage savings
	avgSize := int64(0)
	for _, m := range metrics {
		avgSize += m.CompressedSize
	}
	avgSize /= int64(len(metrics))

	expectedSize := int64(float64(avgSize) * rec.CompressionRecommendation.ExpectedRatio)
	storageSaved := avgSize - expectedSize

	// Cost savings
	monthlySavings := rec.StorageRecommendation.Savings

	return CostSavings{
		TimesSaved:       timeSaved,
		StorageSaved:     storageSaved,
		MonthlyCostSaved: monthlySavings,
		AnnualCostSaved:  monthlySavings * 12,
	}
}

// CalculateMonthlyCost calculates monthly storage cost.
func (cc *CostCalculator) CalculateMonthlyCost(sizeBytes float64, tier StorageTier) float64 {
	sizeGB := sizeBytes / (1024 * 1024 * 1024)
	pricePerGB := cc.tierPricing[tier]
	return sizeGB * pricePerGB
}

// Helper structures

type CompressionStats struct {
	Algorithm   CompressionAlgorithm
	Samples     []CompressionSample
	AvgRatio    float64
	AvgDuration time.Duration
	AvgCPUUsage float64
}

type CompressionSample struct {
	Ratio    float64
	Duration time.Duration
	CPUUsage float64
}

func (cs *CompressionStats) Calculate() {
	if len(cs.Samples) == 0 {
		return
	}

	totalRatio := 0.0
	totalDuration := time.Duration(0)
	totalCPU := 0.0

	for _, s := range cs.Samples {
		totalRatio += s.Ratio
		totalDuration += s.Duration
		totalCPU += s.CPUUsage
	}

	cs.AvgRatio = totalRatio / float64(len(cs.Samples))
	cs.AvgDuration = totalDuration / time.Duration(len(cs.Samples))
	cs.AvgCPUUsage = totalCPU / float64(len(cs.Samples))
}

type WindowStats struct {
	DayOfWeek   time.Weekday
	Hour        int
	Samples     []WindowSample
	AvgDuration time.Duration
	SuccessRate float64
	AvgLoad     float64
}

type WindowSample struct {
	Duration time.Duration
	CPUUsage float64
	DiskIOPS float64
	Success  bool
}

func (ws *WindowStats) Calculate() {
	if len(ws.Samples) == 0 {
		return
	}

	totalDuration := time.Duration(0)
	totalLoad := 0.0
	successes := 0

	for _, s := range ws.Samples {
		totalDuration += s.Duration
		totalLoad += s.CPUUsage
		if s.Success {
			successes++
		}
	}

	ws.AvgDuration = totalDuration / time.Duration(len(ws.Samples))
	ws.AvgLoad = totalLoad / float64(len(ws.Samples))
	ws.SuccessRate = float64(successes) / float64(len(ws.Samples)) * 100.0
}

type ParallelismStats struct {
	Workers        int
	Samples        []ParallelismSample
	AvgDuration    time.Duration
	AvgCPUUsage    float64
	AvgMemoryUsage float64
	AvgDiskIOPS    float64
}

type ParallelismSample struct {
	Duration    time.Duration
	CPUUsage    float64
	MemoryUsage float64
	DiskIOPS    float64
}

func (ps *ParallelismStats) Calculate() {
	if len(ps.Samples) == 0 {
		return
	}

	totalDuration := time.Duration(0)
	totalCPU := 0.0
	totalMemory := 0.0
	totalDisk := 0.0

	for _, s := range ps.Samples {
		totalDuration += s.Duration
		totalCPU += s.CPUUsage
		totalMemory += s.MemoryUsage
		totalDisk += s.DiskIOPS
	}

	ps.AvgDuration = totalDuration / time.Duration(len(ps.Samples))
	ps.AvgCPUUsage = totalCPU / float64(len(ps.Samples))
	ps.AvgMemoryUsage = totalMemory / float64(len(ps.Samples))
	ps.AvgDiskIOPS = totalDisk / float64(len(ps.Samples))
}

// GetRecommendations returns all recommendations.
func (bo *BackupOptimizer) GetRecommendations() map[string]*OptimizationRecommendation {
	bo.mu.RLock()
	defer bo.mu.RUnlock()

	recommendations := make(map[string]*OptimizationRecommendation)
	for k, v := range bo.recommendations {
		recommendations[k] = v
	}
	return recommendations
}

// GetStatistics returns optimizer statistics.
func (bo *BackupOptimizer) GetStatistics() map[string]interface{} {
	bo.mu.RLock()
	defer bo.mu.RUnlock()

	totalSavings := 0.0
	totalDatabases := len(bo.recommendations)

	for _, rec := range bo.recommendations {
		totalSavings += rec.EstimatedSavings.AnnualCostSaved
	}

	return map[string]interface{}{
		"total_databases":        totalDatabases,
		"total_annual_savings":   totalSavings,
		"average_savings_per_db": totalSavings / float64(max(1, totalDatabases)),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
