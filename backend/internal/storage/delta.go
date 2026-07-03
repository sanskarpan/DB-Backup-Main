package storage

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DeltaConfig contains configuration for delta backups.
type DeltaConfig struct {
	// Enable delta backups
	Enabled bool `mapstructure:"enabled"`

	// Storage path for delta files
	DeltaStorePath string `mapstructure:"delta_store_path"`

	// Metadata path
	MetadataPath string `mapstructure:"metadata_path"`

	// Compression enabled
	Compression bool `mapstructure:"compression"`

	// Block size for diff algorithm (default: 4KB)
	BlockSize int `mapstructure:"block_size"`

	// Maximum chain length before requiring new full backup
	MaxChainLength int `mapstructure:"max_chain_length"`

	// Minimum file size for delta (files smaller than this get full backup)
	MinFileSize int64 `mapstructure:"min_file_size"`
}

// DeltaManager manages delta backup operations.
type DeltaManager struct {
	config *DeltaConfig
	mu     sync.RWMutex

	// Chain tracking: baseID -> chain of deltas
	chains map[string]*BackupChain

	// Metrics
	metrics *DeltaMetrics
}

// BackupChain represents a chain of delta backups.
type BackupChain struct {
	BaseID      string
	BaseHash    string
	BaseSize    int64
	Deltas      []*DeltaBackup
	TotalSize   int64
	ChainLength int
	CreatedAt   time.Time
	LastUpdated time.Time
}

// DeltaBackup represents a single delta backup.
type DeltaBackup struct {
	ID         string
	BaseID     string
	DeltaPath  string
	SourceHash string
	TargetHash string
	SourceSize int64
	TargetSize int64
	DeltaSize  int64
	Compressed bool
	CreatedAt  time.Time
	Operations []DeltaOperation
}

// DeltaOperation represents a single diff operation.
type DeltaOperation struct {
	Type   OperationType // insert, delete, copy
	Offset int64         // offset in source
	Length int64         // length of operation
	Data   []byte        // data for insert operations
}

// OperationType represents the type of delta operation.
type OperationType byte

const (
	OpInsert OperationType = 1
	OpDelete OperationType = 2
	OpCopy   OperationType = 3
)

// DeltaMetrics tracks delta backup statistics.
type DeltaMetrics struct {
	TotalDeltaBackups  int64
	TotalFullBackups   int64
	TotalBytesOriginal int64
	TotalBytesDelta    int64
	SpaceSaved         int64
	CompressionRatio   float64
	AverageChainLength float64
	mu                 sync.RWMutex
}

// Block represents a block of data with its hash.
type Block struct {
	Offset int64
	Length int64
	Hash   string
}

