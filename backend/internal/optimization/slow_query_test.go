package optimization

import (
	"testing"
	"time"
)

func TestNewSlowQueryDetector(t *testing.T) {
	threshold := 500 * time.Millisecond
	detector := NewSlowQueryDetector(threshold)

	if detector == nil {
		t.Fatal("Expected detector to be created")
	}

	if detector.threshold != threshold {
		t.Errorf("Expected threshold %v, got %v", threshold, detector.threshold)
	}

	if detector.slowQueries == nil {
		t.Error("Expected slowQueries to be initialized")
	}

	if detector.queryStats == nil {
		t.Error("Expected queryStats to be initialized")
	}

	t.Log("✓ SlowQueryDetector created successfully")
}

func TestDetectSlowQuery(t *testing.T) {
	detector := NewSlowQueryDetector(1 * time.Second)

	// Test query below threshold
	query1 := "SELECT * FROM users WHERE id = 1"
	slowQuery1 := detector.Detect(query1, 500*time.Millisecond)

	if slowQuery1 != nil {
		t.Error("Expected query below threshold not to be detected as slow")
	}

	// Test query above threshold
	query2 := "SELECT * FROM large_table JOIN another_table"
	slowQuery2 := detector.Detect(query2, 2*time.Second)

	if slowQuery2 == nil {
		t.Fatal("Expected query above threshold to be detected as slow")
	}

	if slowQuery2.Query != query2 {
		t.Errorf("Expected query '%s', got '%s'", query2, slowQuery2.Query)
	}

	if slowQuery2.ExecutionTime != 2*time.Second {
		t.Errorf("Expected execution time 2s, got %v", slowQuery2.ExecutionTime)
	}

	if slowQuery2.QueryHash == "" {
		t.Error("Expected query hash to be set")
	}

	if slowQuery2.Severity == "" {
		t.Error("Expected severity to be set")
	}

	t.Log("✓ Slow query detection works correctly")
}

func TestCalculateSeverity(t *testing.T) {
	detector := NewSlowQueryDetector(1 * time.Second)

	tests := []struct {
		executionTime time.Duration
		expected      Severity
	}{
		{1500 * time.Millisecond, SeverityLow},      // 1.5x threshold
		{3 * time.Second, SeverityMedium},           // 3x threshold
		{7 * time.Second, SeverityHigh},             // 7x threshold
		{15 * time.Second, SeverityCritical},        // 15x threshold
	}

	for _, tt := range tests {
		severity := detector.calculateSeverity(tt.executionTime)
		if severity != tt.expected {
			t.Errorf("For execution time %v, expected severity %s, got %s",
				tt.executionTime, tt.expected, severity)
		}
	}

	t.Log("✓ Severity calculation works correctly")
}

func TestGetSlowQueries(t *testing.T) {
	detector := NewSlowQueryDetector(100 * time.Millisecond)

	// Add multiple slow queries
	detector.Detect("SELECT * FROM table1", 200*time.Millisecond)
	detector.Detect("SELECT * FROM table2", 300*time.Millisecond)
	detector.Detect("SELECT * FROM table3", 400*time.Millisecond)
	detector.Detect("SELECT * FROM table4", 500*time.Millisecond)

	// Get last 2
	queries := detector.GetSlowQueries(2)

	if len(queries) != 2 {
		t.Errorf("Expected 2 queries, got %d", len(queries))
	}

	// Verify they are the most recent ones
	if queries[0].Query != "SELECT * FROM table3" {
		t.Errorf("Expected first query to be 'SELECT * FROM table3', got '%s'", queries[0].Query)
	}

	// Get all
	allQueries := detector.GetSlowQueries(0)
	if len(allQueries) != 4 {
		t.Errorf("Expected 4 queries, got %d", len(allQueries))
	}

	t.Log("✓ GetSlowQueries works correctly")
}

func TestSlowQueryStats(t *testing.T) {
	detector := NewSlowQueryDetector(100 * time.Millisecond)

	query := "SELECT * FROM users WHERE age > 18"

	// Detect same query multiple times
	detector.Detect(query, 200*time.Millisecond)
	detector.Detect(query, 300*time.Millisecond)
	detector.Detect(query, 400*time.Millisecond)

	queryHash := calculateQueryHash(query)
	stats := detector.GetQueryStats(queryHash)

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.ExecutionCount != 3 {
		t.Errorf("Expected execution count 3, got %d", stats.ExecutionCount)
	}

	if stats.MinDuration != 200*time.Millisecond {
		t.Errorf("Expected min duration 200ms, got %v", stats.MinDuration)
	}

	if stats.MaxDuration != 400*time.Millisecond {
		t.Errorf("Expected max duration 400ms, got %v", stats.MaxDuration)
	}

	expectedAvg := 300 * time.Millisecond
	if stats.AvgDuration != expectedAvg {
		t.Errorf("Expected avg duration %v, got %v", expectedAvg, stats.AvgDuration)
	}

	t.Log("✓ Query statistics tracking works correctly")
}

