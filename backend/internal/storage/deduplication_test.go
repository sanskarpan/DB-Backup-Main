package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDeduplicationManager(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	if manager.config.ChunkSize != 4*1024*1024 {
		t.Errorf("Expected default chunk size 4MB, got %d", manager.config.ChunkSize)
	}

	if manager.config.HashAlgorithm != "sha256" {
		t.Errorf("Expected default hash algorithm sha256, got %s", manager.config.HashAlgorithm)
	}
}

func TestStoreAndRetrieveChunk(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024, // Small chunks for testing
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Create test data
	testData := []byte("Hello, this is test data for deduplication")
	hash := manager.hashChunk(testData)

	// Store chunk
	err = manager.storeChunk(hash, testData)
	if err != nil {
		t.Fatalf("Failed to store chunk: %v", err)
	}

	// Retrieve chunk
	retrieved, err := manager.retrieveChunk(hash)
	if err != nil {
		t.Fatalf("Failed to retrieve chunk: %v", err)
	}

	if !bytes.Equal(testData, retrieved) {
		t.Errorf("Retrieved data doesn't match original")
	}
}

func TestDeduplicationDetection(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024,
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	testData := []byte("Duplicate data")
	hash := manager.hashChunk(testData)

	// Store same chunk twice
	err = manager.storeChunk(hash, testData)
	if err != nil {
		t.Fatalf("Failed to store chunk first time: %v", err)
	}

	err = manager.storeChunk(hash, testData)
	if err != nil {
		t.Fatalf("Failed to store chunk second time: %v", err)
	}

	// Check that only one unique chunk exists
	metrics := manager.GetMetrics()
	if metrics.UniqueChunks != 1 {
		t.Errorf("Expected 1 unique chunk, got %d", metrics.UniqueChunks)
	}

	// Check reference count
	manager.mu.RLock()
	refCount := manager.refCount[hash]
	manager.mu.RUnlock()

	if refCount != 2 {
		t.Errorf("Expected ref count 2, got %d", refCount)
	}
}

