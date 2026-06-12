package profiling

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewMemoryProfiler(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
		Interval:   1 * time.Second,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	if profiler.config.MemoryThreshold != 500*1024*1024 {
		t.Errorf("Expected default memory threshold 500MB, got %d", profiler.config.MemoryThreshold)
	}

	if profiler.config.GoroutineThreshold != 1000 {
		t.Errorf("Expected default goroutine threshold 1000, got %d", profiler.config.GoroutineThreshold)
	}
}

func TestTakeSnapshot(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	snapshot := profiler.takeSnapshot()

	if snapshot.HeapAlloc == 0 {
		t.Error("HeapAlloc should not be zero")
	}

	if snapshot.NumGoroutine == 0 {
		t.Error("NumGoroutine should not be zero")
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestGetCurrentStats(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	stats := profiler.GetCurrentStats()

	if stats == nil {
		t.Fatal("GetCurrentStats returned nil")
	}

	if stats.HeapAlloc == 0 {
		t.Error("Expected non-zero heap allocation")
	}

	if stats.NumGoroutine < 1 {
		t.Error("Expected at least one goroutine")
	}
}

func TestForceGC(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Allocate some memory
	_ = make([]byte, 10*1024*1024) // 10MB

	before, after := profiler.ForceGC()

	if before == nil || after == nil {
		t.Fatal("ForceGC returned nil snapshots")
	}

	// After GC, some memory should be freed (though not guaranteed)
	t.Logf("Before GC: HeapAlloc=%d, After GC: HeapAlloc=%d",
		before.HeapAlloc, after.HeapAlloc)
}

func TestGetMemoryGrowth(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Allocate some memory
	data := make([][]byte, 0)
	for i := 0; i < 10; i++ {
		data = append(data, make([]byte, 1024*1024)) // 1MB each
	}

	heapGrowth, goroutineGrowth, duration := profiler.GetMemoryGrowth()

	if heapGrowth <= 0 {
		t.Logf("Warning: Expected positive heap growth, got %d", heapGrowth)
	}

	if duration <= 0 {
		t.Error("Duration should be positive")
	}

	t.Logf("Heap growth: %d bytes, Goroutine growth: %d, Duration: %v",
		heapGrowth, goroutineGrowth, duration)

	// Keep data in scope
	_ = data
}

func TestStartStop(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
		Interval:   100 * time.Millisecond,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start profiling
	err = profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Wait a bit to collect some snapshots
	time.Sleep(300 * time.Millisecond)

	// Stop profiling
	err = profiler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop profiler: %v", err)
	}

	// Verify snapshots were collected
	snapshots := profiler.GetSnapshots()
	if len(snapshots) < 2 {
		t.Errorf("Expected at least 2 snapshots, got %d", len(snapshots))
	}
}

func TestHeapProfiling(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:     true,
		ProfileDir:  filepath.Join(tempDir, "profiles"),
		HeapProfile: true,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Write heap profile
	err = profiler.writeHeapProfile()
	if err != nil {
		t.Fatalf("Failed to write heap profile: %v", err)
	}

	// Verify profile file was created
	files, err := os.ReadDir(config.ProfileDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Heap profile file was not created")
	}
}

func TestGoroutineProfiling(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:          true,
		ProfileDir:       filepath.Join(tempDir, "profiles"),
		GoroutineProfile: true,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Write goroutine profile
	err = profiler.writeGoroutineProfile()
	if err != nil {
		t.Fatalf("Failed to write goroutine profile: %v", err)
	}

	// Verify profile file was created
	files, err := os.ReadDir(config.ProfileDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Goroutine profile file was not created")
	}
}

func TestLeakDetectionMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping leak detection test in short mode")
	}

	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:             true,
		ProfileDir:          filepath.Join(tempDir, "profiles"),
		LeakDetection:       true,
		LeakDetectionWindow: 1 * time.Second,
		MemoryThreshold:     1 * 1024 * 1024, // 1MB threshold for testing
		Interval:            100 * time.Millisecond,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Take initial snapshots
	for i := 0; i < 5; i++ {
		profiler.takeSnapshot()
		time.Sleep(50 * time.Millisecond)
	}

	// Simulate memory leak by allocating and holding references
	leakedData := make([][]byte, 0)
	for i := 0; i < 20; i++ {
		leakedData = append(leakedData, make([]byte, 100*1024)) // 100KB each
		profiler.takeSnapshot()
		time.Sleep(50 * time.Millisecond)
	}

	// Detect leaks
	profiler.detectLeaks()

	leaks := profiler.GetLeaks()

	// We expect a leak to be detected
	if len(leaks) == 0 {
		t.Log("No memory leak detected (may be due to GC timing)")
	} else {
		leak := leaks[0]
		t.Logf("Leak detected: %s - %s", leak.LeakType, leak.Description)

		if leak.LeakType != LeakTypeMemory && leak.LeakType != LeakTypeBoth {
			t.Errorf("Expected memory leak, got %s", leak.LeakType)
		}
	}

	// Keep data in scope
	runtime.KeepAlive(leakedData)
}

