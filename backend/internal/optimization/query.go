package optimization

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// QueryOptimizer provides query optimization capabilities.
type QueryOptimizer struct {
	analyzer          QueryAnalyzer
	indexRecommender  IndexRecommender
	slowQueryDetector *SlowQueryDetector
	rewriter          QueryRewriter
	statsCollector    *StatsCollector
	config            *OptimizerConfig
	mu                sync.RWMutex
}

// OptimizerConfig holds optimizer configuration.
type OptimizerConfig struct {
	// Slow query threshold
	SlowQueryThreshold time.Duration

	// Enable automatic index recommendations
	EnableIndexRecommendations bool

	// Enable query rewriting
	EnableQueryRewriting bool

	// Enable statistics collection
	EnableStatsCollection bool

	// Maximum query plan cache size
	MaxPlanCacheSize int

	// Statistics retention period
	StatsRetentionPeriod time.Duration

	// Index recommendation confidence threshold (0.0-1.0)
	IndexConfidenceThreshold float64

	// Enable query plan analysis
	EnablePlanAnalysis bool

	// Query timeout for analysis
	AnalysisTimeout time.Duration
}

// QueryAnalyzer analyzes query execution plans.
type QueryAnalyzer interface {
	// AnalyzeQueryPlan analyzes the execution plan for a query
	AnalyzeQueryPlan(ctx context.Context, query string) (*QueryPlan, error)

	// GetDatabaseType returns the database type
	GetDatabaseType() DatabaseType

	// EstimateCost estimates the cost of executing a query
	EstimateCost(ctx context.Context, query string) (float64, error)

	// GetTableStatistics gets statistics for a table
	GetTableStatistics(ctx context.Context, table string) (*TableStatistics, error)
}

// IndexRecommender recommends indexes for query optimization.
type IndexRecommender interface {
	// RecommendIndexes recommends indexes based on query patterns
	RecommendIndexes(ctx context.Context, queries []string) ([]*IndexRecommendation, error)

	// AnalyzeIndexUsage analyzes current index usage
	AnalyzeIndexUsage(ctx context.Context) ([]*IndexUsage, error)

	// ValidateIndex checks if an index would be beneficial
	ValidateIndex(ctx context.Context, table string, columns []string) (*IndexValidation, error)
}

// QueryRewriter rewrites queries for better performance.
type QueryRewriter interface {
	// RewriteQuery rewrites a query for optimization
	RewriteQuery(ctx context.Context, query string) (*RewrittenQuery, error)

	// GetRewriteRules returns available rewrite rules
	GetRewriteRules() []RewriteRule

	// ApplyRule applies a specific rewrite rule
	ApplyRule(ctx context.Context, query string, rule RewriteRule) (string, error)
}

// DatabaseType represents the type of database.
type DatabaseType string

const (
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypeMongoDB    DatabaseType = "mongodb"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
)

// QueryPlan represents a query execution plan.
type QueryPlan struct {
	Query           string
	DatabaseType    DatabaseType
	EstimatedCost   float64
	EstimatedRows   int64
	ActualCost      float64
	ActualRows      int64
	ExecutionTime   time.Duration
	PlanNodes       []*PlanNode
	UsedIndexes     []string
	MissingIndexes  []string
	Warnings        []string
	Recommendations []string
	AnalyzedAt      time.Time
}

// PlanNode represents a node in the query execution plan.
type PlanNode struct {
	NodeType  string
	Operation string
	Table     string
	Index     string
	Rows      int64
	Cost      float64
	Width     int
	Filters   []string
	JoinType  string
	Children  []*PlanNode
	ExtraInfo map[string]interface{}
}

// IndexRecommendation represents a recommended index.
type IndexRecommendation struct {
	Table           string
	Columns         []string
	IndexType       IndexType
	Reason          string
	EstimatedGain   float64
	Confidence      float64
	AffectedQueries []string
	Priority        Priority
	CreatedAt       time.Time
}

// IndexType represents the type of index.
type IndexType string

const (
	IndexTypeBTree    IndexType = "btree"
	IndexTypeHash     IndexType = "hash"
	IndexTypeGIN      IndexType = "gin"
	IndexTypeGiST     IndexType = "gist"
	IndexTypeBRIN     IndexType = "brin"
	IndexTypeFullText IndexType = "fulltext"
)