// NewDeltaManager creates a new delta backup manager.
func NewDeltaManager(config *DeltaConfig) (*DeltaManager, error) {
	if config == nil {
		return nil, fmt.Errorf("delta config is required")
	}

	// Set defaults
	if config.DeltaStorePath == "" {
		config.DeltaStorePath = "/var/lib/db-backup/deltas"
	}
	if config.MetadataPath == "" {
		config.MetadataPath = "/var/lib/db-backup/delta-metadata"
	}
	if config.BlockSize == 0 {
		config.BlockSize = 4096 // 4KB blocks
	}
	if config.MaxChainLength == 0 {
		config.MaxChainLength = 10 // Max 10 deltas in chain
	}
	if config.MinFileSize == 0 {
		config.MinFileSize = 1024 * 1024 // 1MB minimum
	}

	// Create directories
	if err := os.MkdirAll(config.DeltaStorePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create delta store: %w", err)
	}
	if err := os.MkdirAll(config.MetadataPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	manager := &DeltaManager{
		config:  config,
		chains:  make(map[string]*BackupChain),
		metrics: &DeltaMetrics{},
	}

	return manager, nil
}

// CreateDelta creates a delta backup from source to target.
func (m *DeltaManager) CreateDelta(sourcePath, targetPath, baseID string) (*DeltaBackup, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("delta backups are not enabled")
	}

	// Read source and target files
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source: %w", err)
	}

	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read target: %w", err)
	}

	// Check if file is large enough for delta
	if int64(len(targetData)) < m.config.MinFileSize {
		return nil, fmt.Errorf("file too small for delta backup (use full backup)")
	}

	// Calculate hashes
	sourceHash := calculateHash(sourceData)
	targetHash := calculateHash(targetData)

	// Generate delta operations
	operations := m.generateDelta(sourceData, targetData)

	// Serialize delta
	deltaData, err := m.serializeDelta(operations)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize delta: %w", err)
	}

	// Compress if enabled
	if m.config.Compression {
		deltaData, err = compressData(deltaData)
		if err != nil {
			return nil, fmt.Errorf("failed to compress delta: %w", err)
		}
	}

	// Generate delta ID
	deltaID := generateDeltaID()

	// Save delta to disk
	deltaPath := filepath.Join(m.config.DeltaStorePath, deltaID+".delta")
	if err := os.WriteFile(deltaPath, deltaData, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write delta: %w", err)
	}

	delta := &DeltaBackup{
		ID:         deltaID,
		BaseID:     baseID,
		DeltaPath:  deltaPath,
		SourceHash: sourceHash,
		TargetHash: targetHash,
		SourceSize: int64(len(sourceData)),
		TargetSize: int64(len(targetData)),
		DeltaSize:  int64(len(deltaData)),
		Compressed: m.config.Compression,
		CreatedAt:  time.Now(),
		Operations: operations,
	}

	// Update chain
	m.addToChain(baseID, delta)

	// Update metrics
	m.updateMetrics(delta)

	return delta, nil
}

