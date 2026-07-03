package optimization

import (
	"testing"
	"time"
)

func TestNewStatsCollector(t *testing.T) {
	retention := 24 * time.Hour
	collector := NewStatsCollector(retention)

	if collector == nil {
		t.Fatal("Expected collector to be created")
	}

	if collector.retentionPeriod != retention {
		t.Errorf("Expected retention period %v, got %v", retention, collector.retentionPeriod)
	}

	if collector.statistics == nil {
		t.Error("Expected statistics map to be initialized")
	}

	// Cleanup
	collector.Stop()

	t.Log("✓ StatsCollector created successfully")
}

func TestRecordQuery(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	query := "SELECT * FROM users WHERE id = ?"
	collector.RecordQuery(query, 100*time.Millisecond)

	stats := collector.GetStatistics()
	if len(stats) != 1 {
		t.Fatalf("Expected 1 query statistic, got %d", len(stats))
	}

	queryHash := calculateQueryHash(query)
	queryStat := stats[queryHash]

	if queryStat == nil {
		t.Fatal("Expected query statistic to exist")
	}

	if queryStat.Query != query {
		t.Errorf("Expected query '%s', got '%s'", query, queryStat.Query)
	}

	if queryStat.ExecutionCount != 1 {
		t.Errorf("Expected execution count 1, got %d", queryStat.ExecutionCount)
	}

	if queryStat.TotalTime != 100*time.Millisecond {
		t.Errorf("Expected total time 100ms, got %v", queryStat.TotalTime)
	}

	if queryStat.MeanTime != 100*time.Millisecond {
		t.Errorf("Expected mean time 100ms, got %v", queryStat.MeanTime)
	}

	t.Log("✓ Query recording works correctly")
}

func TestRecordMultipleExecutions(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	query := "SELECT * FROM users WHERE email = ?"

	// Record multiple executions
	collector.RecordQuery(query, 100*time.Millisecond)
	collector.RecordQuery(query, 200*time.Millisecond)
	collector.RecordQuery(query, 300*time.Millisecond)

	queryHash := calculateQueryHash(query)
	queryStat := collector.GetQueryStatistics(queryHash)

	if queryStat == nil {
		t.Fatal("Expected query statistic to exist")
	}

	if queryStat.ExecutionCount != 3 {
		t.Errorf("Expected execution count 3, got %d", queryStat.ExecutionCount)
	}

	expectedTotal := 600 * time.Millisecond
	if queryStat.TotalTime != expectedTotal {
		t.Errorf("Expected total time %v, got %v", expectedTotal, queryStat.TotalTime)
	}

	expectedMean := 200 * time.Millisecond
	if queryStat.MeanTime != expectedMean {
		t.Errorf("Expected mean time %v, got %v", expectedMean, queryStat.MeanTime)
	}

	if queryStat.MinTime != 100*time.Millisecond {
		t.Errorf("Expected min time 100ms, got %v", queryStat.MinTime)
	}

	if queryStat.MaxTime != 300*time.Millisecond {
		t.Errorf("Expected max time 300ms, got %v", queryStat.MaxTime)
	}

	t.Log("✓ Multiple execution recording works correctly")
}

func TestRecordQueryWithDetails(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	query := "SELECT * FROM posts WHERE user_id = ?"
	collector.RecordQueryWithDetails(query, 150*time.Millisecond, 10, 0)
	collector.RecordQueryWithDetails(query, 200*time.Millisecond, 15, 0)

	queryHash := calculateQueryHash(query)
	queryStat := collector.GetQueryStatistics(queryHash)

	if queryStat == nil {
		t.Fatal("Expected query statistic to exist")
	}

	if queryStat.RowsReturned != 25 {
		t.Errorf("Expected rows returned 25, got %d", queryStat.RowsReturned)
	}

	if queryStat.ExecutionCount != 2 {
		t.Errorf("Expected execution count 2, got %d", queryStat.ExecutionCount)
	}

	t.Log("✓ Query recording with details works correctly")
}

