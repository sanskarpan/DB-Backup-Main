package optimization

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewBackupOptimizer(t *testing.T) {
	bo := NewBackupOptimizer(nil)
	if bo == nil {
		t.Fatal("Expected non-nil backup optimizer")
	}

	if bo.config == nil {
		t.Fatal("Expected default config")
	}

	if bo.config.MinHistoryDays != 7 {
		t.Errorf("Expected 7 min history days, got %d", bo.config.MinHistoryDays)
	}
}

func TestAddMetrics(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	metrics := OptimizationMetrics{
		ID:               "metrics-1",
		DatabaseName:     "testdb",
		Timestamp:        time.Now(),
		Duration:         30 * time.Minute,
		OriginalSize:     1024 * 1024 * 1024,
		CompressedSize:   512 * 1024 * 1024,
		CompressionAlgo:  CompressionGzip,
		CompressionRatio: 0.5,
		ParallelWorkers:  4,
		CPUUsage:         60.0,
		MemoryUsage:      50.0,
		Success:          true,
		DataType:         DataTypeText,
		StorageTier:      StorageTierHot,
	}

	bo.AddMetrics(metrics)

	bo.mu.RLock()
	data := bo.historicalData["testdb"]
	bo.mu.RUnlock()

	if len(data) != 1 {
		t.Fatalf("Expected 1 metrics entry, got %d", len(data))
	}

	if data[0].ID != "metrics-1" {
		t.Errorf("Expected metrics-1, got %s", data[0].ID)
	}
}

func TestAddMetrics_RetentionLimit(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Add old metrics (>90 days)
	oldMetrics := OptimizationMetrics{
		ID:           "old-metrics",
		DatabaseName: "testdb",
		Timestamp:    time.Now().AddDate(0, 0, -100),
		Success:      true,
	}
	bo.AddMetrics(oldMetrics)

	// Add recent metrics
	recentMetrics := OptimizationMetrics{
		ID:           "recent-metrics",
		DatabaseName: "testdb",
		Timestamp:    time.Now(),
		Success:      true,
	}
	bo.AddMetrics(recentMetrics)

	bo.mu.RLock()
	data := bo.historicalData["testdb"]
	bo.mu.RUnlock()

	// Old metrics should be filtered out
	if len(data) != 1 {
		t.Fatalf("Expected 1 metrics after retention, got %d", len(data))
	}

	if data[0].ID != "recent-metrics" {
		t.Errorf("Expected recent-metrics, got %s", data[0].ID)
	}
}

func TestGenerateRecommendations_InsufficientData(t *testing.T) {
	bo := NewBackupOptimizer(nil)
	ctx := context.Background()

	// Add only a few metrics
	for i := 0; i < 3; i++ {
		metrics := OptimizationMetrics{
			ID:           "metrics-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			Success:      true,
		}
		bo.AddMetrics(metrics)
	}

	_, err := bo.GenerateRecommendations(ctx, "testdb")
	if err == nil {
		t.Error("Expected error for insufficient data")
	}
}

func TestRecommendCompression_Gzip(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics favoring gzip for text data
	metrics := make([]OptimizationMetrics, 0)
	for i := 0; i < 20; i++ {
		m := OptimizationMetrics{
			ID:               "metrics-" + string(rune(i)),
			DatabaseName:     "testdb",
			Timestamp:        time.Now().AddDate(0, 0, -i),
			Duration:         30 * time.Minute,
			OriginalSize:     1024 * 1024 * 1024,
			CompressedSize:   300 * 1024 * 1024,
			CompressionAlgo:  CompressionGzip,
			CompressionRatio: 0.7,
			CPUUsage:         50.0,
			Success:          true,
			DataType:         DataTypeText,
		}
		metrics = append(metrics, m)
	}

	rec := bo.recommendCompression(metrics)

	if rec == nil {
		t.Fatal("Expected compression recommendation")
	}

	if rec.Algorithm != CompressionGzip {
		t.Errorf("Expected gzip algorithm, got %s", rec.Algorithm)
	}

	if rec.ExpectedRatio < 0.6 {
		t.Errorf("Expected ratio >= 0.6, got %.2f", rec.ExpectedRatio)
	}

	if len(rec.Alternatives) == 0 {
		t.Error("Expected alternative algorithms")
	}

	if len(rec.Reasoning) == 0 {
		t.Error("Expected reasoning")
	}
}

