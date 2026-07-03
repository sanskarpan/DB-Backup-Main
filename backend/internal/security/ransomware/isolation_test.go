package ransomware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIsolationManager(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy("/tmp/quarantine")

	im := NewIsolationManager(policy, detector)

	assert.NotNil(t, im)
	assert.NotNil(t, im.policy)
	assert.NotNil(t, im.detector)
	assert.NotNil(t, im.isolatedBackups)
}

func TestIsolationManager_ScanAndIsolate_NoThreat(t *testing.T) {
	// Create temp directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "safe_backup.sql")
	err := os.WriteFile(testFile, []byte("safe data"), 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	im := NewIsolationManager(policy, detector)

	ctx := context.Background()
	report, err := im.ScanAndIsolate(ctx, testFile)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, ThreatLevelNone, report.ThreatLevel)

	// File should not be isolated
	assert.False(t, im.IsBackupIsolated(testFile))

	// Original file should still exist
	_, err = os.Stat(testFile)
	assert.NoError(t, err)
}

func TestIsolationManager_ScanAndIsolate_HighEntropy(t *testing.T) {
	// Create temp directory and file with high entropy (random bytes)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "encrypted_backup.sql")

	// Create file with high entropy data (simulating encrypted content)
	highEntropyData := make([]byte, 1024)
	for i := range highEntropyData {
		highEntropyData[i] = byte(i % 256)
	}
	err := os.WriteFile(testFile, highEntropyData, 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	im := NewIsolationManager(policy, detector)

	ctx := context.Background()
	report, err := im.ScanAndIsolate(ctx, testFile)

	assert.NoError(t, err)
	assert.NotNil(t, report)

	// High entropy should be detected
	if report.ThreatLevel == ThreatLevelHigh || report.ThreatLevel == ThreatLevelCritical {
		// File should be isolated
		assert.True(t, im.IsBackupIsolated(testFile))
	}
}

func TestIsolationManager_MoveToQuarantine(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	im := NewIsolationManager(policy, detector)

	// Move to quarantine
	quarantinePath, err := im.moveToQuarantine(testFile)

	assert.NoError(t, err)
	assert.NotEmpty(t, quarantinePath)

	// Original file should not exist
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))

	// Quarantine file should exist
	_, err = os.Stat(quarantinePath)
	assert.NoError(t, err)

	// Quarantine directory should be created
	_, err = os.Stat(quarantineDir)
	assert.NoError(t, err)

	// Check file permissions (should be read-only)
	info, err := os.Stat(quarantinePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o400), info.Mode().Perm())
}

func TestIsolationManager_BlockAccess(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "blocked_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	// Block access
	err = im.blockAccess(testFile)
	assert.NoError(t, err)

	// Check permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o000), info.Mode().Perm())
}

func TestIsolationManager_ExecuteIsolation_Quarantine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	im := NewIsolationManager(policy, detector)

	report := &ThreatReport{
		ThreatLevel: ThreatLevelHigh,
		FilePath:    testFile,
		Description: "Test threat",
	}

	isolated, err := im.executeIsolation(testFile, report, IsolationActionQuarantine)

	assert.NoError(t, err)
	assert.NotNil(t, isolated)
	assert.Equal(t, IsolationActionQuarantine, isolated.Action)
	assert.NotEmpty(t, isolated.QuarantinePath)
	assert.Equal(t, testFile, isolated.OriginalPath)
}

func TestIsolationManager_ExecuteIsolation_Block(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	report := &ThreatReport{
		ThreatLevel: ThreatLevelMedium,
		FilePath:    testFile,
		Description: "Test threat",
	}

	isolated, err := im.executeIsolation(testFile, report, IsolationActionBlock)

	assert.NoError(t, err)
	assert.NotNil(t, isolated)
	assert.Equal(t, IsolationActionBlock, isolated.Action)

	// File should still exist but with no permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o000), info.Mode().Perm())
}

