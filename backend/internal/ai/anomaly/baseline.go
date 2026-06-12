// Package anomaly provides AI-powered anomaly detection for backup operations
package anomaly

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// BackupMetric represents metrics for a single backup operation
type BackupMetric struct {
	BackupID     string
	DatabaseName string
	Timestamp    time.Time
	Size         int64   // Backup size in bytes
	Duration     float64 // Duration in seconds
	Success      bool
	ErrorMessage string
	CompressionRatio float64
	FilesProcessed int
}

// BaselineModel represents statistical baseline for backup operations
type BaselineModel struct {
	DatabaseName string

	// Size statistics
	SizeMean   float64
	SizeStdDev float64
	SizeMin    float64
	SizeMax    float64
	SizeP50    float64 // Median
	SizeP95    float64
	SizeP99    float64

	// Duration statistics
	DurationMean   float64
	DurationStdDev float64
	DurationMin    float64
	DurationMax    float64
	DurationP50    float64
	DurationP95    float64
	DurationP99    float64

	// Success rate
	SuccessRate float64
	TotalBackups int
	SuccessfulBackups int
	FailedBackups int

	// Temporal patterns
	HourlyDistribution [24]int // Distribution by hour of day
	DayOfWeekDistribution [7]int // Distribution by day of week

	// Training metadata
	TrainingStartDate time.Time
	TrainingEndDate   time.Time
	LastUpdated       time.Time
	SampleCount       int
}

// BaselineTrainer trains baseline models from historical data
type BaselineTrainer struct {
	mu sync.RWMutex
	models map[string]*BaselineModel // key: database name
	metrics []BackupMetric

	// Configuration
	minSamples int
	windowDays int
}

// NewBaselineTrainer creates a new baseline trainer
func NewBaselineTrainer(windowDays int) *BaselineTrainer {
	if windowDays <= 0 {
		windowDays = 30 // Default to 30 days
	}

	return &BaselineTrainer{
		models:     make(map[string]*BaselineModel),
		metrics:    make([]BackupMetric, 0),
		minSamples: 10, // Minimum samples required for training
		windowDays: windowDays,
	}
}

// AddMetric adds a backup metric to the training dataset
func (bt *BaselineTrainer) AddMetric(metric BackupMetric) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.metrics = append(bt.metrics, metric)
}

// AddMetrics adds multiple backup metrics
func (bt *BaselineTrainer) AddMetrics(metrics []BackupMetric) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.metrics = append(bt.metrics, metrics...)
}

// Train trains baseline models for all databases
func (bt *BaselineTrainer) Train(ctx context.Context) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Filter metrics within window
	cutoffTime := time.Now().AddDate(0, 0, -bt.windowDays)
	filteredMetrics := make([]BackupMetric, 0)

	for _, metric := range bt.metrics {
		if metric.Timestamp.After(cutoffTime) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	if len(filteredMetrics) < bt.minSamples {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"insufficient data for training: need at least 10 samples")
	}

	// Group metrics by database
	dbMetrics := make(map[string][]BackupMetric)
	for _, metric := range filteredMetrics {
		dbMetrics[metric.DatabaseName] = append(dbMetrics[metric.DatabaseName], metric)
	}

	// Train model for each database
	for dbName, metrics := range dbMetrics {
		if len(metrics) < bt.minSamples {
			continue // Skip databases with insufficient data
		}

		model, err := bt.trainDatabase(dbName, metrics)
		if err != nil {
			return err
		}

		bt.models[dbName] = model
	}

	return nil
}

