package profiling

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// MemoryProfilerConfig contains configuration for memory profiling
type MemoryProfilerConfig struct {
	// Enable profiling
	Enabled bool `mapstructure:"enabled"`

	// Output directory for profiles
	ProfileDir string `mapstructure:"profile_dir"`

	// Profiling interval
	Interval time.Duration `mapstructure:"interval"`

	// Enable heap profiling
	HeapProfile bool `mapstructure:"heap_profile"`

	// Enable goroutine profiling
	GoroutineProfile bool `mapstructure:"goroutine_profile"`

	// Enable allocation profiling
	AllocProfile bool `mapstructure:"alloc_profile"`

	// Enable CPU profiling
	CPUProfile bool `mapstructure:"cpu_profile"`

	// Enable execution trace
	ExecutionTrace bool `mapstructure:"execution_trace"`

	// Memory threshold for alerts (in bytes)
	MemoryThreshold int64 `mapstructure:"memory_threshold"`

	// Goroutine threshold for alerts
	GoroutineThreshold int `mapstructure:"goroutine_threshold"`

	// Enable automatic leak detection
	LeakDetection bool `mapstructure:"leak_detection"`

	// Leak detection window
	LeakDetectionWindow time.Duration `mapstructure:"leak_detection_window"`
}

// MemoryProfiler provides memory profiling and leak detection
type MemoryProfiler struct {
	config    *MemoryProfilerConfig
	mu        sync.RWMutex

	// Baseline measurements
	baselineHeap      uint64
	baselineGoroutines int
	baselineTimestamp  time.Time

	// Tracking
	snapshots     []*MemorySnapshot
	leaks         []*LeakReport
	cpuProfileFile *os.File
	traceFile      *os.File

	// Control
	stopChan      chan struct{}
	running       bool
}

// MemorySnapshot represents a point-in-time memory snapshot
type MemorySnapshot struct {
	Timestamp      time.Time
	HeapAlloc      uint64
	HeapSys        uint64
	HeapIdle       uint64
	HeapInuse      uint64
	HeapReleased   uint64
	HeapObjects    uint64
	StackInuse     uint64
	StackSys       uint64
	NumGoroutine   int
	NumCgoCall     int64
	GCCycles       uint32
	NextGC         uint64
	LastGC         time.Time
}

// LeakReport contains information about a detected leak
type LeakReport struct {
	DetectedAt     time.Time
	LeakType       LeakType
	Description    string
	Severity       Severity
	HeapGrowth     int64 // bytes
	GoroutineGrowth int
	Recommendations []string
}

// LeakType represents the type of memory leak
type LeakType string

const (
	LeakTypeMemory     LeakType = "memory"
	LeakTypeGoroutine  LeakType = "goroutine"
	LeakTypeBoth       LeakType = "both"
)

// Severity represents leak severity
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler(config *MemoryProfilerConfig) (*MemoryProfiler, error) {
	if config == nil {
		return nil, fmt.Errorf("profiler config is required")
	}

	// Set defaults
	if config.ProfileDir == "" {
		config.ProfileDir = "/var/lib/db-backup/profiles"
	}
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.MemoryThreshold == 0 {
		config.MemoryThreshold = 500 * 1024 * 1024 // 500MB
	}
	if config.GoroutineThreshold == 0 {
		config.GoroutineThreshold = 1000
	}
	if config.LeakDetectionWindow == 0 {
		config.LeakDetectionWindow = 5 * time.Minute
	}

	// Create profile directory
	if err := os.MkdirAll(config.ProfileDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}

	profiler := &MemoryProfiler{
		config:     config,
		snapshots:  make([]*MemorySnapshot, 0),
		leaks:      make([]*LeakReport, 0),
		stopChan:   make(chan struct{}),
	}

	// Take baseline snapshot
	profiler.takeBaseline()

	return profiler, nil
}

