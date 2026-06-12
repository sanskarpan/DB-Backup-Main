package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchEngine provides full-text search capabilities for backups
type SearchEngine struct {
	indexer *CatalogIndexer
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(indexer *CatalogIndexer) *SearchEngine {
	return &SearchEngine{
		indexer: indexer,
	}
}

// IsAvailable returns true if the search engine is properly configured
func (se *SearchEngine) IsAvailable() bool {
	return se != nil && se.indexer != nil
}

// SearchQuery represents a search query with various filters
type SearchQuery struct {
	// Text is the free-text search query
	Text string

	// DatabaseNames filters by database names
	DatabaseNames []string

	// DatabaseTypes filters by database types (mysql, postgres, etc.)
	DatabaseTypes []string

	// BackupTypes filters by backup types (full, incremental, etc.)
	BackupTypes []string

	// Status filters by backup status (success, failed, etc.)
	Status []string

	// StorageProviders filters by storage providers
	StorageProviders []string

	// Tags filters by tags (must match all specified tags)
	Tags map[string]string

	// DateFrom filters backups created after this date
	DateFrom *time.Time

	// DateTo filters backups created before this date
	DateTo *time.Time

	// MinSize filters backups larger than this size (bytes)
	MinSize *int64

	// MaxSize filters backups smaller than this size (bytes)
	MaxSize *int64

	// MinDuration filters backups longer than this duration (seconds)
	MinDuration *float64

	// MaxDuration filters backups shorter than this duration (seconds)
	MaxDuration *float64

	// Tables filters by table names (contains any of these tables)
	Tables []string

	// Schemas filters by schema names (contains any of these schemas)
	Schemas []string

	// ParentBackupID filters by parent backup ID
	ParentBackupID string

	// IncludeExpired includes expired backups in results
	IncludeExpired bool

	// Limit is the maximum number of results to return
	Limit int

	// Offset is the number of results to skip
	Offset int

	// SortBy specifies the field to sort by (created_at, size_bytes, duration)
	SortBy string

	// SortOrder specifies the sort order (asc, desc)
	SortOrder string
}

// SearchResult represents a search result
type SearchResult struct {
	Backup *BackupDocument
	Score  float64
}

// SearchResults represents search results with metadata
type SearchResults struct {
	Results []*SearchResult
	Total   int64
	Took    int
}

// Search performs a search query
func (se *SearchEngine) Search(ctx context.Context, query *SearchQuery) (*SearchResults, error) {
	if query == nil {
		query = &SearchQuery{}
	}

	// Set defaults
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.SortBy == "" {
		query.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}

	// Build Elasticsearch query
	esQuery := se.buildElasticsearchQuery(query)

	// Execute search
	response, err := se.indexer.client.Search(ctx, se.indexer.indexName, esQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results
	results := make([]*SearchResult, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		backup := &BackupDocument{}
		if err := convertMapToStruct(hit.Source, backup); err != nil {
			return nil, fmt.Errorf("failed to convert result: %w", err)
		}

		score := 0.0
		if hit.Score != nil {
			score = *hit.Score
		}

		results[i] = &SearchResult{
			Backup: backup,
			Score:  score,
		}
	}

	return &SearchResults{
		Results: results,
		Total:   response.GetTotal(),
		Took:    response.Took,
	}, nil
}

// buildElasticsearchQuery builds an Elasticsearch query from SearchQuery
func (se *SearchEngine) buildElasticsearchQuery(query *SearchQuery) map[string]interface{} {
	// Build the bool query
	mustClauses := []map[string]interface{}{}
	filterClauses := []map[string]interface{}{}

	// Text search (multi-match across multiple fields)
	if query.Text != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query.Text,
				"fields": []string{"database_name^3", "storage_path^2", "tables", "schemas", "error_message"},
				"type":   "best_fields",
			},
		})
	}

	// Database names filter
	if len(query.DatabaseNames) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"database_name.keyword": query.DatabaseNames,
			},
		})
	}

	// Database types filter
	if len(query.DatabaseTypes) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"database_type": query.DatabaseTypes,
			},
		})
	}

	// Backup types filter
	if len(query.BackupTypes) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"backup_type": query.BackupTypes,
			},
		})
	}

	// Status filter
	if len(query.Status) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"status": query.Status,
			},
		})
	}

	// Storage providers filter
	if len(query.StorageProviders) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"storage_provider": query.StorageProviders,
			},
		})
	}

	// Tags filter (must match all specified tags)
	for key, value := range query.Tags {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				fmt.Sprintf("tags.%s.keyword", key): value,
			},
		})
	}

	// Date range filter
	if query.DateFrom != nil || query.DateTo != nil {
		rangeFilter := map[string]interface{}{}
		if query.DateFrom != nil {
			rangeFilter["gte"] = query.DateFrom.Format(time.RFC3339)
		}
		if query.DateTo != nil {
			rangeFilter["lte"] = query.DateTo.Format(time.RFC3339)
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"created_at": rangeFilter,
			},
		})
	}

	// Size filter
	if query.MinSize != nil || query.MaxSize != nil {
		rangeFilter := map[string]interface{}{}
		if query.MinSize != nil {
			rangeFilter["gte"] = *query.MinSize
		}
		if query.MaxSize != nil {
			rangeFilter["lte"] = *query.MaxSize
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"size_bytes": rangeFilter,
			},
		})
	}

	// Duration filter
	if query.MinDuration != nil || query.MaxDuration != nil {
		rangeFilter := map[string]interface{}{}
		if query.MinDuration != nil {
			rangeFilter["gte"] = *query.MinDuration
		}
		if query.MaxDuration != nil {
			rangeFilter["lte"] = *query.MaxDuration
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"duration": rangeFilter,
			},
		})
	}

	// Tables filter
	if len(query.Tables) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"tables": query.Tables,
			},
		})
	}

	// Schemas filter
	if len(query.Schemas) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"schemas": query.Schemas,
			},
		})
	}

	// Parent backup ID filter
	if query.ParentBackupID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"parent_backup_id": query.ParentBackupID,
			},
		})
	}

	// Exclude expired backups unless requested
	if !query.IncludeExpired {
		filterClauses = append(filterClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"bool": map[string]interface{}{
							"must_not": map[string]interface{}{
								"exists": map[string]interface{}{
									"field": "expires_at",
								},
							},
						},
					},
					{
						"range": map[string]interface{}{
							"expires_at": map[string]interface{}{
								"gte": time.Now().Format(time.RFC3339),
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		})
	}

	// Build final bool query
	boolQuery := map[string]interface{}{}
	if len(mustClauses) > 0 {
		boolQuery["must"] = mustClauses
	}
	if len(filterClauses) > 0 {
		boolQuery["filter"] = filterClauses
	}

	// If no conditions, use match_all
	if len(mustClauses) == 0 && len(filterClauses) == 0 {
		boolQuery = map[string]interface{}{
			"must": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
	}

	// Build sort
	sort := []map[string]interface{}{
		{
			query.SortBy: map[string]interface{}{
				"order": query.SortOrder,
			},
		},
	}

	// Build final query
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"from": query.Offset,
		"size": query.Limit,
		"sort": sort,
	}

	return esQuery
}

