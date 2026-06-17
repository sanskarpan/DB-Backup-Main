package incremental

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// BlockSize represents different block sizes for change tracking
type BlockSize int

const (
	BlockSize4KB  BlockSize = 4096       // 4 KB blocks
	BlockSize8KB  BlockSize = 8192       // 8 KB blocks
	BlockSize16KB BlockSize = 16384      // 16 KB blocks
	BlockSize64KB BlockSize = 65536      // 64 KB blocks
	BlockSize1MB  BlockSize = 1048576    // 1 MB blocks
)

// BlockMetadata represents metadata for a single block
type BlockMetadata struct {
	ID         string    // Unique block identifier
	Offset     int64     // Byte offset in the file
	Size       int64     // Block size in bytes
	Checksum   string    // SHA-256 checksum
	Changed    bool      // Whether block changed compared to previous snapshot
	FirstSeen  time.Time // When this block was first seen
	LastSeen   time.Time // When this block was last seen
	ModifiedAt time.Time // When this block was last modified
}

// FileSnapshot represents a snapshot of a file at a point in time
type FileSnapshot struct {
	ID           string
	FilePath     string
	FileSize     int64
	BlockSize    BlockSize
	TotalBlocks  int64
	Blocks       []*BlockMetadata
	Checksum     string // Overall file checksum
	CreatedAt    time.Time
	ModifiedTime time.Time
}

// ChangeSet represents the set of changes between two snapshots
type ChangeSet struct {
	ID            string
	BaseSnapshot  string // ID of the base snapshot
	NewSnapshot   string // ID of the new snapshot
	ChangedBlocks []*BlockMetadata
	NewBlocks     []*BlockMetadata
	DeletedBlocks []*BlockMetadata
	TotalChanged  int64 // Total bytes changed
	TotalNew      int64 // Total bytes new
	TotalDeleted  int64 // Total bytes deleted
	ChangeRatio   float64
	CreatedAt     time.Time
}

// ChangeTracker tracks block-level changes for incremental backups
type ChangeTracker struct {
	mu        sync.RWMutex
	snapshots map[string]*FileSnapshot
	changeSets map[string]*ChangeSet
	blockSize BlockSize
}

// NewChangeTracker creates a new change tracker
func NewChangeTracker(blockSize BlockSize) *ChangeTracker {
	return &ChangeTracker{
		snapshots:  make(map[string]*FileSnapshot),
		changeSets: make(map[string]*ChangeSet),
		blockSize:  blockSize,
	}
}

// CreateSnapshot creates a snapshot of a file's blocks
func (ct *ChangeTracker) CreateSnapshot(filePath string) (*FileSnapshot, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := stat.Size()
	totalBlocks := (fileSize + int64(ct.blockSize) - 1) / int64(ct.blockSize)

	snapshot := &FileSnapshot{
		ID:           uid.New("snap"),
		FilePath:     filePath,
		FileSize:     fileSize,
		BlockSize:    ct.blockSize,
		TotalBlocks:  totalBlocks,
		Blocks:       make([]*BlockMetadata, 0, totalBlocks),
		CreatedAt:    time.Now(),
		ModifiedTime: stat.ModTime(),
	}

	// Read file in blocks and compute checksums
	buffer := make([]byte, ct.blockSize)
	offset := int64(0)
	blockIndex := 0

	// Also compute overall file checksum
	fileHasher := sha256.New()

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Compute block checksum
		blockHasher := sha256.New()
		blockHasher.Write(buffer[:n])
		checksum := hex.EncodeToString(blockHasher.Sum(nil))

		// Update file checksum
		fileHasher.Write(buffer[:n])

		block := &BlockMetadata{
			ID:         fmt.Sprintf("%s-block-%d", snapshot.ID, blockIndex),
			Offset:     offset,
			Size:       int64(n),
			Checksum:   checksum,
			Changed:    false,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			ModifiedAt: time.Now(),
		}

		snapshot.Blocks = append(snapshot.Blocks, block)

		offset += int64(n)
		blockIndex++
	}

	snapshot.Checksum = hex.EncodeToString(fileHasher.Sum(nil))

	// Store snapshot
	ct.mu.Lock()
	ct.snapshots[snapshot.ID] = snapshot
	ct.mu.Unlock()

	return snapshot, nil
}