// Priority represents recommendation priority.
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// IndexUsage represents index usage statistics.
type IndexUsage struct {
	SchemaName     string
	TableName      string
	IndexName      string
	IndexType      string
	Columns        []string
	ScanCount      int64
	TupleRead      int64
	TupleFetch     int64
	IndexSize      int64
	UsagePercent   float64
	LastUsed       *time.Time
	Recommendation string
}

// IndexValidation represents index validation result.
type IndexValidation struct {
	Table        string
	Columns      []string
	IsValid      bool
	IsBeneficial bool
	ExpectedGain float64
	Warnings     []string
	Alternatives []*IndexRecommendation
}

// RewrittenQuery represents a rewritten query.
type RewrittenQuery struct {
	OriginalQuery  string
	RewrittenQuery string
	AppliedRules   []string
	EstimatedGain  float64
	Explanation    string
	RewrittenAt    time.Time
}

// RewriteRule represents a query rewrite rule.
type RewriteRule struct {
	Name        string
	Description string
	Pattern     string
	Replacement string
	Conditions  []string
	Priority    int
	Enabled     bool
}

// TableStatistics represents table statistics.
type TableStatistics struct {
	SchemaName    string
	TableName     string
	RowCount      int64
	TableSize     int64
	IndexSize     int64
	DeadTuples    int64
	LastVacuum    *time.Time
	LastAnalyze   *time.Time
	EstimatedRows int64
	AvgRowSize    int
	Bloat         float64
}

// QueryStatistics represents query execution statistics.
type QueryStatistics struct {
	Query          string
	QueryHash      string
	DatabaseType   DatabaseType
	ExecutionCount int64
	TotalTime      time.Duration
	MinTime        time.Duration
	MaxTime        time.Duration
	MeanTime       time.Duration
	StdDevTime     time.Duration
	RowsReturned   int64
	RowsAffected   int64
	CacheHitRatio  float64
	IndexHitRatio  float64
	LastExecuted   time.Time
	FirstExecuted  time.Time
}

// SlowQuery represents a slow query entry.
type SlowQuery struct {
	Query           string
	QueryHash       string
	DatabaseType    DatabaseType
	ExecutionTime   time.Duration
	Timestamp       time.Time
	User            string
	Database        string
	Plan            *QueryPlan
	Recommendations []string
	Severity        Severity
}

// Severity represents query issue severity.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// NewQueryOptimizer creates a new query optimizer.
func NewQueryOptimizer(config *OptimizerConfig, analyzer QueryAnalyzer) *QueryOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	optimizer := &QueryOptimizer{
		analyzer:          analyzer,
		config:            config,
		slowQueryDetector: NewSlowQueryDetector(config.SlowQueryThreshold),
		statsCollector:    NewStatsCollector(config.StatsRetentionPeriod),
	}

	return optimizer
}

// DefaultOptimizerConfig returns default optimizer configuration.
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		SlowQueryThreshold:         1 * time.Second,
		EnableIndexRecommendations: true,
		EnableQueryRewriting:       true,
		EnableStatsCollection:      true,
		MaxPlanCacheSize:           1000,
		StatsRetentionPeriod:       24 * time.Hour,
		IndexConfidenceThreshold:   0.7,
		EnablePlanAnalysis:         true,
		AnalysisTimeout:            10 * time.Second,
	}
}

// AnalyzeQuery analyzes a query and returns optimization recommendations.
func (qo *QueryOptimizer) AnalyzeQuery(ctx context.Context, query string) (*QueryAnalysis, error) {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	start := time.Now()
	analysis := &QueryAnalysis{
		Query:      query,
		AnalyzedAt: start,
	}

	// Analyze query plan
	if qo.config.EnablePlanAnalysis && qo.analyzer != nil {
		planCtx, cancel := context.WithTimeout(ctx, qo.config.AnalysisTimeout)
		defer cancel()

		plan, err := qo.analyzer.AnalyzeQueryPlan(planCtx, query)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to analyze query plan")
			analysis.Warnings = append(analysis.Warnings, fmt.Sprintf("Plan analysis failed: %v", err))
		} else {
			analysis.Plan = plan
			analysis.EstimatedCost = plan.EstimatedCost
			analysis.EstimatedRows = plan.EstimatedRows
		}
	}

	// Collect statistics
	if qo.config.EnableStatsCollection && qo.statsCollector != nil {
		qo.statsCollector.RecordQuery(query, time.Since(start))
	}

	analysis.AnalysisDuration = time.Since(start)

	log.Debug().
		Str("query", truncateQuery(query, 100)).
		Dur("duration", analysis.AnalysisDuration).
		Msg("Query analyzed")

	return analysis, nil
}

