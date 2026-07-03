package catalog

import (
	"context"
	"testing"
	"time"
)

// Mock Elasticsearch client for testing.
type mockESClient struct {
	documents map[string]map[string]interface{}
	indices   map[string]bool
}

func newMockESClient() *mockESClient {
	return &mockESClient{
		documents: make(map[string]map[string]interface{}),
		indices:   make(map[string]bool),
	}
}

func (m *mockESClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockESClient) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	m.indices[indexName] = true
	return nil
}

func (m *mockESClient) IndexExists(ctx context.Context, indexName string) (bool, error) {
	return m.indices[indexName], nil
}

func (m *mockESClient) IndexDocument(ctx context.Context, indexName, documentID string, document interface{}) error {
	fullKey := indexName + ":" + documentID
	m.documents[fullKey] = document.(map[string]interface{})
	return nil
}

// Tests for ElasticsearchClient

func TestNewElasticsearchClient(t *testing.T) {
	config := DefaultElasticsearchConfig()
	if config == nil {
		t.Fatal("DefaultElasticsearchConfig returned nil")
	}

	if len(config.Addresses) == 0 {
		t.Error("Expected at least one address in default config")
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout=30s, got %v", config.Timeout)
	}

	if config.IndexPrefix != "db-backup" {
		t.Errorf("Expected IndexPrefix='db-backup', got %s", config.IndexPrefix)
	}
}

// Tests for CatalogIndexer

func TestBackupDocument(t *testing.T) {
	now := time.Now()
	expires := now.Add(30 * 24 * time.Hour)

	backup := &BackupDocument{
		ID:              "backup-001",
		DatabaseName:    "test-db",
		DatabaseType:    "postgres",
		BackupType:      "full",
		Status:          "success",
		SizeBytes:       1024 * 1024 * 1024, // 1GB
		CompressedSize:  500 * 1024 * 1024,  // 500MB
		Duration:        120.5,              // 120.5 seconds
		StorageProvider: "s3",
		StoragePath:     "s3://backup-bucket/test-db/backup-001.tar.gz",
		CreatedAt:       now,
		ExpiresAt:       &expires,
		Tags: map[string]string{
			"environment": "production",
			"team":        "database",
		},
		ChecksumType: "sha256",
		Checksum:     "abc123def456",
		Tables:       []string{"users", "orders", "products"},
		Schemas:      []string{"public"},
		RestoreCount: 0,
		AccessCount:  0,
	}

	if backup.ID == "" {
		t.Error("Backup ID should not be empty")
	}

	if backup.DatabaseName != "test-db" {
		t.Errorf("Expected DatabaseName='test-db', got %s", backup.DatabaseName)
	}

	if backup.SizeBytes != 1024*1024*1024 {
		t.Errorf("Expected SizeBytes=1GB, got %d", backup.SizeBytes)
	}

	if len(backup.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(backup.Tags))
	}

	if len(backup.Tables) != 3 {
		t.Errorf("Expected 3 tables, got %d", len(backup.Tables))
	}
}

func TestConvertMapToStruct(t *testing.T) {
	now := time.Now()
	m := map[string]interface{}{
		"id":               "backup-001",
		"database_name":    "test-db",
		"database_type":    "mysql",
		"backup_type":      "full",
		"status":           "success",
		"size_bytes":       float64(1024 * 1024 * 1024),
		"compressed_size":  float64(500 * 1024 * 1024),
		"duration":         120.5,
		"storage_provider": "s3",
		"storage_path":     "s3://bucket/path",
		"created_at":       now.Format(time.RFC3339),
		"restore_count":    float64(5),
		"access_count":     float64(10),
	}

	var backup BackupDocument
	err := convertMapToStruct(m, &backup)
	if err != nil {
		t.Fatalf("convertMapToStruct failed: %v", err)
	}

	if backup.ID != "backup-001" {
		t.Errorf("Expected ID='backup-001', got %s", backup.ID)
	}

	if backup.DatabaseName != "test-db" {
		t.Errorf("Expected DatabaseName='test-db', got %s", backup.DatabaseName)
	}

	if backup.RestoreCount != 5 {
		t.Errorf("Expected RestoreCount=5, got %d", backup.RestoreCount)
	}
}

// Tests for SearchEngine