// Start starts the memory profiler
func (p *MemoryProfiler) Start(ctx context.Context) error {
	if !p.config.Enabled {
		return nil
	}

	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("profiler is already running")
	}
	p.running = true
	p.mu.Unlock()

	// Start CPU profiling if enabled
	if p.config.CPUProfile {
		if err := p.startCPUProfile(); err != nil {
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}
	}

	// Start execution trace if enabled
	if p.config.ExecutionTrace {
		if err := p.startTrace(); err != nil {
			return fmt.Errorf("failed to start trace: %w", err)
		}
	}

	// Start monitoring goroutine
	go p.monitorLoop(ctx)

	return nil
}

// Stop stops the memory profiler
func (p *MemoryProfiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	close(p.stopChan)
	p.running = false

	// Stop CPU profiling
	if p.cpuProfileFile != nil {
		pprof.StopCPUProfile()
		p.cpuProfileFile.Close()
		p.cpuProfileFile = nil
	}

	// Stop trace
	if p.traceFile != nil {
		trace.Stop()
		p.traceFile.Close()
		p.traceFile = nil
	}

	return nil
}

// monitorLoop performs periodic memory monitoring
func (p *MemoryProfiler) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.takeSnapshot()

			// Write profiles if enabled
			if p.config.HeapProfile {
				p.writeHeapProfile()
			}
			if p.config.GoroutineProfile {
				p.writeGoroutineProfile()
			}
			if p.config.AllocProfile {
				p.writeAllocProfile()
			}

			// Check for leaks if enabled
			if p.config.LeakDetection {
				p.detectLeaks()
			}
		}
	}
}

// takeBaseline takes a baseline memory snapshot
func (p *MemoryProfiler) takeBaseline() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.baselineHeap = m.HeapAlloc
	p.baselineGoroutines = runtime.NumGoroutine()
	p.baselineTimestamp = time.Now()
}

// takeSnapshot takes a memory snapshot
func (p *MemoryProfiler) takeSnapshot() *MemorySnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := &MemorySnapshot{
		Timestamp:    time.Now(),
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		HeapObjects:  m.HeapObjects,
		StackInuse:   m.StackInuse,
		StackSys:     m.StackSys,
		NumGoroutine: runtime.NumGoroutine(),
		NumCgoCall:   runtime.NumCgoCall(),
		GCCycles:     m.NumGC,
		NextGC:       m.NextGC,
		LastGC:       time.Unix(0, int64(m.LastGC)),
	}

	p.mu.Lock()
	p.snapshots = append(p.snapshots, snapshot)

	// Keep only last 1000 snapshots
	if len(p.snapshots) > 1000 {
		p.snapshots = p.snapshots[len(p.snapshots)-1000:]
	}
	p.mu.Unlock()

	return snapshot
}