func TestStoreAndRestoreFile(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024, // Small chunks for testing
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	testData := bytes.Repeat([]byte("Test data for deduplication. "), 100)
	err = os.WriteFile(testFile, testData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Store file
	manifest, err := manager.StoreFile(testFile)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	if manifest.OriginalSize != int64(len(testData)) {
		t.Errorf("Expected original size %d, got %d", len(testData), manifest.OriginalSize)
	}

	if manifest.TotalChunks == 0 {
		t.Error("Expected at least one chunk")
	}

	// Restore file
	restoredFile := filepath.Join(tempDir, "restored.txt")
	err = manager.RestoreFile(manifest, restoredFile)
	if err != nil {
		t.Fatalf("Failed to restore file: %v", err)
	}

	// Verify restored data
	restoredData, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if !bytes.Equal(testData, restoredData) {
		t.Error("Restored data doesn't match original")
	}
}

func TestDeduplicationRatio(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      100, // Very small chunks for testing
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Create file with repetitive data (high deduplication potential)
	testFile := filepath.Join(tempDir, "repetitive.txt")
	testData := bytes.Repeat([]byte("A"), 1000) // 1000 bytes of 'A'
	err = os.WriteFile(testFile, testData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Store file
	manifest, err := manager.StoreFile(testFile)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// With 100-byte chunks, we expect 10 total chunks
	if manifest.TotalChunks != 10 {
		t.Errorf("Expected 10 total chunks, got %d", manifest.TotalChunks)
	}

	// All chunks should be identical (same content), so unique chunks should be 1
	if manifest.UniqueChunks != 1 {
		t.Errorf("Expected 1 unique chunk, got %d", manifest.UniqueChunks)
	}

	// Deduplication ratio should be 0.1 (1 unique / 10 total)
	expectedRatio := 0.1
	if manifest.DedupeRatio != expectedRatio {
		t.Errorf("Expected dedupe ratio %.2f, got %.2f", expectedRatio, manifest.DedupeRatio)
	}
}

func TestGarbageCollection(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024,
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Store some chunks
	testData1 := []byte("Test data 1")
	testData2 := []byte("Test data 2")

	hash1 := manager.hashChunk(testData1)
	hash2 := manager.hashChunk(testData2)

	manager.storeChunk(hash1, testData1)
	manager.storeChunk(hash2, testData2)

	// Verify 2 chunks exist
	metrics := manager.GetMetrics()
	if metrics.UniqueChunks != 2 {
		t.Errorf("Expected 2 unique chunks, got %d", metrics.UniqueChunks)
	}

	// Manually set ref count to 0 for one chunk
	manager.mu.Lock()
	manager.index[hash1].RefCount = 0
	manager.refCount[hash1] = 0
	manager.mu.Unlock()

	// Run garbage collection
	removed, err := manager.GarbageCollect()
	if err != nil {
		t.Fatalf("GC failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("Expected to remove 1 chunk, removed %d", removed)
	}

	// Verify only 1 chunk remains
	metrics = manager.GetMetrics()
	if metrics.UniqueChunks != 1 {
		t.Errorf("Expected 1 unique chunk after GC, got %d", metrics.UniqueChunks)
	}
}

func TestChunkCache(t *testing.T) {
	cache := NewChunkCache(2) // Small cache for testing

	// Add items
	cache.Put("hash1", []byte("data1"))
	cache.Put("hash2", []byte("data2"))

	// Retrieve items
	data1 := cache.Get("hash1")
	if data1 == nil || string(data1) != "data1" {
		t.Error("Failed to retrieve cached item")
	}

	// Add third item — should evict least recently used (hash2, since hash1 was just Get'd)
	cache.Put("hash3", []byte("data3"))

	// hash1 should still be there (was recently accessed via Get)
	data1again := cache.Get("hash1")
	if data1again == nil {
		t.Error("hash1 should still be in cache (was recently accessed)")
	}

	// hash2 should have been evicted (it was least recently used)
	data2 := cache.Get("hash2")
	if data2 != nil {
		t.Error("hash2 should have been evicted (LRU)")
	}

	// Verify size
	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

func TestHashConsistency(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	testData := []byte("Test data for hashing")

	// Hash same data twice
	hash1 := manager.hashChunk(testData)
	hash2 := manager.hashChunk(testData)

	if hash1 != hash2 {
		t.Error("Hash should be consistent for same data")
	}

	// Verify hash format (SHA256 produces 64 hex characters)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Verify it matches manual SHA256
	expected := sha256.Sum256(testData)
	expectedHex := hex.EncodeToString(expected[:])

	if hash1 != expectedHex {
		t.Error("Hash doesn't match expected SHA256")
	}
}

func TestDeleteManifest(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      100,
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Create and store a file with DISTINCT chunks (different content per chunk)
	testFile := filepath.Join(tempDir, "test.txt")
	chunk1 := bytes.Repeat([]byte("A"), 100)
	chunk2 := bytes.Repeat([]byte("B"), 100)
	testData := append(chunk1, chunk2...)
	os.WriteFile(testFile, testData, 0o644)

	manifest, err := manager.StoreFile(testFile)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	if len(manifest.Chunks) != 2 {
		t.Fatalf("Expected 2 distinct chunks, got %d", len(manifest.Chunks))
	}

	// Get initial ref counts (each distinct chunk should have refCount=1)
	initialRefCounts := make(map[string]int)
	manager.mu.RLock()
	for _, chunk := range manifest.Chunks {
		initialRefCounts[chunk.Hash] = manager.refCount[chunk.Hash]
	}
	manager.mu.RUnlock()

	// Delete manifest
	err = manager.DeleteManifest(manifest)
	if err != nil {
		t.Fatalf("Failed to delete manifest: %v", err)
	}

	// Verify ref counts decreased by 1 for each distinct chunk
	manager.mu.RLock()
	for _, chunk := range manifest.Chunks {
		newCount := manager.refCount[chunk.Hash]
		oldCount := initialRefCounts[chunk.Hash]
		if newCount != oldCount-1 {
			t.Errorf("Expected ref count to decrease by 1, was %d, now %d", oldCount, newCount)
		}
	}
	manager.mu.RUnlock()
}

func TestIndexPersistence(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024,
	}

	// Create first manager and store data
	manager1, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create first manager: %v", err)
	}

	testData := []byte("Persistent test data")
	hash := manager1.hashChunk(testData)
	manager1.storeChunk(hash, testData)

	// Save and close
	err = manager1.Close()
	if err != nil {
		t.Fatalf("Failed to close first manager: %v", err)
	}

	// Create second manager (should load existing index)
	manager2, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}
	defer manager2.Close()

	// Verify chunk exists in new manager
	retrieved, err := manager2.retrieveChunk(hash)
	if err != nil {
		t.Fatalf("Failed to retrieve chunk with new manager: %v", err)
	}

	if !bytes.Equal(testData, retrieved) {
		t.Error("Retrieved data doesn't match after manager reload")
	}
}

func TestMetrics(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      100,
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Store file with some repetition
	testFile := filepath.Join(tempDir, "test.txt")
	// Create data with pattern: AAABBBAAABBB (should have some deduplication)
	testData := append(bytes.Repeat([]byte("A"), 100), bytes.Repeat([]byte("B"), 100)...)
	testData = append(testData, bytes.Repeat([]byte("A"), 100)...)
	testData = append(testData, bytes.Repeat([]byte("B"), 100)...)

	os.WriteFile(testFile, testData, 0o644)

	manifest, err := manager.StoreFile(testFile)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	metrics := manager.GetMetrics()

	if metrics.TotalChunks != int64(manifest.TotalChunks) {
		t.Errorf("Metrics total chunks mismatch")
	}

	if metrics.TotalBytes != manifest.OriginalSize {
		t.Errorf("Metrics total bytes mismatch")
	}

	// Should have deduplication since pattern repeats
	if metrics.UniqueChunks >= metrics.TotalChunks {
		t.Error("Expected some deduplication, but got none")
	}
}

func TestCacheMetrics(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      1024,
		CacheSize:      2,
	}

	manager, err := NewDeduplicationManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	// Store chunks
	data1 := []byte("data1")
	data2 := []byte("data2")
	hash1 := manager.hashChunk(data1)
	hash2 := manager.hashChunk(data2)

	manager.storeChunk(hash1, data1)
	manager.storeChunk(hash2, data2)

	// Clear cache to force a miss on next retrieval
	manager.cache.Clear()

	// First retrieval should be cache miss (loads from disk)
	manager.retrieveChunk(hash1)

	// Second retrieval should be cache hit
	manager.retrieveChunk(hash1)

	metrics := manager.GetMetrics()

	if metrics.CacheHits == 0 {
		t.Error("Expected at least one cache hit")
	}

	if metrics.CacheMisses == 0 {
		t.Error("Expected at least one cache miss")
	}
}

func BenchmarkStoreChunk(b *testing.B) {
	tempDir := b.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      4 * 1024 * 1024,
	}

	manager, _ := NewDeduplicationManager(config)
	defer manager.Close()

	testData := bytes.Repeat([]byte("X"), 4*1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := manager.hashChunk(testData)
		manager.storeChunk(hash, testData)
	}
}

func BenchmarkRetrieveChunk(b *testing.B) {
	tempDir := b.TempDir()

	config := &DeduplicationConfig{
		Enabled:        true,
		ChunkStorePath: filepath.Join(tempDir, "chunks"),
		IndexPath:      filepath.Join(tempDir, "index.json"),
		ChunkSize:      4 * 1024 * 1024,
		CacheSize:      100,
	}

	manager, _ := NewDeduplicationManager(config)
	defer manager.Close()

	testData := bytes.Repeat([]byte("X"), 4*1024*1024)
	hash := manager.hashChunk(testData)
	manager.storeChunk(hash, testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.retrieveChunk(hash)
	}
}