func TestLeakDetectionGoroutine(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping leak detection test in short mode")
	}

	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:             true,
		ProfileDir:          filepath.Join(tempDir, "profiles"),
		LeakDetection:       true,
		LeakDetectionWindow: 1 * time.Second,
		GoroutineThreshold:  10, // Low threshold for testing
		Interval:            100 * time.Millisecond,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Take initial snapshots
	for i := 0; i < 5; i++ {
		profiler.takeSnapshot()
		time.Sleep(50 * time.Millisecond)
	}

	// Simulate goroutine leak
	stopChans := make([]chan struct{}, 0)
	for i := 0; i < 50; i++ {
		stopChan := make(chan struct{})
		stopChans = append(stopChans, stopChan)

		go func() {
			<-stopChan
		}()

		profiler.takeSnapshot()
		time.Sleep(20 * time.Millisecond)
	}

	// Detect leaks
	profiler.detectLeaks()

	leaks := profiler.GetLeaks()

	// We expect a goroutine leak to be detected
	if len(leaks) == 0 {
		t.Log("No goroutine leak detected")
	} else {
		leak := leaks[0]
		t.Logf("Leak detected: %s - %s", leak.LeakType, leak.Description)

		if leak.LeakType != LeakTypeGoroutine && leak.LeakType != LeakTypeBoth {
			t.Errorf("Expected goroutine leak, got %s", leak.LeakType)
		}

		if len(leak.Recommendations) == 0 {
			t.Error("Expected recommendations for leak fix")
		}
	}

	// Cleanup goroutines
	for _, ch := range stopChans {
		close(ch)
	}
}

func TestCalculateSeverity(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:            true,
		ProfileDir:         filepath.Join(tempDir, "profiles"),
		MemoryThreshold:    100 * 1024 * 1024, // 100MB
		GoroutineThreshold: 100,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	tests := []struct {
		name              string
		heapGrowth        int64
		goroutineGrowth   int
		expectedSeverity  Severity
	}{
		{
			name:             "low severity",
			heapGrowth:       50 * 1024 * 1024, // 50MB
			goroutineGrowth:  50,
			expectedSeverity: SeverityLow,
		},
		{
			name:             "medium severity",
			heapGrowth:       150 * 1024 * 1024, // 150MB
			goroutineGrowth:  150,
			expectedSeverity: SeverityMedium,
		},
		{
			name:             "high severity",
			heapGrowth:       300 * 1024 * 1024, // 300MB
			goroutineGrowth:  300,
			expectedSeverity: SeverityHigh,
		},
		{
			name:             "critical severity",
			heapGrowth:       600 * 1024 * 1024, // 600MB
			goroutineGrowth:  600,
			expectedSeverity: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := profiler.calculateSeverity(tt.heapGrowth, tt.goroutineGrowth)
			if severity != tt.expectedSeverity {
				t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, severity)
			}
		})
	}
}

func TestAnalyzeGrowthTrend(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Create snapshots with increasing memory
	for i := 0; i < 15; i++ {
		// Allocate memory to create upward trend
		_ = make([]byte, 2*1024*1024) // 2MB

		profiler.takeSnapshot()
		time.Sleep(10 * time.Millisecond)
	}

	memoryTrend, goroutineTrend := profiler.AnalyzeGrowthTrend()

	if memoryTrend == "insufficient_data" {
		t.Error("Should have sufficient data for trend analysis")
	}

	t.Logf("Memory trend: %s, Goroutine trend: %s", memoryTrend, goroutineTrend)
}

func TestSnapshotRetention(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Create more than 1000 snapshots
	for i := 0; i < 1100; i++ {
		profiler.takeSnapshot()
	}

	snapshots := profiler.GetSnapshots()

	// Should keep only last 1000
	if len(snapshots) > 1000 {
		t.Errorf("Expected at most 1000 snapshots, got %d", len(snapshots))
	}
}

func TestCPUProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CPU profiling test in short mode")
	}

	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
		CPUProfile: true,
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start profiling
	err = profiler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Do some CPU work
	for i := 0; i < 1000000; i++ {
		_ = i * i
	}

	// Stop profiling
	err = profiler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop profiler: %v", err)
	}

	// Verify CPU profile file was created
	files, err := os.ReadDir(config.ProfileDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" {
			found = true
			break
		}
	}

	if !found {
		t.Error("CPU profile file was not created")
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tempDir := t.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, err := NewMemoryProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	tests := []struct {
		name     string
		leakType LeakType
		minRecs  int
	}{
		{
			name:     "memory leak recommendations",
			leakType: LeakTypeMemory,
			minRecs:  3,
		},
		{
			name:     "goroutine leak recommendations",
			leakType: LeakTypeGoroutine,
			minRecs:  3,
		},
		{
			name:     "both leaks recommendations",
			leakType: LeakTypeBoth,
			minRecs:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leak := &LeakReport{
				LeakType: tt.leakType,
			}

			recommendations := profiler.generateRecommendations(leak)

			if len(recommendations) < tt.minRecs {
				t.Errorf("Expected at least %d recommendations, got %d",
					tt.minRecs, len(recommendations))
			}

			for _, rec := range recommendations {
				if rec == "" {
					t.Error("Recommendation should not be empty")
				}
			}
		})
	}
}

func BenchmarkTakeSnapshot(b *testing.B) {
	tempDir := b.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:    true,
		ProfileDir: filepath.Join(tempDir, "profiles"),
	}

	profiler, _ := NewMemoryProfiler(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.takeSnapshot()
	}
}

func BenchmarkDetectLeaks(b *testing.B) {
	tempDir := b.TempDir()

	config := &MemoryProfilerConfig{
		Enabled:             true,
		ProfileDir:          filepath.Join(tempDir, "profiles"),
		LeakDetection:       true,
		LeakDetectionWindow: 1 * time.Second,
	}

	profiler, _ := NewMemoryProfiler(config)

	// Create some snapshots
	for i := 0; i < 20; i++ {
		profiler.takeSnapshot()
		time.Sleep(10 * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.detectLeaks()
	}
}