func TestIsolationManager_ExecuteIsolation_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	report := &ThreatReport{
		ThreatLevel: ThreatLevelCritical,
		FilePath:    testFile,
		Description: "Critical threat",
	}

	isolated, err := im.executeIsolation(testFile, report, IsolationActionDelete)

	assert.NoError(t, err)
	assert.NotNil(t, isolated)
	assert.Equal(t, IsolationActionDelete, isolated.Action)

	// File should not exist
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestIsolationManager_RestoreFromQuarantine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	im := NewIsolationManager(policy, detector)

	// Quarantine the file
	report := &ThreatReport{
		ThreatLevel: ThreatLevelHigh,
		FilePath:    testFile,
	}
	isolated, err := im.executeIsolation(testFile, report, IsolationActionQuarantine)
	require.NoError(t, err)

	// Restore from quarantine
	restorePath := filepath.Join(tmpDir, "restored_backup.sql")
	err = im.RestoreFromQuarantine(testFile, restorePath)

	assert.NoError(t, err)

	// Restored file should exist
	_, err = os.Stat(restorePath)
	assert.NoError(t, err)

	// Quarantine file should not exist
	_, err = os.Stat(isolated.QuarantinePath)
	assert.True(t, os.IsNotExist(err))

	// Backup should be marked as restored
	assert.True(t, isolated.Restored)
	assert.NotNil(t, isolated.RestoredAt)
}

func TestIsolationManager_RestoreFromQuarantine_NotQuarantined(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	// Block the file (not quarantine)
	report := &ThreatReport{
		ThreatLevel: ThreatLevelMedium,
		FilePath:    testFile,
	}
	_, err = im.executeIsolation(testFile, report, IsolationActionBlock)
	require.NoError(t, err)

	// Try to restore - should fail
	err = im.RestoreFromQuarantine(testFile, filepath.Join(tmpDir, "restored.sql"))
	assert.Error(t, err)
}

func TestIsolationManager_ListIsolatedBackups(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	// Isolate multiple backups
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, "backup"+string(rune('0'+i))+".sql")
		err := os.WriteFile(testFile, []byte("data"), 0o644)
		require.NoError(t, err)

		report := &ThreatReport{
			ThreatLevel: ThreatLevelHigh,
			FilePath:    testFile,
		}
		_, err = im.executeIsolation(testFile, report, IsolationActionQuarantine)
		require.NoError(t, err)
	}

	backups := im.ListIsolatedBackups()
	assert.Len(t, backups, 3)
}

func TestIsolationManager_GetIsolatedBackup(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	report := &ThreatReport{
		ThreatLevel: ThreatLevelHigh,
		FilePath:    testFile,
	}
	_, err = im.executeIsolation(testFile, report, IsolationActionQuarantine)
	require.NoError(t, err)

	// Get isolated backup
	isolated, err := im.GetIsolatedBackup(testFile)

	assert.NoError(t, err)
	assert.NotNil(t, isolated)
	assert.Equal(t, testFile, isolated.OriginalPath)
}

func TestIsolationManager_GetIsolatedBackup_NotFound(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy("/tmp/quarantine")
	im := NewIsolationManager(policy, detector)

	_, err := im.GetIsolatedBackup("/nonexistent/backup.sql")
	assert.Error(t, err)
}

func TestIsolationManager_CleanupOldQuarantine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "old_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	quarantineDir := filepath.Join(tmpDir, "quarantine")
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(quarantineDir)
	policy.IsolationRetentionDays = 1 // 1 day retention
	im := NewIsolationManager(policy, detector)

	// Quarantine the file
	report := &ThreatReport{
		ThreatLevel: ThreatLevelHigh,
		FilePath:    testFile,
	}
	isolated, err := im.executeIsolation(testFile, report, IsolationActionQuarantine)
	require.NoError(t, err)

	// Manually set isolation time to 2 days ago
	isolated.IsolatedAt = time.Now().AddDate(0, 0, -2)

	// Cleanup
	ctx := context.Background()
	err = im.CleanupOldQuarantine(ctx)

	assert.NoError(t, err)

	// Quarantine file should be deleted
	_, err = os.Stat(isolated.QuarantinePath)
	assert.True(t, os.IsNotExist(err))

	// Should not be in tracking anymore
	_, err = im.GetIsolatedBackup(testFile)
	assert.Error(t, err)
}