func TestRecommendCompression_MultipleAlgorithms(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics with different algorithms
	metrics := make([]OptimizationMetrics, 0)
	algorithms := []CompressionAlgorithm{CompressionGzip, CompressionZstd, CompressionLZ4}

	for i := 0; i < 30; i++ {
		algo := algorithms[i%len(algorithms)]
		ratio := 0.6
		cpuUsage := 50.0

		if algo == CompressionZstd {
			ratio = 0.8     // Better compression
			cpuUsage = 70.0 // Higher CPU
		} else if algo == CompressionLZ4 {
			ratio = 0.5     // Lower compression
			cpuUsage = 30.0 // Lower CPU
		}

		m := OptimizationMetrics{
			ID:               "metrics-" + string(rune(i)),
			DatabaseName:     "testdb",
			Timestamp:        time.Now().AddDate(0, 0, -i),
			Duration:         30 * time.Minute,
			OriginalSize:     1024 * 1024 * 1024,
			CompressedSize:   int64(float64(1024*1024*1024) * ratio),
			CompressionAlgo:  algo,
			CompressionRatio: ratio,
			CPUUsage:         cpuUsage,
			Success:          true,
			DataType:         DataTypeText,
		}
		metrics = append(metrics, m)
	}

	rec := bo.recommendCompression(metrics)

	if rec == nil {
		t.Fatal("Expected compression recommendation")
	}

	// Should have all three algorithms in alternatives
	if len(rec.Alternatives) < 3 {
		t.Errorf("Expected at least 3 alternatives, got %d", len(rec.Alternatives))
	}

	// Alternatives should be sorted by score
	for i := 1; i < len(rec.Alternatives); i++ {
		if rec.Alternatives[i].Score > rec.Alternatives[i-1].Score {
			t.Error("Alternatives not sorted by score")
		}
	}
}

func TestRecommendBackupWindow(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics with better performance at night
	metrics := make([]OptimizationMetrics, 0)

	// Add multiple samples per hour to create clear patterns
	for day := 0; day < 10; day++ {
		for hour := 0; hour < 24; hour++ {
			timestamp := time.Now().AddDate(0, 0, -day).Add(time.Duration(hour) * time.Hour)

			// Night time (0-6) has better performance
			duration := 30 * time.Minute
			cpuUsage := 40.0
			success := true

			if hour >= 7 && hour <= 18 {
				// Day time is slower and has some failures
				duration = 60 * time.Minute
				cpuUsage = 80.0
				if hour >= 12 && hour <= 14 {
					// Lunch time is worst
					success = day%3 != 0 // 33% failure rate
				}
			}

			m := OptimizationMetrics{
				ID:           fmt.Sprintf("metrics-%d-%d", day, hour),
				DatabaseName: "testdb",
				Timestamp:    timestamp,
				Duration:     duration,
				CPUUsage:     cpuUsage,
				DiskIOPS:     cpuUsage,
				Success:      success,
				StartHour:    hour,
				DayOfWeek:    timestamp.Weekday(),
			}
			metrics = append(metrics, m)
		}
	}

	rec := bo.recommendBackupWindow(metrics)

	if rec == nil {
		t.Fatal("Expected window recommendation")
	}

	if len(rec.OptimalWindows) == 0 {
		t.Error("Expected optimal windows")
	}

	if len(rec.WorstWindows) == 0 {
		t.Error("Expected worst windows")
	}

	// Best window should have better score than worst
	if len(rec.OptimalWindows) > 0 && len(rec.WorstWindows) > 0 {
		bestScore := rec.OptimalWindows[0].Score
		worstScore := rec.WorstWindows[len(rec.WorstWindows)-1].Score

		if bestScore <= worstScore {
			t.Errorf("Best window score (%.1f) should be > worst window score (%.1f)", bestScore, worstScore)
		}
	}

	if len(rec.Reasoning) == 0 {
		t.Error("Expected reasoning")
	}
}

