package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDeltaManager(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager.config.BlockSize != 4096 {
		t.Errorf("Expected default block size 4096, got %d", manager.config.BlockSize)
	}

	if manager.config.MaxChainLength != 10 {
		t.Errorf("Expected default max chain length 10, got %d", manager.config.MaxChainLength)
	}
}

func TestCreateAndApplyDelta(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		BlockSize:      512, // Smaller for testing
		MinFileSize:    1, // Minimum 1 byte, allowing small test files
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create source file
	sourcePath := filepath.Join(tempDir, "source.txt")
	sourceData := []byte("Hello, this is the original content of the file.")
	err = os.WriteFile(sourcePath, sourceData, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create target file (modified version)
	targetPath := filepath.Join(tempDir, "target.txt")
	targetData := []byte("Hello, this is the MODIFIED content of the file with more data.")
	err = os.WriteFile(targetPath, targetData, 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create delta
	delta, err := manager.CreateDelta(sourcePath, targetPath, "base-1")
	if err != nil {
		t.Fatalf("Failed to create delta: %v", err)
	}

	if delta.SourceSize != int64(len(sourceData)) {
		t.Errorf("Expected source size %d, got %d", len(sourceData), delta.SourceSize)
	}

	if delta.TargetSize != int64(len(targetData)) {
		t.Errorf("Expected target size %d, got %d", len(targetData), delta.TargetSize)
	}

	// Delta should be smaller than full target
	if delta.DeltaSize >= delta.TargetSize {
		t.Logf("Warning: Delta size (%d) is not smaller than target (%d)", delta.DeltaSize, delta.TargetSize)
	}

	// Apply delta
	outputPath := filepath.Join(tempDir, "output.txt")
	err = manager.ApplyDelta(sourcePath, delta, outputPath)
	if err != nil {
		t.Fatalf("Failed to apply delta: %v", err)
	}

	// Verify output matches target
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if !bytes.Equal(targetData, outputData) {
		t.Error("Output doesn't match target after applying delta")
		t.Logf("Expected: %s", string(targetData))
		t.Logf("Got: %s", string(outputData))
	}
}

func TestDeltaWithIdenticalFiles(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create two identical files
	data := []byte("Identical content in both files")
	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")

	os.WriteFile(sourcePath, data, 0644)
	os.WriteFile(targetPath, data, 0644)

	// Create delta
	delta, err := manager.CreateDelta(sourcePath, targetPath, "base-1")
	if err != nil {
		t.Fatalf("Failed to create delta: %v", err)
	}

	// Delta should be very small for identical files
	if delta.DeltaSize > int64(len(data))/2 {
		t.Logf("Delta size (%d) is larger than expected for identical files", delta.DeltaSize)
	}

	// Hashes should match
	if delta.SourceHash != delta.TargetHash {
		t.Error("Source and target hashes should match for identical files")
	}
}

func TestDeltaWithCompression(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		Compression:    true,
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create files with repetitive data (compresses well)
	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")

	sourceData := bytes.Repeat([]byte("A"), 1000)
	targetData := bytes.Repeat([]byte("B"), 1000)

	os.WriteFile(sourcePath, sourceData, 0644)
	os.WriteFile(targetPath, targetData, 0644)

	// Create delta
	delta, err := manager.CreateDelta(sourcePath, targetPath, "base-1")
	if err != nil {
		t.Fatalf("Failed to create delta: %v", err)
	}

	if !delta.Compressed {
		t.Error("Delta should be marked as compressed")
	}

	// Apply delta
	outputPath := filepath.Join(tempDir, "output.txt")
	err = manager.ApplyDelta(sourcePath, delta, outputPath)
	if err != nil {
		t.Fatalf("Failed to apply compressed delta: %v", err)
	}

	// Verify output
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if !bytes.Equal(targetData, outputData) {
		t.Error("Output doesn't match target after applying compressed delta")
	}
}

func TestBackupChain(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		BlockSize:      512,
		MaxChainLength: 3,
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	baseID := "test-chain"

	// Create base file
	basePath := filepath.Join(tempDir, "base.txt")
	baseData := []byte("Version 0: Original content")
	os.WriteFile(basePath, baseData, 0644)

	// Create a chain of 3 versions
	currentPath := basePath
	for i := 1; i <= 3; i++ {
		nextPath := filepath.Join(tempDir, "version-"+string(rune('0'+i))+".txt")
		nextData := append(baseData, []byte(" - Update "+string(rune('0'+i)))...)
		os.WriteFile(nextPath, nextData, 0644)

		_, err := manager.CreateDelta(currentPath, nextPath, baseID)
		if err != nil {
			t.Fatalf("Failed to create delta %d: %v", i, err)
		}

		currentPath = nextPath
	}

	// Check chain
	chain, err := manager.GetChain(baseID)
	if err != nil {
		t.Fatalf("Failed to get chain: %v", err)
	}

	if chain.ChainLength != 3 {
		t.Errorf("Expected chain length 3, got %d", chain.ChainLength)
	}

	// Check if needs full backup
	if !manager.NeedsFullBackup(baseID) {
		t.Error("Should need full backup after reaching max chain length")
	}
}

func TestRestoreFromChain(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		BlockSize:      256,
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	baseID := "restore-chain"

	// Create versions
	versions := []string{
		"Version 1 content",
		"Version 2 content with more data",
		"Version 3 content with even more data and changes",
	}

	basePath := filepath.Join(tempDir, "v0.txt")
	os.WriteFile(basePath, []byte(versions[0]), 0644)

	for i := 1; i < len(versions); i++ {
		prevPath := filepath.Join(tempDir, "v"+string(rune('0'+i-1))+".txt")
		nextPath := filepath.Join(tempDir, "v"+string(rune('0'+i))+".txt")
		os.WriteFile(nextPath, []byte(versions[i]), 0644)

		_, err := manager.CreateDelta(prevPath, nextPath, baseID)
		if err != nil {
			t.Fatalf("Failed to create delta: %v", err)
		}
	}

	// Restore version 2 (second delta)
	outputPath := filepath.Join(tempDir, "restored.txt")
	err = manager.RestoreFromChain(basePath, baseID, 2, outputPath)
	if err != nil {
		t.Fatalf("Failed to restore from chain: %v", err)
	}

	// Verify restored data
	restoredData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	expectedData := []byte(versions[2])
	if !bytes.Equal(restoredData, expectedData) {
		t.Error("Restored data doesn't match expected version")
		t.Logf("Expected: %s", string(expectedData))
		t.Logf("Got: %s", string(restoredData))
	}
}

func TestHashVerification(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create files
	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")
	sourceData := []byte("Source data")
	targetData := []byte("Target data")

	os.WriteFile(sourcePath, sourceData, 0644)
	os.WriteFile(targetPath, targetData, 0644)

	// Create delta
	delta, err := manager.CreateDelta(sourcePath, targetPath, "base-1")
	if err != nil {
		t.Fatalf("Failed to create delta: %v", err)
	}

	// Corrupt source file
	corruptedSource := filepath.Join(tempDir, "corrupted.txt")
	os.WriteFile(corruptedSource, []byte("Wrong data"), 0644)

	// Try to apply delta with wrong source
	outputPath := filepath.Join(tempDir, "output.txt")
	err = manager.ApplyDelta(corruptedSource, delta, outputPath)
	if err == nil {
		t.Error("Expected error when applying delta with wrong source")
	}
	if err != nil && err.Error() != "source hash mismatch (corrupted or wrong base)" {
		t.Logf("Got error: %v", err)
	}
}

func TestDeltaMetrics(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create some deltas
	for i := 0; i < 3; i++ {
		sourcePath := filepath.Join(tempDir, "source-"+string(rune('0'+i))+".txt")
		targetPath := filepath.Join(tempDir, "target-"+string(rune('0'+i))+".txt")

		sourceData := bytes.Repeat([]byte("X"), 1000)
		targetData := bytes.Repeat([]byte("Y"), 1000)

		os.WriteFile(sourcePath, sourceData, 0644)
		os.WriteFile(targetPath, targetData, 0644)

		_, err := manager.CreateDelta(sourcePath, targetPath, "base-"+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("Failed to create delta: %v", err)
		}
	}

	metrics := manager.GetMetrics()

	if metrics.TotalDeltaBackups != 3 {
		t.Errorf("Expected 3 delta backups, got %d", metrics.TotalDeltaBackups)
	}

	if metrics.TotalBytesOriginal == 0 {
		t.Error("Expected non-zero original bytes")
	}

	if metrics.SpaceSaved <= 0 {
		t.Logf("Warning: Expected space savings, got %d", metrics.SpaceSaved)
	}
}

func TestListChains(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create multiple chains
	for i := 0; i < 3; i++ {
		baseID := "chain-" + string(rune('A'+i))
		sourcePath := filepath.Join(tempDir, baseID+"-source.txt")
		targetPath := filepath.Join(tempDir, baseID+"-target.txt")

		os.WriteFile(sourcePath, []byte("source"), 0644)
		os.WriteFile(targetPath, []byte("target"), 0644)

		_, err := manager.CreateDelta(sourcePath, targetPath, baseID)
		if err != nil {
			t.Fatalf("Failed to create delta for chain %s: %v", baseID, err)
		}

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	chains := manager.ListChains()

	if len(chains) != 3 {
		t.Errorf("Expected 3 chains, got %d", len(chains))
	}

	// Verify chains are sorted by creation date (newest first)
	for i := 0; i < len(chains)-1; i++ {
		if chains[i].CreatedAt.Before(chains[i+1].CreatedAt) {
			t.Error("Chains should be sorted by creation date (newest first)")
		}
	}
}

func TestBlockIndexBuilding(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		BlockSize:      16, // Small block size for testing
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	data := []byte("This is test data for block indexing")
	blocks := manager.buildBlockIndex(data)

	if len(blocks) == 0 {
		t.Error("Expected at least one block")
	}

	// Verify blocks have correct properties
	for hash, block := range blocks {
		if hash == "" {
			t.Error("Block hash should not be empty")
		}
		if block.Length <= 0 {
			t.Error("Block length should be positive")
		}
		if block.Offset < 0 {
			t.Error("Block offset should not be negative")
		}
	}
}

func TestOperationOptimization(t *testing.T) {
	tempDir := t.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
	}

	manager, err := NewDeltaManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create operations with adjacent inserts
	ops := []DeltaOperation{
		{Type: OpInsert, Offset: 0, Length: 5, Data: []byte("Hello")},
		{Type: OpInsert, Offset: 5, Length: 6, Data: []byte(" World")},
		{Type: OpCopy, Offset: 10, Length: 5},
	}

	optimized := manager.optimizeOperations(ops)

	// First two inserts should be combined
	if len(optimized) != 2 {
		t.Errorf("Expected 2 operations after optimization, got %d", len(optimized))
	}

	if optimized[0].Type != OpInsert {
		t.Error("First operation should be insert")
	}

	if optimized[0].Length != 11 {
		t.Errorf("Combined insert should have length 11, got %d", optimized[0].Length)
	}

	expectedData := []byte("Hello World")
	if !bytes.Equal(optimized[0].Data, expectedData) {
		t.Error("Combined insert data is incorrect")
	}
}

func BenchmarkCreateDelta(b *testing.B) {
	tempDir := b.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, _ := NewDeltaManager(config)

	// Create test files
	sourceData := bytes.Repeat([]byte("A"), 100*1024) // 100KB
	targetData := bytes.Repeat([]byte("B"), 100*1024)

	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")

	os.WriteFile(sourcePath, sourceData, 0644)
	os.WriteFile(targetPath, targetData, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CreateDelta(sourcePath, targetPath, "bench-base")
	}
}

func BenchmarkApplyDelta(b *testing.B) {
	tempDir := b.TempDir()

	config := &DeltaConfig{
		Enabled:        true,
		DeltaStorePath: filepath.Join(tempDir, "deltas"),
		MetadataPath:   filepath.Join(tempDir, "metadata"),
		MinFileSize:    1,
	}

	manager, _ := NewDeltaManager(config)

	// Create delta
	sourceData := bytes.Repeat([]byte("A"), 100*1024)
	targetData := bytes.Repeat([]byte("B"), 100*1024)

	sourcePath := filepath.Join(tempDir, "source.txt")
	targetPath := filepath.Join(tempDir, "target.txt")
	outputPath := filepath.Join(tempDir, "output.txt")

	os.WriteFile(sourcePath, sourceData, 0644)
	os.WriteFile(targetPath, targetData, 0644)

	delta, _ := manager.CreateDelta(sourcePath, targetPath, "bench-base")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ApplyDelta(sourcePath, delta, outputPath)
	}
}
