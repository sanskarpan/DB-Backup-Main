package ai

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ==================== Test Suite for Smart Features ====================

// TestPredictiveScheduler tests the ML-based predictive scheduler.
func TestPredictiveScheduler(t *testing.T) {
	t.Run("AddDataPoint", func(t *testing.T) {
		ps := NewPredictiveScheduler()
		database := "test-db"

		point := &ScheduleDataPoint{
			Timestamp:   time.Now(),
			Hour:        14,
			DayOfWeek:   2, // Tuesday
			Duration:    30 * time.Minute,
			Size:        1073741824, // 1GB
			Success:     true,
			CPUUsage:    0.45,
			MemoryUsage: 0.60,
			SystemLoad:  0.35,
		}

		ps.AddDataPoint(database, point)

		if len(ps.historicalData[database]) != 1 {
			t.Errorf("Expected 1 data point, got %d", len(ps.historicalData[database]))
		}
	})

	t.Run("TrainModel_InsufficientData", func(t *testing.T) {
		ps := NewPredictiveScheduler()
		ctx := context.Background()

		_, err := ps.TrainModel(ctx, "nonexistent-db")
		if err == nil {
			t.Error("Expected error for insufficient data, got nil")
		}
	})

	t.Run("TrainModel_Success", func(t *testing.T) {
		ps := NewPredictiveScheduler()
		ctx := context.Background()
		database := "test-db"

		// Add sufficient data points (at least 10)
		for i := 0; i < 15; i++ {
			point := &ScheduleDataPoint{
				Timestamp:   time.Now().Add(-time.Duration(i) * 24 * time.Hour),
				Hour:        (14 + i) % 24,
				DayOfWeek:   i % 7,
				Duration:    30 * time.Minute,
				Size:        1073741824,
				Success:     i%5 != 0, // 80% success rate
				CPUUsage:    0.45,
				MemoryUsage: 0.60,
				SystemLoad:  0.35,
			}
			ps.AddDataPoint(database, point)
		}

		model, err := ps.TrainModel(ctx, database)
		if err != nil {
			t.Fatalf("Expected successful training, got error: %v", err)
		}

		if model.DatabaseName != database {
			t.Errorf("Expected database name %s, got %s", database, model.DatabaseName)
		}

		if len(model.OptimalHours) == 0 {
			t.Error("Expected optimal hours to be populated")
		}

		if model.Confidence < 0 || model.Confidence > 100 {
			t.Errorf("Expected confidence between 0-100, got %.2f", model.Confidence)
		}
	})

	t.Run("GetRecommendation", func(t *testing.T) {
		ps := NewPredictiveScheduler()
		ctx := context.Background()
		database := "test-db"

		// Add training data
		for i := 0; i < 15; i++ {
			point := &ScheduleDataPoint{
				Timestamp:   time.Now().Add(-time.Duration(i) * 24 * time.Hour),
				Hour:        2, // All at 2 AM
				DayOfWeek:   1, // All on Monday
				Duration:    30 * time.Minute,
				Size:        1073741824,
				Success:     true,
				CPUUsage:    0.30,
				MemoryUsage: 0.50,
				SystemLoad:  0.25,
			}
			ps.AddDataPoint(database, point)
		}

		recommendation, err := ps.GetRecommendation(ctx, database)
		if err != nil {
			t.Fatalf("Expected recommendation, got error: %v", err)
		}

		if recommendation.DatabaseName != database {
			t.Errorf("Expected database %s, got %s", database, recommendation.DatabaseName)
		}

		if recommendation.Confidence < 0 || recommendation.Confidence > 100 {
			t.Errorf("Expected confidence 0-100, got %.2f", recommendation.Confidence)
		}

		if recommendation.CronExpression == "" {
			t.Error("Expected non-empty cron expression")
		}
	})
}