func TestGetTopQueries(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Record queries with different total times
	collector.RecordQuery("SELECT * FROM query1", 1*time.Second)
	collector.RecordQuery("SELECT * FROM query1", 1*time.Second) // Total: 2s

	collector.RecordQuery("SELECT * FROM query2", 500*time.Millisecond)
	collector.RecordQuery("SELECT * FROM query2", 500*time.Millisecond)
	collector.RecordQuery("SELECT * FROM query2", 500*time.Millisecond) // Total: 1.5s

	collector.RecordQuery("SELECT * FROM query3", 100*time.Millisecond) // Total: 100ms

	topQueries := collector.GetTopQueries(2)

	if len(topQueries) != 2 {
		t.Errorf("Expected 2 top queries, got %d", len(topQueries))
	}

	// First should have highest total time
	if topQueries[0].TotalTime != 2*time.Second {
		t.Errorf("Expected first query total time 2s, got %v", topQueries[0].TotalTime)
	}

	// Second should have second highest
	if topQueries[1].TotalTime != 1500*time.Millisecond {
		t.Errorf("Expected second query total time 1.5s, got %v", topQueries[1].TotalTime)
	}

	t.Log("✓ Top queries retrieval works correctly")
}

func TestGetSlowestQueries(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Record queries with different average times
	collector.RecordQuery("SELECT * FROM slow", 1*time.Second)
	collector.RecordQuery("SELECT * FROM medium", 500*time.Millisecond)
	collector.RecordQuery("SELECT * FROM fast", 100*time.Millisecond)

	slowest := collector.GetSlowestQueries(2)

	if len(slowest) != 2 {
		t.Errorf("Expected 2 slowest queries, got %d", len(slowest))
	}

	// First should have highest mean time
	if slowest[0].MeanTime != 1*time.Second {
		t.Errorf("Expected first query mean time 1s, got %v", slowest[0].MeanTime)
	}

	t.Log("✓ Slowest queries retrieval works correctly")
}

func TestGetMostFrequentQueries(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Record queries with different frequencies
	frequent := "SELECT * FROM frequent"
	medium := "SELECT * FROM medium"
	rare := "SELECT * FROM rare"

	for i := 0; i < 10; i++ {
		collector.RecordQuery(frequent, 100*time.Millisecond)
	}

	for i := 0; i < 5; i++ {
		collector.RecordQuery(medium, 100*time.Millisecond)
	}

	collector.RecordQuery(rare, 100*time.Millisecond)

	mostFrequent := collector.GetMostFrequentQueries(2)

	if len(mostFrequent) != 2 {
		t.Errorf("Expected 2 most frequent queries, got %d", len(mostFrequent))
	}

	// First should be most frequent
	if mostFrequent[0].ExecutionCount != 10 {
		t.Errorf("Expected first query execution count 10, got %d", mostFrequent[0].ExecutionCount)
	}

	// Second should have second highest count
	if mostFrequent[1].ExecutionCount != 5 {
		t.Errorf("Expected second query execution count 5, got %d", mostFrequent[1].ExecutionCount)
	}

	t.Log("✓ Most frequent queries retrieval works correctly")
}

func TestClearStatistics(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Add some data
	collector.RecordQuery("SELECT * FROM test", 100*time.Millisecond)
	collector.RecordQuery("SELECT * FROM another", 200*time.Millisecond)

	totalQueries := collector.GetTotalQueries()
	if totalQueries != 2 {
		t.Errorf("Expected 2 total queries, got %d", totalQueries)
	}

	// Clear
	collector.Clear()

	totalQueries = collector.GetTotalQueries()
	if totalQueries != 0 {
		t.Errorf("Expected 0 total queries after clear, got %d", totalQueries)
	}

	stats := collector.GetStatistics()
	if len(stats) != 0 {
		t.Errorf("Expected empty statistics after clear, got %d", len(stats))
	}

	t.Log("✓ Statistics clearing works correctly")
}

func TestGetTotalExecutions(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Record multiple executions across different queries
	collector.RecordQuery("SELECT * FROM query1", 100*time.Millisecond)
	collector.RecordQuery("SELECT * FROM query1", 100*time.Millisecond)
	collector.RecordQuery("SELECT * FROM query1", 100*time.Millisecond)

	collector.RecordQuery("SELECT * FROM query2", 100*time.Millisecond)
	collector.RecordQuery("SELECT * FROM query2", 100*time.Millisecond)

	totalExecutions := collector.GetTotalExecutions()
	if totalExecutions != 5 {
		t.Errorf("Expected 5 total executions, got %d", totalExecutions)
	}

	t.Log("✓ Total executions counting works correctly")
}

func TestStatsCollectorConcurrency(t *testing.T) {
	collector := NewStatsCollector(24 * time.Hour)
	defer collector.Stop()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.RecordQuery("SELECT * FROM concurrent_test", 100*time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	totalExecutions := collector.GetTotalExecutions()
	if totalExecutions != 1000 {
		t.Errorf("Expected 1000 total executions, got %d", totalExecutions)
	}

	t.Log("✓ Concurrent statistics collection works correctly")
}