// ParseQueryString parses a simple query string into a SearchQuery
// Format: "database:mydb type:mysql status:success size:>1GB date:2024-01-01..2024-12-31 searchtext"
func (se *SearchEngine) ParseQueryString(queryString string) (*SearchQuery, error) {
	query := &SearchQuery{}

	// Split into tokens
	tokens := strings.Fields(queryString)
	textTokens := []string{}

	for _, token := range tokens {
		// Check if token contains a colon (filter)
		if strings.Contains(token, ":") {
			parts := strings.SplitN(token, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.ToLower(parts[0])
			value := parts[1]

			switch key {
			case "database", "db":
				query.DatabaseNames = append(query.DatabaseNames, value)
			case "type", "dbtype":
				query.DatabaseTypes = append(query.DatabaseTypes, value)
			case "backup_type", "backuptype":
				query.BackupTypes = append(query.BackupTypes, value)
			case "status":
				query.Status = append(query.Status, value)
			case "provider", "storage":
				query.StorageProviders = append(query.StorageProviders, value)
			case "table":
				query.Tables = append(query.Tables, value)
			case "schema":
				query.Schemas = append(query.Schemas, value)
			case "date":
				// Parse date range (format: YYYY-MM-DD..YYYY-MM-DD or YYYY-MM-DD)
				if strings.Contains(value, "..") {
					dates := strings.Split(value, "..")
					if len(dates) == 2 {
						from, err := time.Parse("2006-01-02", dates[0])
						if err == nil {
							query.DateFrom = &from
						}
						to, err := time.Parse("2006-01-02", dates[1])
						if err == nil {
							query.DateTo = &to
						}
					}
				} else {
					// Single date - search for that day
					date, err := time.Parse("2006-01-02", value)
					if err == nil {
						query.DateFrom = &date
						endDate := date.Add(24 * time.Hour)
						query.DateTo = &endDate
					}
				}
			case "size":
				// Parse size (format: >1GB, <500MB, 1GB..10GB)
				// Simplified version - just store as is for now
				// Full implementation would parse size units
			}
		} else {
			// Regular text token
			textTokens = append(textTokens, token)
		}
	}

	// Join text tokens
	if len(textTokens) > 0 {
		query.Text = strings.Join(textTokens, " ")
	}

	return query, nil
}

// Suggest provides search suggestions based on partial input
func (se *SearchEngine) Suggest(ctx context.Context, prefix string, field string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	// Use aggregation to get unique values
	aggQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"prefix": map[string]interface{}{
				field: map[string]interface{}{
					"value": prefix,
				},
			},
		},
		"size": 0,
		"aggs": map[string]interface{}{
			"suggestions": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": field,
					"size":  limit,
				},
			},
		},
	}

	response, err := se.indexer.client.Search(ctx, se.indexer.indexName, aggQuery)
	if err != nil {
		return nil, fmt.Errorf("suggestion query failed: %w", err)
	}

	// Parse aggregation results
	suggestions := []string{}
	if aggs, ok := response.Aggregations["suggestions"].(map[string]interface{}); ok {
		if buckets, ok := aggs["buckets"].([]interface{}); ok {
			for _, bucket := range buckets {
				if b, ok := bucket.(map[string]interface{}); ok {
					if key, ok := b["key"].(string); ok {
						suggestions = append(suggestions, key)
					}
				}
			}
		}
	}

	return suggestions, nil
}