// detectLeaks analyzes snapshots to detect memory leaks
func (p *MemoryProfiler) detectLeaks() {
	p.mu.RLock()
	if len(p.snapshots) < 2 {
		p.mu.RUnlock()
		return
	}

	// Get snapshots from the detection window
	windowStart := time.Now().Add(-p.config.LeakDetectionWindow)
	windowSnapshots := make([]*MemorySnapshot, 0)

	for _, snap := range p.snapshots {
		if snap.Timestamp.After(windowStart) {
			windowSnapshots = append(windowSnapshots, snap)
		}
	}
	p.mu.RUnlock()

	if len(windowSnapshots) < 2 {
		return
	}

	// Calculate growth rates
	firstSnap := windowSnapshots[0]
	lastSnap := windowSnapshots[len(windowSnapshots)-1]

	heapGrowth := int64(lastSnap.HeapAlloc) - int64(firstSnap.HeapAlloc)
	goroutineGrowth := lastSnap.NumGoroutine - firstSnap.NumGoroutine

	// Check for memory leak
	memoryLeak := heapGrowth > int64(p.config.MemoryThreshold)
	goroutineLeak := goroutineGrowth > p.config.GoroutineThreshold

	if memoryLeak || goroutineLeak {
		leak := &LeakReport{
			DetectedAt:      time.Now(),
			HeapGrowth:      heapGrowth,
			GoroutineGrowth: goroutineGrowth,
		}

		if memoryLeak && goroutineLeak {
			leak.LeakType = LeakTypeBoth
			leak.Description = fmt.Sprintf("Both memory and goroutine leaks detected. Heap grew by %d MB, goroutines grew by %d",
				heapGrowth/(1024*1024), goroutineGrowth)
			leak.Severity = SeverityCritical
		} else if memoryLeak {
			leak.LeakType = LeakTypeMemory
			leak.Description = fmt.Sprintf("Memory leak detected. Heap grew by %d MB in %v",
				heapGrowth/(1024*1024), p.config.LeakDetectionWindow)
			leak.Severity = p.calculateSeverity(heapGrowth, 0)
		} else {
			leak.LeakType = LeakTypeGoroutine
			leak.Description = fmt.Sprintf("Goroutine leak detected. Count grew by %d in %v",
				goroutineGrowth, p.config.LeakDetectionWindow)
			leak.Severity = p.calculateSeverity(0, goroutineGrowth)
		}

		leak.Recommendations = p.generateRecommendations(leak)

		p.mu.Lock()
		p.leaks = append(p.leaks, leak)
		p.mu.Unlock()

		// Log the leak
		fmt.Printf("[LEAK DETECTED] %s - %s\n", leak.Severity, leak.Description)
		for _, rec := range leak.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}
}

// calculateSeverity calculates leak severity
func (p *MemoryProfiler) calculateSeverity(heapGrowth int64, goroutineGrowth int) Severity {
	if heapGrowth > int64(p.config.MemoryThreshold)*5 || goroutineGrowth > p.config.GoroutineThreshold*5 {
		return SeverityCritical
	}
	if heapGrowth > int64(p.config.MemoryThreshold)*2 || goroutineGrowth > p.config.GoroutineThreshold*2 {
		return SeverityHigh
	}
	if heapGrowth > int64(p.config.MemoryThreshold) || goroutineGrowth > p.config.GoroutineThreshold {
		return SeverityMedium
	}
	return SeverityLow
}

// generateRecommendations generates leak fix recommendations
func (p *MemoryProfiler) generateRecommendations(leak *LeakReport) []string {
	recommendations := make([]string, 0)

	switch leak.LeakType {
	case LeakTypeMemory:
		recommendations = append(recommendations,
			"Review heap profile to identify allocation hotspots",
			"Check for unbounded slice/map growth",
			"Verify proper closure of resources (files, connections)",
			"Look for circular references preventing GC",
		)
	case LeakTypeGoroutine:
		recommendations = append(recommendations,
			"Review goroutine profile to identify leak sources",
			"Check for goroutines waiting on never-closing channels",
			"Verify context cancellation is properly propagated",
			"Look for missing return statements in goroutines",
		)
	case LeakTypeBoth:
		recommendations = append(recommendations,
			"CRITICAL: Both memory and goroutine leaks detected",
			"Review both heap and goroutine profiles",
			"Check for goroutines holding references to large data structures",
			"Verify proper cleanup in defer statements",
			"Consider immediate service restart if leak is severe",
		)
	}

	return recommendations
}

