package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DeduplicationConfig contains configuration for deduplication.
type DeduplicationConfig struct {
	// Enable deduplication
	Enabled bool `mapstructure:"enabled"`

	// Chunk size for content-defined chunking (CDC)
	ChunkSize int `mapstructure:"chunk_size"` // Default: 4MB

	// Minimum chunk size
	MinChunkSize int `mapstructure:"min_chunk_size"` // Default: 2MB

	// Maximum chunk size
	MaxChunkSize int `mapstructure:"max_chunk_size"` // Default: 8MB

	// Hash algorithm (sha256, blake2b)
	HashAlgorithm string `mapstructure:"hash_algorithm"`

	// Storage path for deduplicated chunks
	ChunkStorePath string `mapstructure:"chunk_store_path"`

	// Index path for chunk metadata
	IndexPath string `mapstructure:"index_path"`

	// Enable compression on chunks
	CompressChunks bool `mapstructure:"compress_chunks"`

	// Garbage collection interval
	GCInterval time.Duration `mapstructure:"gc_interval"`

	// Cache size for chunk index (in entries)
	CacheSize int `mapstructure:"cache_size"`
}

// DeduplicationManager manages content-addressable storage with deduplication.
type DeduplicationManager struct {
	config *DeduplicationConfig
	mu     sync.RWMutex

	// Chunk index: hash -> chunk metadata
	index map[string]*ChunkMetadata

	// Reference count: hash -> count
	refCount map[string]int

	// LRU cache for frequently accessed chunks
	cache *ChunkCache

	// Metrics
	metrics *DeduplicationMetrics
}

// ChunkMetadata contains metadata about a deduplicated chunk.
type ChunkMetadata struct {
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	CompressedSize int64     `json:"compressed_size,omitempty"`
	RefCount       int       `json:"ref_count"`
	FirstSeen      time.Time `json:"first_seen"`
	LastAccessed   time.Time `json:"last_accessed"`
	Compressed     bool      `json:"compressed"`
	StoragePath    string    `json:"storage_path"`
}

// ChunkReference represents a reference to a chunk in a file.
type ChunkReference struct {
	Hash   string `json:"hash"`
	Offset int64  `json:"offset"`
	Size   int64  `json:"size"`
}

// FileManifest contains the chunk list for a deduplicated file.
type FileManifest struct {
	OriginalPath string           `json:"original_path"`
	OriginalSize int64            `json:"original_size"`
	Chunks       []ChunkReference `json:"chunks"`
	TotalChunks  int              `json:"total_chunks"`
	UniqueChunks int              `json:"unique_chunks"`
	DedupeRatio  float64          `json:"dedupe_ratio"`
	CreatedAt    time.Time        `json:"created_at"`
}

// DeduplicationMetrics tracks deduplication statistics.
type DeduplicationMetrics struct {
	TotalChunks        int64
	UniqueChunks       int64
	TotalBytes         int64
	UniqueBytes        int64
	SavedBytes         int64
	DeduplicationRatio float64
	CacheHits          int64
	CacheMisses        int64
	mu                 sync.RWMutex
}

// ChunkCache implements a simple LRU cache.
type ChunkCache struct {
	maxSize int
	cache   map[string]*cacheEntry
	mu      sync.RWMutex
}

type cacheEntry struct {
	data     []byte
	accessed time.Time
}

// NewDeduplicationManager creates a new deduplication manager.
func NewDeduplicationManager(config *DeduplicationConfig) (*DeduplicationManager, error) {
	if config == nil {
		return nil, fmt.Errorf("deduplication config is required")
	}

	// Set defaults
	if config.ChunkSize == 0 {
		config.ChunkSize = 4 * 1024 * 1024 // 4MB
	}
	if config.MinChunkSize == 0 {
		config.MinChunkSize = 2 * 1024 * 1024 // 2MB
	}
	if config.MaxChunkSize == 0 {
		config.MaxChunkSize = 8 * 1024 * 1024 // 8MB
	}
	if config.HashAlgorithm == "" {
		config.HashAlgorithm = "sha256"
	}
	if config.ChunkStorePath == "" {
		config.ChunkStorePath = "/var/lib/db-backup/chunks"
	}
	if config.IndexPath == "" {
		config.IndexPath = "/var/lib/db-backup/dedup-index.json"
	}
	if config.CacheSize == 0 {
		config.CacheSize = 1000 // Cache 1000 chunks
	}

	// Create directories
	if err := os.MkdirAll(config.ChunkStorePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chunk store: %w", err)
	}

	indexDir := filepath.Dir(config.IndexPath)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	manager := &DeduplicationManager{
		config:   config,
		index:    make(map[string]*ChunkMetadata),
		refCount: make(map[string]int),
		cache:    NewChunkCache(config.CacheSize),
		metrics:  &DeduplicationMetrics{},
	}

	// Load existing index
	if err := manager.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return manager, nil
}