// TestBackupQualityScorer tests the backup quality scoring system.
func TestBackupQualityScorer(t *testing.T) {
	t.Run("CalculateScore_PerfectBackup", func(t *testing.T) {
		bqs := NewBackupQualityScorer()
		ctx := context.Background()

		metrics := &BackupMetrics{
			DatabaseName:     "test-db",
			Size:             10737418240, // 10GB
			CompressedSize:   3221225472,  // 3GB
			Duration:         30 * time.Minute,
			Success:          true,
			ErrorCount:       0,
			ChecksumValid:    true,
			Encrypted:        true,
			CompressionRatio: 0.30,
			Age:              12 * time.Hour,
			RedundancyLevel:  3,
			LastValidated:    time.Now().Add(-24 * time.Hour),
			RecoveryTested:   true,
			IncrementalChain: 3,
		}

		score, err := bqs.CalculateScore(ctx, metrics)
		if err != nil {
			t.Fatalf("Expected successful scoring, got error: %v", err)
		}

		if score.OverallScore < 90 {
			t.Errorf("Expected score >= 90 for perfect backup, got %.2f", score.OverallScore)
		}

		if score.Grade != "A" && score.Grade != "A+" {
			t.Errorf("Expected grade A or A+, got %s", score.Grade)
		}

		if len(score.ComponentScores) != 6 {
			t.Errorf("Expected 6 component scores, got %d", len(score.ComponentScores))
		}
	})

	t.Run("CalculateScore_FailedBackup", func(t *testing.T) {
		bqs := NewBackupQualityScorer()
		ctx := context.Background()

		metrics := &BackupMetrics{
			DatabaseName:     "test-db",
			Size:             0, // Zero size
			CompressedSize:   0,
			Duration:         5 * time.Minute,
			Success:          false, // Failed
			ErrorCount:       5,
			ChecksumValid:    false,
			Encrypted:        false,
			CompressionRatio: 0.0,
			Age:              72 * time.Hour, // 3 days old
			RedundancyLevel:  0,
			LastValidated:    time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
			RecoveryTested:   false,
			IncrementalChain: 15,
		}

		score, err := bqs.CalculateScore(ctx, metrics)
		if err != nil {
			t.Fatalf("Expected successful scoring, got error: %v", err)
		}

		if score.OverallScore > 50 {
			t.Errorf("Expected score < 50 for failed backup, got %.2f", score.OverallScore)
		}

		if score.Grade == "A" || score.Grade == "A+" {
			t.Errorf("Expected poor grade, got %s", score.Grade)
		}

		if len(score.Issues) == 0 {
			t.Error("Expected issues to be identified")
		}

		if len(score.Recommendations) == 0 {
			t.Error("Expected recommendations to be provided")
		}
	})
}

