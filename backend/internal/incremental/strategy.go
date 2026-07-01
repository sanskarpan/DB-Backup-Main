package incremental

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"        // Full backup
	BackupTypeIncremental BackupType = "incremental" // Incremental backup
	BackupTypeSynthetic   BackupType = "synthetic"   // Synthetic full backup
)

// BackupManifest represents metadata for a backup
type BackupManifest struct {
	ID             string
	Type           BackupType
	DatabaseID     string
	DatabaseName   string
	SourcePath     string
	BackupPath     string
	ParentBackupID string // ID of parent backup (for incrementals)
	BaseBackupID   string // ID of the base full backup
	SnapshotID     string // ID of the snapshot used
	ChangeSetID    string // ID of the change set (for incrementals)
	FileSize       int64
	CompressedSize int64
	BlockSize      BlockSize
	TotalBlocks    int64
	ChangedBlocks  int64
	Compression    string
	Checksum       string
	CreatedAt      time.Time
	CompletedAt    time.Time
	Duration       time.Duration
	Status         string // "pending", "in_progress", "completed", "failed"
	Error          string
	Metadata       map[string]interface{}
}

// BackupBlock represents a backed up block with its data
type BackupBlock struct {
	Offset   int64
	Size     int64
	Checksum string
	Data     []byte
}

// IncrementalForeverStrategy implements the incremental forever backup strategy
type IncrementalForeverStrategy struct {
	mu                 sync.RWMutex
	tracker            *ChangeTracker
	manifests          map[string]*BackupManifest
	backupDir          string
	compressionEnabled bool
	blockSize          BlockSize
}

// snapshotsSubdir is the sub-directory under backupDir where block snapshots are
// persisted. Keeping snapshots in their own directory keeps them from colliding
// with the "*-manifest.json" / "*.backup" files in backupDir.
const snapshotsSubdir = "snapshots"

// NewIncrementalForeverStrategy creates a new incremental forever strategy.
//
// State is durable: block snapshots are persisted under backupDir/snapshots and
// backup manifests under backupDir. Any previously persisted snapshots and
// manifests are loaded on construction so that incremental chains, retention and
// restore survive process restarts.
func NewIncrementalForeverStrategy(backupDir string, blockSize BlockSize, enableCompression bool) *IncrementalForeverStrategy {
	snapshotDir := filepath.Join(backupDir, snapshotsSubdir)

	tracker, err := NewChangeTrackerWithDir(blockSize, snapshotDir)
	if err != nil {
		// If persistence cannot be set up (e.g. unwritable path) fall back to an
		// in-memory tracker rather than failing construction.
		tracker = NewChangeTracker(blockSize)
	}

	ifs := &IncrementalForeverStrategy{
		tracker:            tracker,
		manifests:          make(map[string]*BackupManifest),
		backupDir:          backupDir,
		compressionEnabled: enableCompression,
		blockSize:          blockSize,
	}

	// Reload persisted manifests so the in-memory map reflects prior runs.
	ifs.loadManifests()

	return ifs
}

// loadManifests scans backupDir for persisted manifest files and loads them into
// the in-memory map by actually calling LoadManifest. Corrupt or unreadable
// manifest files are skipped so a single bad file does not block startup.
func (ifs *IncrementalForeverStrategy) loadManifests() {
	entries, err := os.ReadDir(ifs.backupDir)
	if err != nil {
		return
	}

	ifs.mu.Lock()
	defer ifs.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-manifest.json") {
			continue
		}

		manifest, err := ifs.LoadManifest(filepath.Join(ifs.backupDir, entry.Name()))
		if err != nil {
			continue
		}

		ifs.manifests[manifest.ID] = manifest
	}
}

