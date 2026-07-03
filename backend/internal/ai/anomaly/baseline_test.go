package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaselineTrainer(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	assert.NotNil(t, trainer)
	assert.Equal(t, 30, trainer.windowDays)
	assert.Equal(t, 10, trainer.minSamples)
	assert.NotNil(t, trainer.models)
	assert.NotNil(t, trainer.metrics)
}

func TestNewBaselineTrainer_DefaultWindow(t *testing.T) {
	trainer := NewBaselineTrainer(0)

	assert.NotNil(t, trainer)
	assert.Equal(t, 30, trainer.windowDays) // Should default to 30
}

func TestBaselineTrainer_AddMetric(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	metric := BackupMetric{
		BackupID:     "backup-1",
		DatabaseName: "test-db",
		Timestamp:    time.Now(),
		Size:         1000000,
		Duration:     60.5,
		Success:      true,
	}

	trainer.AddMetric(metric)

	assert.Len(t, trainer.metrics, 1)
	assert.Equal(t, metric, trainer.metrics[0])
}

func TestBaselineTrainer_AddMetrics(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	metrics := []BackupMetric{
		{BackupID: "1", DatabaseName: "db1", Timestamp: time.Now(), Size: 1000, Duration: 10, Success: true},
		{BackupID: "2", DatabaseName: "db1", Timestamp: time.Now(), Size: 1100, Duration: 12, Success: true},
		{BackupID: "3", DatabaseName: "db1", Timestamp: time.Now(), Size: 1050, Duration: 11, Success: true},
	}

	trainer.AddMetrics(metrics)

	assert.Len(t, trainer.metrics, 3)
}

func TestBaselineTrainer_Train(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	// Add sufficient metrics for training
	now := time.Now()
	for i := 0; i < 20; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Size:         1000000 + int64(i*10000),
			Duration:     60.0 + float64(i),
			Success:      i%10 != 0, // 10% failure rate
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)

	assert.NoError(t, err)

	// Verify model was created
	model, err := trainer.GetModel("test-db")
	assert.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, "test-db", model.DatabaseName)
	assert.Equal(t, 20, model.SampleCount)
	assert.Greater(t, model.SizeMean, float64(0))
	assert.Greater(t, model.DurationMean, float64(0))
}