// TestBackupSizeEstimator tests the size estimation ML model.
func TestBackupSizeEstimator(t *testing.T) {
	t.Run("AddSizeDataPoint", func(t *testing.T) {
		bse := NewBackupSizeEstimator()
		database := "test-db"

		point := SizeDataPoint{
			Timestamp: time.Now(),
			Size:      1073741824, // 1GB
			Type:      "full",
		}

		bse.AddSizeDataPoint(database, point)

		if len(bse.historicalSizes[database]) != 1 {
			t.Errorf("Expected 1 data point, got %d", len(bse.historicalSizes[database]))
		}
	})

	t.Run("TrainGrowthModel_Success", func(t *testing.T) {
		bse := NewBackupSizeEstimator()
		ctx := context.Background()
		database := "test-db"

		// Add growing data points
		baseSize := int64(10737418240) // 10GB
		for i := 0; i < 10; i++ {
			point := SizeDataPoint{
				Timestamp: time.Now().Add(-time.Duration(9-i) * 24 * time.Hour),
				Size:      baseSize + int64(i)*107374182, // Growing by ~100MB/day
				Type:      "full",
			}
			bse.AddSizeDataPoint(database, point)
		}

		model, err := bse.TrainGrowthModel(ctx, database)
		if err != nil {
			t.Fatalf("Expected successful training, got error: %v", err)
		}

		if model.GrowthRatePerDay <= 0 {
			t.Errorf("Expected positive growth rate, got %.2f", model.GrowthRatePerDay)
		}

		if model.R2 < 0 || model.R2 > 1 {
			t.Errorf("Expected R² between 0-1, got %.3f", model.R2)
		}
	})

	t.Run("EstimateSize", func(t *testing.T) {
		bse := NewBackupSizeEstimator()
		ctx := context.Background()
		database := "test-db"

		// Add training data
		baseSize := int64(10737418240)
		for i := 0; i < 10; i++ {
			point := SizeDataPoint{
				Timestamp: time.Now().Add(-time.Duration(9-i) * 24 * time.Hour),
				Size:      baseSize + int64(i)*107374182,
				Type:      "full",
			}
			bse.AddSizeDataPoint(database, point)
		}

		// Estimate size 7 days in future
		targetDate := time.Now().Add(7 * 24 * time.Hour)
		estimation, err := bse.EstimateSize(ctx, database, targetDate)
		if err != nil {
			t.Fatalf("Expected size estimation, got error: %v", err)
		}

		if estimation.EstimatedSize <= 0 {
			t.Errorf("Expected positive estimated size, got %d", estimation.EstimatedSize)
		}

		if estimation.Confidence < 0 || estimation.Confidence > 100 {
			t.Errorf("Expected confidence 0-100, got %.2f", estimation.Confidence)
		}

		if estimation.EstimatedSize < estimation.LowerBound || estimation.EstimatedSize > estimation.UpperBound {
			t.Error("Estimated size should be within confidence bounds")
		}
	})
}

// TestAutoCategorizer tests the auto-categorization system.
func TestAutoCategorizer(t *testing.T) {
	t.Run("Categorize_Production", func(t *testing.T) {
		ac := NewAutoCategorizer()
		ctx := context.Background()

		result, err := ac.Categorize(ctx, "prod-mysql-backup", "production-db", "Main production database", nil)
		if err != nil {
			t.Fatalf("Expected successful categorization, got error: %v", err)
		}

		if result.PrimaryCategory != "Production" {
			t.Errorf("Expected Production category, got %s", result.PrimaryCategory)
		}

		if result.Confidence < 50 {
			t.Errorf("Expected reasonable confidence, got %.2f", result.Confidence)
		}

		if len(result.SuggestedTags) == 0 {
			t.Error("Expected suggested tags")
		}
	})

	t.Run("Categorize_Compliance", func(t *testing.T) {
		ac := NewAutoCategorizer()
		ctx := context.Background()

		result, err := ac.Categorize(ctx, "gdpr-backup", "compliance-db", "GDPR compliance audit database", nil)
		if err != nil {
			t.Fatalf("Expected successful categorization, got error: %v", err)
		}

		if result.PrimaryCategory != "Compliance" {
			t.Errorf("Expected Compliance category, got %s", result.PrimaryCategory)
		}
	})
}

// TestDuplicateDetector tests the duplicate detection system.
func TestDuplicateDetector(t *testing.T) {
	t.Run("IndexBackup", func(t *testing.T) {
		dd := NewDuplicateDetector()

		content := []byte("test backup content")
		err := dd.IndexBackup("backup-1", "db-1", int64(len(content)), content, nil)
		if err != nil {
			t.Fatalf("Expected successful indexing, got error: %v", err)
		}

		if len(dd.backupHashes) != 1 {
			t.Errorf("Expected 1 indexed backup, got %d", len(dd.backupHashes))
		}
	})

	t.Run("DetectDuplicates_ExactMatch", func(t *testing.T) {
		dd := NewDuplicateDetector()
		ctx := context.Background()

		content := []byte("test backup content")

		// Index first backup
		dd.IndexBackup("backup-1", "db-1", int64(len(content)), content, nil)

		// Detect duplicates for same content
		report, err := dd.DetectDuplicates(ctx, "backup-2", "db-2", content)
		if err != nil {
			t.Fatalf("Expected successful detection, got error: %v", err)
		}

		if report.TotalDuplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", report.TotalDuplicates)
		}

		if len(report.Duplicates) > 0 && report.Duplicates[0].MatchType != "exact" {
			t.Errorf("Expected exact match, got %s", report.Duplicates[0].MatchType)
		}
	})
}

