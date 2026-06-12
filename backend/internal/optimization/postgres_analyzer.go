package optimization

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PostgreSQLAnalyzer implements QueryAnalyzer for PostgreSQL
type PostgreSQLAnalyzer struct {
	db *sql.DB
}

// NewPostgreSQLAnalyzer creates a new PostgreSQL query analyzer
func NewPostgreSQLAnalyzer(db *sql.DB) *PostgreSQLAnalyzer {
	return &PostgreSQLAnalyzer{
		db: db,
	}
}

// AnalyzeQueryPlan analyzes the execution plan for a PostgreSQL query
func (pa *PostgreSQLAnalyzer) AnalyzeQueryPlan(ctx context.Context, query string) (*QueryPlan, error) {
	// Use EXPLAIN (ANALYZE, FORMAT JSON) for detailed plan
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, FORMAT JSON) %s", query)

	rows, err := pa.db.QueryContext(ctx, explainQuery)
	if err != nil {
		// Fallback to EXPLAIN without ANALYZE if query has errors
		explainQuery = fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", query)
		rows, err = pa.db.QueryContext(ctx, explainQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to explain query: %w", err)
		}
	}
	defer rows.Close()

	var planJSON string
	if rows.Next() {
		if err := rows.Scan(&planJSON); err != nil {
			return nil, fmt.Errorf("failed to scan explain result: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading explain result: %w", err)
	}

	// Parse JSON plan
	var plans []struct {
		Plan struct {
			NodeType       string  `json:"Node Type"`
			TotalCost      float64 `json:"Total Cost"`
			PlanRows       int64   `json:"Plan Rows"`
			PlanWidth      int     `json:"Plan Width"`
			ActualRows     int64   `json:"Actual Rows"`
			ActualTime     float64 `json:"Actual Total Time"`
			RelationName   string  `json:"Relation Name"`
			IndexName      string  `json:"Index Name"`
			Filter         string  `json:"Filter"`
			JoinType       string  `json:"Join Type"`
			Plans          []interface{} `json:"Plans"`
		} `json:"Plan"`
		ExecutionTime float64 `json:"Execution Time"`
	}

	if err := json.Unmarshal([]byte(planJSON), &plans); err != nil {
		return nil, fmt.Errorf("failed to parse explain JSON: %w", err)
	}

	if len(plans) == 0 {
		return nil, fmt.Errorf("empty explain result")
	}

	planData := plans[0]

	// Create query plan
	plan := &QueryPlan{
		Query:         query,
		DatabaseType:  DatabaseTypePostgreSQL,
		EstimatedCost: planData.Plan.TotalCost,
		EstimatedRows: planData.Plan.PlanRows,
		ActualRows:    planData.Plan.ActualRows,
		ExecutionTime: time.Duration(planData.ExecutionTime * float64(time.Millisecond)),
		AnalyzedAt:    time.Now(),
		PlanNodes:     make([]*PlanNode, 0),
		UsedIndexes:   make([]string, 0),
		Warnings:      make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Parse plan nodes
	rootNode := pa.parsePlanNode(&planData.Plan)
	if rootNode != nil {
		plan.PlanNodes = append(plan.PlanNodes, rootNode)
	}

	// Extract used indexes
	plan.UsedIndexes = pa.extractUsedIndexes(rootNode)

	// Generate recommendations
	plan.Recommendations = pa.generateRecommendations(plan, rootNode)

	// Detect warnings
	plan.Warnings = pa.detectWarnings(rootNode)

	return plan, nil
}

// GetDatabaseType returns the database type
func (pa *PostgreSQLAnalyzer) GetDatabaseType() DatabaseType {
	return DatabaseTypePostgreSQL
}

// EstimateCost estimates the cost of executing a query
func (pa *PostgreSQLAnalyzer) EstimateCost(ctx context.Context, query string) (float64, error) {
	explainQuery := fmt.Sprintf("EXPLAIN (FORMAT JSON) %s", query)

	rows, err := pa.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to explain query: %w", err)
	}
	defer rows.Close()

	var planJSON string
	if rows.Next() {
		if err := rows.Scan(&planJSON); err != nil {
			return 0, fmt.Errorf("failed to scan explain result: %w", err)
		}
	}

	var plans []struct {
		Plan struct {
			TotalCost float64 `json:"Total Cost"`
		} `json:"Plan"`
	}

	if err := json.Unmarshal([]byte(planJSON), &plans); err != nil {
		return 0, fmt.Errorf("failed to parse explain JSON: %w", err)
	}

	if len(plans) == 0 {
		return 0, fmt.Errorf("empty explain result")
	}

	return plans[0].Plan.TotalCost, nil
}