func TestBaselineTrainer_Train_InsufficientData(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	// Add only 5 metrics (below minimum of 10)
	for i := 0; i < 5; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    time.Now(),
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestBaselineTrainer_Train_MultipleDatabase(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	now := time.Now()

	// Add metrics for multiple databases
	for _, db := range []string{"db1", "db2", "db3"} {
		for i := 0; i < 15; i++ {
			trainer.AddMetric(BackupMetric{
				BackupID:     "backup-" + db + "-" + string(rune('0'+i)),
				DatabaseName: db,
				Timestamp:    now.Add(-time.Duration(i) * time.Hour),
				Size:         1000000 + int64(i*10000),
				Duration:     60.0 + float64(i),
				Success:      true,
			})
		}
	}

	ctx := context.Background()
	err := trainer.Train(ctx)

	assert.NoError(t, err)

	// Verify models for all databases
	models := trainer.GetAllModels()
	assert.Len(t, models, 3)

	for _, db := range []string{"db1", "db2", "db3"} {
		model, exists := models[db]
		assert.True(t, exists)
		assert.NotNil(t, model)
		assert.Equal(t, 15, model.SampleCount)
	}
}

func TestBaselineModel_Statistics(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	now := time.Now()

	// Add controlled dataset
	sizes := []int64{1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000}

	for i, size := range sizes {
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Size:         size,
			Duration:     float64(i + 50),
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	model, err := trainer.GetModel("test-db")
	require.NoError(t, err)

	// Verify statistics
	assert.Equal(t, float64(1000), model.SizeMin)
	assert.Equal(t, float64(2000), model.SizeMax)
	assert.InDelta(t, 1500.0, model.SizeMean, 10.0) // Mean should be around 1500
	assert.Greater(t, model.SizeStdDev, float64(0))
	assert.Equal(t, float64(1500), model.SizeP50) // Median
}

func TestBaselineModel_SuccessRate(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	now := time.Now()

	// Add 20 metrics with 15 successes (75% success rate)
	for i := 0; i < 20; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Size:         1000000,
			Duration:     60.0,
			Success:      i < 15, // First 15 succeed
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	model, err := trainer.GetModel("test-db")
	require.NoError(t, err)

	assert.Equal(t, 20, model.TotalBackups)
	assert.Equal(t, 15, model.SuccessfulBackups)
	assert.Equal(t, 5, model.FailedBackups)
	assert.InDelta(t, 0.75, model.SuccessRate, 0.01)
}

func TestBaselineModel_TemporalDistribution(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	now := time.Now()

	// Add backups at specific hours
	for i := 0; i < 10; i++ {
		// All backups at 2 AM
		timestamp := time.Date(now.Year(), now.Month(), now.Day()-i, 2, 0, 0, 0, time.UTC)
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    timestamp,
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	model, err := trainer.GetModel("test-db")
	require.NoError(t, err)

	// All backups should be at hour 2
	assert.Equal(t, 10, model.HourlyDistribution[2])
	for i := 0; i < 24; i++ {
		if i != 2 {
			assert.Equal(t, 0, model.HourlyDistribution[i])
		}
	}
}

func TestBaselineTrainer_IncrementalUpdate(t *testing.T) {
	trainer := NewBaselineTrainer(30)

	now := time.Now()

	// Initial training
	for i := 0; i < 15; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "backup-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.Add(-time.Duration(i) * time.Hour),
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	ctx := context.Background()
	err := trainer.Train(ctx)
	require.NoError(t, err)

	model1, _ := trainer.GetModel("test-db")
	oldCount := model1.SampleCount

	// Add new metrics
	newMetrics := []BackupMetric{
		{BackupID: "new-1", DatabaseName: "test-db", Timestamp: now, Size: 1100000, Duration: 65.0, Success: true},
		{BackupID: "new-2", DatabaseName: "test-db", Timestamp: now, Size: 1200000, Duration: 70.0, Success: true},
	}

	err = trainer.IncrementalUpdate("test-db", newMetrics)
	assert.NoError(t, err)

	model2, _ := trainer.GetModel("test-db")
	assert.Greater(t, model2.SampleCount, oldCount)
}

func TestBaselineTrainer_ClearOldMetrics(t *testing.T) {
	trainer := NewBaselineTrainer(7) // 7-day window

	now := time.Now()

	// Add old and new metrics
	for i := 0; i < 10; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "old-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.AddDate(0, 0, -10), // 10 days ago
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	for i := 0; i < 5; i++ {
		trainer.AddMetric(BackupMetric{
			BackupID:     "new-" + string(rune('0'+i)),
			DatabaseName: "test-db",
			Timestamp:    now.AddDate(0, 0, -3), // 3 days ago
			Size:         1000000,
			Duration:     60.0,
			Success:      true,
		})
	}

	assert.Len(t, trainer.metrics, 15)

	trainer.ClearOldMetrics()

	// Only new metrics within 7-day window should remain
	assert.Len(t, trainer.metrics, 5)
}

func TestStatisticalFunctions(t *testing.T) {
	t.Run("mean", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5}
		assert.InDelta(t, 3.0, mean(values), 0.01)
	})

	t.Run("stdDev", func(t *testing.T) {
		values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
		m := mean(values)
		sd := stdDev(values, m)
		assert.InDelta(t, 2.0, sd, 0.1)
	})

	t.Run("min", func(t *testing.T) {
		values := []float64{5, 2, 9, 1, 7}
		assert.Equal(t, 1.0, minFloat(values))
	})

	t.Run("max", func(t *testing.T) {
		values := []float64{5, 2, 9, 1, 7}
		assert.Equal(t, 9.0, maxFloat(values))
	})

	t.Run("percentile", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		assert.InDelta(t, 5.5, percentile(values, 0.50), 0.1) // Median
		assert.InDelta(t, 9.5, percentile(values, 0.95), 0.1) // 95th percentile
	})
}