// writeHeapProfile writes a heap profile to disk
func (p *MemoryProfiler) writeHeapProfile() error {
	filename := filepath.Join(p.config.ProfileDir,
		fmt.Sprintf("heap-%d.prof", time.Now().Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.GC() // Get up-to-date statistics
	return pprof.WriteHeapProfile(f)
}

// writeGoroutineProfile writes a goroutine profile to disk
func (p *MemoryProfiler) writeGoroutineProfile() error {
	filename := filepath.Join(p.config.ProfileDir,
		fmt.Sprintf("goroutine-%d.prof", time.Now().Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.Lookup("goroutine").WriteTo(f, 0)
}

// writeAllocProfile writes an allocation profile to disk
func (p *MemoryProfiler) writeAllocProfile() error {
	filename := filepath.Join(p.config.ProfileDir,
		fmt.Sprintf("allocs-%d.prof", time.Now().Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.Lookup("allocs").WriteTo(f, 0)
}

// startCPUProfile starts CPU profiling
func (p *MemoryProfiler) startCPUProfile() error {
	filename := filepath.Join(p.config.ProfileDir,
		fmt.Sprintf("cpu-%d.prof", time.Now().Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	p.cpuProfileFile = f
	return pprof.StartCPUProfile(f)
}

// startTrace starts execution trace
func (p *MemoryProfiler) startTrace() error {
	filename := filepath.Join(p.config.ProfileDir,
		fmt.Sprintf("trace-%d.out", time.Now().Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	p.traceFile = f
	return trace.Start(f)
}

// GetSnapshots returns all memory snapshots
func (p *MemoryProfiler) GetSnapshots() []*MemorySnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	snapshots := make([]*MemorySnapshot, len(p.snapshots))
	copy(snapshots, p.snapshots)
	return snapshots
}

// GetLeaks returns all detected leaks
func (p *MemoryProfiler) GetLeaks() []*LeakReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	leaks := make([]*LeakReport, len(p.leaks))
	copy(leaks, p.leaks)
	return leaks
}

// GetCurrentStats returns current memory statistics
func (p *MemoryProfiler) GetCurrentStats() *MemorySnapshot {
	return p.takeSnapshot()
}

// ForceGC forces a garbage collection and returns before/after stats
func (p *MemoryProfiler) ForceGC() (before, after *MemorySnapshot) {
	before = p.takeSnapshot()
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Let GC complete
	after = p.takeSnapshot()
	return
}

// GetMemoryGrowth returns memory growth since baseline
func (p *MemoryProfiler) GetMemoryGrowth() (heapGrowth int64, goroutineGrowth int, duration time.Duration) {
	// takeSnapshot acquires mu.Lock internally, so snapshot must be taken before acquiring RLock
	current := p.takeSnapshot()

	p.mu.RLock()
	defer p.mu.RUnlock()

	heapGrowth = int64(current.HeapAlloc) - int64(p.baselineHeap)
	goroutineGrowth = current.NumGoroutine - p.baselineGoroutines
	duration = time.Since(p.baselineTimestamp)

	return
}

// AnalyzeGrowthTrend analyzes if memory/goroutines are trending upward
func (p *MemoryProfiler) AnalyzeGrowthTrend() (memoryTrend, goroutineTrend string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.snapshots) < 10 {
		return "insufficient_data", "insufficient_data"
	}

	// Take last 10 snapshots
	recent := p.snapshots[len(p.snapshots)-10:]

	// Calculate linear trend
	memorySlope := p.calculateSlope(recent, func(s *MemorySnapshot) float64 {
		return float64(s.HeapAlloc)
	})

	goroutineSlope := p.calculateSlope(recent, func(s *MemorySnapshot) float64 {
		return float64(s.NumGoroutine)
	})

	memoryTrend = p.classifyTrend(memorySlope)
	goroutineTrend = p.classifyTrend(goroutineSlope)

	return
}

// calculateSlope calculates the slope of a metric over time
func (p *MemoryProfiler) calculateSlope(snapshots []*MemorySnapshot, getValue func(*MemorySnapshot) float64) float64 {
	if len(snapshots) < 2 {
		return 0
	}

	n := float64(len(snapshots))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, snap := range snapshots {
		x := float64(i)
		y := getValue(snap)

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Linear regression slope: (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	return slope
}

// classifyTrend classifies the trend based on slope
func (p *MemoryProfiler) classifyTrend(slope float64) string {
	if slope > 1000000 { // >1MB per snapshot
		return "rapidly_increasing"
	} else if slope > 100000 { // >100KB per snapshot
		return "increasing"
	} else if slope > -100000 && slope < 100000 {
		return "stable"
	} else if slope < -100000 {
		return "decreasing"
	}
	return "stable"
}
