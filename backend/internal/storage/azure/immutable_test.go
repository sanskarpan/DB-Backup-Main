package azure

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
)

func TestImmutabilityPolicyMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     ImmutabilityPolicyMode
		expected string
	}{
		{
			name:     "unlocked mode",
			mode:     ImmutabilityPolicyModeUnlocked,
			expected: "Unlocked",
		},
		{
			name:     "locked mode",
			mode:     ImmutabilityPolicyModeLocked,
			expected: "Locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

func TestImmutableBlobConfig(t *testing.T) {
	t.Run("create config with days", func(t *testing.T) {
		config := &ImmutableBlobConfig{
			ImmutabilityPeriodDays:     30,
			LegalHold:                  false,
			Mode:                       ImmutabilityPolicyModeUnlocked,
			AllowProtectedAppendWrites: false,
		}

		assert.Equal(t, 30, config.ImmutabilityPeriodDays)
		assert.False(t, config.LegalHold)
		assert.Equal(t, ImmutabilityPolicyModeUnlocked, config.Mode)
		assert.False(t, config.AllowProtectedAppendWrites)
	})

	t.Run("create config with locked mode", func(t *testing.T) {
		config := &ImmutableBlobConfig{
			ImmutabilityPeriodDays:     90,
			LegalHold:                  true,
			Mode:                       ImmutabilityPolicyModeLocked,
			AllowProtectedAppendWrites: true,
		}

		assert.Equal(t, 90, config.ImmutabilityPeriodDays)
		assert.True(t, config.LegalHold)
		assert.Equal(t, ImmutabilityPolicyModeLocked, config.Mode)
		assert.True(t, config.AllowProtectedAppendWrites)
	})
}

func TestContainerImmutabilityPolicy(t *testing.T) {
	t.Run("default policy", func(t *testing.T) {
		policy := &ContainerImmutabilityPolicy{
			ImmutabilityPeriodDays:        30,
			State:                         "Unlocked",
			AllowProtectedAppendWrites:    false,
			AllowProtectedAppendWritesAll: false,
		}

		assert.Equal(t, 30, policy.ImmutabilityPeriodDays)
		assert.Equal(t, "Unlocked", policy.State)
		assert.False(t, policy.AllowProtectedAppendWrites)
		assert.False(t, policy.AllowProtectedAppendWritesAll)
	})

	t.Run("locked policy", func(t *testing.T) {
		policy := &ContainerImmutabilityPolicy{
			ImmutabilityPeriodDays:        90,
			State:                         "Locked",
			AllowProtectedAppendWrites:    true,
			AllowProtectedAppendWritesAll: false,
		}

		assert.Equal(t, 90, policy.ImmutabilityPeriodDays)
		assert.Equal(t, "Locked", policy.State)
		assert.True(t, policy.AllowProtectedAppendWrites)
		assert.False(t, policy.AllowProtectedAppendWritesAll)
	})
}

func TestAzureImmutableBlobInfo_IsProtected(t *testing.T) {
	tests := []struct {
		name     string
		info     AzureImmutableBlobInfo
		expected bool
	}{
		{
			name: "protected by legal hold",
			info: AzureImmutableBlobInfo{
				LegalHold: true,
			},
			expected: true,
		},
		{
			name: "protected by immutability policy",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:             false,
			},
			expected: true,
		},
		{
			name: "not protected - policy expired",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, -1)),
				LegalHold:             false,
			},
			expected: false,
		},
		{
			name: "not protected - no policy",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: nil,
				LegalHold:             false,
			},
			expected: false,
		},
		{
			name: "protected by both legal hold and policy",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:             true,
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

func TestAzureImmutableBlobInfo_DaysUntilUnlock(t *testing.T) {
	tests := []struct {
		name     string
		info     AzureImmutableBlobInfo
		expected int
	}{
		{
			name: "legal hold indefinite",
			info: AzureImmutableBlobInfo{
				LegalHold: true,
			},
			expected: -1,
		},
		{
			name: "30 days retention",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 30)),
				LegalHold:             false,
			},
			expected: 30,
		},
		{
			name: "expired retention",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, -5)),
				LegalHold:             false,
			},
			expected: 0,
		},
		{
			name: "no retention",
			info: AzureImmutableBlobInfo{
				ImmutabilityExpiresOn: nil,
				LegalHold:             false,
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

func TestAzureImmutableBlobInfo_IsLocked(t *testing.T) {
	tests := []struct {
		name     string
		info     AzureImmutableBlobInfo
		expected bool
	}{
		{
			name: "locked policy",
			info: AzureImmutableBlobInfo{
				ImmutabilityPolicyMode: "Locked",
			},
			expected: true,
		},
		{
			name: "unlocked policy",
			info: AzureImmutableBlobInfo{
				ImmutabilityPolicyMode: "Unlocked",
			},
			expected: false,
		},
		{
			name: "no policy mode",
			info: AzureImmutableBlobInfo{
				ImmutabilityPolicyMode: "",
			},
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

func TestAzureImmutableBlobInfo_ProtectionScenarios(t *testing.T) {
	t.Run("blob lifecycle", func(t *testing.T) {
		// Day 0: Create blob with 30-day retention
		expiryDate := time.Now().AddDate(0, 0, 30)
		info := AzureImmutableBlobInfo{
			BlobName:               "backup-2025-01-01.sql",
			ImmutabilityExpiresOn:  &expiryDate,
			ImmutabilityPolicyMode: "Unlocked",
			LegalHold:              false,
		}

		// Should be protected
		assert.True(t, info.IsProtected())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 30, info.DaysUntilUnlock(), 1)
		assert.False(t, info.IsLocked())

		// Apply legal hold
		info.LegalHold = true
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())

		// Lock the policy
		info.ImmutabilityPolicyMode = "Locked"
		assert.True(t, info.IsLocked())

		// Remove legal hold (policy still protects)
		info.LegalHold = false
		assert.True(t, info.IsProtected())

		// Simulate retention expiration
		pastDate := time.Now().AddDate(0, 0, -1)
		info.ImmutabilityExpiresOn = &pastDate
		assert.False(t, info.IsProtected())
		assert.Equal(t, 0, info.DaysUntilUnlock())
	})

	t.Run("locked policy protection", func(t *testing.T) {
		expiryDate := time.Now().AddDate(0, 0, 90)
		info := AzureImmutableBlobInfo{
			BlobName:               "backup-locked.sql",
			ImmutabilityPolicyMode: "Locked",
			ImmutabilityExpiresOn:  &expiryDate,
			LegalHold:              false,
		}

		assert.True(t, info.IsProtected())
		assert.True(t, info.IsLocked())
		// Allow 1 day variance due to time precision
		assert.InDelta(t, 90, info.DaysUntilUnlock(), 1)
		assert.Equal(t, "Locked", info.ImmutabilityPolicyMode)
	})

	t.Run("append writes allowed", func(t *testing.T) {
		config := &ImmutableBlobConfig{
			ImmutabilityPeriodDays:     60,
			LegalHold:                  false,
			Mode:                       ImmutabilityPolicyModeUnlocked,
			AllowProtectedAppendWrites: true,
		}

		assert.True(t, config.AllowProtectedAppendWrites)
		assert.Equal(t, 60, config.ImmutabilityPeriodDays)
		assert.False(t, config.LegalHold)
		assert.Equal(t, ImmutabilityPolicyModeUnlocked, config.Mode)
	})
}

func TestImmutabilityPolicyModeValidation(t *testing.T) {
	t.Run("valid modes", func(t *testing.T) {
		validModes := []ImmutabilityPolicyMode{
			ImmutabilityPolicyModeUnlocked,
			ImmutabilityPolicyModeLocked,
		}

		for _, mode := range validModes {
			assert.NotEmpty(t, string(mode))
			assert.Contains(t, []string{"Unlocked", "Locked"}, string(mode))
		}
	})
}

func TestImmutableBlobConfig_Validation(t *testing.T) {
	t.Run("unlocked policy can be modified", func(t *testing.T) {
		config := &ImmutableBlobConfig{
			ImmutabilityPeriodDays:     30,
			Mode:                       ImmutabilityPolicyModeUnlocked,
			LegalHold:                  false,
			AllowProtectedAppendWrites: true,
		}

		assert.Equal(t, ImmutabilityPolicyModeUnlocked, config.Mode)
		assert.True(t, config.AllowProtectedAppendWrites)
		assert.Equal(t, 30, config.ImmutabilityPeriodDays)
		assert.False(t, config.LegalHold)
	})

	t.Run("locked policy cannot be modified", func(t *testing.T) {
		config := &ImmutableBlobConfig{
			ImmutabilityPeriodDays:     90,
			Mode:                       ImmutabilityPolicyModeLocked,
			LegalHold:                  false,
			AllowProtectedAppendWrites: false,
		}

		assert.Equal(t, ImmutabilityPolicyModeLocked, config.Mode)
		assert.False(t, config.AllowProtectedAppendWrites)
		assert.Equal(t, 90, config.ImmutabilityPeriodDays)
		assert.False(t, config.LegalHold)
	})
}

func TestAzureImmutableBlobInfo_SizeAndMetadata(t *testing.T) {
	t.Run("blob with metadata", func(t *testing.T) {
		lastModified := time.Now().AddDate(0, 0, -7)
		etag := azcore.ETag("\"0x8D9F5C8E9F8D0A1\"")

		info := AzureImmutableBlobInfo{
			BlobName:              "backup-100mb.sql",
			LastModified:          &lastModified,
			ETag:                  &etag,
			ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 30)),
		}

		assert.NotNil(t, info.LastModified)
		assert.NotNil(t, info.ETag)
		assert.Equal(t, "backup-100mb.sql", info.BlobName)
		assert.True(t, info.IsProtected())
	})

	t.Run("blob without metadata", func(t *testing.T) {
		info := AzureImmutableBlobInfo{
			BlobName:  "backup-minimal.sql",
			LegalHold: false,
		}

		assert.Nil(t, info.LastModified)
		assert.Nil(t, info.ETag)
		assert.Nil(t, info.ImmutabilityExpiresOn)
		assert.False(t, info.IsProtected())
	})
}

func TestContainerImmutabilityPolicy_Scenarios(t *testing.T) {
	t.Run("container with default retention", func(t *testing.T) {
		policy := &ContainerImmutabilityPolicy{
			ImmutabilityPeriodDays:        7,
			State:                         "Unlocked",
			AllowProtectedAppendWrites:    false,
			AllowProtectedAppendWritesAll: false,
		}

		assert.Equal(t, 7, policy.ImmutabilityPeriodDays)
		assert.Equal(t, "Unlocked", policy.State)
		assert.False(t, policy.AllowProtectedAppendWrites)
		assert.False(t, policy.AllowProtectedAppendWritesAll)
	})

	t.Run("container with locked retention", func(t *testing.T) {
		policy := &ContainerImmutabilityPolicy{
			ImmutabilityPeriodDays:        365,
			State:                         "Locked",
			AllowProtectedAppendWrites:    true,
			AllowProtectedAppendWritesAll: true,
		}

		assert.Equal(t, 365, policy.ImmutabilityPeriodDays)
		assert.Equal(t, "Locked", policy.State)
		assert.True(t, policy.AllowProtectedAppendWrites)
		assert.True(t, policy.AllowProtectedAppendWritesAll)
	})
}

func TestAzureImmutableBlobInfo_EdgeCases(t *testing.T) {
	t.Run("protection with nil expiry but legal hold", func(t *testing.T) {
		info := AzureImmutableBlobInfo{
			BlobName:              "test.sql",
			ImmutabilityExpiresOn: nil,
			LegalHold:             true,
		}

		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())
	})

	t.Run("expired policy with legal hold", func(t *testing.T) {
		pastDate := time.Now().AddDate(0, 0, -10)
		info := AzureImmutableBlobInfo{
			BlobName:              "test.sql",
			ImmutabilityExpiresOn: &pastDate,
			LegalHold:             true,
		}

		// Legal hold takes precedence
		assert.True(t, info.IsProtected())
		assert.Equal(t, -1, info.DaysUntilUnlock())
	})

	t.Run("exactly at expiry boundary", func(t *testing.T) {
		// Expiry is right now (within seconds)
		expiryDate := time.Now()
		info := AzureImmutableBlobInfo{
			BlobName:              "test.sql",
			ImmutabilityExpiresOn: &expiryDate,
			LegalHold:             false,
		}

		// Should be at the boundary (0 days or barely protected)
		days := info.DaysUntilUnlock()
		assert.True(t, days <= 1 && days >= 0, "Expected 0-1 days at boundary, got %d", days)
	})
}

