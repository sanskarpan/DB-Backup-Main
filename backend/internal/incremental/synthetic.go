package incremental

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// SyntheticBackupGenerator generates synthetic full backups.
type SyntheticBackupGenerator struct {
	mu       sync.RWMutex
	strategy *IncrementalForeverStrategy
	tempDir  string
}

// NewSyntheticBackupGenerator creates a new synthetic backup generator.
func NewSyntheticBackupGenerator(strategy *IncrementalForeverStrategy, tempDir string) *SyntheticBackupGenerator {
	return &SyntheticBackupGenerator{
		strategy: strategy,
		tempDir:  tempDir,
	}
}

// GenerateSyntheticFull generates a synthetic full backup from a backup chain.
func (sbg *SyntheticBackupGenerator) GenerateSyntheticFull(latestIncrementalID string) (*BackupManifest, error) {
	sbg.mu.Lock()
	defer sbg.mu.Unlock()

	// Build the backup chain
	chain, err := sbg.strategy.GetBackupChain(latestIncrementalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("empty backup chain")
	}

	baseManifest := chain[0]
	if baseManifest.Type != BackupTypeFull {
		return nil, fmt.Errorf("first backup in chain must be full backup")
	}

	// Create temporary restore file
	tempRestorePath := filepath.Join(sbg.tempDir, fmt.Sprintf("synthetic-restore-%s", uid.Hex(8)))
	defer os.Remove(tempRestorePath)

	// Restore to temporary location
	err = sbg.strategy.RestoreFromBackup(latestIncrementalID, tempRestorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to restore for synthetic: %w", err)
	}

	// Get latest incremental manifest
	latestManifest, exists := sbg.strategy.GetManifest(latestIncrementalID)
	if !exists {
		return nil, fmt.Errorf("latest manifest not found")
	}

	// Create synthetic full backup
	syntheticManifest := &BackupManifest{
		ID:           uid.New("synth"),
		Type:         BackupTypeSynthetic,
		DatabaseID:   latestManifest.DatabaseID,
		DatabaseName: latestManifest.DatabaseName,
		SourcePath:   latestManifest.SourcePath,
		BaseBackupID: baseManifest.BaseBackupID,
		BlockSize:    latestManifest.BlockSize,
		Compression:  latestManifest.Compression,
		Status:       "in_progress",
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	syntheticManifest.Metadata["derived_from_chain"] = len(chain)
	syntheticManifest.Metadata["base_backup_id"] = baseManifest.ID
	syntheticManifest.Metadata["latest_incremental_id"] = latestIncrementalID

	// Create backup path
	backupFileName := fmt.Sprintf("%s-%s.backup", syntheticManifest.ID, syntheticManifest.Type)
	syntheticManifest.BackupPath = filepath.Join(sbg.strategy.backupDir, backupFileName)

	// Copy restored file to backup location with compression
	err = sbg.strategy.copyWithCompression(tempRestorePath, syntheticManifest.BackupPath, syntheticManifest.Compression)
	if err != nil {
		syntheticManifest.Status = "failed"
		syntheticManifest.Error = err.Error()
		return syntheticManifest, fmt.Errorf("failed to create synthetic backup: %w", err)
	}

	// Create snapshot of synthetic backup
	snapshot, err := sbg.strategy.tracker.CreateSnapshot(tempRestorePath)
	if err != nil {
		syntheticManifest.Status = "failed"
		syntheticManifest.Error = err.Error()
		return syntheticManifest, fmt.Errorf("failed to create snapshot: %w", err)
	}

	syntheticManifest.SnapshotID = snapshot.ID
	syntheticManifest.FileSize = snapshot.FileSize
	syntheticManifest.TotalBlocks = snapshot.TotalBlocks
	syntheticManifest.Checksum = snapshot.Checksum

	// Get compressed size
	stat, err := os.Stat(syntheticManifest.BackupPath)
	if err == nil {
		syntheticManifest.CompressedSize = stat.Size()
	}

	syntheticManifest.CompletedAt = time.Now()
	syntheticManifest.Duration = syntheticManifest.CompletedAt.Sub(syntheticManifest.CreatedAt)
	syntheticManifest.Status = "completed"

	// Save manifest
	err = sbg.strategy.saveManifest(syntheticManifest)
	if err != nil {
		return syntheticManifest, fmt.Errorf("failed to save manifest: %w", err)
	}

	// Add to strategy's manifest map
	sbg.strategy.mu.Lock()
	sbg.strategy.manifests[syntheticManifest.ID] = syntheticManifest
	sbg.strategy.mu.Unlock()

	return syntheticManifest, nil
}

// ShouldGenerateSynthetic determines if a synthetic full should be generated.
func (sbg *SyntheticBackupGenerator) ShouldGenerateSynthetic(latestBackupID string, threshold int) (bool, error) {
	// Get backup chain
	chain, err := sbg.strategy.GetBackupChain(latestBackupID)
	if err != nil {
		return false, fmt.Errorf("failed to get backup chain: %w", err)
	}

	// Count incrementals in chain
	incrementalCount := 0
	for _, manifest := range chain {
		if manifest.Type == BackupTypeIncremental {
			incrementalCount++
		}
	}

	return incrementalCount >= threshold, nil
}

// GetSyntheticBackupRecommendation provides recommendation on synthetic backup.
func (sbg *SyntheticBackupGenerator) GetSyntheticBackupRecommendation(latestBackupID string) (map[string]interface{}, error) {
	chain, err := sbg.strategy.GetBackupChain(latestBackupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	recommendation := map[string]interface{}{
		"chain_length": len(chain),
	}

	if len(chain) == 0 {
		recommendation["recommended"] = false
		recommendation["reason"] = "No backup chain found"
		return recommendation, nil
	}

	incrementalCount := 0
	var baseManifest *BackupManifest

	for _, manifest := range chain {
		if manifest.Type == BackupTypeFull || manifest.Type == BackupTypeSynthetic {
			baseManifest = manifest
		}
		if manifest.Type == BackupTypeIncremental {
			incrementalCount++
		}
	}

	recommendation["incremental_count"] = incrementalCount
	recommendation["base_backup_age"] = time.Since(baseManifest.CreatedAt).Hours()

	// Recommendation logic
	if incrementalCount >= 10 {
		recommendation["recommended"] = true
		recommendation["reason"] = "Long incremental chain (10+)"
		recommendation["priority"] = "high"
	} else if incrementalCount >= 5 {
		recommendation["recommended"] = true
		recommendation["reason"] = "Moderate incremental chain (5+)"
		recommendation["priority"] = "medium"
	} else if time.Since(baseManifest.CreatedAt) > 7*24*time.Hour {
		recommendation["recommended"] = true
		recommendation["reason"] = "Base backup older than 7 days"
		recommendation["priority"] = "low"
	} else {
		recommendation["recommended"] = false
		recommendation["reason"] = "Chain is still optimal"
	}

	// Estimate synthetic backup size
	if len(chain) > 0 {
		latestManifest := chain[len(chain)-1]
		recommendation["estimated_size"] = latestManifest.FileSize
		recommendation["estimated_compressed_size"] = latestManifest.CompressedSize
	}

	return recommendation, nil
}

// VerifySyntheticBackup verifies a synthetic backup's integrity.
func (sbg *SyntheticBackupGenerator) VerifySyntheticBackup(syntheticID string) error {
	manifest, exists := sbg.strategy.GetManifest(syntheticID)
	if !exists {
		return fmt.Errorf("synthetic backup %s not found", syntheticID)
	}

	if manifest.Type != BackupTypeSynthetic {
		return fmt.Errorf("backup %s is not a synthetic backup", syntheticID)
	}

	// Verify backup file exists
	_, err := os.Stat(manifest.BackupPath)
	if err != nil {
		return fmt.Errorf("synthetic backup file not found: %w", err)
	}

	// Verify snapshot exists
	_, exists = sbg.strategy.tracker.GetSnapshot(manifest.SnapshotID)
	if !exists {
		return fmt.Errorf("snapshot %s not found", manifest.SnapshotID)
	}

	return nil
}

// ConvertToBase converts a synthetic full backup to a base backup for new incrementals.
func (sbg *SyntheticBackupGenerator) ConvertToBase(syntheticID string) error {
	sbg.strategy.mu.Lock()
	defer sbg.strategy.mu.Unlock()

	manifest, exists := sbg.strategy.manifests[syntheticID]
	if !exists {
		return fmt.Errorf("synthetic backup %s not found", syntheticID)
	}

	if manifest.Type != BackupTypeSynthetic {
		return fmt.Errorf("backup %s is not a synthetic backup", syntheticID)
	}

	// Update manifest to make it a base backup
	manifest.Type = BackupTypeFull
	manifest.BaseBackupID = manifest.ID
	manifest.Metadata["converted_from_synthetic"] = true
	manifest.Metadata["converted_at"] = time.Now()

	// Save updated manifest
	err := sbg.strategy.saveManifest(manifest)
	if err != nil {
		return fmt.Errorf("failed to save updated manifest: %w", err)
	}

	return nil
}

// CleanupOldChain removes old backups after synthetic full is created.
func (sbg *SyntheticBackupGenerator) CleanupOldChain(syntheticID string, keepLatestN int) ([]string, error) {
	sbg.strategy.mu.Lock()
	defer sbg.strategy.mu.Unlock()

	manifest, exists := sbg.strategy.manifests[syntheticID]
	if !exists {
		return nil, fmt.Errorf("synthetic backup %s not found", syntheticID)
	}

	if manifest.Type != BackupTypeSynthetic {
		return nil, fmt.Errorf("backup %s is not a synthetic backup", syntheticID)
	}

	// Get the chain that was used to create this synthetic
	latestIncrID, ok := manifest.Metadata["latest_incremental_id"].(string)
	if !ok {
		return nil, fmt.Errorf("cannot determine source chain")
	}

	// Build the old chain (this will be unlocked above, so we need to access strategy without lock)
	sbg.strategy.mu.Unlock()
	chain, err := sbg.strategy.GetBackupChain(latestIncrID)
	sbg.strategy.mu.Lock()

	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	removed := make([]string, 0)

	// Keep the latest N incrementals, remove the rest including the old base
	if len(chain) > keepLatestN {
		for i := 0; i < len(chain)-keepLatestN; i++ {
			oldManifest := chain[i]

			// Remove backup file
			err := os.Remove(oldManifest.BackupPath)
			if err != nil && !os.IsNotExist(err) {
				return removed, fmt.Errorf("failed to remove backup file %s: %w", oldManifest.BackupPath, err)
			}

			// Remove manifest file
			manifestPath := filepath.Join(sbg.strategy.backupDir, fmt.Sprintf("%s-manifest.json", oldManifest.ID))
			err = os.Remove(manifestPath)
			if err != nil && !os.IsNotExist(err) {
				return removed, fmt.Errorf("failed to remove manifest file %s: %w", manifestPath, err)
			}

			// Remove from map
			delete(sbg.strategy.manifests, oldManifest.ID)
			removed = append(removed, oldManifest.ID)
		}
	}

	return removed, nil
}

// EstimateSyntheticCreationTime estimates how long it will take to create a synthetic.
func (sbg *SyntheticBackupGenerator) EstimateSyntheticCreationTime(latestBackupID string) (time.Duration, error) {
	chain, err := sbg.strategy.GetBackupChain(latestBackupID)
	if err != nil {
		return 0, fmt.Errorf("failed to get backup chain: %w", err)
	}

	if len(chain) == 0 {
		return 0, fmt.Errorf("empty backup chain")
	}

	// Get latest manifest to estimate file size
	latestManifest := chain[len(chain)-1]
	fileSize := latestManifest.FileSize

	// Estimate based on file size and assumed I/O speed
	// Assume 100 MB/s read + write speed
	ioSpeed := int64(100 * 1024 * 1024) // 100 MB/s

	// Need to read from base, apply incrementals, write synthetic
	// Estimate 2x file size worth of I/O
	totalIO := fileSize * 2
	estimatedSeconds := totalIO / ioSpeed

	// Add overhead for decompression, compression, snapshot creation
	overhead := time.Duration(10+len(chain)*2) * time.Second

	return time.Duration(estimatedSeconds)*time.Second + overhead, nil
}

// GetSyntheticBackups returns all synthetic backups.
func (sbg *SyntheticBackupGenerator) GetSyntheticBackups() []*BackupManifest {
	sbg.strategy.mu.RLock()
	defer sbg.strategy.mu.RUnlock()

	synthetics := make([]*BackupManifest, 0)
	for _, manifest := range sbg.strategy.manifests {
		if manifest.Type == BackupTypeSynthetic {
			synthetics = append(synthetics, manifest)
		}
	}

	return synthetics
}

// ScheduleSyntheticBackup schedules a synthetic backup to be created.
func (sbg *SyntheticBackupGenerator) ScheduleSyntheticBackup(latestBackupID string, scheduleTime time.Time) (*SyntheticSchedule, error) {
	schedule := &SyntheticSchedule{
		ID:             uid.New("sched"),
		LatestBackupID: latestBackupID,
		ScheduledTime:  scheduleTime,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	return schedule, nil
}

// SyntheticSchedule represents a scheduled synthetic backup.
type SyntheticSchedule struct {
	ID             string
	LatestBackupID string
	ScheduledTime  time.Time
	Status         string // "pending", "running", "completed", "failed"
	SyntheticID    string // ID of created synthetic backup
	Error          string
	CreatedAt      time.Time
	CompletedAt    time.Time
}

// GetStatistics returns statistics about synthetic backups.
func (sbg *SyntheticBackupGenerator) GetStatistics() map[string]interface{} {
	synthetics := sbg.GetSyntheticBackups()

	totalSize := int64(0)
	totalCompressedSize := int64(0)

	for _, manifest := range synthetics {
		totalSize += manifest.FileSize
		totalCompressedSize += manifest.CompressedSize
	}

	stats := map[string]interface{}{
		"total_synthetic_backups": len(synthetics),
		"total_size":              totalSize,
		"total_compressed_size":   totalCompressedSize,
	}

	if totalSize > 0 {
		stats["compression_ratio"] = float64(totalCompressedSize) / float64(totalSize) * 100
	}

	return stats
}
