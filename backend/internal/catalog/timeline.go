package catalog

import (
	"context"
	"fmt"
	"time"
)

// TimelineEngine provides timeline visualization and relationship mapping.
type TimelineEngine struct {
	indexer *CatalogIndexer
}

// NewTimelineEngine creates a new timeline engine.
func NewTimelineEngine(indexer *CatalogIndexer) *TimelineEngine {
	return &TimelineEngine{
		indexer: indexer,
	}
}

// TimelineEntry represents a single entry in the timeline.
type TimelineEntry struct {
	ID           string                 `json:"id"`
	DatabaseName string                 `json:"database_name"`
	BackupType   string                 `json:"backup_type"`
	Status       string                 `json:"status"`
	SizeBytes    int64                  `json:"size_bytes"`
	Duration     float64                `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Timeline represents a collection of timeline entries.
type Timeline struct {
	Entries    []*TimelineEntry `json:"entries"`
	StartDate  time.Time        `json:"start_date"`
	EndDate    time.Time        `json:"end_date"`
	TotalCount int64            `json:"total_count"`
}

// TimelineOptions configures timeline generation.
type TimelineOptions struct {
	// DatabaseName filters timeline by database
	DatabaseName string

	// StartDate is the start of the timeline
	StartDate time.Time

	// EndDate is the end of the timeline
	EndDate time.Time

	// Interval is the grouping interval (hour, day, week, month)
	Interval string

	// Limit is the maximum number of entries
	Limit int
}

// GetTimeline generates a visual timeline of backups.
func (te *TimelineEngine) GetTimeline(ctx context.Context, options *TimelineOptions) (*Timeline, error) {
	if options == nil {
		options = &TimelineOptions{}
	}

	// Set defaults
	if options.EndDate.IsZero() {
		options.EndDate = time.Now()
	}
	if options.StartDate.IsZero() {
		options.StartDate = options.EndDate.AddDate(0, -1, 0) // Last month
	}
	if options.Limit <= 0 {
		options.Limit = 1000
	}

	// Build query
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"range": map[string]interface{}{
							"created_at": map[string]interface{}{
								"gte": options.StartDate.Format(time.RFC3339),
								"lte": options.EndDate.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"size": options.Limit,
		"sort": []map[string]interface{}{
			{
				"created_at": map[string]interface{}{
					"order": "asc",
				},
			},
		},
	}

	// Add database filter if specified
	if options.DatabaseName != "" {
		boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
		filters := boolQuery["filter"].([]map[string]interface{})
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"database_name.keyword": options.DatabaseName,
			},
		})
		boolQuery["filter"] = filters
	}

	// Execute search
	response, err := te.indexer.client.Search(ctx, te.indexer.indexName, query)
	if err != nil {
		return nil, fmt.Errorf("timeline query failed: %w", err)
	}

	// Convert results to timeline entries
	entries := make([]*TimelineEntry, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		backup := &BackupDocument{}
		if err := convertMapToStruct(hit.Source, backup); err != nil {
			return nil, fmt.Errorf("failed to convert result: %w", err)
		}

		entries[i] = &TimelineEntry{
			ID:           backup.ID,
			DatabaseName: backup.DatabaseName,
			BackupType:   backup.BackupType,
			Status:       backup.Status,
			SizeBytes:    backup.SizeBytes,
			Duration:     backup.Duration,
			Timestamp:    backup.CreatedAt,
			Metadata:     backup.Metadata,
		}
	}

	return &Timeline{
		Entries:    entries,
		StartDate:  options.StartDate,
		EndDate:    options.EndDate,
		TotalCount: response.GetTotal(),
	}, nil
}

// BackupRelationship represents a relationship between backups.
type BackupRelationship struct {
	ParentBackup *BackupDocument   `json:"parent_backup"`
	ChildBackups []*BackupDocument `json:"child_backups"`
	TotalSize    int64             `json:"total_size"`
	ChainLength  int               `json:"chain_length"`
}

// GetBackupChain retrieves the full backup chain for a given backup.
func (te *TimelineEngine) GetBackupChain(ctx context.Context, backupID string) (*BackupRelationship, error) {
	// Get the backup
	backup, err := te.indexer.GetBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}

	// Find the root (full backup) if this is an incremental
	root := backup
	if backup.ParentBackupID != "" {
		root, err = te.findRootBackup(ctx, backup)
		if err != nil {
			return nil, err
		}
	}

	// Get all child backups
	children, err := te.getChildBackups(ctx, root.ID)
	if err != nil {
		return nil, err
	}

	// Calculate total size and chain length
	totalSize := root.SizeBytes
	for _, child := range children {
		totalSize += child.SizeBytes
	}

	return &BackupRelationship{
		ParentBackup: root,
		ChildBackups: children,
		TotalSize:    totalSize,
		ChainLength:  1 + len(children),
	}, nil
}

// findRootBackup finds the root (full) backup in a chain.
func (te *TimelineEngine) findRootBackup(ctx context.Context, backup *BackupDocument) (*BackupDocument, error) {
	current := backup
	visited := make(map[string]bool) // Prevent infinite loops

	for current.ParentBackupID != "" {
		// Check for circular reference
		if visited[current.ID] {
			return nil, fmt.Errorf("circular backup chain detected for backup %s", current.ID)
		}
		visited[current.ID] = true

		parent, err := te.indexer.GetBackup(ctx, current.ParentBackupID)
		if err != nil {
			return nil, fmt.Errorf("failed to find parent backup %s: %w", current.ParentBackupID, err)
		}

		current = parent
	}

	return current, nil
}

// getChildBackups retrieves all child backups recursively.
func (te *TimelineEngine) getChildBackups(ctx context.Context, parentID string) ([]*BackupDocument, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"parent_backup_id": parentID,
			},
		},
		"size": 100, // Reasonable limit for chain length
		"sort": []map[string]interface{}{
			{
				"created_at": map[string]interface{}{
					"order": "asc",
				},
			},
		},
	}

	response, err := te.indexer.client.Search(ctx, te.indexer.indexName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get child backups: %w", err)
	}

	children := make([]*BackupDocument, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		backup := &BackupDocument{}
		if err := convertMapToStruct(hit.Source, backup); err != nil {
			return nil, fmt.Errorf("failed to convert result: %w", err)
		}
		children[i] = backup
	}

	return children, nil
}

// BackupDependency represents a dependency between backups.
type BackupDependency struct {
	BackupID       string   `json:"backup_id"`
	DependsOn      []string `json:"depends_on"`
	RequiredBy     []string `json:"required_by"`
	IsRestoreable  bool     `json:"is_restoreable"`
	MissingBackups []string `json:"missing_backups,omitempty"`
}

// GetBackupDependencies analyzes dependencies for a backup.
func (te *TimelineEngine) GetBackupDependencies(ctx context.Context, backupID string) (*BackupDependency, error) {
	backup, err := te.indexer.GetBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}

	dependency := &BackupDependency{
		BackupID:   backupID,
		DependsOn:  []string{},
		RequiredBy: []string{},
	}

	// If this is an incremental backup, it depends on its parent chain
	if backup.ParentBackupID != "" {
		// Build dependency chain
		current := backup
		visited := make(map[string]bool)

		for current.ParentBackupID != "" {
			if visited[current.ID] {
				return nil, fmt.Errorf("circular dependency detected")
			}
			visited[current.ID] = true

			dependency.DependsOn = append(dependency.DependsOn, current.ParentBackupID)

			// Check if parent exists
			parent, err := te.indexer.GetBackup(ctx, current.ParentBackupID)
			if err != nil {
				dependency.MissingBackups = append(dependency.MissingBackups, current.ParentBackupID)
				dependency.IsRestoreable = false
				break
			}

			current = parent
		}
	}

	// Find backups that depend on this one
	children, err := te.getChildBackups(ctx, backupID)
	if err == nil {
		for _, child := range children {
			dependency.RequiredBy = append(dependency.RequiredBy, child.ID)
		}
	}

	// Backup is restoreable if it's a full backup or all dependencies exist
	dependency.IsRestoreable = len(dependency.MissingBackups) == 0

	return dependency, nil
}

// DatabaseTimeline represents backup timeline for a specific database.
type DatabaseTimeline struct {
	DatabaseName string           `json:"database_name"`
	FirstBackup  time.Time        `json:"first_backup"`
	LastBackup   time.Time        `json:"last_backup"`
	TotalBackups int64            `json:"total_backups"`
	FullBackups  int64            `json:"full_backups"`
	IncrBackups  int64            `json:"incremental_backups"`
	SuccessRate  float64          `json:"success_rate"`
	AverageSize  int64            `json:"average_size"`
	TotalSize    int64            `json:"total_size"`
	BackupChains int              `json:"backup_chains"`
	Timeline     []*TimelineEntry `json:"timeline"`
}

// GetDatabaseTimeline generates a timeline for a specific database.
func (te *TimelineEngine) GetDatabaseTimeline(ctx context.Context, databaseName string, days int) (*DatabaseTimeline, error) {
	if days <= 0 {
		days = 30 // Default to last 30 days
	}

	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	// Get timeline
	timeline, err := te.GetTimeline(ctx, &TimelineOptions{
		DatabaseName: databaseName,
		StartDate:    startDate,
		EndDate:      endDate,
		Limit:        10000,
	})
	if err != nil {
		return nil, err
	}

	// Calculate statistics using aggregations
	aggQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"database_name.keyword": databaseName,
						},
					},
					{
						"range": map[string]interface{}{
							"created_at": map[string]interface{}{
								"gte": startDate.Format(time.RFC3339),
								"lte": endDate.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"size": 0,
		"aggs": map[string]interface{}{
			"by_type": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "backup_type",
				},
			},
			"by_status": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "status",
				},
			},
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
			"first_backup": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "created_at",
				},
			},
			"last_backup": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "created_at",
				},
			},
		},
	}

	response, err := te.indexer.client.Search(ctx, te.indexer.indexName, aggQuery)
	if err != nil {
		return nil, fmt.Errorf("aggregation query failed: %w", err)
	}

	dbTimeline := &DatabaseTimeline{
		DatabaseName: databaseName,
		TotalBackups: timeline.TotalCount,
		Timeline:     timeline.Entries,
	}

	// Parse aggregations
	if aggs := response.Aggregations; aggs != nil {
		// Backup counts by type
		if byType := parseTermsAggregation(aggs, "by_type"); byType != nil {
			dbTimeline.FullBackups = byType["full"]
			dbTimeline.IncrBackups = byType["incremental"]
			dbTimeline.BackupChains = int(byType["full"]) // Approximate
		}

		// Success rate
		if byStatus := parseTermsAggregation(aggs, "by_status"); byStatus != nil {
			success := byStatus["success"]
			total := timeline.TotalCount
			if total > 0 {
				dbTimeline.SuccessRate = float64(success) / float64(total) * 100
			}
		}

		// Size statistics
		if totalSize, ok := aggs["total_size"].(map[string]interface{}); ok {
			if value, ok := totalSize["value"].(float64); ok {
				dbTimeline.TotalSize = int64(value)
			}
		}
		if avgSize, ok := aggs["avg_size"].(map[string]interface{}); ok {
			if value, ok := avgSize["value"].(float64); ok {
				dbTimeline.AverageSize = int64(value)
			}
		}

		// First and last backup timestamps
		if firstBackup, ok := aggs["first_backup"].(map[string]interface{}); ok {
			if value, ok := firstBackup["value_as_string"].(string); ok {
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					dbTimeline.FirstBackup = t
				}
			}
		}
		if lastBackup, ok := aggs["last_backup"].(map[string]interface{}); ok {
			if value, ok := lastBackup["value_as_string"].(string); ok {
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					dbTimeline.LastBackup = t
				}
			}
		}
	}

	return dbTimeline, nil
}

// CompareDatabases compares backup timelines across multiple databases.
func (te *TimelineEngine) CompareDatabases(ctx context.Context, databaseNames []string, days int) (map[string]*DatabaseTimeline, error) {
	results := make(map[string]*DatabaseTimeline)

	for _, dbName := range databaseNames {
		timeline, err := te.GetDatabaseTimeline(ctx, dbName, days)
		if err != nil {
			return nil, fmt.Errorf("failed to get timeline for %s: %w", dbName, err)
		}
		results[dbName] = timeline
	}

	return results, nil
}