func TestGetTopSlowQueries(t *testing.T) {
	detector := NewSlowQueryDetector(100 * time.Millisecond)

	// Add queries with different average times
	query1 := "SELECT * FROM fast_table"
	query2 := "SELECT * FROM medium_table"
	query3 := "SELECT * FROM slow_table"

	// Fast query
	detector.Detect(query1, 200*time.Millisecond)
	detector.Detect(query1, 200*time.Millisecond)

	// Medium query
	detector.Detect(query2, 500*time.Millisecond)
	detector.Detect(query2, 500*time.Millisecond)

	// Slow query
	detector.Detect(query3, 1*time.Second)
	detector.Detect(query3, 1*time.Second)

	topQueries := detector.GetTopSlowQueries(2)

	if len(topQueries) != 2 {
		t.Errorf("Expected 2 top queries, got %d", len(topQueries))
	}

	// First should be the slowest
	if topQueries[0].AvgDuration != 1*time.Second {
		t.Errorf("Expected slowest query to have avg duration 1s, got %v", topQueries[0].AvgDuration)
	}

	// Second should be medium
	if topQueries[1].AvgDuration != 500*time.Millisecond {
		t.Errorf("Expected second query to have avg duration 500ms, got %v", topQueries[1].AvgDuration)
	}

	t.Log("✓ Top slow queries retrieval works correctly")
}

func TestClearSlowQueries(t *testing.T) {
	detector := NewSlowQueryDetector(100 * time.Millisecond)

	// Add some queries
	detector.Detect("SELECT * FROM table1", 200*time.Millisecond)
	detector.Detect("SELECT * FROM table2", 300*time.Millisecond)

	count := detector.GetCount()
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	// Clear
	detector.Clear()

	count = detector.GetCount()
	if count != 0 {
		t.Errorf("Expected count to be 0 after clear, got %d", count)
	}

	queries := detector.GetSlowQueries(10)
	if len(queries) != 0 {
		t.Errorf("Expected no queries after clear, got %d", len(queries))
	}

	t.Log("✓ Clear works correctly")
}

func TestSetThreshold(t *testing.T) {
	detector := NewSlowQueryDetector(1 * time.Second)

	newThreshold := 500 * time.Millisecond
	detector.SetThreshold(newThreshold)

	if detector.GetThreshold() != newThreshold {
		t.Errorf("Expected threshold %v, got %v", newThreshold, detector.GetThreshold())
	}

	// Test detection with new threshold
	slowQuery := detector.Detect("SELECT * FROM test", 600*time.Millisecond)
	if slowQuery == nil {
		t.Error("Expected query to be detected as slow with new threshold")
	}

	fastQuery := detector.Detect("SELECT * FROM test2", 400*time.Millisecond)
	if fastQuery != nil {
		t.Error("Expected query not to be detected as slow with new threshold")
	}

	t.Log("✓ Threshold setting works correctly")
}

func TestGenerateRecommendations(t *testing.T) {
	detector := NewSlowQueryDetector(1 * time.Second)

	tests := []struct {
		executionTime time.Duration
		minRecsCount  int
	}{
		{15 * time.Second, 1},  // Should get basic recommendations
		{35 * time.Second, 2},  // Should get more recommendations
		{65 * time.Second, 3},  // Should get even more recommendations
	}

	for _, tt := range tests {
		recs := detector.generateRecommendations(tt.executionTime)
		if len(recs) < tt.minRecsCount {
			t.Errorf("For execution time %v, expected at least %d recommendations, got %d",
				tt.executionTime, tt.minRecsCount, len(recs))
		}
	}

	t.Log("✓ Recommendation generation works correctly")
}

func TestCalculateQueryHash(t *testing.T) {
	query1 := "SELECT * FROM users WHERE id = 1"
	query2 := "SELECT * FROM users WHERE id = 2"
	query3 := "SELECT * FROM users WHERE id = 1" // Same as query1

	hash1 := calculateQueryHash(query1)
	hash2 := calculateQueryHash(query2)
	hash3 := calculateQueryHash(query3)

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	if hash1 == hash2 {
		t.Error("Expected different queries to have different hashes")
	}

	if hash1 != hash3 {
		t.Error("Expected same queries to have same hash")
	}

	t.Log("✓ Query hash calculation works correctly")
}