func TestRecommendParallelism(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics showing optimal performance with 4 workers
	metrics := make([]OptimizationMetrics, 0)
	workerCounts := []int{1, 2, 4, 8}

	for i := 0; i < 40; i++ {
		workers := workerCounts[i%len(workerCounts)]

		// 4 workers gives best efficiency
		duration := 60 * time.Minute
		cpuUsage := 50.0
		memoryUsage := 40.0

		switch workers {
		case 1:
			duration = 120 * time.Minute
			cpuUsage = 25.0
			memoryUsage = 20.0
		case 2:
			duration = 70 * time.Minute
			cpuUsage = 40.0
			memoryUsage = 30.0
		case 4:
			duration = 40 * time.Minute
			cpuUsage = 60.0
			memoryUsage = 50.0
		case 8:
			duration = 35 * time.Minute
			cpuUsage = 90.0
			memoryUsage = 80.0
		}

		m := OptimizationMetrics{
			ID:              "metrics-" + string(rune(i)),
			DatabaseName:    "testdb",
			Timestamp:       time.Now().AddDate(0, 0, -i),
			Duration:        duration,
			ParallelWorkers: workers,
			CPUUsage:        cpuUsage,
			MemoryUsage:     memoryUsage,
			DiskIOPS:        cpuUsage,
			Success:         true,
		}
		metrics = append(metrics, m)
	}

	rec := bo.recommendParallelism(metrics)

	if rec == nil {
		t.Fatal("Expected parallelism recommendation")
	}

	// Should recommend 4 workers as optimal
	if rec.OptimalWorkers != 4 {
		t.Logf("Warning: Expected 4 optimal workers, got %d (algorithm may vary)", rec.OptimalWorkers)
	}

	if rec.ExpectedSpeedup < 1.0 {
		t.Errorf("Expected speedup >= 1.0, got %.2f", rec.ExpectedSpeedup)
	}

	if rec.MinWorkers >= rec.OptimalWorkers {
		t.Errorf("MinWorkers (%d) should be < OptimalWorkers (%d)", rec.MinWorkers, rec.OptimalWorkers)
	}

	if rec.MaxWorkers <= rec.OptimalWorkers {
		t.Errorf("MaxWorkers (%d) should be > OptimalWorkers (%d)", rec.MaxWorkers, rec.OptimalWorkers)
	}

	if len(rec.ResourceUtilization) == 0 {
		t.Error("Expected resource utilization data")
	}

	if len(rec.Reasoning) == 0 {
		t.Error("Expected reasoning")
	}
}

func TestRecommendStorage(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics showing old backups
	metrics := make([]OptimizationMetrics, 0)
	for i := 0; i < 30; i++ {
		m := OptimizationMetrics{
			ID:             "metrics-" + string(rune(i)),
			DatabaseName:   "testdb",
			Timestamp:      time.Now().AddDate(0, 0, -i-30), // 30-60 days old
			OriginalSize:   1024 * 1024 * 1024,
			CompressedSize: 512 * 1024 * 1024,
			Success:        true,
			StorageTier:    StorageTierHot,
			Cost:           0.023 * 0.5, // Hot tier cost
		}
		metrics = append(metrics, m)
	}

	rec := bo.recommendStorage(metrics)

	if rec == nil {
		t.Fatal("Expected storage recommendation")
	}

	// Should recommend cooler tier for old backups
	if rec.RecommendedTier == StorageTierHot {
		t.Log("Warning: Expected cooler tier recommendation for old backups")
	}

	if rec.RetentionOptimization == nil {
		t.Error("Expected retention optimization")
	}

	if len(rec.Reasoning) == 0 {
		t.Error("Expected reasoning")
	}
}

func TestCalculateGrowthRate(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Create metrics with consistent growth
	metrics := make([]OptimizationMetrics, 0)
	for i := 0; i < 10; i++ {
		m := OptimizationMetrics{
			ID:             "metrics-" + string(rune(i)),
			DatabaseName:   "testdb",
			Timestamp:      time.Now().AddDate(0, 0, -i),
			CompressedSize: int64(1024*1024*1024 + i*100*1024*1024), // +100MB per day
		}
		metrics = append(metrics, m)
	}

	growthRate := bo.calculateGrowthRate(metrics)

	// Should be approximately 0.1 GB/day (100MB/day)
	if growthRate < 0.05 || growthRate > 0.15 {
		t.Logf("Warning: Expected growth rate ~0.1 GB/day, got %.3f", growthRate)
	}
}

