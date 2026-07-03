package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestEngine returns an in-memory engine seeded with a fixed set of
// documents that cover every filter the tests exercise.
func newTestEngine(t *testing.T) *InMemorySearchEngine {
	t.Helper()

	e := NewInMemorySearchEngine()
	ctx := context.Background()

	base := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(-48 * time.Hour)

	docs := []*BackupDocument{
		{
			ID:              "b1",
			DatabaseName:    "orders",
			DatabaseType:    "postgres",
			BackupType:      "full",
			Status:          "success",
			SizeBytes:       1000,
			Duration:        10,
			StorageProvider: "s3",
			StoragePath:     "s3://bucket/orders/b1.tar.gz",
			CreatedAt:       base,
			Tags:            map[string]string{"env": "prod", "team": "core"},
			Tables:          []string{"users", "orders"},
			Schemas:         []string{"public"},
			ExpiresAt:       &future,
		},
		{
			ID:              "b2",
			DatabaseName:    "orders_archive",
			DatabaseType:    "mysql",
			BackupType:      "incremental",
			Status:          "success",
			SizeBytes:       5000,
			Duration:        50,
			StorageProvider: "gcs",
			StoragePath:     "gcs://bucket/orders_archive/b2.tar.gz",
			CreatedAt:       base.Add(24 * time.Hour),
			Tags:            map[string]string{"env": "staging"},
			Tables:          []string{"orders"},
			Schemas:         []string{"analytics"},
		},
		{
			ID:              "b3",
			DatabaseName:    "billing",
			DatabaseType:    "postgres",
			BackupType:      "full",
			Status:          "failed",
			SizeBytes:       200,
			Duration:        2,
			StorageProvider: "s3",
			StoragePath:     "s3://bucket/billing/b3.tar.gz",
			CreatedAt:       base.Add(72 * time.Hour),
			Tags:            map[string]string{"env": "prod"},
			ErrorMessage:    "connection refused",
		},
		{
			ID:              "b4",
			DatabaseName:    "billing_expired",
			DatabaseType:    "postgres",
			BackupType:      "full",
			Status:          "success",
			SizeBytes:       300,
			Duration:        3,
			StorageProvider: "azure",
			StoragePath:     "azure://bucket/billing_expired/b4.tar.gz",
			CreatedAt:       base.Add(96 * time.Hour),
			ExpiresAt:       &past,
		},
	}

	if err := e.IndexBackups(ctx, docs); err != nil {
		t.Fatalf("IndexBackups failed: %v", err)
	}
	return e
}

// ids extracts the document IDs from search results.
func ids(res *SearchResults) []string {
	out := make([]string, 0, len(res.Results))
	for _, r := range res.Results {
		out = append(out, r.Backup.ID)
	}
	return out
}