// RecommendIndexes recommends indexes for a set of queries.
func (qo *QueryOptimizer) RecommendIndexes(ctx context.Context, queries []string) ([]*IndexRecommendation, error) {
	if !qo.config.EnableIndexRecommendations || qo.indexRecommender == nil {
		return nil, fmt.Errorf("index recommendations are disabled")
	}

	qo.mu.RLock()
	defer qo.mu.RUnlock()

	recommendations, err := qo.indexRecommender.RecommendIndexes(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("failed to recommend indexes: %w", err)
	}

	// Filter by confidence threshold
	filtered := make([]*IndexRecommendation, 0)
	for _, rec := range recommendations {
		if rec.Confidence >= qo.config.IndexConfidenceThreshold {
			filtered = append(filtered, rec)
		}
	}

	log.Info().
		Int("total_recommendations", len(recommendations)).
		Int("filtered_recommendations", len(filtered)).
		Float64("confidence_threshold", qo.config.IndexConfidenceThreshold).
		Msg("Index recommendations generated")

	return filtered, nil
}

// DetectSlowQuery detects and records slow queries.
func (qo *QueryOptimizer) DetectSlowQuery(query string, executionTime time.Duration) *SlowQuery {
	if qo.slowQueryDetector == nil {
		return nil
	}

	slowQuery := qo.slowQueryDetector.Detect(query, executionTime)
	if slowQuery != nil {
		log.Warn().
			Str("query", truncateQuery(query, 100)).
			Dur("execution_time", executionTime).
			Str("severity", string(slowQuery.Severity)).
			Msg("Slow query detected")
	}

	return slowQuery
}

// RewriteQuery rewrites a query for optimization.
func (qo *QueryOptimizer) RewriteQuery(ctx context.Context, query string) (*RewrittenQuery, error) {
	if !qo.config.EnableQueryRewriting || qo.rewriter == nil {
		return nil, fmt.Errorf("query rewriting is disabled")
	}

	qo.mu.RLock()
	defer qo.mu.RUnlock()

	rewritten, err := qo.rewriter.RewriteQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite query: %w", err)
	}

	if rewritten.OriginalQuery != rewritten.RewrittenQuery {
		log.Info().
			Str("original", truncateQuery(rewritten.OriginalQuery, 100)).
			Str("rewritten", truncateQuery(rewritten.RewrittenQuery, 100)).
			Strs("rules", rewritten.AppliedRules).
			Float64("gain", rewritten.EstimatedGain).
			Msg("Query rewritten")
	}

	return rewritten, nil
}

// GetStatistics returns query statistics.
func (qo *QueryOptimizer) GetStatistics() map[string]*QueryStatistics {
	if qo.statsCollector == nil {
		return make(map[string]*QueryStatistics)
	}

	return qo.statsCollector.GetStatistics()
}

// GetSlowQueries returns detected slow queries.
func (qo *QueryOptimizer) GetSlowQueries(limit int) []*SlowQuery {
	if qo.slowQueryDetector == nil {
		return []*SlowQuery{}
	}

	return qo.slowQueryDetector.GetSlowQueries(limit)
}

// ClearStatistics clears collected statistics.
func (qo *QueryOptimizer) ClearStatistics() {
	if qo.statsCollector != nil {
		qo.statsCollector.Clear()
	}
	if qo.slowQueryDetector != nil {
		qo.slowQueryDetector.Clear()
	}

	log.Info().Msg("Optimizer statistics cleared")
}

// SetIndexRecommender sets the index recommender.
func (qo *QueryOptimizer) SetIndexRecommender(recommender IndexRecommender) {
	qo.mu.Lock()
	defer qo.mu.Unlock()
	qo.indexRecommender = recommender
}

// SetQueryRewriter sets the query rewriter.
func (qo *QueryOptimizer) SetQueryRewriter(rewriter QueryRewriter) {
	qo.mu.Lock()
	defer qo.mu.Unlock()
	qo.rewriter = rewriter
}

// QueryAnalysis represents the result of query analysis.
type QueryAnalysis struct {
	Query            string
	Plan             *QueryPlan
	EstimatedCost    float64
	EstimatedRows    int64
	Recommendations  []*IndexRecommendation
	Warnings         []string
	AnalyzedAt       time.Time
	AnalysisDuration time.Duration
}

// Helper function to truncate query for logging.
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}
