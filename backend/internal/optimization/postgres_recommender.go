package optimization

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// PostgreSQLIndexRecommender implements IndexRecommender for PostgreSQL
type PostgreSQLIndexRecommender struct {
	db *sql.DB
}

// NewPostgreSQLIndexRecommender creates a new PostgreSQL index recommender
func NewPostgreSQLIndexRecommender(db *sql.DB) *PostgreSQLIndexRecommender {
	return &PostgreSQLIndexRecommender{
		db: db,
	}
}

// RecommendIndexes recommends indexes based on query patterns
func (pir *PostgreSQLIndexRecommender) RecommendIndexes(ctx context.Context, queries []string) ([]*IndexRecommendation, error) {
	recommendations := make([]*IndexRecommendation, 0)

	// Analyze each query
	for _, query := range queries {
		recs := pir.analyzeQueryForIndexes(query)
		recommendations = append(recommendations, recs...)
	}

	// Deduplicate and prioritize recommendations
	recommendations = pir.deduplicateRecommendations(recommendations)
	recommendations = pir.prioritizeRecommendations(recommendations)

	return recommendations, nil
}

// AnalyzeIndexUsage analyzes current index usage
func (pir *PostgreSQLIndexRecommender) AnalyzeIndexUsage(ctx context.Context) ([]*IndexUsage, error) {
	query := `
		SELECT
			schemaname,
			tablename,
			indexname,
			idx_scan,
			idx_tup_read,
			idx_tup_fetch,
			pg_relation_size(indexrelid) as index_size
		FROM pg_stat_user_indexes
		ORDER BY idx_scan ASC
	`

	rows, err := pir.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query index usage: %w", err)
	}
	defer rows.Close()

	usages := make([]*IndexUsage, 0)

	for rows.Next() {
		var usage IndexUsage
		err := rows.Scan(
			&usage.SchemaName,
			&usage.TableName,
			&usage.IndexName,
			&usage.ScanCount,
			&usage.TupleRead,
			&usage.TupleFetch,
			&usage.IndexSize,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index usage: %w", err)
		}

		// Calculate usage percentage (simplified)
		if usage.TupleRead > 0 {
			usage.UsagePercent = float64(usage.ScanCount) / float64(usage.TupleRead) * 100
		}

		// Generate recommendation
		if usage.ScanCount == 0 && usage.IndexSize > 1024*1024 { // Unused index > 1MB
			usage.Recommendation = "Consider dropping this unused index"
		} else if usage.UsagePercent < 1.0 {
			usage.Recommendation = "Low usage index - review if needed"
		}

		usages = append(usages, &usage)
	}

	return usages, rows.Err()
}

// ValidateIndex checks if an index would be beneficial
func (pir *PostgreSQLIndexRecommender) ValidateIndex(ctx context.Context, table string, columns []string) (*IndexValidation, error) {
	validation := &IndexValidation{
		Table:   table,
		Columns: columns,
		Warnings: make([]string, 0),
		Alternatives: make([]*IndexRecommendation, 0),
	}

	// Check if index already exists
	exists, err := pir.checkIndexExists(ctx, table, columns)
	if err != nil {
		return nil, err
	}

	if exists {
		validation.IsValid = false
		validation.IsBeneficial = false
		validation.Warnings = append(validation.Warnings, "Index already exists on these columns")
		return validation, nil
	}

	// Estimate benefit
	selectivity, err := pir.estimateSelectivity(ctx, table, columns)
	if err != nil {
		validation.Warnings = append(validation.Warnings, fmt.Sprintf("Could not estimate selectivity: %v", err))
	}

	validation.IsValid = true

	// High selectivity = beneficial index
	if selectivity > 0.01 && selectivity < 0.9 {
		validation.IsBeneficial = true
		validation.ExpectedGain = (1.0 - selectivity) * 100
	} else if selectivity >= 0.9 {
		validation.IsBeneficial = false
		validation.Warnings = append(validation.Warnings, "Low selectivity - index may not be beneficial")
	} else {
		validation.IsBeneficial = true
		validation.ExpectedGain = 80.0 // High selectivity = good gain
	}

	return validation, nil
}

// Helper methods