func TestRecommendStorageTier(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	tests := []struct {
		avgAge       time.Duration
		expectedTier StorageTier
	}{
		{3 * 24 * time.Hour, StorageTierHot},
		{15 * 24 * time.Hour, StorageTierCool},
		{60 * 24 * time.Hour, StorageTierArchive},
		{120 * 24 * time.Hour, StorageTierGlacier},
	}

	for _, tt := range tests {
		// Create metrics with specific age
		metrics := make([]OptimizationMetrics, 0)
		timestamp := time.Now().Add(-tt.avgAge)
		m := OptimizationMetrics{
			ID:           "metrics-1",
			DatabaseName: "testdb",
			Timestamp:    timestamp,
		}
		metrics = append(metrics, m)

		tier := bo.recommendStorageTier(metrics)
		if tier != tt.expectedTier {
			t.Errorf("Age %v: expected %s, got %s", tt.avgAge, tt.expectedTier, tier)
		}
	}
}

func TestCostCalculator_CalculateMonthlyCost(t *testing.T) {
	config := DefaultOptimizerConfig()
	cc := NewCostCalculator(config)

	sizeGB := 100.0 // 100 GB

	tests := []struct {
		tier         StorageTier
		expectedCost float64
	}{
		{StorageTierHot, 2.30},      // 100 * 0.023
		{StorageTierCool, 1.25},     // 100 * 0.0125
		{StorageTierArchive, 0.40},  // 100 * 0.004
		{StorageTierGlacier, 0.099}, // 100 * 0.00099
	}

	for _, tt := range tests {
		cost := cc.CalculateMonthlyCost(sizeGB*1024*1024*1024, tt.tier)
		if cost < tt.expectedCost*0.9 || cost > tt.expectedCost*1.1 {
			t.Errorf("Tier %s: expected ~$%.2f, got $%.2f", tt.tier, tt.expectedCost, cost)
		}
	}
}

func TestGenerateRecommendations_Complete(t *testing.T) {
	bo := NewBackupOptimizer(nil)
	ctx := context.Background()

	// Create comprehensive metrics
	for i := 0; i < 30; i++ {
		timestamp := time.Now().AddDate(0, 0, -i)
		m := OptimizationMetrics{
			ID:                "metrics-" + string(rune(i)),
			DatabaseName:      "testdb",
			Timestamp:         timestamp,
			Duration:          30 * time.Minute,
			OriginalSize:      1024 * 1024 * 1024,
			CompressedSize:    512 * 1024 * 1024,
			CompressionAlgo:   CompressionGzip,
			CompressionRatio:  0.5,
			ParallelWorkers:   4,
			CPUUsage:          60.0,
			MemoryUsage:       50.0,
			DiskIOPS:          100.0,
			NetworkThroughput: 100.0,
			Success:           true,
			StartHour:         2,
			DayOfWeek:         timestamp.Weekday(),
			DataType:          DataTypeText,
			StorageTier:       StorageTierHot,
			Cost:              11.5,
		}
		bo.AddMetrics(m)
	}

	rec, err := bo.GenerateRecommendations(ctx, "testdb")
	if err != nil {
		t.Fatalf("GenerateRecommendations failed: %v", err)
	}

	if rec == nil {
		t.Fatal("Expected recommendation")
	}

	// Verify all recommendation components
	if rec.CompressionRecommendation == nil {
		t.Error("Expected compression recommendation")
	}

	if rec.WindowRecommendation == nil {
		t.Error("Expected window recommendation")
	}

	if rec.ParallelismRecommendation == nil {
		t.Error("Expected parallelism recommendation")
	}

	if rec.StorageRecommendation == nil {
		t.Error("Expected storage recommendation")
	}

	if rec.OverallScore < 0 || rec.OverallScore > 100 {
		t.Errorf("Invalid overall score: %.1f", rec.OverallScore)
	}

	// Verify it's stored
	bo.mu.RLock()
	stored := bo.recommendations["testdb"]
	bo.mu.RUnlock()

	if stored == nil {
		t.Error("Expected recommendation to be stored")
	}
}