// ApplyDelta applies a delta to reconstruct the target.
func (m *DeltaManager) ApplyDelta(sourcePath string, delta *DeltaBackup, outputPath string) error {
	// Read source data
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Verify source hash
	if calculateHash(sourceData) != delta.SourceHash {
		return fmt.Errorf("source hash mismatch (corrupted or wrong base)")
	}

	// Read delta data
	deltaData, err := os.ReadFile(delta.DeltaPath)
	if err != nil {
		return fmt.Errorf("failed to read delta: %w", err)
	}

	// Decompress if needed
	if delta.Compressed {
		deltaData, err = decompressData(deltaData)
		if err != nil {
			return fmt.Errorf("failed to decompress delta: %w", err)
		}
	}

	// Deserialize operations
	operations, err := m.deserializeDelta(deltaData)
	if err != nil {
		return fmt.Errorf("failed to deserialize delta: %w", err)
	}

	// Apply operations
	targetData, err := m.applyOperations(sourceData, operations)
	if err != nil {
		return fmt.Errorf("failed to apply operations: %w", err)
	}

	// Verify target hash
	if calculateHash(targetData) != delta.TargetHash {
		return fmt.Errorf("target hash mismatch (corrupted delta)")
	}

	// Write output
	if err := os.WriteFile(outputPath, targetData, 0o644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// generateDelta generates delta operations using a rolling hash algorithm.
func (m *DeltaManager) generateDelta(source, target []byte) []DeltaOperation {
	operations := make([]DeltaOperation, 0)

	// Build block index for source
	sourceBlocks := m.buildBlockIndex(source)

	// Process target and find matching/non-matching regions
	targetPos := 0
	pendingInsert := make([]byte, 0)

	for targetPos < len(target) {
		// Try to find a matching block
		matchFound := false
		matchOffset := int64(0)
		matchLength := 0

		// Look for blocks of decreasing size
		for blockSize := m.config.BlockSize; blockSize >= m.config.BlockSize/4 && blockSize > 0; blockSize /= 2 {
			if targetPos+blockSize > len(target) {
				continue
			}

			targetBlock := target[targetPos : targetPos+blockSize]
			blockHash := calculateHash(targetBlock)

			if block, exists := sourceBlocks[blockHash]; exists {
				// Found matching block
				matchOffset = block.Offset
				matchLength = blockSize
				matchFound = true
				break
			}
		}

		if matchFound {
			// Flush pending insert
			if len(pendingInsert) > 0 {
				operations = append(operations, DeltaOperation{
					Type:   OpInsert,
					Offset: int64(targetPos - len(pendingInsert)),
					Length: int64(len(pendingInsert)),
					Data:   append([]byte(nil), pendingInsert...),
				})
				pendingInsert = pendingInsert[:0]
			}

			// Add copy operation
			operations = append(operations, DeltaOperation{
				Type:   OpCopy,
				Offset: matchOffset,
				Length: int64(matchLength),
			})

			targetPos += matchLength
		} else {
			// No match, add to pending insert
			if targetPos < len(target) {
				pendingInsert = append(pendingInsert, target[targetPos])
			}
			targetPos++
		}
	}

	// Flush remaining insert
	if len(pendingInsert) > 0 {
		operations = append(operations, DeltaOperation{
			Type:   OpInsert,
			Offset: int64(targetPos - len(pendingInsert)),
			Length: int64(len(pendingInsert)),
			Data:   pendingInsert,
		})
	}

	return m.optimizeOperations(operations)
}

// buildBlockIndex builds a hash index of source blocks.
func (m *DeltaManager) buildBlockIndex(data []byte) map[string]*Block {
	blocks := make(map[string]*Block)

	for offset := 0; offset < len(data); offset += m.config.BlockSize / 2 {
		length := m.config.BlockSize
		if offset+length > len(data) {
			length = len(data) - offset
		}

		if length == 0 {
			break
		}

		block := data[offset : offset+length]
		hash := calculateHash(block)

		blocks[hash] = &Block{
			Offset: int64(offset),
			Length: int64(length),
			Hash:   hash,
		}
	}

	return blocks
}

// optimizeOperations combines adjacent insert operations.
func (m *DeltaManager) optimizeOperations(ops []DeltaOperation) []DeltaOperation {
	if len(ops) == 0 {
		return ops
	}

	optimized := make([]DeltaOperation, 0, len(ops))
	current := ops[0]

	for i := 1; i < len(ops); i++ {
		next := ops[i]

		// Combine adjacent inserts
		if current.Type == OpInsert && next.Type == OpInsert &&
			current.Offset+current.Length == next.Offset {
			current.Data = append(current.Data, next.Data...)
			current.Length += next.Length
		} else {
			optimized = append(optimized, current)
			current = next
		}
	}

	optimized = append(optimized, current)
	return optimized
}

// applyOperations applies delta operations to source data.
func (m *DeltaManager) applyOperations(source []byte, operations []DeltaOperation) ([]byte, error) {
	result := make([]byte, 0)

	for _, op := range operations {
		switch op.Type {
		case OpInsert:
			result = append(result, op.Data...)

		case OpCopy:
			if op.Offset < 0 || op.Offset+op.Length > int64(len(source)) {
				return nil, fmt.Errorf("invalid copy operation: offset=%d, length=%d, source_size=%d",
					op.Offset, op.Length, len(source))
			}
			result = append(result, source[op.Offset:op.Offset+op.Length]...)

		case OpDelete:
			// Delete is implicit - we just don't copy

		default:
			return nil, fmt.Errorf("unknown operation type: %d", op.Type)
		}
	}

	return result, nil
}

// serializeDelta serializes delta operations.
func (m *DeltaManager) serializeDelta(operations []DeltaOperation) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(operations); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// deserializeDelta deserializes delta operations.
func (m *DeltaManager) deserializeDelta(data []byte) ([]DeltaOperation, error) {
	var operations []DeltaOperation

	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&operations); err != nil {
		return nil, err
	}

	return operations, nil
}

// addToChain adds a delta to a backup chain.
func (m *DeltaManager) addToChain(baseID string, delta *DeltaBackup) {
	m.mu.Lock()
	defer m.mu.Unlock()

	chain, exists := m.chains[baseID]
	if !exists {
		chain = &BackupChain{
			BaseID:    baseID,
			Deltas:    make([]*DeltaBackup, 0),
			CreatedAt: time.Now(),
		}
		m.chains[baseID] = chain
	}

	chain.Deltas = append(chain.Deltas, delta)
	chain.TotalSize += delta.DeltaSize
	chain.ChainLength = len(chain.Deltas)
	chain.LastUpdated = time.Now()
}

// NeedsFullBackup determines if a new full backup is needed.
func (m *DeltaManager) NeedsFullBackup(baseID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, exists := m.chains[baseID]
	if !exists {
		return true // No chain exists, need full backup
	}

	return chain.ChainLength >= m.config.MaxChainLength
}

// GetChain retrieves a backup chain.
func (m *DeltaManager) GetChain(baseID string) (*BackupChain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, exists := m.chains[baseID]
	if !exists {
		return nil, fmt.Errorf("chain not found: %s", baseID)
	}

	return chain, nil
}

// RestoreFromChain restores a file by applying all deltas in a chain.
func (m *DeltaManager) RestoreFromChain(basePath string, chainID string, version int, outputPath string) error {
	chain, err := m.GetChain(chainID)
	if err != nil {
		return err
	}

	if version < 0 || version > len(chain.Deltas) {
		return fmt.Errorf("invalid version: %d (chain has %d deltas)", version, len(chain.Deltas))
	}

	// Start with base file
	currentPath := basePath
	tempFiles := make([]string, 0)

	// Apply deltas up to requested version
	for i := 0; i < version; i++ {
		delta := chain.Deltas[i]
		tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("delta-restore-%d.tmp", i))
		tempFiles = append(tempFiles, tempPath)

		if err := m.ApplyDelta(currentPath, delta, tempPath); err != nil {
			// Cleanup temp files
			for _, tf := range tempFiles {
				os.Remove(tf)
			}
			return fmt.Errorf("failed to apply delta %d: %w", i, err)
		}

		currentPath = tempPath
	}

	// Copy final result to output
	finalData, err := os.ReadFile(currentPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, finalData, 0o644); err != nil {
		return err
	}

	// Cleanup temp files
	for _, tf := range tempFiles {
		os.Remove(tf)
	}

	return nil
}