func (pir *PostgreSQLIndexRecommender) analyzeQueryForIndexes(query string) []*IndexRecommendation {
	recommendations := make([]*IndexRecommendation, 0)

	// Parse WHERE clause
	whereColumns := pir.extractWhereColumns(query)
	if len(whereColumns) > 0 {
		for table, columns := range whereColumns {
			rec := &IndexRecommendation{
				Table:     table,
				Columns:   columns,
				IndexType: IndexTypeBTree,
				Reason:    "Frequently used in WHERE clause",
				Confidence: 0.8,
				Priority:   PriorityMedium,
				CreatedAt:  time.Now(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Parse JOIN conditions
	joinColumns := pir.extractJoinColumns(query)
	if len(joinColumns) > 0 {
		for table, columns := range joinColumns {
			rec := &IndexRecommendation{
				Table:     table,
				Columns:   columns,
				IndexType: IndexTypeBTree,
				Reason:    "Used in JOIN condition",
				Confidence: 0.9,
				Priority:   PriorityHigh,
				CreatedAt:  time.Now(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Parse ORDER BY
	orderColumns := pir.extractOrderByColumns(query)
	if len(orderColumns) > 0 {
		for table, columns := range orderColumns {
			rec := &IndexRecommendation{
				Table:     table,
				Columns:   columns,
				IndexType: IndexTypeBTree,
				Reason:    "Used in ORDER BY clause",
				Confidence: 0.7,
				Priority:   PriorityMedium,
				CreatedAt:  time.Now(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

func (pir *PostgreSQLIndexRecommender) extractWhereColumns(query string) map[string][]string {
	result := make(map[string][]string)

	// Simple regex-based extraction (simplified for demo)
	whereRegex := regexp.MustCompile(`WHERE\s+(\w+)\.(\w+)\s*=`)
	matches := whereRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) == 3 {
			table := match[1]
			column := match[2]
			result[table] = append(result[table], column)
		}
	}

	return result
}

func (pir *PostgreSQLIndexRecommender) extractJoinColumns(query string) map[string][]string {
	result := make(map[string][]string)

	// Simple regex-based extraction
	joinRegex := regexp.MustCompile(`JOIN\s+(\w+)\s+ON\s+\w+\.(\w+)\s*=`)
	matches := joinRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) == 3 {
			table := match[1]
			column := match[2]
			result[table] = append(result[table], column)
		}
	}

	return result
}

func (pir *PostgreSQLIndexRecommender) extractOrderByColumns(query string) map[string][]string {
	result := make(map[string][]string)

	// Simple regex-based extraction
	orderRegex := regexp.MustCompile(`ORDER\s+BY\s+(\w+)\.(\w+)`)
	matches := orderRegex.FindAllStringSubmatch(query, -1)

	for _, match := range matches {
		if len(match) == 3 {
			table := match[1]
			column := match[2]
			result[table] = append(result[table], column)
		}
	}

	return result
}

func (pir *PostgreSQLIndexRecommender) checkIndexExists(ctx context.Context, table string, columns []string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = $1
		AND indexdef LIKE $2
	`

	columnPattern := "%" + strings.Join(columns, "%") + "%"

	var count int
	err := pir.db.QueryRowContext(ctx, query, table, columnPattern).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (pir *PostgreSQLIndexRecommender) estimateSelectivity(ctx context.Context, table string, columns []string) (float64, error) {
	if len(columns) == 0 {
		return 0, fmt.Errorf("no columns specified")
	}

	// Simple selectivity estimation
	query := fmt.Sprintf("SELECT COUNT(DISTINCT %s)::float / COUNT(*)::float FROM %s", columns[0], table)

	var selectivity float64
	err := pir.db.QueryRowContext(ctx, query).Scan(&selectivity)
	if err != nil {
		return 0, err
	}

	return selectivity, nil
}

func (pir *PostgreSQLIndexRecommender) deduplicateRecommendations(recommendations []*IndexRecommendation) []*IndexRecommendation {
	seen := make(map[string]bool)
	result := make([]*IndexRecommendation, 0)

	for _, rec := range recommendations {
		key := fmt.Sprintf("%s:%s", rec.Table, strings.Join(rec.Columns, ","))
		if !seen[key] {
			seen[key] = true
			result = append(result, rec)
		}
	}

	return result
}

func (pir *PostgreSQLIndexRecommender) prioritizeRecommendations(recommendations []*IndexRecommendation) []*IndexRecommendation {
	// Sort by priority and confidence
	n := len(recommendations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if pir.comparePriority(recommendations[j], recommendations[j+1]) {
				recommendations[j], recommendations[j+1] = recommendations[j+1], recommendations[j]
			}
		}
	}

	return recommendations
}

func (pir *PostgreSQLIndexRecommender) comparePriority(a, b *IndexRecommendation) bool {
	priorityOrder := map[Priority]int{
		PriorityCritical: 4,
		PriorityHigh:     3,
		PriorityMedium:   2,
		PriorityLow:      1,
	}

	if priorityOrder[a.Priority] < priorityOrder[b.Priority] {
		return true
	}

	if priorityOrder[a.Priority] == priorityOrder[b.Priority] {
		return a.Confidence < b.Confidence
	}

	return false
}