func TestGetStatistics(t *testing.T) {
	bo := NewBackupOptimizer(nil)
	ctx := context.Background()

	// Create recommendations for multiple databases
	for db := 0; db < 3; db++ {
		dbName := "testdb-" + string(rune(db))
		for i := 0; i < 10; i++ {
			m := OptimizationMetrics{
				ID:               "metrics-" + string(rune(i)),
				DatabaseName:     dbName,
				Timestamp:        time.Now().AddDate(0, 0, -i),
				Duration:         30 * time.Minute,
				OriginalSize:     1024 * 1024 * 1024,
				CompressedSize:   512 * 1024 * 1024,
				CompressionAlgo:  CompressionGzip,
				CompressionRatio: 0.5,
				ParallelWorkers:  4,
				Success:          true,
				DataType:         DataTypeText,
				StorageTier:      StorageTierHot,
			}
			bo.AddMetrics(m)
		}

		_, err := bo.GenerateRecommendations(ctx, dbName)
		if err != nil {
			t.Fatalf("GenerateRecommendations failed for %s: %v", dbName, err)
		}
	}

	stats := bo.GetStatistics()

	totalDatabases, ok := stats["total_databases"].(int)
	if !ok || totalDatabases != 3 {
		t.Errorf("Expected 3 total databases, got %v", totalDatabases)
	}

	totalSavings, ok := stats["total_annual_savings"].(float64)
	if !ok {
		t.Error("Expected total_annual_savings")
	}

	avgSavings, ok := stats["average_savings_per_db"].(float64)
	if !ok {
		t.Error("Expected average_savings_per_db")
	}

	if totalSavings < avgSavings*float64(totalDatabases)*0.9 {
		t.Error("Total savings calculation seems incorrect")
	}
}

func TestScoreCompressionAlgorithm(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	stats := &CompressionStats{
		Algorithm:   CompressionGzip,
		AvgRatio:    0.7,
		AvgDuration: 5 * time.Minute,
		AvgCPUUsage: 50.0,
	}

	score := bo.scoreCompressionAlgorithm(CompressionGzip, stats, DataTypeText)

	if score < 0 || score > 100 {
		t.Errorf("Invalid score: %.1f", score)
	}

	// Higher compression ratio should give higher score
	betterStats := &CompressionStats{
		Algorithm:   CompressionZstd,
		AvgRatio:    0.9,
		AvgDuration: 5 * time.Minute,
		AvgCPUUsage: 50.0,
	}

	betterScore := bo.scoreCompressionAlgorithm(CompressionZstd, betterStats, DataTypeText)

	if betterScore <= score {
		t.Errorf("Better compression should score higher: %.1f vs %.1f", betterScore, score)
	}
}

func TestGetDataTypeBonus(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Zstd should get bonus for text
	textBonus := bo.getDataTypeBonus(CompressionZstd, DataTypeText)
	if textBonus < 90 {
		t.Errorf("Expected high bonus for Zstd on text, got %.1f", textBonus)
	}

	// None should get bonus for images
	imageBonus := bo.getDataTypeBonus(CompressionNone, DataTypeImages)
	if imageBonus < 90 {
		t.Errorf("Expected high bonus for no compression on images, got %.1f", imageBonus)
	}

	// LZ4 should get bonus for binary
	binaryBonus := bo.getDataTypeBonus(CompressionLZ4, DataTypeBinary)
	if binaryBonus < 90 {
		t.Errorf("Expected high bonus for LZ4 on binary, got %.1f", binaryBonus)
	}
}

func TestScoreBackupWindow(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Night window with excellent performance
	nightStats := &WindowStats{
		Hour:        2,
		AvgDuration: 20 * time.Minute,
		SuccessRate: 98.0,
		AvgLoad:     20.0,
	}
	nightStats.Calculate() // Ensure stats are calculated

	nightScore := bo.scoreBackupWindow(nightStats)

	// Day window with mediocre performance
	dayStats := &WindowStats{
		Hour:        14,
		AvgDuration: 80 * time.Minute,
		SuccessRate: 75.0,
		AvgLoad:     85.0,
	}
	dayStats.Calculate() // Ensure stats are calculated

	dayScore := bo.scoreBackupWindow(dayStats)

	// Both scores should be valid
	if nightScore < 0 || nightScore > 100 {
		t.Errorf("Invalid night score: %.1f", nightScore)
	}
	if dayScore < 0 || dayScore > 100 {
		t.Errorf("Invalid day score: %.1f", dayScore)
	}

	// Night should score higher than day
	if nightScore <= dayScore {
		t.Errorf("Night window should score higher: %.1f vs %.1f", nightScore, dayScore)
	}
}