// RemoveBackup deletes a backup's persisted artifacts (data file, manifest file
// and associated snapshot) as well as its in-memory entry. It returns a real
// error if any artifact cannot be removed so callers never treat a failed
// deletion as success.
func (ifs *IncrementalForeverStrategy) RemoveBackup(backupID string) error {
	ifs.mu.Lock()
	manifest, exists := ifs.manifests[backupID]
	if !exists {
		ifs.mu.Unlock()
		return fmt.Errorf("backup %s not found", backupID)
	}
	backupPath := manifest.BackupPath
	snapshotID := manifest.SnapshotID
	ifs.mu.Unlock()

	// Remove backup data file.
	if backupPath != "" {
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove backup file %s: %w", backupPath, err)
		}
	}

	// Remove manifest file.
	manifestPath := filepath.Join(ifs.backupDir, fmt.Sprintf("%s-manifest.json", backupID))
	if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove manifest file %s: %w", manifestPath, err)
	}

	// Remove the associated snapshot artifact.
	if snapshotID != "" {
		if err := ifs.tracker.RemoveSnapshot(snapshotID); err != nil {
			return err
		}
	}

	// Only drop the in-memory entry once on-disk artifacts are gone.
	ifs.mu.Lock()
	delete(ifs.manifests, backupID)
	ifs.mu.Unlock()

	return nil
}

