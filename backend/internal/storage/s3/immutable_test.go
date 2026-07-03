package s3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObjectLockMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     ObjectLockMode
		expected string
	}{
		{
			name:     "governance mode",
			mode:     ObjectLockModeGovernance,
			expected: "GOVERNANCE",
		},
		{
			name:     "compliance mode",
			mode:     ObjectLockModeCompliance,
			expected: "COMPLIANCE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

func TestObjectLockConfig(t *testing.T) {
	t.Run("create config with days", func(t *testing.T) {
		config := &ObjectLockConfig{
			Mode:          ObjectLockModeGovernance,
			RetentionDays: 30,
			LegalHold:     false,
		}

		assert.Equal(t, ObjectLockModeGovernance, config.Mode)
		assert.Equal(t, 30, config.RetentionDays)
		assert.False(t, config.LegalHold)
		assert.Nil(t, config.RetainUntilDate)
	})

	t.Run("create config with date", func(t *testing.T) {
		retainDate := time.Now().AddDate(0, 0, 60)
		config := &ObjectLockConfig{
			Mode:            ObjectLockModeCompliance,
			RetainUntilDate: &retainDate,
			LegalHold:       true,
		}

		assert.Equal(t, ObjectLockModeCompliance, config.Mode)
		assert.NotNil(t, config.RetainUntilDate)
		assert.True(t, config.LegalHold)
	})
}

func TestImmutableBackupConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := &ImmutableBackupConfig{
			Enabled:              true,
			DefaultMode:          ObjectLockModeGovernance,
			DefaultRetentionDays: 30,
			AllowBypass:          false,
		}

		assert.True(t, config.Enabled)
		assert.Equal(t, ObjectLockModeGovernance, config.DefaultMode)
		assert.Equal(t, 30, config.DefaultRetentionDays)
		assert.False(t, config.AllowBypass)
	})

	t.Run("compliance config", func(t *testing.T) {
		config := &ImmutableBackupConfig{
			Enabled:              true,
			DefaultMode:          ObjectLockModeCompliance,
			DefaultRetentionDays: 90,
			AllowBypass:          false,
		}

		assert.True(t, config.Enabled)
		assert.Equal(t, ObjectLockModeCompliance, config.DefaultMode)
		assert.Equal(t, 90, config.DefaultRetentionDays)
		assert.False(t, config.AllowBypass)
	})
}