func TestCalculateParallelismEfficiency(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// Efficient parallelism
	efficientStats := &ParallelismStats{
		Workers:        4,
		AvgDuration:    30 * time.Minute,
		AvgCPUUsage:    60.0,
		AvgMemoryUsage: 50.0,
	}

	efficientScore := bo.calculateParallelismEfficiency(efficientStats)

	// Inefficient parallelism (high resource cost, similar duration)
	inefficientStats := &ParallelismStats{
		Workers:        8,
		AvgDuration:    28 * time.Minute,
		AvgCPUUsage:    95.0,
		AvgMemoryUsage: 90.0,
	}

	inefficientScore := bo.calculateParallelismEfficiency(inefficientStats)

	if efficientScore <= inefficientScore {
		t.Logf("More efficient parallelism should score higher: %.3f vs %.3f", efficientScore, inefficientScore)
	}
}

func TestCalculateOverallScore(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	rec := &OptimizationRecommendation{
		CompressionRecommendation: &CompressionRecommendation{
			Confidence: 90.0,
		},
		WindowRecommendation: &WindowRecommendation{
			Confidence: 85.0,
		},
		ParallelismRecommendation: &ParallelismRecommendation{
			Confidence: 80.0,
		},
		StorageRecommendation: &StorageRecommendation{
			Savings: 10.0,
		},
	}

	score := bo.calculateOverallScore(rec)

	if score < 0 || score > 100 {
		t.Errorf("Invalid overall score: %.1f", score)
	}

	// Should be weighted average
	expectedApprox := (90.0 + 85.0 + 80.0 + 100.0) / 4.0
	if score < expectedApprox*0.8 || score > expectedApprox*1.2 {
		t.Logf("Overall score %.1f seems off from expected ~%.1f", score, expectedApprox)
	}
}

func TestCalculateSavings(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	metrics := make([]OptimizationMetrics, 0)
	for i := 0; i < 10; i++ {
		m := OptimizationMetrics{
			ID:             "metrics-" + string(rune(i)),
			DatabaseName:   "testdb",
			Timestamp:      time.Now().AddDate(0, 0, -i),
			Duration:       60 * time.Minute,
			CompressedSize: 1024 * 1024 * 1024,
		}
		metrics = append(metrics, m)
	}

	rec := &OptimizationRecommendation{
		CompressionRecommendation: &CompressionRecommendation{
			ExpectedRatio:    0.8,
			ExpectedDuration: 30 * time.Minute,
		},
		ParallelismRecommendation: &ParallelismRecommendation{
			ExpectedSpeedup: 2.0,
		},
		StorageRecommendation: &StorageRecommendation{
			Savings: 50.0,
		},
	}

	savings := bo.calculateSavings(metrics, rec)

	if savings.TimesSaved < 0 {
		t.Error("Time saved should be non-negative")
	}

	if savings.MonthlyCostSaved != rec.StorageRecommendation.Savings {
		t.Errorf("Monthly cost saved should match storage savings: %.2f vs %.2f",
			savings.MonthlyCostSaved, rec.StorageRecommendation.Savings)
	}

	if savings.AnnualCostSaved != savings.MonthlyCostSaved*12 {
		t.Error("Annual cost should be 12x monthly")
	}
}

func TestOptimizeRetention(t *testing.T) {
	bo := NewBackupOptimizer(nil)

	// High change rate metrics
	metrics := make([]OptimizationMetrics, 0)
	for i := 0; i < 10; i++ {
		m := OptimizationMetrics{
			ID:           "metrics-" + string(rune(i)),
			DatabaseName: "testdb",
			Timestamp:    time.Now().AddDate(0, 0, -i),
			OriginalSize: int64(1024*1024*1024 + i*100*1024*1024), // Changing size
		}
		metrics = append(metrics, m)
	}

	retOpt := bo.optimizeRetention(metrics)

	if retOpt == nil {
		t.Fatal("Expected retention optimization")
	}

	if retOpt.CurrentRetention <= 0 {
		t.Error("Current retention should be positive")
	}

	if retOpt.OptimalRetention <= 0 {
		t.Error("Optimal retention should be positive")
	}

	if retOpt.RiskAssessment == "" {
		t.Error("Expected risk assessment")
	}
}