// TestNLPSearchEngine tests the NLP search engine.
func TestNLPSearchEngine(t *testing.T) {
	t.Run("IndexBackup", func(t *testing.T) {
		nse := NewNLPSearchEngine()

		backup := &SearchableBackup{
			BackupID:     "backup-1",
			DatabaseName: "production-mysql",
			Description:  "Daily production database backup",
			Tags:         []string{"mysql", "production", "daily"},
			Timestamp:    time.Now(),
			Size:         1073741824,
		}

		err := nse.IndexBackup(backup)
		if err != nil {
			t.Fatalf("Expected successful indexing, got error: %v", err)
		}

		if len(nse.indexedBackups) != 1 {
			t.Errorf("Expected 1 indexed backup, got %d", len(nse.indexedBackups))
		}
	})

	t.Run("Search", func(t *testing.T) {
		nse := NewNLPSearchEngine()
		ctx := context.Background()

		// Index multiple backups
		backups := []*SearchableBackup{
			{
				BackupID:     "backup-1",
				DatabaseName: "production-mysql",
				Description:  "Daily production MySQL database backup",
				Tags:         []string{"mysql", "production", "daily"},
				Timestamp:    time.Now(),
				Size:         1073741824,
			},
			{
				BackupID:     "backup-2",
				DatabaseName: "staging-postgres",
				Description:  "Weekly staging PostgreSQL backup",
				Tags:         []string{"postgres", "staging", "weekly"},
				Timestamp:    time.Now().Add(-7 * 24 * time.Hour),
				Size:         536870912,
			},
		}

		for _, backup := range backups {
			nse.IndexBackup(backup)
		}

		// Search for "production mysql"
		response, err := nse.Search(ctx, "production mysql", nil)
		if err != nil {
			t.Fatalf("Expected successful search, got error: %v", err)
		}

		if response.TotalResults == 0 {
			t.Error("Expected search results")
		}

		if len(response.Results) > 0 && response.Results[0].BackupID != "backup-1" {
			t.Errorf("Expected backup-1 as top result, got %s", response.Results[0].BackupID)
		}
	})
}

// TestSmartRetentionPolicyGenerator tests retention policy generation.
func TestSmartRetentionPolicyGenerator(t *testing.T) {
	t.Run("GeneratePolicy", func(t *testing.T) {
		srpg := NewSmartRetentionPolicyGenerator()
		ctx := context.Background()

		recommendation, err := srpg.GeneratePolicy(
			ctx,
			"production-db",
			10737418240, // 10GB
			"daily",
			[]string{"GDPR", "HIPAA"},
			500.0, // $500 budget
		)
		if err != nil {
			t.Fatalf("Expected successful policy generation, got error: %v", err)
		}

		policy := recommendation.RecommendedPolicy
		if policy.DatabaseName != "production-db" {
			t.Errorf("Expected database production-db, got %s", policy.DatabaseName)
		}

		if policy.YearlyRetention != 6 {
			t.Errorf("Expected 6 years retention for GDPR/HIPAA, got %d", policy.YearlyRetention)
		}

		if len(recommendation.Alternatives) == 0 {
			t.Error("Expected alternative policies")
		}

		if recommendation.CostAnalysis == nil {
			t.Error("Expected cost analysis")
		}

		if recommendation.RiskAssessment == nil {
			t.Error("Expected risk assessment")
		}
	})
}