// Helper function to create time pointers.
func timePtr(t time.Time) *time.Time {
	return &t
}

// Benchmark tests.
func BenchmarkAzureIsProtected(b *testing.B) {
	info := AzureImmutableBlobInfo{
		ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 30)),
		LegalHold:             false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.IsProtected()
	}
}

func BenchmarkAzureDaysUntilUnlock(b *testing.B) {
	info := AzureImmutableBlobInfo{
		ImmutabilityExpiresOn: timePtr(time.Now().AddDate(0, 0, 60)),
		LegalHold:             false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.DaysUntilUnlock()
	}
}

func BenchmarkAzureIsLocked(b *testing.B) {
	info := AzureImmutableBlobInfo{
		ImmutabilityPolicyMode: "Locked",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.IsLocked()
	}
}

// Integration tests would go here (require actual Azure Blob Storage with immutability enabled)
// These are commented out to avoid requiring Azure credentials in unit tests

/*
func TestAzureProvider_SetContainerImmutabilityPolicy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test Azure provider
	provider := setupTestAzureProvider(t)
	ctx := context.Background()

	// Test setting immutability policy
	err := provider.SetContainerImmutabilityPolicy(ctx, 30, false)
	require.NoError(t, err)

	// Verify policy is set
	// (Would need to check container configuration)
}

func TestAzureProvider_SetBlobImmutabilityPolicy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestAzureProvider(t)
	ctx := context.Background()

	blobName := "test-immutable.sql"
	expiryTime := time.Now().AddDate(0, 0, 30)

	// Set blob-level immutability policy
	err := provider.SetBlobImmutabilityPolicy(ctx, blobName, expiryTime, ImmutabilityPolicyModeUnlocked)
	require.NoError(t, err)

	// Verify policy
	info, err := provider.GetBlobImmutabilityPolicy(ctx, blobName)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.True(t, info.IsProtected())
}

func TestAzureProvider_LegalHold_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestAzureProvider(t)
	ctx := context.Background()

	blobName := "test-legal-hold.sql"

	// Set legal hold
	err := provider.SetBlobLegalHold(ctx, blobName, true)
	require.NoError(t, err)

	// Verify legal hold
	hasLegalHold, err := provider.GetBlobLegalHold(ctx, blobName)
	require.NoError(t, err)
	assert.True(t, hasLegalHold)

	// Remove legal hold
	err = provider.SetBlobLegalHold(ctx, blobName, false)
	require.NoError(t, err)

	// Verify legal hold removed
	hasLegalHold, err = provider.GetBlobLegalHold(ctx, blobName)
	require.NoError(t, err)
	assert.False(t, hasLegalHold)
}

func TestAzureProvider_LockImmutabilityPolicy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestAzureProvider(t)
	ctx := context.Background()

	// WARNING: This test will PERMANENTLY lock the policy
	// Only run in a dedicated test container that can be deleted

	// Set unlocked policy first
	err := provider.SetContainerImmutabilityPolicy(ctx, 1, false) // 1 day minimum
	require.NoError(t, err)

	// Lock the policy (IRREVERSIBLE!)
	err = provider.LockContainerImmutabilityPolicy(ctx)
	require.NoError(t, err)

	// Verify policy is locked
	// (Would need to verify via container properties)
}
*/