// contains reports whether want is present in got.
func contains(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

func TestInMemoryIsAvailable(t *testing.T) {
	e := NewInMemorySearchEngine()
	if !e.IsAvailable() {
		t.Fatal("expected IsAvailable() to be true")
	}
	var iface SearchEngineInterface = e
	if !iface.IsAvailable() {
		t.Fatal("expected interface IsAvailable() to be true")
	}
}

func TestInMemorySearchByText(t *testing.T) {
	e := newTestEngine(t)
	res, err := e.Search(context.Background(), &SearchQuery{Text: "orders"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	got := ids(res)
	// b1 (name+table), b2 (name+table). b3/b4 don't mention orders. b4 excluded (expired) anyway.
	if len(got) != 2 || !contains(got, "b1") || !contains(got, "b2") {
		t.Fatalf("unexpected text results: %v", got)
	}

	// Error message text search.
	res, err = e.Search(context.Background(), &SearchQuery{Text: "refused"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got := ids(res); len(got) != 1 || got[0] != "b3" {
		t.Fatalf("expected only b3 for 'refused', got %v", got)
	}
}

func TestInMemorySearchByType(t *testing.T) {
	e := newTestEngine(t)
	res, err := e.Search(context.Background(), &SearchQuery{DatabaseTypes: []string{"postgres"}})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	got := ids(res)
	// b1, b3 (b4 postgres but expired/excluded).
	if len(got) != 2 || !contains(got, "b1") || !contains(got, "b3") {
		t.Fatalf("unexpected type results: %v", got)
	}
}

func TestInMemorySearchByTag(t *testing.T) {
	e := newTestEngine(t)
	res, err := e.Search(context.Background(), &SearchQuery{Tags: map[string]string{"env": "prod"}})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	got := ids(res)
	if len(got) != 2 || !contains(got, "b1") || !contains(got, "b3") {
		t.Fatalf("unexpected tag results: %v", got)
	}

	// Two tags must all match.
	res, err = e.Search(context.Background(), &SearchQuery{Tags: map[string]string{"env": "prod", "team": "core"}})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got := ids(res); len(got) != 1 || got[0] != "b1" {
		t.Fatalf("expected only b1 for prod+core, got %v", got)
	}
}

func TestInMemorySearchBySizeRange(t *testing.T) {
	e := newTestEngine(t)
	minSize := int64(400)
	maxSize := int64(6000)
	res, err := e.Search(context.Background(), &SearchQuery{MinSize: &minSize, MaxSize: &maxSize})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	// b1=1000, b2=5000 in range. b3=200 below, b4=300 below (and expired).
	got := ids(res)
	if len(got) != 2 || !contains(got, "b1") || !contains(got, "b2") {
		t.Fatalf("unexpected size results: %v", got)
	}
}

func TestInMemorySearchByDateRange(t *testing.T) {
	e := newTestEngine(t)
	from := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, time.January, 3, 12, 0, 0, 0, time.UTC)
	res, err := e.Search(context.Background(), &SearchQuery{DateFrom: &from, DateTo: &to})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	// b2 = Jan 2, b3 = Jan 4 (out). Only b2 within window.
	if got := ids(res); len(got) != 1 || got[0] != "b2" {
		t.Fatalf("expected only b2 in date window, got %v", got)
	}
}

func TestInMemorySearchExcludesExpired(t *testing.T) {
	e := newTestEngine(t)
	res, err := e.Search(context.Background(), &SearchQuery{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got := ids(res); contains(got, "b4") {
		t.Fatalf("expired b4 should be excluded by default: %v", got)
	}
	if res.Total != 3 {
		t.Fatalf("expected 3 non-expired docs, got %d", res.Total)
	}

	// IncludeExpired brings it back.
	res, err = e.Search(context.Background(), &SearchQuery{IncludeExpired: true})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got := ids(res); !contains(got, "b4") || res.Total != 4 {
		t.Fatalf("expected b4 included with IncludeExpired, got %v (total %d)", got, res.Total)
	}
}

func TestInMemorySearchSortAndPaginate(t *testing.T) {
	e := newTestEngine(t)
	res, err := e.Search(context.Background(), &SearchQuery{
		IncludeExpired: true,
		SortBy:         "size_bytes",
		SortOrder:      "asc",
		Limit:          2,
		Offset:         0,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	// Sizes asc: b3=200, b4=300, b1=1000, b2=5000 -> first page b3,b4.
	got := ids(res)
	if len(got) != 2 || got[0] != "b3" || got[1] != "b4" {
		t.Fatalf("unexpected first page: %v", got)
	}
	if res.Total != 4 {
		t.Fatalf("expected total 4, got %d", res.Total)
	}

	// Second page.
	res, err = e.Search(context.Background(), &SearchQuery{
		IncludeExpired: true,
		SortBy:         "size_bytes",
		SortOrder:      "asc",
		Limit:          2,
		Offset:         2,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if got := ids(res); len(got) != 2 || got[0] != "b1" || got[1] != "b2" {
		t.Fatalf("unexpected second page: %v", got)
	}
}

func TestInMemorySuggest(t *testing.T) {
	e := newTestEngine(t)
	got, err := e.Suggest(context.Background(), "orders", "database_name", 10)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if len(got) != 2 || got[0] != "orders" || got[1] != "orders_archive" {
		t.Fatalf("unexpected suggestions: %v", got)
	}

	// Field with .keyword suffix is normalized.
	got, err = e.Suggest(context.Background(), "bil", "database_name.keyword", 10)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if len(got) != 2 || !contains(got, "billing") || !contains(got, "billing_expired") {
		t.Fatalf("unexpected keyword suggestions: %v", got)
	}

	// Limit is respected.
	got, err = e.Suggest(context.Background(), "", "database_name", 1)
	if err != nil {
		t.Fatalf("Suggest failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion with limit, got %v", got)
	}
}

func TestInMemoryStats(t *testing.T) {
	e := newTestEngine(t)
	stats, err := e.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalBackups != 4 {
		t.Fatalf("expected 4 total, got %d", stats.TotalBackups)
	}
	if stats.TotalSize != 6500 {
		t.Fatalf("expected total size 6500, got %d", stats.TotalSize)
	}
	if stats.AverageSize != 1625 {
		t.Fatalf("expected avg size 1625, got %d", stats.AverageSize)
	}
	if stats.ByStatus["success"] != 3 || stats.ByStatus["failed"] != 1 {
		t.Fatalf("unexpected by_status: %v", stats.ByStatus)
	}
	if stats.ByDatabaseType["postgres"] != 3 || stats.ByDatabaseType["mysql"] != 1 {
		t.Fatalf("unexpected by_database_type: %v", stats.ByDatabaseType)
	}
	if stats.ByType["full"] != 3 || stats.ByType["incremental"] != 1 {
		t.Fatalf("unexpected by_type: %v", stats.ByType)
	}
	if stats.ByProvider["s3"] != 2 {
		t.Fatalf("unexpected by_provider: %v", stats.ByProvider)
	}
}

func TestInMemoryGetAndDelete(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()

	doc, err := e.GetBackup(ctx, "b1")
	if err != nil {
		t.Fatalf("GetBackup failed: %v", err)
	}
	if doc.DatabaseName != "orders" {
		t.Fatalf("unexpected doc: %+v", doc)
	}

	// Returned copy must not mutate stored state.
	doc.DatabaseName = "mutated"
	again, err := e.GetBackup(ctx, "b1")
	if err != nil {
		t.Fatalf("GetBackup failed: %v", err)
	}
	if again.DatabaseName != "orders" {
		t.Fatalf("stored doc was mutated through returned pointer")
	}

	if err := e.DeleteBackup(ctx, "b1"); err != nil {
		t.Fatalf("DeleteBackup failed: %v", err)
	}
	if _, err := e.GetBackup(ctx, "b1"); err == nil {
		t.Fatal("expected error for deleted doc")
	}
	// Deleting a missing doc is not an error.
	if err := e.DeleteBackup(ctx, "missing"); err != nil {
		t.Fatalf("deleting missing doc should be a no-op, got %v", err)
	}
}

func TestInMemoryValidationErrors(t *testing.T) {
	e := NewInMemorySearchEngine()
	ctx := context.Background()

	if err := e.IndexBackup(ctx, nil); err == nil {
		t.Fatal("expected error for nil backup")
	}
	if err := e.IndexBackup(ctx, &BackupDocument{}); err == nil {
		t.Fatal("expected error for empty ID")
	}
	if _, err := e.GetBackup(ctx, ""); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestInMemoryParseQueryString(t *testing.T) {
	e := NewInMemorySearchEngine()
	q, err := e.ParseQueryString("db:orders type:postgres status:success freetext")
	if err != nil {
		t.Fatalf("ParseQueryString failed: %v", err)
	}
	if len(q.DatabaseNames) != 1 || q.DatabaseNames[0] != "orders" {
		t.Fatalf("unexpected database names: %v", q.DatabaseNames)
	}
	if len(q.DatabaseTypes) != 1 || q.DatabaseTypes[0] != "postgres" {
		t.Fatalf("unexpected database types: %v", q.DatabaseTypes)
	}
	if q.Text != "freetext" {
		t.Fatalf("unexpected text: %q", q.Text)
	}
}

func TestInMemoryPersistenceSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	e1, err := NewPersistentInMemorySearchEngine(dir)
	if err != nil {
		t.Fatalf("construct persistent engine: %v", err)
	}
	doc := &BackupDocument{
		ID:              "p1",
		DatabaseName:    "persisted",
		DatabaseType:    "postgres",
		BackupType:      "full",
		Status:          "success",
		SizeBytes:       4096,
		StorageProvider: "s3",
		CreatedAt:       time.Now(),
		Tags:            map[string]string{"env": "prod"},
	}
	if err := e1.IndexBackup(ctx, doc); err != nil {
		t.Fatalf("IndexBackup failed: %v", err)
	}

	// The JSON file exists.
	if _, err := os.ReadFile(filepath.Join(dir, memoryStoreFileName)); err != nil {
		t.Fatalf("expected persistence file: %v", err)
	}

	// Simulate a restart by building a new engine from the same dir.
	e2, err := NewPersistentInMemorySearchEngine(dir)
	if err != nil {
		t.Fatalf("reconstruct persistent engine: %v", err)
	}
	got, err := e2.GetBackup(ctx, "p1")
	if err != nil {
		t.Fatalf("GetBackup after restart failed: %v", err)
	}
	if got.DatabaseName != "persisted" || got.Tags["env"] != "prod" {
		t.Fatalf("restored doc mismatch: %+v", got)
	}

	stats, err := e2.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalBackups != 1 {
		t.Fatalf("expected 1 backup after restart, got %d", stats.TotalBackups)
	}
}

func TestNewPersistentInMemorySearchEngineRequiresDir(t *testing.T) {
	if _, err := NewPersistentInMemorySearchEngine(""); err == nil {
		t.Fatal("expected error for empty dir")
	}
}