// TestAIAdvisor tests the AI advisor conversational mode.
func TestAIAdvisor(t *testing.T) {
	t.Run("StartConversation", func(t *testing.T) {
		aa := NewAIAdvisor()

		err := aa.StartConversation("session-1", "user-1")
		if err != nil {
			t.Fatalf("Expected successful session start, got error: %v", err)
		}

		if len(aa.conversationHistory["session-1"]) != 1 {
			t.Errorf("Expected 1 welcome message, got %d", len(aa.conversationHistory["session-1"]))
		}
	})

	t.Run("Ask_Optimization", func(t *testing.T) {
		aa := NewAIAdvisor()
		ctx := context.Background()

		aa.StartConversation("session-1", "user-1")

		response, err := aa.Ask(ctx, "session-1", "How can I optimize my backups?")
		if err != nil {
			t.Fatalf("Expected successful response, got error: %v", err)
		}

		if response.Message == "" {
			t.Error("Expected non-empty response message")
		}

		if len(response.Recommendations) == 0 {
			t.Error("Expected recommendations")
		}

		if len(response.NextSteps) == 0 {
			t.Error("Expected next steps")
		}
	})

	t.Run("Ask_CostAnalysis", func(t *testing.T) {
		aa := NewAIAdvisor()
		ctx := context.Background()

		aa.StartConversation("session-1", "user-1")

		_, err := aa.Ask(ctx, "session-1", "How can I reduce costs?")
		if err != nil {
			t.Fatalf("Expected successful response, got error: %v", err)
		}

		intent := aa.detectIntent("How can I reduce costs?")
		if intent != "cost_analysis" {
			t.Errorf("Expected cost_analysis intent, got %s", intent)
		}
	})
}

// TestAutomatedTroubleshootingEngine tests the troubleshooting engine.
func TestAutomatedTroubleshootingEngine(t *testing.T) {
	t.Run("Diagnose_DiskSpace", func(t *testing.T) {
		ate := NewAutomatedTroubleshootingEngine()
		ctx := context.Background()

		result, err := ate.Diagnose(ctx, "test-db", "Error: no space left on device", map[string]interface{}{
			"disk_usage_percent": 95.5,
		})
		if err != nil {
			t.Fatalf("Expected successful diagnosis, got error: %v", err)
		}

		if result.Category != "disk_space" {
			t.Errorf("Expected disk_space category, got %s", result.Category)
		}

		if result.Severity != "critical" {
			t.Errorf("Expected critical severity, got %s", result.Severity)
		}

		if !result.AutoFixApplied {
			t.Error("Expected auto-fix to be applied")
		}

		if len(result.Resolution) == 0 {
			t.Error("Expected resolution steps")
		}
	})

	t.Run("Diagnose_Checksum", func(t *testing.T) {
		ate := NewAutomatedTroubleshootingEngine()
		ctx := context.Background()

		result, err := ate.Diagnose(ctx, "test-db", "Checksum validation failed: corrupted data", nil)
		if err != nil {
			t.Fatalf("Expected successful diagnosis, got error: %v", err)
		}

		if result.Category != "data_integrity" {
			t.Errorf("Expected data_integrity category, got %s", result.Category)
		}

		if result.Confidence < 50 {
			t.Errorf("Expected reasonable confidence, got %.2f", result.Confidence)
		}
	})

	t.Run("GenerateReport", func(t *testing.T) {
		ate := NewAutomatedTroubleshootingEngine()
		ctx := context.Background()

		// Diagnose some issues
		ate.Diagnose(ctx, "test-db", "no space left on device", nil)
		ate.Diagnose(ctx, "test-db", "connection timeout", nil)

		report, err := ate.GenerateReport(ctx, "test-db")
		if err != nil {
			t.Fatalf("Expected successful report generation, got error: %v", err)
		}

		if report.TotalIssues != 2 {
			t.Errorf("Expected 2 issues, got %d", report.TotalIssues)
		}

		if report.CriticalCount == 0 {
			t.Error("Expected some critical issues")
		}

		if len(report.Recommendations) == 0 {
			t.Error("Expected recommendations")
		}
	})
}