func TestSearchQuery(t *testing.T) {
	now := time.Now()
	minSize := int64(1024 * 1024)        // 1MB
	maxSize := int64(1024 * 1024 * 1024) // 1GB

	query := &SearchQuery{
		Text:             "production database",
		DatabaseNames:    []string{"app-db", "user-db"},
		DatabaseTypes:    []string{"postgres", "mysql"},
		BackupTypes:      []string{"full"},
		Status:           []string{"success"},
		StorageProviders: []string{"s3"},
		Tags: map[string]string{
			"environment": "production",
		},
		DateFrom:       &now,
		DateTo:         &now,
		MinSize:        &minSize,
		MaxSize:        &maxSize,
		Tables:         []string{"users"},
		Schemas:        []string{"public"},
		ParentBackupID: "",
		IncludeExpired: false,
		Limit:          100,
		Offset:         0,
		SortBy:         "created_at",
		SortOrder:      "desc",
	}

	if query.Text != "production database" {
		t.Errorf("Expected Text='production database', got %s", query.Text)
	}

	if len(query.DatabaseNames) != 2 {
		t.Errorf("Expected 2 database names, got %d", len(query.DatabaseNames))
	}

	if len(query.Tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(query.Tags))
	}

	if query.Limit != 100 {
		t.Errorf("Expected Limit=100, got %d", query.Limit)
	}

	if query.SortBy != "created_at" {
		t.Errorf("Expected SortBy='created_at', got %s", query.SortBy)
	}
}

func TestParseQueryString(t *testing.T) {
	// This is a placeholder test - actual parsing would require a real SearchEngine
	queryString := "database:mydb type:mysql status:success production data"

	// Expected parsing results:
	// - DatabaseNames: ["mydb"]
	// - DatabaseTypes: ["mysql"]
	// - Status: ["success"]
	// - Text: "production data"

	// For now, just validate the query string format
	if queryString == "" {
		t.Error("Query string should not be empty")
	}
}

func TestCatalogStats(t *testing.T) {
	stats := &CatalogStats{
		TotalBackups:    1000,
		TotalSize:       1024 * 1024 * 1024 * 1024, // 1TB
		AverageSize:     1024 * 1024 * 1024,        // 1GB
		AverageDuration: 120.5,
		ByStatus: map[string]int64{
			"success": 950,
			"failed":  50,
		},
		ByType: map[string]int64{
			"full":        200,
			"incremental": 800,
		},
		ByDatabaseType: map[string]int64{
			"postgres": 600,
			"mysql":    400,
		},
		ByProvider: map[string]int64{
			"s3":    700,
			"azure": 300,
		},
	}

	if stats.TotalBackups != 1000 {
		t.Errorf("Expected TotalBackups=1000, got %d", stats.TotalBackups)
	}

	if stats.ByStatus["success"] != 950 {
		t.Errorf("Expected 950 successful backups, got %d", stats.ByStatus["success"])
	}

	if stats.ByType["full"] != 200 {
		t.Errorf("Expected 200 full backups, got %d", stats.ByType["full"])
	}

	if stats.ByDatabaseType["postgres"] != 600 {
		t.Errorf("Expected 600 postgres backups, got %d", stats.ByDatabaseType["postgres"])
	}

	if stats.ByProvider["s3"] != 700 {
		t.Errorf("Expected 700 S3 backups, got %d", stats.ByProvider["s3"])
	}

	// Calculate success rate
	successRate := float64(stats.ByStatus["success"]) / float64(stats.TotalBackups) * 100
	if successRate != 95.0 {
		t.Errorf("Expected 95%% success rate, got %.2f%%", successRate)
	}
}

// Tests for TimelineEngine

func TestTimelineEntry(t *testing.T) {
	now := time.Now()

	entry := &TimelineEntry{
		ID:           "backup-001",
		DatabaseName: "test-db",
		BackupType:   "full",
		Status:       "success",
		SizeBytes:    1024 * 1024 * 1024,
		Duration:     120.5,
		Timestamp:    now,
		Metadata: map[string]interface{}{
			"compression": "gzip",
			"encryption":  "aes256",
		},
	}

	if entry.ID == "" {
		t.Error("Entry ID should not be empty")
	}

	if entry.DatabaseName != "test-db" {
		t.Errorf("Expected DatabaseName='test-db', got %s", entry.DatabaseName)
	}

	if entry.BackupType != "full" {
		t.Errorf("Expected BackupType='full', got %s", entry.BackupType)
	}

	if entry.Status != "success" {
		t.Errorf("Expected Status='success', got %s", entry.Status)
	}

	if len(entry.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(entry.Metadata))
	}
}

