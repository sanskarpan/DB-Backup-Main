package optimization

import (
	"context"
	"testing"
	"time"
)

// Mock analyzer for testing
type mockAnalyzer struct {
	planToReturn *QueryPlan
	costToReturn float64
	statsToReturn *TableStatistics
	errorToReturn error
}

func (ma *mockAnalyzer) AnalyzeQueryPlan(ctx context.Context, query string) (*QueryPlan, error) {
	if ma.errorToReturn != nil {
		return nil, ma.errorToReturn
	}
	if ma.planToReturn != nil {
		return ma.planToReturn, nil
	}
	return &QueryPlan{
		Query:         query,
		DatabaseType:  DatabaseTypePostgreSQL,
		EstimatedCost: 100.0,
		EstimatedRows: 1000,
		PlanNodes:     []*PlanNode{},
		UsedIndexes:   []string{},
		Warnings:      []string{},
		Recommendations: []string{},
		AnalyzedAt:    time.Now(),
	}, nil
}

func (ma *mockAnalyzer) GetDatabaseType() DatabaseType {
	return DatabaseTypePostgreSQL
}

func (ma *mockAnalyzer) EstimateCost(ctx context.Context, query string) (float64, error) {
	if ma.errorToReturn != nil {
		return 0, ma.errorToReturn
	}
	return ma.costToReturn, nil
}

func (ma *mockAnalyzer) GetTableStatistics(ctx context.Context, table string) (*TableStatistics, error) {
	if ma.errorToReturn != nil {
		return nil, ma.errorToReturn
	}
	if ma.statsToReturn != nil {
		return ma.statsToReturn, nil
	}
	return &TableStatistics{
		TableName: table,
		RowCount:  10000,
		TableSize: 1024 * 1024,
	}, nil
}

func TestNewQueryOptimizer(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := DefaultOptimizerConfig()

	optimizer := NewQueryOptimizer(config, analyzer)
	if optimizer == nil {
		t.Fatal("Expected optimizer to be created")
	}

	if optimizer.analyzer == nil {
		t.Error("Expected analyzer to be set")
	}

	if optimizer.config == nil {
		t.Error("Expected config to be set")
	}

	if optimizer.slowQueryDetector == nil {
		t.Error("Expected slow query detector to be initialized")
	}

	if optimizer.statsCollector == nil {
		t.Error("Expected stats collector to be initialized")
	}

	t.Log("✓ QueryOptimizer created successfully")
}

func TestDefaultOptimizerConfig(t *testing.T) {
	config := DefaultOptimizerConfig()

	if config == nil {
		t.Fatal("Expected config to be created")
	}

	if config.SlowQueryThreshold != 1*time.Second {
		t.Errorf("Expected slow query threshold to be 1s, got %v", config.SlowQueryThreshold)
	}

	if !config.EnableIndexRecommendations {
		t.Error("Expected index recommendations to be enabled")
	}

	if !config.EnableQueryRewriting {
		t.Error("Expected query rewriting to be enabled")
	}

	if !config.EnableStatsCollection {
		t.Error("Expected stats collection to be enabled")
	}

	if config.IndexConfidenceThreshold != 0.7 {
		t.Errorf("Expected index confidence threshold to be 0.7, got %f", config.IndexConfidenceThreshold)
	}

	t.Log("✓ Default config has correct values")
}