// GetStats returns statistics about the catalog
func (se *SearchEngine) GetStats(ctx context.Context) (*CatalogStats, error) {
	// Use aggregations to get statistics
	aggQuery := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"total_size": map[string]interface{}{
				"sum": map[string]interface{}{
					"field": "size_bytes",
				},
			},
			"avg_size": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "size_bytes",
				},
			},
			"avg_duration": map[string]interface{}{
				"avg": map[string]interface{}{
					"field": "duration",
				},
			},
			"by_status": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "status",
				},
			},
			"by_type": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "backup_type",
				},
			},
			"by_database_type": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "database_type",
				},
			},
			"by_provider": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "storage_provider",
				},
			},
		},
	}

	response, err := se.indexer.client.Search(ctx, se.indexer.indexName, aggQuery)
	if err != nil {
		return nil, fmt.Errorf("stats query failed: %w", err)
	}

	stats := &CatalogStats{
		TotalBackups: response.GetTotal(),
	}

	// Parse aggregations
	if aggs := response.Aggregations; aggs != nil {
		if totalSize, ok := aggs["total_size"].(map[string]interface{}); ok {
			if value, ok := totalSize["value"].(float64); ok {
				stats.TotalSize = int64(value)
			}
		}
		if avgSize, ok := aggs["avg_size"].(map[string]interface{}); ok {
			if value, ok := avgSize["value"].(float64); ok {
				stats.AverageSize = int64(value)
			}
		}
		if avgDuration, ok := aggs["avg_duration"].(map[string]interface{}); ok {
			if value, ok := avgDuration["value"].(float64); ok {
				stats.AverageDuration = value
			}
		}

		stats.ByStatus = parseTermsAggregation(aggs, "by_status")
		stats.ByType = parseTermsAggregation(aggs, "by_type")
		stats.ByDatabaseType = parseTermsAggregation(aggs, "by_database_type")
		stats.ByProvider = parseTermsAggregation(aggs, "by_provider")
	}

	return stats, nil
}

// parseTermsAggregation parses a terms aggregation from Elasticsearch response
func parseTermsAggregation(aggs map[string]interface{}, name string) map[string]int64 {
	result := make(map[string]int64)

	if agg, ok := aggs[name].(map[string]interface{}); ok {
		if buckets, ok := agg["buckets"].([]interface{}); ok {
			for _, bucket := range buckets {
				if b, ok := bucket.(map[string]interface{}); ok {
					key, _ := b["key"].(string)
					count := int64(0)
					if docCount, ok := b["doc_count"].(float64); ok {
						count = int64(docCount)
					}
					result[key] = count
				}
			}
		}
	}

	return result
}

// CatalogStats represents statistics about the backup catalog
type CatalogStats struct {
	TotalBackups    int64             `json:"total_backups"`
	TotalSize       int64             `json:"total_size"`
	AverageSize     int64             `json:"average_size"`
	AverageDuration float64           `json:"average_duration"`
	ByStatus        map[string]int64  `json:"by_status"`
	ByType          map[string]int64  `json:"by_type"`
	ByDatabaseType  map[string]int64  `json:"by_database_type"`
	ByProvider      map[string]int64  `json:"by_provider"`
}