// trainDatabase trains a baseline model for a specific database
func (bt *BaselineTrainer) trainDatabase(dbName string, metrics []BackupMetric) (*BaselineModel, error) {
	if len(metrics) == 0 {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeValidation, "no metrics for database")
	}

	model := &BaselineModel{
		DatabaseName: dbName,
		SampleCount:  len(metrics),
		LastUpdated:  time.Now(),
	}

	// Extract size and duration values
	sizes := make([]float64, 0, len(metrics))
	durations := make([]float64, 0, len(metrics))
	successCount := 0

	for _, m := range metrics {
		sizes = append(sizes, float64(m.Size))
		durations = append(durations, m.Duration)

		if m.Success {
			successCount++
		}

		// Populate temporal distributions
		hour := m.Timestamp.Hour()
		model.HourlyDistribution[hour]++

		dayOfWeek := int(m.Timestamp.Weekday())
		model.DayOfWeekDistribution[dayOfWeek]++

		// Track date range
		if model.TrainingStartDate.IsZero() || m.Timestamp.Before(model.TrainingStartDate) {
			model.TrainingStartDate = m.Timestamp
		}
		if m.Timestamp.After(model.TrainingEndDate) {
			model.TrainingEndDate = m.Timestamp
		}
	}

	// Calculate size statistics
	model.SizeMean = mean(sizes)
	model.SizeStdDev = stdDev(sizes, model.SizeMean)
	model.SizeMin = min(sizes)
	model.SizeMax = max(sizes)
	model.SizeP50 = percentile(sizes, 0.50)
	model.SizeP95 = percentile(sizes, 0.95)
	model.SizeP99 = percentile(sizes, 0.99)

	// Calculate duration statistics
	model.DurationMean = mean(durations)
	model.DurationStdDev = stdDev(durations, model.DurationMean)
	model.DurationMin = min(durations)
	model.DurationMax = max(durations)
	model.DurationP50 = percentile(durations, 0.50)
	model.DurationP95 = percentile(durations, 0.95)
	model.DurationP99 = percentile(durations, 0.99)

	// Calculate success rate
	model.TotalBackups = len(metrics)
	model.SuccessfulBackups = successCount
	model.FailedBackups = len(metrics) - successCount
	model.SuccessRate = float64(successCount) / float64(len(metrics))

	return model, nil
}

// GetModel returns the baseline model for a database
func (bt *BaselineTrainer) GetModel(dbName string) (*BaselineModel, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	model, exists := bt.models[dbName]
	if !exists {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound,
			"no baseline model for database")
	}

	return model, nil
}

// GetAllModels returns all trained baseline models
func (bt *BaselineTrainer) GetAllModels() map[string]*BaselineModel {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	// Return a copy to prevent external modifications
	models := make(map[string]*BaselineModel, len(bt.models))
	for k, v := range bt.models {
		models[k] = v
	}

	return models
}

// IncrementalUpdate updates a baseline model with new metrics
func (bt *BaselineTrainer) IncrementalUpdate(dbName string, newMetrics []BackupMetric) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// Add new metrics to dataset
	bt.metrics = append(bt.metrics, newMetrics...)

	// Get metrics for this database
	dbMetrics := make([]BackupMetric, 0)
	cutoffTime := time.Now().AddDate(0, 0, -bt.windowDays)

	for _, metric := range bt.metrics {
		if metric.DatabaseName == dbName && metric.Timestamp.After(cutoffTime) {
			dbMetrics = append(dbMetrics, metric)
		}
	}

	if len(dbMetrics) < bt.minSamples {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"insufficient data for incremental update")
	}

	// Retrain model
	model, err := bt.trainDatabase(dbName, dbMetrics)
	if err != nil {
		return err
	}

	bt.models[dbName] = model
	return nil
}

// ClearOldMetrics removes metrics older than the training window
func (bt *BaselineTrainer) ClearOldMetrics() {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -bt.windowDays)
	filtered := make([]BackupMetric, 0)

	for _, metric := range bt.metrics {
		if metric.Timestamp.After(cutoffTime) {
			filtered = append(filtered, metric)
		}
	}

	bt.metrics = filtered
}

// Statistical helper functions

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}

	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}

	return minVal
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	return maxVal
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate index
	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