// GetTableStatistics gets statistics for a table
func (pa *PostgreSQLAnalyzer) GetTableStatistics(ctx context.Context, table string) (*TableStatistics, error) {
	// Parse schema and table name
	parts := strings.Split(table, ".")
	schema := "public"
	tableName := table

	if len(parts) == 2 {
		schema = parts[0]
		tableName = parts[1]
	}

	query := `
		SELECT
			schemaname,
			tablename,
			n_live_tup as row_count,
			pg_total_relation_size(schemaname||'.'||tablename) as table_size,
			pg_indexes_size(schemaname||'.'||tablename) as index_size,
			n_dead_tup as dead_tuples,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_user_tables
		WHERE schemaname = $1 AND tablename = $2
	`

	var stats TableStatistics
	var lastVacuum, lastAutoVacuum, lastAnalyze, lastAutoAnalyze sql.NullTime

	err := pa.db.QueryRowContext(ctx, query, schema, tableName).Scan(
		&stats.SchemaName,
		&stats.TableName,
		&stats.RowCount,
		&stats.TableSize,
		&stats.IndexSize,
		&stats.DeadTuples,
		&lastVacuum,
		&lastAutoVacuum,
		&lastAnalyze,
		&lastAutoAnalyze,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get table statistics: %w", err)
	}

	// Use the most recent vacuum time
	if lastVacuum.Valid {
		stats.LastVacuum = &lastVacuum.Time
	} else if lastAutoVacuum.Valid {
		stats.LastVacuum = &lastAutoVacuum.Time
	}

	// Use the most recent analyze time
	if lastAnalyze.Valid {
		stats.LastAnalyze = &lastAnalyze.Time
	} else if lastAutoAnalyze.Valid {
		stats.LastAnalyze = &lastAutoAnalyze.Time
	}

	// Calculate estimated rows and average row size
	if stats.TableSize > 0 && stats.RowCount > 0 {
		stats.AvgRowSize = int(stats.TableSize / stats.RowCount)
	}

	// Calculate bloat percentage
	if stats.RowCount > 0 {
		stats.Bloat = float64(stats.DeadTuples) / float64(stats.RowCount+stats.DeadTuples) * 100
	}

	return &stats, nil
}

// Helper methods

func (pa *PostgreSQLAnalyzer) parsePlanNode(nodeData interface{}) *PlanNode {
	nodeMap, ok := nodeData.(map[string]interface{})
	if !ok {
		return nil
	}

	node := &PlanNode{
		ExtraInfo: make(map[string]interface{}),
	}

	// Extract common fields
	if val, ok := nodeMap["Node Type"].(string); ok {
		node.NodeType = val
		node.Operation = val
	}

	if val, ok := nodeMap["Relation Name"].(string); ok {
		node.Table = val
	}

	if val, ok := nodeMap["Index Name"].(string); ok {
		node.Index = val
	}

	if val, ok := nodeMap["Plan Rows"].(float64); ok {
		node.Rows = int64(val)
	}

	if val, ok := nodeMap["Total Cost"].(float64); ok {
		node.Cost = val
	}

	if val, ok := nodeMap["Plan Width"].(int); ok {
		node.Width = val
	}

	if val, ok := nodeMap["Filter"].(string); ok {
		node.Filters = []string{val}
	}

	if val, ok := nodeMap["Join Type"].(string); ok {
		node.JoinType = val
	}

	// Parse child nodes
	if plans, ok := nodeMap["Plans"].([]interface{}); ok {
		node.Children = make([]*PlanNode, 0, len(plans))
		for _, childData := range plans {
			if childNode := pa.parsePlanNode(childData); childNode != nil {
				node.Children = append(node.Children, childNode)
			}
		}
	}

	return node
}

func (pa *PostgreSQLAnalyzer) extractUsedIndexes(node *PlanNode) []string {
	if node == nil {
		return []string{}
	}

	indexes := make([]string, 0)

	if node.Index != "" {
		indexes = append(indexes, node.Index)
	}

	for _, child := range node.Children {
		childIndexes := pa.extractUsedIndexes(child)
		indexes = append(indexes, childIndexes...)
	}

	return indexes
}

func (pa *PostgreSQLAnalyzer) generateRecommendations(plan *QueryPlan, node *PlanNode) []string {
	recommendations := make([]string, 0)

	// Check for sequential scans on large tables
	if node != nil && node.NodeType == "Seq Scan" && node.Rows > 1000 {
		recommendations = append(recommendations, fmt.Sprintf("Consider adding an index on table '%s' to avoid sequential scan", node.Table))
	}

	// Check for high cost
	if plan.EstimatedCost > 10000 {
		recommendations = append(recommendations, "Query has high estimated cost - consider optimization")
	}

	// Check for missing indexes
	if len(plan.UsedIndexes) == 0 && plan.EstimatedRows > 100 {
		recommendations = append(recommendations, "No indexes used - consider adding appropriate indexes")
	}

	// Recursively check child nodes
	if node != nil {
		for _, child := range node.Children {
			childRecs := pa.generateRecommendations(plan, child)
			recommendations = append(recommendations, childRecs...)
		}
	}

	return recommendations
}

func (pa *PostgreSQLAnalyzer) detectWarnings(node *PlanNode) []string {
	warnings := make([]string, 0)

	if node == nil {
		return warnings
	}

	// Warn about nested loops with many iterations
	if node.NodeType == "Nested Loop" && node.Rows > 10000 {
		warnings = append(warnings, "Nested loop with high row count detected - may be inefficient")
	}

	// Warn about hash joins with large tables
	if node.NodeType == "Hash Join" && node.Rows > 100000 {
		warnings = append(warnings, "Hash join on large dataset - ensure sufficient work_mem")
	}

	// Recursively check children
	for _, child := range node.Children {
		childWarnings := pa.detectWarnings(child)
		warnings = append(warnings, childWarnings...)
	}

	return warnings
}