func TestAnalyzeQuery(t *testing.T) {
	analyzer := &mockAnalyzer{
		planToReturn: &QueryPlan{
			Query:         "SELECT * FROM users WHERE id = 1",
			DatabaseType:  DatabaseTypePostgreSQL,
			EstimatedCost: 150.0,
			EstimatedRows: 1,
			UsedIndexes:   []string{"users_pkey"},
		},
	}

	config := DefaultOptimizerConfig()
	optimizer := NewQueryOptimizer(config, analyzer)

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = 1"

	analysis, err := optimizer.AnalyzeQuery(ctx, query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis to be returned")
	}

	if analysis.Query != query {
		t.Errorf("Expected query '%s', got '%s'", query, analysis.Query)
	}

	if analysis.Plan == nil {
		t.Error("Expected plan to be set")
	}

	if analysis.EstimatedCost != 150.0 {
		t.Errorf("Expected estimated cost 150.0, got %f", analysis.EstimatedCost)
	}

	if analysis.EstimatedRows != 1 {
		t.Errorf("Expected estimated rows 1, got %d", analysis.EstimatedRows)
	}

	t.Log("✓ Query analysis works correctly")
}

func TestOptimizerDetectSlowQuery(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := &OptimizerConfig{
		SlowQueryThreshold: 100 * time.Millisecond,
	}

	optimizer := NewQueryOptimizer(config, analyzer)

	// Test fast query
	fastQuery := "SELECT * FROM users LIMIT 10"
	slowQuery1 := optimizer.DetectSlowQuery(fastQuery, 50*time.Millisecond)
	if slowQuery1 != nil {
		t.Error("Expected fast query not to be detected as slow")
	}

	// Test slow query
	slowQueryStr := "SELECT * FROM large_table"
	slowQuery2 := optimizer.DetectSlowQuery(slowQueryStr, 500*time.Millisecond)
	if slowQuery2 == nil {
		t.Error("Expected slow query to be detected")
	}

	if slowQuery2 != nil {
		if slowQuery2.Query != slowQueryStr {
			t.Errorf("Expected query '%s', got '%s'", slowQueryStr, slowQuery2.Query)
		}

		if slowQuery2.ExecutionTime != 500*time.Millisecond {
			t.Errorf("Expected execution time 500ms, got %v", slowQuery2.ExecutionTime)
		}

		if slowQuery2.Severity == "" {
			t.Error("Expected severity to be set")
		}
	}

	t.Log("✓ Slow query detection works correctly")
}

func TestGetStatistics(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := DefaultOptimizerConfig()
	optimizer := NewQueryOptimizer(config, analyzer)

	// Record some queries
	optimizer.statsCollector.RecordQuery("SELECT * FROM users", 100*time.Millisecond)
	optimizer.statsCollector.RecordQuery("SELECT * FROM users", 150*time.Millisecond)
	optimizer.statsCollector.RecordQuery("SELECT * FROM posts", 200*time.Millisecond)

	stats := optimizer.GetStatistics()

	if len(stats) != 2 {
		t.Errorf("Expected 2 unique queries, got %d", len(stats))
	}

	t.Log("✓ Statistics retrieval works correctly")
}

func TestOptimizerGetSlowQueries(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := &OptimizerConfig{
		SlowQueryThreshold: 100 * time.Millisecond,
	}
	optimizer := NewQueryOptimizer(config, analyzer)

	// Record some slow queries
	optimizer.DetectSlowQuery("SELECT * FROM table1", 500*time.Millisecond)
	optimizer.DetectSlowQuery("SELECT * FROM table2", 600*time.Millisecond)
	optimizer.DetectSlowQuery("SELECT * FROM table3", 700*time.Millisecond)

	slowQueries := optimizer.GetSlowQueries(2)

	if len(slowQueries) != 2 {
		t.Errorf("Expected 2 slow queries, got %d", len(slowQueries))
	}

	t.Log("✓ Slow queries retrieval works correctly")
}

func TestOptimizerClearStatistics(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := DefaultOptimizerConfig()
	optimizer := NewQueryOptimizer(config, analyzer)

	// Add some data
	optimizer.statsCollector.RecordQuery("SELECT * FROM users", 100*time.Millisecond)
	optimizer.DetectSlowQuery("SELECT * FROM large_table", 2*time.Second)

	// Clear
	optimizer.ClearStatistics()

	stats := optimizer.GetStatistics()
	slowQueries := optimizer.GetSlowQueries(10)

	if len(stats) != 0 {
		t.Errorf("Expected statistics to be cleared, got %d", len(stats))
	}

	if len(slowQueries) != 0 {
		t.Errorf("Expected slow queries to be cleared, got %d", len(slowQueries))
	}

	t.Log("✓ Statistics clearing works correctly")
}

