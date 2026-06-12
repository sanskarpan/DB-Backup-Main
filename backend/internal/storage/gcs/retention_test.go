package gcs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetentionMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     RetentionMode
		expected string
	}{
		{
			name:     "unlocked mode",
			mode:     RetentionModeUnlocked,
			expected: "Unlocked",
		},
		{
			name:     "locked mode",
			mode:     RetentionModeLocked,
			expected: "Locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

func TestRetentionPolicyConfig(t *testing.T) {
	t.Run("create policy with period", func(t *testing.T) {
		config := &RetentionPolicyConfig{
			RetentionPeriod: 30 * 24 * time.Hour, // 30 days
			EffectiveTime:   time.Now(),
			IsLocked:        false,
		}

		assert.Equal(t, 30*24*time.Hour, config.RetentionPeriod)
		assert.False(t, config.IsLocked)
		assert.NotZero(t, config.EffectiveTime)
	})

	t.Run("create locked policy", func(t *testing.T) {
		config := &RetentionPolicyConfig{
			RetentionPeriod: 90 * 24 * time.Hour, // 90 days
			EffectiveTime:   time.Now(),
			IsLocked:        true,
		}

		assert.Equal(t, 90*24*time.Hour, config.RetentionPeriod)
		assert.True(t, config.IsLocked)
	})
}

func TestObjectRetentionConfig(t *testing.T) {
	t.Run("create object retention", func(t *testing.T) {
		retainUntil := time.Now().AddDate(0, 0, 60)
		config := &ObjectRetentionConfig{
			Mode:            RetentionModeUnlocked,
			RetainUntilTime: retainUntil,
		}

		assert.Equal(t, RetentionModeUnlocked, config.Mode)
		assert.Equal(t, retainUntil, config.RetainUntilTime)
	})

	t.Run("create locked object retention", func(t *testing.T) {
		retainUntil := time.Now().AddDate(0, 0, 365)
		config := &ObjectRetentionConfig{
			Mode:            RetentionModeLocked,
			RetainUntilTime: retainUntil,
		}

		assert.Equal(t, RetentionModeLocked, config.Mode)
		assert.True(t, config.RetainUntilTime.After(time.Now()))
	})
}

func TestObjectHoldConfig(t *testing.T) {
	t.Run("event-based hold only", func(t *testing.T) {
		config := &ObjectHoldConfig{
			EventBasedHold: true,
			TemporaryHold:  false,
		}

		assert.True(t, config.EventBasedHold)
		assert.False(t, config.TemporaryHold)
	})

	t.Run("temporary hold only", func(t *testing.T) {
		config := &ObjectHoldConfig{
			EventBasedHold: false,
			TemporaryHold:  true,
		}

		assert.False(t, config.EventBasedHold)
		assert.True(t, config.TemporaryHold)
	})

	t.Run("both holds", func(t *testing.T) {
		config := &ObjectHoldConfig{
			EventBasedHold: true,
			TemporaryHold:  true,
		}

		assert.True(t, config.EventBasedHold)
		assert.True(t, config.TemporaryHold)
	})

	t.Run("no holds", func(t *testing.T) {
		config := &ObjectHoldConfig{
			EventBasedHold: false,
			TemporaryHold:  false,
		}

		assert.False(t, config.EventBasedHold)
		assert.False(t, config.TemporaryHold)
	})
}

func TestGCSObjectRetentionInfo_IsProtected(t *testing.T) {
	tests := []struct {
		name     string
		info     GCSObjectRetentionInfo
		expected bool
	}{
		{
			name: "protected by event-based hold",
			info: GCSObjectRetentionInfo{
				EventBasedHold: true,
			},
			expected: true,
		},
		{
			name: "protected by temporary hold",
			info: GCSObjectRetentionInfo{
				TemporaryHold: true,
			},
			expected: true,
		},
		{
			name: "protected by object retention",
			info: GCSObjectRetentionInfo{
				RetainUntilTime: timePtr(time.Now().AddDate(0, 0, 30)),
			},
			expected: true,
		},
		{
			name: "protected by bucket retention",
			info: GCSObjectRetentionInfo{
				Created:               time.Now().AddDate(0, 0, -5), // Created 5 days ago
				BucketRetentionPeriod: 30 * 24 * time.Hour,          // 30-day retention
			},
			expected: true,
		},
		{
			name: "not protected - retention expired",
			info: GCSObjectRetentionInfo{
				RetainUntilTime: timePtr(time.Now().AddDate(0, 0, -1)),
			},
			expected: false,
		},
		{
			name: "not protected - no retention",
			info: GCSObjectRetentionInfo{
				Created: time.Now().AddDate(0, 0, -60), // Created 60 days ago
			},
			expected: false,
		},
		{
			name: "protected by multiple mechanisms",
			info: GCSObjectRetentionInfo{
				EventBasedHold:        true,
				TemporaryHold:         true,
				RetainUntilTime:       timePtr(time.Now().AddDate(0, 0, 30)),
				Created:               time.Now().AddDate(0, 0, -5),
				BucketRetentionPeriod: 30 * 24 * time.Hour,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.IsProtected()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCSObjectRetentionInfo_DaysUntilUnlock(t *testing.T) {
	tests := []struct {
		name     string
		info     GCSObjectRetentionInfo
		expected int
	}{
		{
			name: "event-based hold indefinite",
			info: GCSObjectRetentionInfo{
				EventBasedHold: true,
			},
			expected: -1,
		},
		{
			name: "temporary hold indefinite",
			info: GCSObjectRetentionInfo{
				TemporaryHold: true,
			},
			expected: -1,
		},
		{
			name: "30 days object retention",
			info: GCSObjectRetentionInfo{
				RetainUntilTime: timePtr(time.Now().AddDate(0, 0, 30)),
			},
			expected: 30,
		},
		{
			name: "30 days bucket retention",
			info: GCSObjectRetentionInfo{
				Created:               time.Now().AddDate(0, 0, -5), // Created 5 days ago
				BucketRetentionPeriod: 30 * 24 * time.Hour,          // 30-day retention
			},
			expected: 25, // 30 - 5 = 25 days remaining
		},
		{
			name: "expired retention",
			info: GCSObjectRetentionInfo{
				RetainUntilTime: timePtr(time.Now().AddDate(0, 0, -5)),
			},
			expected: 0,
		},
		{
			name: "no retention",
			info: GCSObjectRetentionInfo{
				Created: time.Now().AddDate(0, 0, -60),
			},
			expected: 0,
		},
		{
			name: "longer of object and bucket retention",
			info: GCSObjectRetentionInfo{
				RetainUntilTime:       timePtr(time.Now().AddDate(0, 0, 60)),  // 60 days
				Created:               time.Now().AddDate(0, 0, -5),            // Created 5 days ago
				BucketRetentionPeriod: 30 * 24 * time.Hour,                    // 30-day retention
			},
			expected: 60, // Object retention is longer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.DaysUntilUnlock()

			// For time-based tests, allow 1-day variance
			if tt.expected > 0 {
				assert.InDelta(t, tt.expected, result, 1,
					"Expected %d days, got %d days", tt.expected, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGCSObjectRetentionInfo_IsLocked(t *testing.T) {
	tests := []struct {
		name     string
		info     GCSObjectRetentionInfo
		expected bool
	}{
		{
			name: "locked object retention",
			info: GCSObjectRetentionInfo{
				RetentionMode: "Locked",
			},
			expected: true,
		},
		{
			name: "locked bucket retention",
			info: GCSObjectRetentionInfo{
				BucketRetentionLocked: true,
			},
			expected: true,
		},
		{
			name: "unlocked object retention",
			info: GCSObjectRetentionInfo{
				RetentionMode: "Unlocked",
			},
			expected: false,
		},
		{
			name: "no retention",
			info: GCSObjectRetentionInfo{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.IsLocked()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGCSObjectRetentionInfo_ProtectionScenarios(t *testing.T) {
	t.Run("object lifecycle with bucket retention", func(t *testing.T) {
		// Day 0: Create object with bucket retention
		info := GCSObjectRetentionInfo{
			ObjectName:            "backup-2025-01-01.sql",
			Created:               time.Now(),
			BucketRetentionPeriod: 30 * 24 * time.Hour,
			BucketRetentionLocked: false,
		}

		// Should be protected
		assert.True(t, info.IsProtected())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 30, info.DaysUntilUnlock(), 1)
		assert.False(t, info.IsLocked())

		// Apply event-based hold
		info.EventBasedHold = true
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())

		// Lock bucket retention
		info.BucketRetentionLocked = true
		assert.True(t, info.IsLocked())

		// Remove event-based hold (bucket retention still protects)
		info.EventBasedHold = false
		assert.True(t, info.IsProtected())

		// Simulate retention expiration (31 days passed)
		info.Created = time.Now().AddDate(0, 0, -31)
		assert.False(t, info.IsProtected())
		assert.Equal(t, 0, info.DaysUntilUnlock())
	})

	t.Run("object with explicit retention", func(t *testing.T) {
		retainUntil := time.Now().AddDate(0, 0, 90)
		info := GCSObjectRetentionInfo{
			ObjectName:      "backup-compliance.sql",
			Created:         time.Now(),
			RetentionMode:   "Locked",
			RetainUntilTime: &retainUntil,
		}

		assert.True(t, info.IsProtected())
		assert.True(t, info.IsLocked())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 90, info.DaysUntilUnlock(), 1)
		assert.Equal(t, "Locked", info.RetentionMode)
	})

	t.Run("temporary hold workflow", func(t *testing.T) {
		info := GCSObjectRetentionInfo{
			ObjectName:    "backup-investigation.sql",
			Created:       time.Now(),
			TemporaryHold: true,
		}

		// Protected by temporary hold
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())

		// Remove temporary hold
		info.TemporaryHold = false
		assert.False(t, info.IsProtected())
		assert.Equal(t, 0, info.DaysUntilUnlock())
	})
}

func TestRetentionModeValidation(t *testing.T) {
	t.Run("valid modes", func(t *testing.T) {
		validModes := []RetentionMode{
			RetentionModeUnlocked,
			RetentionModeLocked,
		}

		for _, mode := range validModes {
			assert.NotEmpty(t, string(mode))
			assert.Contains(t, []string{"Unlocked", "Locked"}, string(mode))
		}
	})
}

func TestGCSObjectRetentionInfo_EdgeCases(t *testing.T) {
	t.Run("protection with nil retain time but holds", func(t *testing.T) {
		info := GCSObjectRetentionInfo{
			ObjectName:     "test.sql",
			RetainUntilTime: nil,
			EventBasedHold:  true,
		}

		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())
	})

	t.Run("expired bucket retention with hold", func(t *testing.T) {
		info := GCSObjectRetentionInfo{
			ObjectName:            "test.sql",
			Created:               time.Now().AddDate(0, 0, -60), // Created 60 days ago
			BucketRetentionPeriod: 30 * 24 * time.Hour,           // 30-day retention
			TemporaryHold:         true,
		}

		// Still protected by temporary hold
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())
	})

	t.Run("exactly at retention boundary", func(t *testing.T) {
		// Retention expires right now (within seconds)
		retainUntil := time.Now()
		info := GCSObjectRetentionInfo{
			ObjectName:      "test.sql",
			Created:         time.Now(),
			RetainUntilTime: &retainUntil,
		}

		// Should be at the boundary (0 days or barely protected)
		days := info.DaysUntilUnlock()
		assert.True(t, days <= 1 && days >= 0, "Expected 0-1 days at boundary, got %d", days)
	})

	t.Run("bucket and object retention both apply", func(t *testing.T) {
		// Object retention: 60 days
		// Bucket retention: 30 days (created 5 days ago, so 25 days remaining)
		// Should use the longer of the two
		info := GCSObjectRetentionInfo{
			ObjectName:            "test.sql",
			Created:               time.Now().AddDate(0, 0, -5),
			BucketRetentionPeriod: 30 * 24 * time.Hour,
			RetainUntilTime:       timePtr(time.Now().AddDate(0, 0, 60)),
		}

		assert.True(t, info.IsProtected())
		// Should return the longer period (60 days from object retention)
		assert.InDelta(t, 60, info.DaysUntilUnlock(), 1)
	})
}

func TestGCSObjectRetentionInfo_SizeAndMetadata(t *testing.T) {
	t.Run("object with full metadata", func(t *testing.T) {
		retainUntil := time.Now().AddDate(0, 0, 30)
		info := GCSObjectRetentionInfo{
			ObjectName:      "backup-100mb.sql",
			Size:            104857600, // 100MB
			Created:         time.Now().AddDate(0, 0, -7),
			Updated:         time.Now().AddDate(0, 0, -1),
			StorageClass:    "STANDARD",
			ContentType:     "application/sql",
			RetainUntilTime: &retainUntil,
		}

		assert.Equal(t, int64(104857600), info.Size)
		assert.Equal(t, "STANDARD", info.StorageClass)
		assert.Equal(t, "application/sql", info.ContentType)
		assert.NotZero(t, info.Created)
		assert.NotZero(t, info.Updated)
		assert.True(t, info.IsProtected())
	})

	t.Run("object with minimal metadata", func(t *testing.T) {
		info := GCSObjectRetentionInfo{
			ObjectName: "backup-minimal.sql",
			Created:    time.Now(),
		}

		assert.Equal(t, int64(0), info.Size)
		assert.Empty(t, info.StorageClass)
		assert.Empty(t, info.ContentType)
		assert.NotZero(t, info.Created)
		assert.False(t, info.IsProtected())
	})
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}

// Benchmark tests
func BenchmarkGCSIsProtected(b *testing.B) {
	info := GCSObjectRetentionInfo{
		RetainUntilTime: timePtr(time.Now().AddDate(0, 0, 30)),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.IsProtected()
	}
}

func BenchmarkGCSDaysUntilUnlock(b *testing.B) {
	info := GCSObjectRetentionInfo{
		RetainUntilTime: timePtr(time.Now().AddDate(0, 0, 60)),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.DaysUntilUnlock()
	}
}

func BenchmarkGCSIsLocked(b *testing.B) {
	info := GCSObjectRetentionInfo{
		RetentionMode: "Locked",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.IsLocked()
	}
}

// Integration tests would go here (require actual GCS bucket with retention enabled)
// These are commented out to avoid requiring GCP credentials in unit tests

/*
func TestGCSProvider_SetBucketRetentionPolicy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test GCS provider
	provider := setupTestGCSProvider(t)
	ctx := context.Background()

	// Test setting retention policy
	err := provider.SetBucketRetentionPolicy(ctx, 30*24*time.Hour)
	require.NoError(t, err)

	// Verify policy is set
	policy, err := provider.GetBucketRetentionPolicy(ctx)
	require.NoError(t, err)
	assert.NotNil(t, policy)
	assert.Equal(t, 30*24*time.Hour, policy.RetentionPeriod)
	assert.False(t, policy.IsLocked)
}

func TestGCSProvider_SetObjectRetention_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestGCSProvider(t)
	ctx := context.Background()

	objectName := "test-retention.sql"
	retainUntil := time.Now().AddDate(0, 0, 30)

	// Set object retention
	err := provider.SetObjectRetention(ctx, objectName, retainUntil, RetentionModeUnlocked)
	require.NoError(t, err)

	// Verify retention
	info, err := provider.GetObjectRetention(ctx, objectName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.IsProtected())
}

func TestGCSProvider_ObjectHold_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestGCSProvider(t)
	ctx := context.Background()

	objectName := "test-hold.sql"

	// Set event-based hold
	err := provider.SetObjectHold(ctx, objectName, &ObjectHoldConfig{
		EventBasedHold: true,
		TemporaryHold:  false,
	})
	require.NoError(t, err)

	// Verify hold
	info, err := provider.GetObjectRetention(ctx, objectName)
	require.NoError(t, err)
	assert.True(t, info.EventBasedHold)
	assert.False(t, info.TemporaryHold)
	assert.True(t, info.IsProtected())

	// Remove hold
	err = provider.SetObjectHold(ctx, objectName, &ObjectHoldConfig{
		EventBasedHold: false,
		TemporaryHold:  false,
	})
	require.NoError(t, err)

	// Verify hold removed
	info, err = provider.GetObjectRetention(ctx, objectName)
	require.NoError(t, err)
	assert.False(t, info.EventBasedHold)
}
*/