// CompareSnapshots compares two snapshots and creates a change set
func (ct *ChangeTracker) CompareSnapshots(baseSnapshotID, newSnapshotID string) (*ChangeSet, error) {
	ct.mu.RLock()
	baseSnapshot, baseExists := ct.snapshots[baseSnapshotID]
	newSnapshot, newExists := ct.snapshots[newSnapshotID]
	ct.mu.RUnlock()

	if !baseExists {
		return nil, fmt.Errorf("base snapshot %s not found", baseSnapshotID)
	}

	if !newExists {
		return nil, fmt.Errorf("new snapshot %s not found", newSnapshotID)
	}

	changeSet := &ChangeSet{
		ID:            uid.New("changeset"),
		BaseSnapshot:  baseSnapshotID,
		NewSnapshot:   newSnapshotID,
		ChangedBlocks: make([]*BlockMetadata, 0),
		NewBlocks:     make([]*BlockMetadata, 0),
		DeletedBlocks: make([]*BlockMetadata, 0),
		CreatedAt:     time.Now(),
	}

	// Build a map of base blocks by offset for quick lookup
	baseBlockMap := make(map[int64]*BlockMetadata)
	for _, block := range baseSnapshot.Blocks {
		baseBlockMap[block.Offset] = block
	}

	// Compare new blocks against base
	for _, newBlock := range newSnapshot.Blocks {
		baseBlock, exists := baseBlockMap[newBlock.Offset]

		if !exists {
			// This is a new block (file grew)
			newBlock.Changed = true
			changeSet.NewBlocks = append(changeSet.NewBlocks, newBlock)
			changeSet.TotalNew += newBlock.Size
		} else if newBlock.Checksum != baseBlock.Checksum {
			// Block changed
			newBlock.Changed = true
			changeSet.ChangedBlocks = append(changeSet.ChangedBlocks, newBlock)
			changeSet.TotalChanged += newBlock.Size
			delete(baseBlockMap, newBlock.Offset)
		} else {
			// Block unchanged
			newBlock.Changed = false
			delete(baseBlockMap, newBlock.Offset)
		}
	}

	// Remaining blocks in baseBlockMap are deleted blocks
	for _, deletedBlock := range baseBlockMap {
		changeSet.DeletedBlocks = append(changeSet.DeletedBlocks, deletedBlock)
		changeSet.TotalDeleted += deletedBlock.Size
	}

	// Calculate change ratio
	totalBytes := newSnapshot.FileSize
	if totalBytes > 0 {
		changeSet.ChangeRatio = float64(changeSet.TotalChanged+changeSet.TotalNew) / float64(totalBytes) * 100
	}

	// Store change set
	ct.mu.Lock()
	ct.changeSets[changeSet.ID] = changeSet
	ct.mu.Unlock()

	return changeSet, nil
}

// DetectChanges detects changes in a file compared to its last snapshot
func (ct *ChangeTracker) DetectChanges(filePath string, lastSnapshotID string) (*ChangeSet, error) {
	// Create new snapshot
	newSnapshot, err := ct.CreateSnapshot(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create new snapshot: %w", err)
	}

	// Compare with last snapshot
	return ct.CompareSnapshots(lastSnapshotID, newSnapshot.ID)
}

// GetSnapshot retrieves a snapshot by ID
func (ct *ChangeTracker) GetSnapshot(snapshotID string) (*FileSnapshot, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	snapshot, exists := ct.snapshots[snapshotID]
	return snapshot, exists
}

// GetChangeSet retrieves a change set by ID
func (ct *ChangeTracker) GetChangeSet(changeSetID string) (*ChangeSet, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	changeSet, exists := ct.changeSets[changeSetID]
	return changeSet, exists
}

// GetLatestSnapshot returns the most recent snapshot for a file
func (ct *ChangeTracker) GetLatestSnapshot(filePath string) (*FileSnapshot, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var latest *FileSnapshot
	for _, snapshot := range ct.snapshots {
		if snapshot.FilePath == filePath {
			if latest == nil || snapshot.CreatedAt.After(latest.CreatedAt) {
				latest = snapshot
			}
		}
	}

	return latest, latest != nil
}