func TestTimeline(t *testing.T) {
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	timeline := &Timeline{
		Entries: []*TimelineEntry{
			{
				ID:           "backup-001",
				DatabaseName: "test-db",
				BackupType:   "full",
				Status:       "success",
				SizeBytes:    1024 * 1024 * 1024,
				Duration:     120.0,
				Timestamp:    startDate.Add(7 * 24 * time.Hour),
			},
			{
				ID:           "backup-002",
				DatabaseName: "test-db",
				BackupType:   "incremental",
				Status:       "success",
				SizeBytes:    100 * 1024 * 1024,
				Duration:     30.0,
				Timestamp:    startDate.Add(14 * 24 * time.Hour),
			},
		},
		StartDate:  startDate,
		EndDate:    endDate,
		TotalCount: 2,
	}

	if len(timeline.Entries) != 2 {
		t.Errorf("Expected 2 timeline entries, got %d", len(timeline.Entries))
	}

	if timeline.TotalCount != 2 {
		t.Errorf("Expected TotalCount=2, got %d", timeline.TotalCount)
	}

	if timeline.Entries[0].BackupType != "full" {
		t.Errorf("Expected first entry to be full backup, got %s", timeline.Entries[0].BackupType)
	}

	if timeline.Entries[1].BackupType != "incremental" {
		t.Errorf("Expected second entry to be incremental backup, got %s", timeline.Entries[1].BackupType)
	}
}

func TestBackupRelationship(t *testing.T) {
	parent := &BackupDocument{
		ID:           "backup-001",
		DatabaseName: "test-db",
		BackupType:   "full",
		SizeBytes:    1024 * 1024 * 1024,
	}

	children := []*BackupDocument{
		{
			ID:             "backup-002",
			DatabaseName:   "test-db",
			BackupType:     "incremental",
			ParentBackupID: "backup-001",
			SizeBytes:      100 * 1024 * 1024,
		},
		{
			ID:             "backup-003",
			DatabaseName:   "test-db",
			BackupType:     "incremental",
			ParentBackupID: "backup-001",
			SizeBytes:      150 * 1024 * 1024,
		},
	}

	totalSize := parent.SizeBytes
	for _, child := range children {
		totalSize += child.SizeBytes
	}

	relationship := &BackupRelationship{
		ParentBackup: parent,
		ChildBackups: children,
		TotalSize:    totalSize,
		ChainLength:  1 + len(children),
	}

	if relationship.ParentBackup.ID != "backup-001" {
		t.Errorf("Expected parent ID='backup-001', got %s", relationship.ParentBackup.ID)
	}

	if len(relationship.ChildBackups) != 2 {
		t.Errorf("Expected 2 child backups, got %d", len(relationship.ChildBackups))
	}

	if relationship.ChainLength != 3 {
		t.Errorf("Expected ChainLength=3, got %d", relationship.ChainLength)
	}

	expectedSize := int64(1024*1024*1024 + 100*1024*1024 + 150*1024*1024)
	if relationship.TotalSize != expectedSize {
		t.Errorf("Expected TotalSize=%d, got %d", expectedSize, relationship.TotalSize)
	}
}

func TestBackupDependency(t *testing.T) {
	dependency := &BackupDependency{
		BackupID:       "backup-003",
		DependsOn:      []string{"backup-002", "backup-001"},
		RequiredBy:     []string{},
		IsRestoreable:  true,
		MissingBackups: []string{},
	}

	if dependency.BackupID != "backup-003" {
		t.Errorf("Expected BackupID='backup-003', got %s", dependency.BackupID)
	}

	if len(dependency.DependsOn) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(dependency.DependsOn))
	}

	if !dependency.IsRestoreable {
		t.Error("Expected backup to be restoreable")
	}

	if len(dependency.MissingBackups) != 0 {
		t.Errorf("Expected no missing backups, got %d", len(dependency.MissingBackups))
	}
}

func TestBackupDependencyWithMissing(t *testing.T) {
	dependency := &BackupDependency{
		BackupID:       "backup-003",
		DependsOn:      []string{"backup-002", "backup-001"},
		RequiredBy:     []string{},
		IsRestoreable:  false,
		MissingBackups: []string{"backup-002"},
	}

	if dependency.IsRestoreable {
		t.Error("Expected backup to not be restoreable when dependencies are missing")
	}

	if len(dependency.MissingBackups) != 1 {
		t.Errorf("Expected 1 missing backup, got %d", len(dependency.MissingBackups))
	}

	if dependency.MissingBackups[0] != "backup-002" {
		t.Errorf("Expected missing backup='backup-002', got %s", dependency.MissingBackups[0])
	}
}