func TestSetIndexRecommender(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := DefaultOptimizerConfig()
	optimizer := NewQueryOptimizer(config, analyzer)

	// Set a mock recommender
	recommender := &mockIndexRecommender{}
	optimizer.SetIndexRecommender(recommender)

	if optimizer.indexRecommender == nil {
		t.Error("Expected index recommender to be set")
	}

	t.Log("✓ Index recommender setter works correctly")
}

func TestSetQueryRewriter(t *testing.T) {
	analyzer := &mockAnalyzer{}
	config := DefaultOptimizerConfig()
	optimizer := NewQueryOptimizer(config, analyzer)

	// Set a mock rewriter
	rewriter := &mockQueryRewriter{}
	optimizer.SetQueryRewriter(rewriter)

	if optimizer.rewriter == nil {
		t.Error("Expected query rewriter to be set")
	}

	t.Log("✓ Query rewriter setter works correctly")
}

// Mock implementations for testing

type mockIndexRecommender struct {
	recommendations []*IndexRecommendation
	usages          []*IndexUsage
	validation      *IndexValidation
	errorToReturn   error
}

func (mir *mockIndexRecommender) RecommendIndexes(ctx context.Context, queries []string) ([]*IndexRecommendation, error) {
	if mir.errorToReturn != nil {
		return nil, mir.errorToReturn
	}
	return mir.recommendations, nil
}

func (mir *mockIndexRecommender) AnalyzeIndexUsage(ctx context.Context) ([]*IndexUsage, error) {
	if mir.errorToReturn != nil {
		return nil, mir.errorToReturn
	}
	return mir.usages, nil
}

func (mir *mockIndexRecommender) ValidateIndex(ctx context.Context, table string, columns []string) (*IndexValidation, error) {
	if mir.errorToReturn != nil {
		return nil, mir.errorToReturn
	}
	return mir.validation, nil
}

type mockQueryRewriter struct {
	rewritten     *RewrittenQuery
	rules         []RewriteRule
	errorToReturn error
}

func (mqr *mockQueryRewriter) RewriteQuery(ctx context.Context, query string) (*RewrittenQuery, error) {
	if mqr.errorToReturn != nil {
		return nil, mqr.errorToReturn
	}
	if mqr.rewritten != nil {
		return mqr.rewritten, nil
	}
	return &RewrittenQuery{
		OriginalQuery:  query,
		RewrittenQuery: query,
		AppliedRules:   []string{},
		EstimatedGain:  0,
		RewrittenAt:    time.Now(),
	}, nil
}

func (mqr *mockQueryRewriter) GetRewriteRules() []RewriteRule {
	return mqr.rules
}

func (mqr *mockQueryRewriter) ApplyRule(ctx context.Context, query string, rule RewriteRule) (string, error) {
	if mqr.errorToReturn != nil {
		return "", mqr.errorToReturn
	}
	return query, nil
}

func TestTruncateQuery(t *testing.T) {
	longQuery := "SELECT * FROM users WHERE id = 1 AND name = 'test' AND email = 'test@example.com' AND age > 18 AND status = 'active'"

	truncated := truncateQuery(longQuery, 50)

	if len(truncated) > 53 { // 50 + "..."
		t.Errorf("Expected truncated query to be <= 53 chars, got %d", len(truncated))
	}

	shortQuery := "SELECT * FROM users"
	notTruncated := truncateQuery(shortQuery, 50)

	if notTruncated != shortQuery {
		t.Error("Expected short query not to be truncated")
	}

	t.Log("✓ Query truncation works correctly")
}