// ListSnapshots returns all snapshots for a file
func (ct *ChangeTracker) ListSnapshots(filePath string) []*FileSnapshot {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	snapshots := make([]*FileSnapshot, 0)
	for _, snapshot := range ct.snapshots {
		if snapshot.FilePath == filePath {
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots
}

// GetStatistics returns statistics about the change tracker
func (ct *ChangeTracker) GetStatistics() map[string]interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	totalBlocks := int64(0)
	changedBlocks := int64(0)
	totalBytes := int64(0)
	changedBytes := int64(0)

	for _, changeSet := range ct.changeSets {
		changedBlocks += int64(len(changeSet.ChangedBlocks) + len(changeSet.NewBlocks))
		changedBytes += changeSet.TotalChanged + changeSet.TotalNew
	}

	for _, snapshot := range ct.snapshots {
		totalBlocks += snapshot.TotalBlocks
		totalBytes += snapshot.FileSize
	}

	stats := map[string]interface{}{
		"total_snapshots":  len(ct.snapshots),
		"total_changesets": len(ct.changeSets),
		"total_blocks":     totalBlocks,
		"changed_blocks":   changedBlocks,
		"total_bytes":      totalBytes,
		"changed_bytes":    changedBytes,
		"block_size":       ct.blockSize,
	}

	if totalBlocks > 0 {
		stats["change_ratio"] = float64(changedBlocks) / float64(totalBlocks) * 100
	}

	return stats
}

// CleanupOldSnapshots removes snapshots older than the specified duration
func (ct *ChangeTracker) CleanupOldSnapshots(maxAge time.Duration) int {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, snapshot := range ct.snapshots {
		if snapshot.CreatedAt.Before(cutoff) {
			delete(ct.snapshots, id)
			removed++
		}
	}

	return removed
}

// GetChangedBlockData extracts data for changed blocks from a file
func (ct *ChangeTracker) GetChangedBlockData(filePath string, changeSet *ChangeSet) (map[int64][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	blockData := make(map[int64][]byte)

	// Read changed blocks
	for _, block := range changeSet.ChangedBlocks {
		data := make([]byte, block.Size)

		_, err := file.Seek(block.Offset, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to offset %d: %w", block.Offset, err)
		}

		n, err := file.Read(data)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read block at offset %d: %w", block.Offset, err)
		}

		blockData[block.Offset] = data[:n]
	}

	// Read new blocks
	for _, block := range changeSet.NewBlocks {
		data := make([]byte, block.Size)

		_, err := file.Seek(block.Offset, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to seek to offset %d: %w", block.Offset, err)
		}

		n, err := file.Read(data)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read block at offset %d: %w", block.Offset, err)
		}

		blockData[block.Offset] = data[:n]
	}

	return blockData, nil
}

// VerifyBlockChecksum verifies the checksum of a block
func VerifyBlockChecksum(data []byte, expectedChecksum string) bool {
	hasher := sha256.New()
	hasher.Write(data)
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	return actualChecksum == expectedChecksum
}

// ComputeBlockChecksum computes the SHA-256 checksum of a block
func ComputeBlockChecksum(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// EstimateIncrementalSize estimates the size of an incremental backup
func (ct *ChangeTracker) EstimateIncrementalSize(changeSet *ChangeSet) int64 {
	size := int64(0)

	for _, block := range changeSet.ChangedBlocks {
		size += block.Size
	}

	for _, block := range changeSet.NewBlocks {
		size += block.Size
	}

	// Add metadata overhead (estimated)
	metadataOverhead := int64(len(changeSet.ChangedBlocks)+len(changeSet.NewBlocks)) * 256

	return size + metadataOverhead
}

// IsFullBackupNeeded determines if a full backup is needed based on change ratio
func (ct *ChangeTracker) IsFullBackupNeeded(changeSet *ChangeSet, threshold float64) bool {
	return changeSet.ChangeRatio >= threshold
}

// GetBlockSize returns the current block size
func (ct *ChangeTracker) GetBlockSize() BlockSize {
	return ct.blockSize
}

// SetBlockSize updates the block size for future snapshots
func (ct *ChangeTracker) SetBlockSize(blockSize BlockSize) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.blockSize = blockSize
}