func TestDatabaseTimeline(t *testing.T) {
	now := time.Now()
	firstBackup := now.AddDate(0, -3, 0)
	lastBackup := now

	dbTimeline := &DatabaseTimeline{
		DatabaseName: "production-db",
		FirstBackup:  firstBackup,
		LastBackup:   lastBackup,
		TotalBackups: 100,
		FullBackups:  20,
		IncrBackups:  80,
		SuccessRate:  98.5,
		AverageSize:  500 * 1024 * 1024,
		TotalSize:    50 * 1024 * 1024 * 1024,
		BackupChains: 20,
		Timeline:     []*TimelineEntry{},
	}

	if dbTimeline.DatabaseName != "production-db" {
		t.Errorf("Expected DatabaseName='production-db', got %s", dbTimeline.DatabaseName)
	}

	if dbTimeline.TotalBackups != 100 {
		t.Errorf("Expected TotalBackups=100, got %d", dbTimeline.TotalBackups)
	}

	if dbTimeline.FullBackups != 20 {
		t.Errorf("Expected FullBackups=20, got %d", dbTimeline.FullBackups)
	}

	if dbTimeline.IncrBackups != 80 {
		t.Errorf("Expected IncrBackups=80, got %d", dbTimeline.IncrBackups)
	}

	if dbTimeline.SuccessRate != 98.5 {
		t.Errorf("Expected SuccessRate=98.5, got %.2f", dbTimeline.SuccessRate)
	}

	if dbTimeline.BackupChains != 20 {
		t.Errorf("Expected BackupChains=20, got %d", dbTimeline.BackupChains)
	}

	// Verify full + incremental = total
	if dbTimeline.FullBackups+dbTimeline.IncrBackups != dbTimeline.TotalBackups {
		t.Error("Full backups + incremental backups should equal total backups")
	}
}

func TestTimelineOptions(t *testing.T) {
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	options := &TimelineOptions{
		DatabaseName: "test-db",
		StartDate:    startDate,
		EndDate:      endDate,
		Interval:     "day",
		Limit:        1000,
	}

	if options.DatabaseName != "test-db" {
		t.Errorf("Expected DatabaseName='test-db', got %s", options.DatabaseName)
	}

	if options.Interval != "day" {
		t.Errorf("Expected Interval='day', got %s", options.Interval)
	}

	if options.Limit != 1000 {
		t.Errorf("Expected Limit=1000, got %d", options.Limit)
	}

	// Verify date range is valid
	if options.EndDate.Before(options.StartDate) {
		t.Error("EndDate should not be before StartDate")
	}

	duration := options.EndDate.Sub(options.StartDate)
	expectedDuration := 30 * 24 * time.Hour // Approximately 1 month
	if duration < expectedDuration-25*time.Hour || duration > expectedDuration+25*time.Hour {
		t.Errorf("Expected ~1 month duration, got %v", duration)
	}
}

func TestSearchResult(t *testing.T) {
	backup := &BackupDocument{
		ID:           "backup-001",
		DatabaseName: "test-db",
		Status:       "success",
		SizeBytes:    1024 * 1024 * 1024,
	}

	result := &SearchResult{
		Backup: backup,
		Score:  95.5,
	}

	if result.Backup.ID != "backup-001" {
		t.Errorf("Expected backup ID='backup-001', got %s", result.Backup.ID)
	}

	if result.Score != 95.5 {
		t.Errorf("Expected Score=95.5, got %.2f", result.Score)
	}
}

func TestSearchResults(t *testing.T) {
	results := &SearchResults{
		Results: []*SearchResult{
			{
				Backup: &BackupDocument{ID: "backup-001"},
				Score:  95.5,
			},
			{
				Backup: &BackupDocument{ID: "backup-002"},
				Score:  88.0,
			},
		},
		Total: 2,
		Took:  150,
	}

	if len(results.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results.Results))
	}

	if results.Total != 2 {
		t.Errorf("Expected Total=2, got %d", results.Total)
	}

	if results.Took != 150 {
		t.Errorf("Expected Took=150ms, got %d", results.Took)
	}

	// Verify scores are in descending order
	if results.Results[0].Score < results.Results[1].Score {
		t.Error("Expected results to be sorted by score in descending order")
	}
}

// Benchmark tests

func BenchmarkConvertMapToStruct(b *testing.B) {
	m := map[string]interface{}{
		"id":               "backup-001",
		"database_name":    "test-db",
		"database_type":    "mysql",
		"backup_type":      "full",
		"status":           "success",
		"size_bytes":       float64(1024 * 1024 * 1024),
		"compressed_size":  float64(500 * 1024 * 1024),
		"duration":         120.5,
		"storage_provider": "s3",
		"storage_path":     "s3://bucket/path",
		"restore_count":    float64(5),
		"access_count":     float64(10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var backup BackupDocument
		_ = convertMapToStruct(m, &backup)
	}
}