// TestSmartFeaturesManager tests the integrated features manager.
func TestSmartFeaturesManager(t *testing.T) {
	t.Run("NewSmartFeaturesManager", func(t *testing.T) {
		sfm := NewSmartFeaturesManager()

		if sfm.predictiveScheduler == nil {
			t.Error("Expected predictive scheduler to be initialized")
		}

		if sfm.qualityScorer == nil {
			t.Error("Expected quality scorer to be initialized")
		}

		if sfm.sizeEstimator == nil {
			t.Error("Expected size estimator to be initialized")
		}

		if sfm.autoCategorizer == nil {
			t.Error("Expected auto categorizer to be initialized")
		}

		if sfm.duplicateDetector == nil {
			t.Error("Expected duplicate detector to be initialized")
		}

		if sfm.nlpSearchEngine == nil {
			t.Error("Expected NLP search engine to be initialized")
		}

		if sfm.retentionPolicyGen == nil {
			t.Error("Expected retention policy generator to be initialized")
		}

		if sfm.aiAdvisor == nil {
			t.Error("Expected AI advisor to be initialized")
		}

		if sfm.troubleshootingEngine == nil {
			t.Error("Expected troubleshooting engine to be initialized")
		}

		if sfm.failurePredictorAPI == nil {
			t.Error("Expected failure predictor API to be initialized")
		}
	})

	t.Run("GetPredictiveScheduler", func(t *testing.T) {
		sfm := NewSmartFeaturesManager()
		ps := sfm.GetPredictiveScheduler()

		if ps == nil {
			t.Error("Expected non-nil predictive scheduler")
		}
	})

	t.Run("GetQualityScorer", func(t *testing.T) {
		sfm := NewSmartFeaturesManager()
		qs := sfm.GetQualityScorer()

		if qs == nil {
			t.Error("Expected non-nil quality scorer")
		}
	})
}

// Benchmark tests.
func BenchmarkPredictiveScheduler_TrainModel(b *testing.B) {
	ps := NewPredictiveScheduler()
	ctx := context.Background()
	database := "bench-db"

	// Setup test data
	for i := 0; i < 100; i++ {
		point := &ScheduleDataPoint{
			Timestamp:   time.Now().Add(-time.Duration(i) * 24 * time.Hour),
			Hour:        i % 24,
			DayOfWeek:   i % 7,
			Duration:    30 * time.Minute,
			Size:        1073741824,
			Success:     true,
			CPUUsage:    0.45,
			MemoryUsage: 0.60,
			SystemLoad:  0.35,
		}
		ps.AddDataPoint(database, point)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps.TrainModel(ctx, database)
	}
}

func BenchmarkBackupQualityScorer_CalculateScore(b *testing.B) {
	bqs := NewBackupQualityScorer()
	ctx := context.Background()

	metrics := &BackupMetrics{
		DatabaseName:     "bench-db",
		Size:             10737418240,
		CompressedSize:   3221225472,
		Duration:         30 * time.Minute,
		Success:          true,
		ErrorCount:       0,
		ChecksumValid:    true,
		Encrypted:        true,
		CompressionRatio: 0.30,
		Age:              12 * time.Hour,
		RedundancyLevel:  3,
		LastValidated:    time.Now().Add(-24 * time.Hour),
		RecoveryTested:   true,
		IncrementalChain: 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bqs.CalculateScore(ctx, metrics)
	}
}

func BenchmarkNLPSearchEngine_Search(b *testing.B) {
	nse := NewNLPSearchEngine()
	ctx := context.Background()

	// Index 1000 backups
	for i := 0; i < 1000; i++ {
		backup := &SearchableBackup{
			BackupID:     fmt.Sprintf("backup-%d", i),
			DatabaseName: fmt.Sprintf("database-%d", i%10),
			Description:  "Daily production backup",
			Tags:         []string{"production", "daily"},
			Timestamp:    time.Now(),
			Size:         1073741824,
		}
		nse.IndexBackup(backup)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nse.Search(ctx, "production backup", nil)
	}
}