// updateMetrics updates delta backup metrics.
func (m *DeltaManager) updateMetrics(delta *DeltaBackup) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalDeltaBackups++
	m.metrics.TotalBytesOriginal += delta.TargetSize
	m.metrics.TotalBytesDelta += delta.DeltaSize
	m.metrics.SpaceSaved = m.metrics.TotalBytesOriginal - m.metrics.TotalBytesDelta

	if m.metrics.TotalBytesOriginal > 0 {
		m.metrics.CompressionRatio = float64(m.metrics.TotalBytesDelta) / float64(m.metrics.TotalBytesOriginal)
	}

	// Calculate average chain length
	totalChainLength := 0
	chainCount := 0
	for _, chain := range m.chains {
		totalChainLength += chain.ChainLength
		chainCount++
	}
	if chainCount > 0 {
		m.metrics.AverageChainLength = float64(totalChainLength) / float64(chainCount)
	}
}

// GetMetrics returns current delta metrics.
func (m *DeltaManager) GetMetrics() *DeltaMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return &DeltaMetrics{
		TotalDeltaBackups:  m.metrics.TotalDeltaBackups,
		TotalFullBackups:   m.metrics.TotalFullBackups,
		TotalBytesOriginal: m.metrics.TotalBytesOriginal,
		TotalBytesDelta:    m.metrics.TotalBytesDelta,
		SpaceSaved:         m.metrics.SpaceSaved,
		CompressionRatio:   m.metrics.CompressionRatio,
		AverageChainLength: m.metrics.AverageChainLength,
	}
}

// ListChains returns all backup chains sorted by creation date.
func (m *DeltaManager) ListChains() []*BackupChain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]*BackupChain, 0, len(m.chains))
	for _, chain := range m.chains {
		chains = append(chains, chain)
	}

	sort.Slice(chains, func(i, j int) bool {
		return chains[i].CreatedAt.After(chains[j].CreatedAt)
	})

	return chains
}

// Helper functions

func calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func generateDeltaID() string {
	return fmt.Sprintf("delta-%d-%s", time.Now().Unix(), calculateHash([]byte(time.Now().String())))[:32]
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