// StoreFile stores a file with deduplication.
func (m *DeduplicationManager) StoreFile(sourcePath string) (*FileManifest, error) {
	if !m.config.Enabled {
		return nil, fmt.Errorf("deduplication is not enabled")
	}

	// Open source file
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	manifest := &FileManifest{
		OriginalPath: sourcePath,
		OriginalSize: fileInfo.Size(),
		Chunks:       make([]ChunkReference, 0),
		CreatedAt:    time.Now(),
	}

	uniqueChunks := make(map[string]bool)
	offset := int64(0)

	// Read and chunk the file
	buffer := make([]byte, m.config.ChunkSize)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		if n == 0 {
			break
		}

		chunk := buffer[:n]
		hash := m.hashChunk(chunk)

		// Store chunk if not exists
		if storeErr := m.storeChunk(hash, chunk); storeErr != nil {
			return nil, fmt.Errorf("failed to store chunk: %w", storeErr)
		}

		// Add to manifest
		manifest.Chunks = append(manifest.Chunks, ChunkReference{
			Hash:   hash,
			Offset: offset,
			Size:   int64(n),
		})

		uniqueChunks[hash] = true
		offset += int64(n)

		if err == io.EOF {
			break
		}
	}

	manifest.TotalChunks = len(manifest.Chunks)
	manifest.UniqueChunks = len(uniqueChunks)

	if manifest.TotalChunks > 0 {
		manifest.DedupeRatio = float64(manifest.UniqueChunks) / float64(manifest.TotalChunks)
	}

	// Update metrics
	m.updateMetrics(manifest)

	return manifest, nil
}

// RestoreFile restores a file from its manifest.
func (m *DeduplicationManager) RestoreFile(manifest *FileManifest, destPath string) error {
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Restore chunks in order
	for _, chunkRef := range manifest.Chunks {
		chunk, err := m.retrieveChunk(chunkRef.Hash)
		if err != nil {
			return fmt.Errorf("failed to retrieve chunk %s: %w", chunkRef.Hash, err)
		}

		if _, err := destFile.Write(chunk); err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}
	}

	return nil
}

// storeChunk stores a chunk if it doesn't already exist.
func (m *DeduplicationManager) storeChunk(hash string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if chunk already exists
	if meta, exists := m.index[hash]; exists {
		meta.RefCount++
		m.refCount[hash]++
		meta.LastAccessed = time.Now()
		return nil
	}

	// Compress chunk if enabled
	var dataToStore []byte
	compressed := false
	compressedSize := int64(0)

	if m.config.CompressChunks {
		// Use gzip compression
		// Simplified - in production use proper compression
		compressed = true
		dataToStore = data
		compressedSize = int64(len(dataToStore))
	} else {
		dataToStore = data
	}

	// Store chunk to disk
	chunkPath := m.getChunkPath(hash)
	if err := os.MkdirAll(filepath.Dir(chunkPath), 0o755); err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}
	if err := os.WriteFile(chunkPath, dataToStore, 0o600); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	// Update index
	meta := &ChunkMetadata{
		Hash:           hash,
		Size:           int64(len(data)),
		CompressedSize: compressedSize,
		RefCount:       1,
		FirstSeen:      time.Now(),
		LastAccessed:   time.Now(),
		Compressed:     compressed,
		StoragePath:    chunkPath,
	}

	m.index[hash] = meta
	m.refCount[hash] = 1

	// Update metrics
	m.metrics.mu.Lock()
	m.metrics.UniqueChunks++
	m.metrics.UniqueBytes += int64(len(data))
	m.metrics.mu.Unlock()

	// Cache the chunk — copy data so callers can safely reuse the source buffer
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.cache.Put(hash, dataCopy)

	return nil
}

// retrieveChunk retrieves a chunk by hash.
func (m *DeduplicationManager) retrieveChunk(hash string) ([]byte, error) {
	// Check cache first
	if data := m.cache.Get(hash); data != nil {
		m.metrics.mu.Lock()
		m.metrics.CacheHits++
		m.metrics.mu.Unlock()
		return data, nil
	}

	m.metrics.mu.Lock()
	m.metrics.CacheMisses++
	m.metrics.mu.Unlock()

	m.mu.RLock()
	meta, exists := m.index[hash]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("chunk not found: %s", hash)
	}

	// Read from disk
	data, err := os.ReadFile(meta.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	// Decompress if needed
	if meta.Compressed {
		// Decompress data (simplified - use proper decompression in production)
	}

	// Update access time
	m.mu.Lock()
	meta.LastAccessed = time.Now()
	m.mu.Unlock()

	// Cache the chunk
	m.cache.Put(hash, data)

	return data, nil
}