// PerformFullBackup performs an initial full backup
func (ifs *IncrementalForeverStrategy) PerformFullBackup(sourcePath, databaseID, databaseName string) (*BackupManifest, error) {
	manifest := &BackupManifest{
		ID:           uid.New("full"),
		Type:         BackupTypeFull,
		DatabaseID:   databaseID,
		DatabaseName: databaseName,
		SourcePath:   sourcePath,
		BlockSize:    ifs.blockSize,
		Compression:  "none",
		Status:       "in_progress",
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	if ifs.compressionEnabled {
		manifest.Compression = "gzip"
	}

	// Create backup path
	backupFileName := fmt.Sprintf("%s-%s.backup", manifest.ID, manifest.Type)
	manifest.BackupPath = filepath.Join(ifs.backupDir, backupFileName)

	// Create snapshot
	snapshot, err := ifs.tracker.CreateSnapshot(sourcePath)
	if err != nil {
		manifest.Status = "failed"
		manifest.Error = err.Error()
		return manifest, fmt.Errorf("failed to create snapshot: %w", err)
	}

	manifest.SnapshotID = snapshot.ID
	manifest.FileSize = snapshot.FileSize
	manifest.TotalBlocks = snapshot.TotalBlocks
	manifest.Checksum = snapshot.Checksum
	manifest.BaseBackupID = manifest.ID

	// Copy file to backup location
	err = ifs.copyWithCompression(sourcePath, manifest.BackupPath, manifest.Compression)
	if err != nil {
		manifest.Status = "failed"
		manifest.Error = err.Error()
		return manifest, fmt.Errorf("failed to copy file: %w", err)
	}

	// Get compressed size
	stat, err := os.Stat(manifest.BackupPath)
	if err == nil {
		manifest.CompressedSize = stat.Size()
	}

	manifest.CompletedAt = time.Now()
	manifest.Duration = manifest.CompletedAt.Sub(manifest.CreatedAt)
	manifest.Status = "completed"

	// Save manifest
	err = ifs.saveManifest(manifest)
	if err != nil {
		return manifest, fmt.Errorf("failed to save manifest: %w", err)
	}

	ifs.mu.Lock()
	ifs.manifests[manifest.ID] = manifest
	ifs.mu.Unlock()

	return manifest, nil
}

// PerformIncrementalBackup performs an incremental backup
func (ifs *IncrementalForeverStrategy) PerformIncrementalBackup(sourcePath, databaseID, databaseName string, baseBackupID string) (*BackupManifest, error) {
	// Find base backup
	ifs.mu.RLock()
	baseManifest, exists := ifs.manifests[baseBackupID]
	ifs.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("base backup %s not found", baseBackupID)
	}

	manifest := &BackupManifest{
		ID:             uid.New("incr"),
		Type:           BackupTypeIncremental,
		DatabaseID:     databaseID,
		DatabaseName:   databaseName,
		SourcePath:     sourcePath,
		ParentBackupID: baseBackupID,
		BaseBackupID:   baseManifest.BaseBackupID,
		BlockSize:      ifs.blockSize,
		Compression:    "none",
		Status:         "in_progress",
		CreatedAt:      time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	if ifs.compressionEnabled {
		manifest.Compression = "gzip"
	}

	// Create backup path
	backupFileName := fmt.Sprintf("%s-%s.backup", manifest.ID, manifest.Type)
	manifest.BackupPath = filepath.Join(ifs.backupDir, backupFileName)

	// Detect changes
	changeSet, err := ifs.tracker.DetectChanges(sourcePath, baseManifest.SnapshotID)
	if err != nil {
		manifest.Status = "failed"
		manifest.Error = err.Error()
		return manifest, fmt.Errorf("failed to detect changes: %w", err)
	}

	manifest.ChangeSetID = changeSet.ID
	manifest.SnapshotID = changeSet.NewSnapshot
	manifest.ChangedBlocks = int64(len(changeSet.ChangedBlocks) + len(changeSet.NewBlocks))

	// Get new snapshot
	newSnapshot, _ := ifs.tracker.GetSnapshot(changeSet.NewSnapshot)
	if newSnapshot != nil {
		manifest.FileSize = newSnapshot.FileSize
		manifest.TotalBlocks = newSnapshot.TotalBlocks
		manifest.Checksum = newSnapshot.Checksum
	}

	// Extract changed block data
	changedData, err := ifs.tracker.GetChangedBlockData(sourcePath, changeSet)
	if err != nil {
		manifest.Status = "failed"
		manifest.Error = err.Error()
		return manifest, fmt.Errorf("failed to get changed blocks: %w", err)
	}

	// Save incremental backup (changed blocks only)
	err = ifs.saveIncrementalBackup(manifest.BackupPath, changedData, changeSet, manifest.Compression)
	if err != nil {
		manifest.Status = "failed"
		manifest.Error = err.Error()
		return manifest, fmt.Errorf("failed to save incremental backup: %w", err)
	}

	// Get compressed size
	stat, err := os.Stat(manifest.BackupPath)
	if err == nil {
		manifest.CompressedSize = stat.Size()
	}

	manifest.CompletedAt = time.Now()
	manifest.Duration = manifest.CompletedAt.Sub(manifest.CreatedAt)
	manifest.Status = "completed"

	// Save manifest
	err = ifs.saveManifest(manifest)
	if err != nil {
		return manifest, fmt.Errorf("failed to save manifest: %w", err)
	}

	ifs.mu.Lock()
	ifs.manifests[manifest.ID] = manifest
	ifs.mu.Unlock()

	return manifest, nil
}

// RestoreFromBackup restores data from a backup chain
func (ifs *IncrementalForeverStrategy) RestoreFromBackup(backupID, targetPath string) error {
	// Build restore chain
	chain, err := ifs.buildRestoreChain(backupID)
	if err != nil {
		return fmt.Errorf("failed to build restore chain: %w", err)
	}

	// Start with base full backup
	baseManifest := chain[0]

	// Extract base backup to target
	err = ifs.extractWithDecompression(baseManifest.BackupPath, targetPath, baseManifest.Compression)
	if err != nil {
		return fmt.Errorf("failed to extract base backup: %w", err)
	}

	// Apply incrementals in order
	for i := 1; i < len(chain); i++ {
		incrManifest := chain[i]

		err = ifs.applyIncrementalBackup(targetPath, incrManifest)
		if err != nil {
			return fmt.Errorf("failed to apply incremental backup %s: %w", incrManifest.ID, err)
		}
	}

	return nil
}

// buildRestoreChain builds the chain of backups needed for restore
func (ifs *IncrementalForeverStrategy) buildRestoreChain(backupID string) ([]*BackupManifest, error) {
	ifs.mu.RLock()
	defer ifs.mu.RUnlock()

	manifest, exists := ifs.manifests[backupID]
	if !exists {
		return nil, fmt.Errorf("backup %s not found", backupID)
	}

	chain := make([]*BackupManifest, 0)

	// Walk backwards to base backup
	current := manifest
	for current != nil {
		chain = append([]*BackupManifest{current}, chain...)

		if current.Type == BackupTypeFull {
			break
		}

		if current.ParentBackupID == "" {
			return nil, fmt.Errorf("incomplete backup chain: no parent for %s", current.ID)
		}

		parent, exists := ifs.manifests[current.ParentBackupID]
		if !exists {
			return nil, fmt.Errorf("parent backup %s not found", current.ParentBackupID)
		}

		current = parent
	}

	return chain, nil
}

// applyIncrementalBackup applies an incremental backup to a file
func (ifs *IncrementalForeverStrategy) applyIncrementalBackup(filePath string, manifest *BackupManifest) error {
	// Load incremental backup data
	backupData, err := ifs.loadIncrementalBackup(manifest.BackupPath, manifest.Compression)
	if err != nil {
		return fmt.Errorf("failed to load incremental backup: %w", err)
	}

	// Open target file for writing
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open target file: %w", err)
	}
	defer file.Close()

	// Apply each changed block
	for offset, data := range backupData {
		_, err := file.Seek(offset, 0)
		if err != nil {
			return fmt.Errorf("failed to seek to offset %d: %w", offset, err)
		}

		_, err = file.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write block at offset %d: %w", offset, err)
		}
	}

	return nil
}