func TestIsolationManager_IsBackupIsolated(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")
	err := os.WriteFile(testFile, []byte("test data"), 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	// Initially not isolated
	assert.False(t, im.IsBackupIsolated(testFile))

	// Isolate
	report := &ThreatReport{
		ThreatLevel: ThreatLevelHigh,
		FilePath:    testFile,
	}
	_, err = im.executeIsolation(testFile, report, IsolationActionBlock)
	require.NoError(t, err)

	// Should be isolated
	assert.True(t, im.IsBackupIsolated(testFile))
}

func TestIsolationManager_GetIsolationStats(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	im := NewIsolationManager(policy, detector)

	// Isolate backups with different actions and threat levels
	testCases := []struct {
		action IsolationAction
		level  ThreatLevel
	}{
		{IsolationActionQuarantine, ThreatLevelCritical},
		{IsolationActionQuarantine, ThreatLevelHigh},
		{IsolationActionBlock, ThreatLevelMedium},
		{IsolationActionAlert, ThreatLevelLow},
	}

	for i, tc := range testCases {
		testFile := filepath.Join(tmpDir, "backup"+string(rune('0'+i))+".sql")
		err := os.WriteFile(testFile, []byte("data"), 0o644)
		require.NoError(t, err)

		report := &ThreatReport{
			ThreatLevel: tc.level,
			FilePath:    testFile,
		}
		_, err = im.executeIsolation(testFile, report, tc.action)
		require.NoError(t, err)
	}

	stats := im.GetIsolationStats()

	assert.Equal(t, 4, stats.TotalIsolated)
	assert.Equal(t, 2, stats.ByAction[IsolationActionQuarantine])
	assert.Equal(t, 1, stats.ByAction[IsolationActionBlock])
	assert.Equal(t, 1, stats.ByAction[IsolationActionAlert])
	assert.Equal(t, 1, stats.ByThreatLevel[ThreatLevelCritical])
	assert.Equal(t, 1, stats.ByThreatLevel[ThreatLevelHigh])
	assert.Equal(t, 1, stats.ByThreatLevel[ThreatLevelMedium])
	assert.Equal(t, 1, stats.ByThreatLevel[ThreatLevelLow])
	assert.Equal(t, 0, stats.Restored)
}

func TestIsolationManager_SetNotificationCallback(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy("/tmp/quarantine")
	im := NewIsolationManager(policy, detector)

	callback := func(isolated *IsolatedBackup) {
		// Callback function for testing
	}

	im.SetNotificationCallback(callback)

	assert.NotNil(t, im.onIsolation)
}

func TestIsolationPolicy_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_backup.sql")

	// Create file with high entropy
	highEntropyData := make([]byte, 1024)
	for i := range highEntropyData {
		highEntropyData[i] = byte(i % 256)
	}
	err := os.WriteFile(testFile, highEntropyData, 0o644)
	require.NoError(t, err)

	detector := NewDetector(DefaultDetectorConfig())
	policy := DefaultIsolationPolicy(tmpDir)
	policy.Enabled = false // Disable isolation
	im := NewIsolationManager(policy, detector)

	ctx := context.Background()
	report, err := im.ScanAndIsolate(ctx, testFile)

	// Should return nil when disabled
	assert.NoError(t, err)
	assert.Nil(t, report)

	// File should not be isolated
	assert.False(t, im.IsBackupIsolated(testFile))
}