// hashChunk calculates the hash of a chunk.
func (m *DeduplicationManager) hashChunk(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// getChunkPath returns the filesystem path for a chunk.
func (m *DeduplicationManager) getChunkPath(hash string) string {
	// Use first 2 characters as subdirectory for better filesystem performance
	subdir := hash[:2]
	subdirPath := filepath.Join(m.config.ChunkStorePath, subdir)
	return filepath.Join(subdirPath, hash)
}

// updateMetrics updates deduplication metrics.
func (m *DeduplicationManager) updateMetrics(manifest *FileManifest) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalChunks += int64(manifest.TotalChunks)
	m.metrics.TotalBytes += manifest.OriginalSize

	savedBytes := manifest.OriginalSize - (int64(manifest.UniqueChunks) * int64(m.config.ChunkSize))
	if savedBytes > 0 {
		m.metrics.SavedBytes += savedBytes
	}

	if m.metrics.TotalBytes > 0 {
		m.metrics.DeduplicationRatio = float64(m.metrics.UniqueBytes) / float64(m.metrics.TotalBytes)
	}
}

// GetMetrics returns current deduplication metrics.
func (m *DeduplicationManager) GetMetrics() *DeduplicationMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return &DeduplicationMetrics{
		TotalChunks:        m.metrics.TotalChunks,
		UniqueChunks:       m.metrics.UniqueChunks,
		TotalBytes:         m.metrics.TotalBytes,
		UniqueBytes:        m.metrics.UniqueBytes,
		SavedBytes:         m.metrics.SavedBytes,
		DeduplicationRatio: m.metrics.DeduplicationRatio,
		CacheHits:          m.metrics.CacheHits,
		CacheMisses:        m.metrics.CacheMisses,
	}
}

// GarbageCollect removes unreferenced chunks.
func (m *DeduplicationManager) GarbageCollect() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0

	for hash, meta := range m.index {
		if meta.RefCount != 0 {
			continue
		}

		// Remove chunk file
		if err := os.Remove(meta.StoragePath); err != nil && !os.IsNotExist(err) {
			return removed, fmt.Errorf("failed to remove chunk: %w", err)
		}

		// Remove from index
		delete(m.index, hash)
		delete(m.refCount, hash)
		removed++

		// Update metrics
		m.metrics.mu.Lock()
		m.metrics.UniqueChunks--
		m.metrics.UniqueBytes -= meta.Size
		m.metrics.mu.Unlock()
	}

	return removed, nil
}

// DeleteManifest decrements reference counts for all chunks in a manifest.
func (m *DeduplicationManager) DeleteManifest(manifest *FileManifest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, chunkRef := range manifest.Chunks {
		if meta, exists := m.index[chunkRef.Hash]; exists {
			meta.RefCount--
			m.refCount[chunkRef.Hash]--

			if meta.RefCount < 0 {
				meta.RefCount = 0
				m.refCount[chunkRef.Hash] = 0
			}
		}
	}

	return nil
}

// saveIndex saves the chunk index to disk.
func (m *DeduplicationManager) saveIndex() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(m.config.IndexPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// loadIndex loads the chunk index from disk.
func (m *DeduplicationManager) loadIndex() error {
	if _, err := os.Stat(m.config.IndexPath); os.IsNotExist(err) {
		return nil // Index doesn't exist yet
	}

	data, err := os.ReadFile(m.config.IndexPath)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	if err := json.Unmarshal(data, &m.index); err != nil {
		return fmt.Errorf("failed to unmarshal index: %w", err)
	}

	// Rebuild refCount map
	for hash, meta := range m.index {
		m.refCount[hash] = meta.RefCount
	}

	return nil
}

// Close saves the index and performs cleanup.
func (m *DeduplicationManager) Close() error {
	return m.saveIndex()
}

// ChunkCache implementation

// NewChunkCache creates a new chunk cache.
func NewChunkCache(maxSize int) *ChunkCache {
	return &ChunkCache{
		maxSize: maxSize,
		cache:   make(map[string]*cacheEntry),
	}
}

// Get retrieves a chunk from cache.
func (c *ChunkCache) Get(hash string) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, exists := c.cache[hash]; exists {
		entry.accessed = time.Now()
		return entry.data
	}

	return nil
}

// Put adds a chunk to cache.
func (c *ChunkCache) Put(hash string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already exists, update
	if entry, exists := c.cache[hash]; exists {
		entry.data = data
		entry.accessed = time.Now()
		return
	}

	// Evict if at capacity
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	// Add new entry
	c.cache[hash] = &cacheEntry{
		data:     data,
		accessed: time.Now(),
	}
}

// evictOldest removes the least recently accessed entry.
func (c *ChunkCache) evictOldest() {
	var oldestHash string
	var oldestTime time.Time

	for hash, entry := range c.cache {
		if oldestHash == "" || entry.accessed.Before(oldestTime) {
			oldestHash = hash
			oldestTime = entry.accessed
		}
	}

	if oldestHash != "" {
		delete(c.cache, oldestHash)
	}
}

// Clear clears the cache.
func (c *ChunkCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
}

// Size returns the number of entries in cache.
func (c *ChunkCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}