// saveIncrementalBackup saves an incremental backup
func (ifs *IncrementalForeverStrategy) saveIncrementalBackup(backupPath string, blockData map[int64][]byte, changeSet *ChangeSet, compression string) error {
	// Create backup structure
	backup := struct {
		ChangeSetID   string
		ChangedBlocks []BackupBlock
		NewBlocks     []BackupBlock
	}{
		ChangeSetID:   changeSet.ID,
		ChangedBlocks: make([]BackupBlock, 0),
		NewBlocks:     make([]BackupBlock, 0),
	}

	// Add changed blocks
	for _, block := range changeSet.ChangedBlocks {
		data, exists := blockData[block.Offset]
		if exists {
			backup.ChangedBlocks = append(backup.ChangedBlocks, BackupBlock{
				Offset:   block.Offset,
				Size:     block.Size,
				Checksum: block.Checksum,
				Data:     data,
			})
		}
	}

	// Add new blocks
	for _, block := range changeSet.NewBlocks {
		data, exists := blockData[block.Offset]
		if exists {
			backup.NewBlocks = append(backup.NewBlocks, BackupBlock{
				Offset:   block.Offset,
				Size:     block.Size,
				Checksum: block.Checksum,
				Data:     data,
			})
		}
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(backup)
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	// Write to file with optional compression
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	if compression == "gzip" {
		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()

		_, err = gzipWriter.Write(jsonData)
		if err != nil {
			return fmt.Errorf("failed to write compressed data: %w", err)
		}
	} else {
		_, err = file.Write(jsonData)
		if err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
	}

	return nil
}

// loadIncrementalBackup loads an incremental backup
func (ifs *IncrementalForeverStrategy) loadIncrementalBackup(backupPath string, compression string) (map[int64][]byte, error) {
	file, err := os.Open(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	if compression == "gzip" {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Read JSON data
	var backup struct {
		ChangeSetID   string
		ChangedBlocks []BackupBlock
		NewBlocks     []BackupBlock
	}

	decoder := json.NewDecoder(reader)
	err = decoder.Decode(&backup)
	if err != nil {
		return nil, fmt.Errorf("failed to decode backup data: %w", err)
	}

	// Convert to map
	blockData := make(map[int64][]byte)

	for _, block := range backup.ChangedBlocks {
		blockData[block.Offset] = block.Data
	}

	for _, block := range backup.NewBlocks {
		blockData[block.Offset] = block.Data
	}

	return blockData, nil
}

// copyWithCompression copies a file with optional compression
func (ifs *IncrementalForeverStrategy) copyWithCompression(sourcePath, destPath, compression string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if compression == "gzip" {
		gzipWriter := gzip.NewWriter(destFile)
		defer gzipWriter.Close()

		_, err = io.Copy(gzipWriter, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to copy with compression: %w", err)
		}
	} else {
		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to copy: %w", err)
		}
	}

	return nil
}

// extractWithDecompression extracts a file with optional decompression
func (ifs *IncrementalForeverStrategy) extractWithDecompression(sourcePath, destPath, compression string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if compression == "gzip" {
		gzipReader, err := gzip.NewReader(sourceFile)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()

		_, err = io.Copy(destFile, gzipReader)
		if err != nil {
			return fmt.Errorf("failed to extract with decompression: %w", err)
		}
	} else {
		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return fmt.Errorf("failed to extract: %w", err)
		}
	}

	return nil
}

// saveManifest saves a backup manifest to disk
func (ifs *IncrementalForeverStrategy) saveManifest(manifest *BackupManifest) error {
	manifestPath := filepath.Join(ifs.backupDir, fmt.Sprintf("%s-manifest.json", manifest.ID))

	if err := writeJSONAtomic(manifestPath, manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// LoadManifest loads a backup manifest from disk
func (ifs *IncrementalForeverStrategy) LoadManifest(manifestPath string) (*BackupManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest BackupManifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// GetManifest retrieves a backup manifest by ID
func (ifs *IncrementalForeverStrategy) GetManifest(backupID string) (*BackupManifest, bool) {
	ifs.mu.RLock()
	defer ifs.mu.RUnlock()

	manifest, exists := ifs.manifests[backupID]
	return manifest, exists
}

// ListBackups lists all backup manifests
func (ifs *IncrementalForeverStrategy) ListBackups() []*BackupManifest {
	ifs.mu.RLock()
	defer ifs.mu.RUnlock()

	manifests := make([]*BackupManifest, 0, len(ifs.manifests))
	for _, manifest := range ifs.manifests {
		manifests = append(manifests, manifest)
	}

	return manifests
}

// GetBackupChain returns the backup chain for a given backup
func (ifs *IncrementalForeverStrategy) GetBackupChain(backupID string) ([]*BackupManifest, error) {
	return ifs.buildRestoreChain(backupID)
}

// VerifyBackup verifies the integrity of a backup
func (ifs *IncrementalForeverStrategy) VerifyBackup(backupID string) error {
	manifest, exists := ifs.GetManifest(backupID)
	if !exists {
		return fmt.Errorf("backup %s not found", backupID)
	}

	// Check if backup file exists
	_, err := os.Stat(manifest.BackupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// For incrementals, verify we can load the data
	if manifest.Type == BackupTypeIncremental {
		_, err := ifs.loadIncrementalBackup(manifest.BackupPath, manifest.Compression)
		if err != nil {
			return fmt.Errorf("failed to load incremental backup: %w", err)
		}
	}

	return nil
}

// GetStatistics returns statistics about backups
func (ifs *IncrementalForeverStrategy) GetStatistics() map[string]interface{} {
	ifs.mu.RLock()
	defer ifs.mu.RUnlock()

	fullCount := 0
	incrCount := 0
	totalSize := int64(0)
	compressedSize := int64(0)

	for _, manifest := range ifs.manifests {
		switch manifest.Type {
		case BackupTypeFull:
			fullCount++
		case BackupTypeIncremental:
			incrCount++
		}
		totalSize += manifest.FileSize
		compressedSize += manifest.CompressedSize
	}

	stats := map[string]interface{}{
		"total_backups":       len(ifs.manifests),
		"full_backups":        fullCount,
		"incremental_backups": incrCount,
		"total_size":          totalSize,
		"compressed_size":     compressedSize,
		"block_size":          ifs.blockSize,
		"compression_enabled": ifs.compressionEnabled,
	}

	if totalSize > 0 {
		stats["compression_ratio"] = float64(compressedSize) / float64(totalSize) * 100
	}

	return stats
}