func TestImmutableBackupInfo_IsProtected(t *testing.T) {
	tests := []struct {
		name     string
		info     ImmutableBackupInfo
		expected bool
	}{
		{
			name: "protected by legal hold",
			info: ImmutableBackupInfo{
				LegalHold: true,
			},
			expected: true,
		},
		{
			name: "protected by retention",
			info: ImmutableBackupInfo{
				RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:       false,
			},
			expected: true,
		},
		{
			name: "not protected - retention expired",
			info: ImmutableBackupInfo{
				RetainUntilDate: timePtr(time.Now().AddDate(0, 0, -1)),
				LegalHold:       false,
			},
			expected: false,
		},
		{
			name: "not protected - no retention",
			info: ImmutableBackupInfo{
				RetainUntilDate: nil,
				LegalHold:       false,
			},
			expected: false,
		},
		{
			name: "protected by both",
			info: ImmutableBackupInfo{
				RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:       true,
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

func TestImmutableBackupInfo_DaysUntilUnlock(t *testing.T) {
	tests := []struct {
		name     string
		info     ImmutableBackupInfo
		expected int
	}{
		{
			name: "legal hold indefinite",
			info: ImmutableBackupInfo{
				LegalHold: true,
			},
			expected: -1,
		},
		{
			name: "30 days retention",
			info: ImmutableBackupInfo{
				RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:       false,
			},
			expected: 30,
		},
		{
			name: "expired retention",
			info: ImmutableBackupInfo{
				RetainUntilDate: timePtr(time.Now().AddDate(0, 0, -5)),
				LegalHold:       false,
			},
			expected: 0,
		},
		{
			name: "no retention",
			info: ImmutableBackupInfo{
				RetainUntilDate: nil,
				LegalHold:       false,
			},
			expected: 0,
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

func TestImmutableBackupInfo_ProtectionScenarios(t *testing.T) {
	t.Run("backup lifecycle", func(t *testing.T) {
		// Day 0: Create backup with 30-day retention
		retainDate := time.Now().AddDate(0, 0, 30)
		info := ImmutableBackupInfo{
			Key:             "backup-2025-01-01.sql",
			RetainUntilDate: &retainDate,
			LegalHold:       false,
		}

		// Should be protected
		assert.True(t, info.IsProtected())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 30, info.DaysUntilUnlock(), 1)

		// Apply legal hold
		info.LegalHold = true
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())

		// Remove legal hold
		info.LegalHold = false
		assert.True(t, info.IsProtected())

		// Simulate retention expiration
		pastDate := time.Now().AddDate(0, 0, -1)
		info.RetainUntilDate = &pastDate
		assert.False(t, info.IsProtected())
		assert.Equal(t, 0, info.DaysUntilUnlock())
	})

	t.Run("compliance mode protection", func(t *testing.T) {
		retainDate := time.Now().AddDate(0, 0, 90)
		info := ImmutableBackupInfo{
			Key:             "backup-compliance.sql",
			RetentionMode:   string(ObjectLockModeCompliance),
			RetainUntilDate: &retainDate,
			LegalHold:       false,
		}

		assert.True(t, info.IsProtected())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 90, info.DaysUntilUnlock(), 1)
		assert.Equal(t, "COMPLIANCE", info.RetentionMode)
	})
}

func TestObjectLockModeValidation(t *testing.T) {
	t.Run("valid modes", func(t *testing.T) {
		validModes := []ObjectLockMode{
			ObjectLockModeGovernance,
			ObjectLockModeCompliance,
		}

		for _, mode := range validModes {
			assert.NotEmpty(t, string(mode))
			assert.Contains(t, []string{"GOVERNANCE", "COMPLIANCE"}, string(mode))
		}
	})
}

func TestImmutableBackupConfig_Validation(t *testing.T) {
	t.Run("governance with bypass allowed", func(t *testing.T) {
		config := &ImmutableBackupConfig{
			Enabled:              true,
			DefaultMode:          ObjectLockModeGovernance,
			DefaultRetentionDays: 30,
			AllowBypass:          true,
		}

		assert.True(t, config.Enabled)
		assert.True(t, config.AllowBypass)
		assert.Equal(t, ObjectLockModeGovernance, config.DefaultMode)
		assert.Equal(t, 30, config.DefaultRetentionDays)
	})

	t.Run("compliance never allows bypass", func(t *testing.T) {
		config := &ImmutableBackupConfig{
			Enabled:              true,
			DefaultMode:          ObjectLockModeCompliance,
			DefaultRetentionDays: 90,
			AllowBypass:          false, // Must always be false for compliance
		}

		assert.True(t, config.Enabled)
		assert.False(t, config.AllowBypass)
		assert.Equal(t, ObjectLockModeCompliance, config.DefaultMode)
		assert.Equal(t, 90, config.DefaultRetentionDays)
	})
}

func TestImmutableBackupInfo_SizeAndMetadata(t *testing.T) {
	t.Run("backup with size metadata", func(t *testing.T) {
		size := int64(1024 * 1024 * 100) // 100MB
		lastModified := time.Now().AddDate(0, 0, -7)

		info := ImmutableBackupInfo{
			Key:             "backup-100mb.sql",
			Size:            &size,
			LastModified:    &lastModified,
			RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 30)),
		}

		assert.NotNil(t, info.Size)
		assert.Equal(t, int64(104857600), *info.Size)
		assert.NotNil(t, info.LastModified)
		assert.True(t, info.IsProtected())
	})
}

// Helper function to create time pointers.
func timePtr(t time.Time) *time.Time {
	return &t
}

// Integration tests would go here (require actual S3 bucket with Object Lock enabled)
// These are commented out to avoid requiring AWS credentials in unit tests

/*
func TestS3Provider_EnableObjectLock_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test S3 provider
	provider := setupTestS3Provider(t)
	ctx := context.Background()

	// Test enabling Object Lock
	err := provider.EnableObjectLock(ctx)
	require.NoError(t, err)

	// Verify Object Lock is enabled
	// (Would need to check bucket configuration)
}

func TestS3Provider_UploadImmutable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestS3Provider(t)
	ctx := context.Background()

	// Create test file
	testFile := createTestFile(t)
	defer os.Remove(testFile)

	// Upload with Object Lock
	lockConfig := &ObjectLockConfig{
		Mode:          ObjectLockModeGovernance,
		RetentionDays: 30,
		LegalHold:     false,
	}

	err := provider.UploadImmutable(ctx, testFile, "test-backup.sql", lockConfig)
	require.NoError(t, err)

	// Verify retention
	retention, err := provider.GetObjectRetention(ctx, "test-backup.sql")
	require.NoError(t, err)
	assert.NotNil(t, retention)
	assert.Equal(t, ObjectLockModeGovernance, retention.Mode)
}

func TestS3Provider_LegalHold_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestS3Provider(t)
	ctx := context.Background()

	remotePath := "test-legal-hold.sql"

	// Set legal hold
	err := provider.SetLegalHold(ctx, remotePath, true)
	require.NoError(t, err)

	// Verify legal hold
	hasLegalHold, err := provider.GetLegalHoldStatus(ctx, remotePath)
	require.NoError(t, err)
	assert.True(t, hasLegalHold)

	// Remove legal hold
	err = provider.SetLegalHold(ctx, remotePath, false)
	require.NoError(t, err)

	// Verify legal hold removed
	hasLegalHold, err = provider.GetLegalHoldStatus(ctx, remotePath)
	require.NoError(t, err)
	assert.False(t, hasLegalHold)
}
*/

// Benchmark tests.
func BenchmarkIsProtected(b *testing.B) {
	info := ImmutableBackupInfo{
		RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 30)),
		LegalHold:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.IsProtected()
	}
}

func BenchmarkDaysUntilUnlock(b *testing.B) {
	info := ImmutableBackupInfo{
		RetainUntilDate: timePtr(time.Now().AddDate(0, 0, 60)),
		LegalHold:       false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.DaysUntilUnlock()
	}
}
