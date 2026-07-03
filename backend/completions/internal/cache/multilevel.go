package cache

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry represents a cache entry.
type Entry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	Hits      int         `json:"hits"`
	Size      int         `json:"size"`
}

// MultiLevelCache implements memory + disk caching with compression.
type MultiLevelCache struct {
	mu            sync.RWMutex
	memCache      map[string]*Entry
	diskCacheDir  string
	maxMemorySize int // in bytes
	currentSize   int
	ttl           time.Duration
	compress      bool
}

// NewMultiLevelCache creates a new multi-level cache.
func NewMultiLevelCache(diskCacheDir string, ttl time.Duration) *MultiLevelCache {
	return &MultiLevelCache{
		memCache:      make(map[string]*Entry),
		diskCacheDir:  diskCacheDir,
		maxMemorySize: 10 * 1024 * 1024, // 10MB default
		ttl:           ttl,
		compress:      true,
	}
}

// Get retrieves value from cache (memory first, then disk).
func (c *MultiLevelCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()

	// Check memory cache first
	if entry, exists := c.memCache[key]; exists {
		if time.Now().Before(entry.ExpiresAt) {
			entry.Hits++
			c.mu.RUnlock()
			return entry.Value, true
		}
		// Expired
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.memCache, key)
		c.currentSize -= entry.Size
		c.mu.Unlock()
		return nil, false
	}

	c.mu.RUnlock()

	// Check disk cache
	value, err := c.getFromDisk(key)
	if err != nil {
		return nil, false
	}

	// Promote to memory cache; a promotion failure is non-fatal since we
	// already have the value from disk.
	if err := c.Set(key, value); err != nil {
		return value, true
	}

	return value, true
}

// Set stores value in cache (memory and disk).
func (c *MultiLevelCache) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Serialize value to calculate size
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	entry := &Entry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
		Hits:      0,
		Size:      len(data),
	}

	// Check if we need to evict from memory
	if c.currentSize+entry.Size > c.maxMemorySize {
		c.evictLRU()
	}

	// Store in memory
	if old, exists := c.memCache[key]; exists {
		c.currentSize -= old.Size
	}
	c.memCache[key] = entry
	c.currentSize += entry.Size

	// Store on disk asynchronously; disk persistence is best-effort so a
	// failure here is non-fatal.
	go func() {
		if err := c.saveToDisk(key, entry); err != nil {
			return
		}
	}()

	return nil
}

// Delete removes value from cache.
func (c *MultiLevelCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.memCache[key]; exists {
		c.currentSize -= entry.Size
		delete(c.memCache, key)
	}

	// Delete from disk
	diskPath := filepath.Join(c.diskCacheDir, key+".cache")
	os.Remove(diskPath)
}

// Clear clears all cache.
func (c *MultiLevelCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memCache = make(map[string]*Entry)
	c.currentSize = 0

	return os.RemoveAll(c.diskCacheDir)
}

// Stats returns cache statistics.
func (c *MultiLevelCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	for _, entry := range c.memCache {
		totalHits += entry.Hits
	}

	diskFiles := 0
	diskSize := int64(0)
	if entries, err := os.ReadDir(c.diskCacheDir); err == nil {
		diskFiles = len(entries)
		for _, entry := range entries {
			if info, err := entry.Info(); err == nil {
				diskSize += info.Size()
			}
		}
	}

	return map[string]interface{}{
		"memory_entries": len(c.memCache),
		"memory_size":    c.currentSize,
		"max_memory":     c.maxMemorySize,
		"disk_files":     diskFiles,
		"disk_size":      diskSize,
		"total_hits":     totalHits,
		"ttl_seconds":    c.ttl.Seconds(),
	}
}

// Prefetch loads multiple keys into cache.
func (c *MultiLevelCache) Prefetch(keys []string, loader func(key string) (interface{}, error)) error {
	for _, key := range keys {
		// Check if already in cache
		if _, exists := c.Get(key); exists {
			continue
		}

		// Load value
		value, err := loader(key)
		if err != nil {
			continue // Skip errors, continue prefetching
		}

		// Store in cache; skip on error and continue prefetching.
		if err := c.Set(key, value); err != nil {
			continue
		}
	}

	return nil
}

// evictLRU evicts least recently used (lowest hits) entry.
func (c *MultiLevelCache) evictLRU() {
	if len(c.memCache) == 0 {
		return
	}

	var lruKey string
	var lruHits int = -1
	var oldestTime time.Time

	for key, entry := range c.memCache {
		if lruHits == -1 || entry.Hits < lruHits {
			lruKey = key
			lruHits = entry.Hits
			oldestTime = entry.ExpiresAt
		} else if entry.Hits == lruHits && entry.ExpiresAt.Before(oldestTime) {
			lruKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if lruKey != "" {
		if entry, exists := c.memCache[lruKey]; exists {
			c.currentSize -= entry.Size
			delete(c.memCache, lruKey)
		}
	}
}

// getFromDisk retrieves value from disk cache.
func (c *MultiLevelCache) getFromDisk(key string) (interface{}, error) {
	diskPath := filepath.Join(c.diskCacheDir, key+".cache")

	file, err := os.Open(diskPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var reader io.Reader = file

	// Decompress if enabled
	if c.compress {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var entry Entry
	if err := json.NewDecoder(reader).Decode(&entry); err != nil {
		return nil, err
	}

	// Check expiry
	if time.Now().After(entry.ExpiresAt) {
		os.Remove(diskPath)
		return nil, os.ErrNotExist
	}

	return entry.Value, nil
}

// saveToDisk saves value to disk cache.
func (c *MultiLevelCache) saveToDisk(key string, entry *Entry) error {
	if err := os.MkdirAll(c.diskCacheDir, 0o755); err != nil {
		return err
	}

	diskPath := filepath.Join(c.diskCacheDir, key+".cache")

	file, err := os.Create(diskPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer io.Writer = file

	// Compress if enabled
	if c.compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	return json.NewEncoder(writer).Encode(entry)
}

// Compress compresses data using gzip.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	if _, err := gzWriter.Write(data); err != nil {
		return nil, err
	}

	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompress decompresses gzip data.
func Decompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	return io.ReadAll(gzReader)
}
