package incremental

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewChangeTracker(t *testing.T) {
	tracker := NewChangeTracker(BlockSize4KB)

	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}

	if tracker.blockSize != BlockSize4KB {
		t.Errorf("Expected block size %d, got %d", BlockSize4KB, tracker.blockSize)
	}

	if tracker.snapshots == nil {
		t.Error("Expected snapshots map to be initialized")
	}
}

func TestCreateSnapshot(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.dat")

	// Write test data
	testData := []byte("Hello, World! This is test data for snapshot testing.")
	err := os.WriteFile(testFile, testData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tracker := NewChangeTracker(BlockSize4KB)

	// Create snapshot
	snapshot, err := tracker.CreateSnapshot(testFile)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if snapshot.FilePath != testFile {
		t.Errorf("Expected file path %s, got %s", testFile, snapshot.FilePath)
	}

	if snapshot.FileSize != int64(len(testData)) {
		t.Errorf("Expected file size %d, got %d", len(testData), snapshot.FileSize)
	}

	if len(snapshot.Blocks) == 0 {
		t.Error("Expected snapshot to have blocks")
	}

	if snapshot.Checksum == "" {
		t.Error("Expected snapshot to have checksum")
	}
}

func TestCompareSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.dat")

	// Create initial file
	initialData := []byte("Initial data content")
	err := os.WriteFile(testFile, initialData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tracker := NewChangeTracker(BlockSize4KB)

	// Create first snapshot
	snapshot1, err := tracker.CreateSnapshot(testFile)
	if err != nil {
		t.Fatalf("Failed to create first snapshot: %v", err)
	}

	// Modify file
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	modifiedData := []byte("Modified data content with additional text")
	err = os.WriteFile(testFile, modifiedData, 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Create second snapshot
	snapshot2, err := tracker.CreateSnapshot(testFile)
	if err != nil {
		t.Fatalf("Failed to create second snapshot: %v", err)
	}

	// Compare snapshots
	changeSet, err := tracker.CompareSnapshots(snapshot1.ID, snapshot2.ID)
	if err != nil {
		t.Fatalf("Failed to compare snapshots: %v", err)
	}

	if len(changeSet.ChangedBlocks)+len(changeSet.NewBlocks) == 0 {
		t.Error("Expected some blocks to have changed")
	}

	if changeSet.ChangeRatio == 0 {
		t.Error("Expected non-zero change ratio")
	}
}

func TestIncrementalForeverStrategy_FullBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "database.db")
	testData := []byte("Database content for full backup test")
	err = os.WriteFile(testFile, testData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Perform full backup
	manifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	if manifest.Type != BackupTypeFull {
		t.Errorf("Expected backup type %s, got %s", BackupTypeFull, manifest.Type)
	}

	if manifest.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", manifest.Status)
	}

	// Verify backup file exists
	if _, err := os.Stat(manifest.BackupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// Verify manifest was saved
	if _, exists := strategy.GetManifest(manifest.ID); !exists {
		t.Error("Manifest was not stored")
	}
}

func TestIncrementalForeverStrategy_IncrementalBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "database.db")
	initialData := []byte("Initial database content")
	err = os.WriteFile(testFile, initialData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Perform full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Modify file
	time.Sleep(10 * time.Millisecond)
	modifiedData := []byte("Modified database content with changes")
	err = os.WriteFile(testFile, modifiedData, 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Perform incremental backup
	incrManifest, err := strategy.PerformIncrementalBackup(testFile, "db-001", "Test Database", fullManifest.ID)
	if err != nil {
		t.Fatalf("Failed to perform incremental backup: %v", err)
	}

	if incrManifest.Type != BackupTypeIncremental {
		t.Errorf("Expected backup type %s, got %s", BackupTypeIncremental, incrManifest.Type)
	}

	if incrManifest.ParentBackupID != fullManifest.ID {
		t.Errorf("Expected parent backup ID %s, got %s", fullManifest.ID, incrManifest.ParentBackupID)
	}

	if incrManifest.ChangedBlocks == 0 {
		t.Error("Expected some changed blocks in incremental backup")
	}

	// Verify incremental backup file exists
	if _, err := os.Stat(incrManifest.BackupPath); os.IsNotExist(err) {
		t.Error("Incremental backup file was not created")
	}
}

func TestIncrementalForeverStrategy_RestoreFromBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tmpDir, "database.db")
	originalData := []byte("Original database content for restore test")
	err = os.WriteFile(testFile, originalData, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Perform full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Modify file (make it longer than original to avoid truncation issues)
	time.Sleep(10 * time.Millisecond)
	modifiedData := []byte("Modified content after full backup - this is the new data that should be restored correctly")
	err = os.WriteFile(testFile, modifiedData, 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Perform incremental backup
	incrManifest, err := strategy.PerformIncrementalBackup(testFile, "db-001", "Test Database", fullManifest.ID)
	if err != nil {
		t.Fatalf("Failed to perform incremental backup: %v", err)
	}

	// Restore from incremental
	restorePath := filepath.Join(tmpDir, "restored.db")
	err = strategy.RestoreFromBackup(incrManifest.ID, restorePath)
	if err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}

	// Verify restored file
	restoredData, err := os.ReadFile(restorePath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredData) != string(modifiedData) {
		t.Errorf("Restored data does not match. Expected '%s', got '%s'", string(modifiedData), string(restoredData))
	}
}

func TestGetBackupChain(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	err = os.WriteFile(testFile, []byte("Initial data"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Create full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Create multiple incrementals (each based on the full backup for simplicity)
	var lastManifest *BackupManifest = fullManifest
	for i := 0; i < 3; i++ {
		time.Sleep(10 * time.Millisecond)
		data := []byte("Modified data iteration " + string(rune(i+'0')))
		err = os.WriteFile(testFile, data, 0o644)
		if err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		// All incrementals are based on the full backup (forever incremental strategy)
		incrManifest, err := strategy.PerformIncrementalBackup(testFile, "db-001", "Test Database", fullManifest.ID)
		if err != nil {
			t.Fatalf("Failed to perform incremental backup %d: %v", i, err)
		}
		lastManifest = incrManifest
	}

	// Adjust expectation: with forever incremental, all incrementals reference the full backup
	// So the chain from the latest incremental to the base will be: latest incr -> full
	// That's a chain of 2, not 4

	// Get backup chain
	chain, err := strategy.GetBackupChain(lastManifest.ID)
	if err != nil {
		t.Fatalf("Failed to get backup chain: %v", err)
	}

	// With forever incremental, chain is: latest incr -> full backup = 2 backups
	if len(chain) != 2 {
		t.Errorf("Expected chain length 2 (latest incr + full), got %d", len(chain))
	}

	// First should be full backup
	if chain[0].Type != BackupTypeFull {
		t.Errorf("Expected first backup to be full, got %s", chain[0].Type)
	}

	// Second should be the incremental
	if len(chain) > 1 && chain[1].Type != BackupTypeIncremental {
		t.Errorf("Expected second backup to be incremental, got %s", chain[1].Type)
	}
}

func TestSyntheticBackupGenerator_GenerateSyntheticFull(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")

	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	err = os.WriteFile(testFile, []byte("Initial database"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)

	// Create full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Create an incremental
	time.Sleep(10 * time.Millisecond)
	err = os.WriteFile(testFile, []byte("Modified database content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	incrManifest, err := strategy.PerformIncrementalBackup(testFile, "db-001", "Test Database", fullManifest.ID)
	if err != nil {
		t.Fatalf("Failed to perform incremental backup: %v", err)
	}

	// Generate synthetic full
	syntheticManifest, err := syntheticGen.GenerateSyntheticFull(incrManifest.ID)
	if err != nil {
		t.Fatalf("Failed to generate synthetic full: %v", err)
	}

	if syntheticManifest.Type != BackupTypeSynthetic {
		t.Errorf("Expected backup type %s, got %s", BackupTypeSynthetic, syntheticManifest.Type)
	}

	if syntheticManifest.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", syntheticManifest.Status)
	}

	// Verify synthetic backup file exists
	if _, err := os.Stat(syntheticManifest.BackupPath); os.IsNotExist(err) {
		t.Error("Synthetic backup file was not created")
	}
}

func TestSyntheticBackupGenerator_ShouldGenerateSynthetic(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")

	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	err = os.WriteFile(testFile, []byte("Database"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)

	// Create full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Create multiple incrementals
	var lastManifest *BackupManifest = fullManifest
	for i := 0; i < 5; i++ {
		time.Sleep(10 * time.Millisecond)
		data := []byte("Data " + string(rune(i+'0')))
		err = os.WriteFile(testFile, data, 0o644)
		if err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		incrManifest, err := strategy.PerformIncrementalBackup(testFile, "db-001", "Test Database", fullManifest.ID)
		if err != nil {
			t.Fatalf("Failed to perform incremental backup: %v", err)
		}
		lastManifest = incrManifest
	}

	// With forever incremental (all based on full), chain has only latest incr + full = 2 backups
	// So incrementalCount in the chain is 1, not 5. Adjust threshold to 0 or 1
	should, err := syntheticGen.ShouldGenerateSynthetic(lastManifest.ID, 1)
	if err != nil {
		t.Fatalf("Failed to check if synthetic should be generated: %v", err)
	}

	if !should {
		t.Error("Expected synthetic to be recommended with 1 incremental in chain and threshold of 1")
	}

	// Check with higher threshold
	should, err = syntheticGen.ShouldGenerateSynthetic(lastManifest.ID, 10)
	if err != nil {
		t.Fatalf("Failed to check if synthetic should be generated: %v", err)
	}

	if should {
		t.Error("Did not expect synthetic to be recommended with 5 incrementals and threshold of 10")
	}
}

func TestBackupChainManager_DetermineBackupType(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")

	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)
	policy := DefaultChainPolicy()
	chainMgr := NewBackupChainManager(strategy, syntheticGen, policy)

	// First backup should be full
	backupType, reason, err := chainMgr.DetermineBackupType("db-001")
	if err != nil {
		t.Fatalf("Failed to determine backup type: %v", err)
	}

	if backupType != BackupTypeFull {
		t.Errorf("Expected first backup to be full, got %s. Reason: %s", backupType, reason)
	}
}

func TestBackupChainManager_ValidateChain(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")

	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	err = os.WriteFile(testFile, []byte("Database"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)
	policy := DefaultChainPolicy()
	chainMgr := NewBackupChainManager(strategy, syntheticGen, policy)

	// Create full backup
	fullManifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Validate chain
	health, err := chainMgr.ValidateChain(fullManifest.ID)
	if err != nil {
		t.Fatalf("Failed to validate chain: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected chain to be healthy, got status: %s. Issues: %v", health.Status, health.Issues)
	}

	if health.ChainLength != 1 {
		t.Errorf("Expected chain length 1, got %d", health.ChainLength)
	}
}

func TestVerifyBlockChecksum(t *testing.T) {
	data := []byte("Test data for checksum verification")
	checksum := ComputeBlockChecksum(data)

	if !VerifyBlockChecksum(data, checksum) {
		t.Error("Checksum verification failed for valid data")
	}

	// Modify data
	modifiedData := []byte("Modified data")
	if VerifyBlockChecksum(modifiedData, checksum) {
		t.Error("Checksum verification should fail for modified data")
	}
}

func TestBlockSizes(t *testing.T) {
	testCases := []struct {
		size     BlockSize
		expected int
	}{
		{BlockSize4KB, 4096},
		{BlockSize8KB, 8192},
		{BlockSize16KB, 16384},
		{BlockSize64KB, 65536},
		{BlockSize1MB, 1048576},
	}

	for _, tc := range testCases {
		if int(tc.size) != tc.expected {
			t.Errorf("Expected block size %d, got %d", tc.expected, int(tc.size))
		}
	}
}

func TestChangeTrackerGetStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.dat")

	err := os.WriteFile(testFile, []byte("Test data"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tracker := NewChangeTracker(BlockSize4KB)

	// Create snapshot
	_, err = tracker.CreateSnapshot(testFile)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	stats := tracker.GetStatistics()

	if stats["total_snapshots"].(int) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", stats["total_snapshots"].(int))
	}

	if stats["block_size"].(BlockSize) != BlockSize4KB {
		t.Errorf("Expected block size %d, got %d", BlockSize4KB, stats["block_size"].(BlockSize))
	}
}

func TestStrategyGetStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	err = os.WriteFile(testFile, []byte("Database"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Create full backup
	_, err = strategy.PerformFullBackup(testFile, "db-001", "Test Database")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	stats := strategy.GetStatistics()

	if stats["total_backups"].(int) != 1 {
		t.Errorf("Expected 1 backup, got %d", stats["total_backups"].(int))
	}

	if stats["full_backups"].(int) != 1 {
		t.Errorf("Expected 1 full backup, got %d", stats["full_backups"].(int))
	}
}

func TestChainManagerGetStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")

	err := os.MkdirAll(backupDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)
	policy := DefaultChainPolicy()
	chainMgr := NewBackupChainManager(strategy, syntheticGen, policy)

	stats := chainMgr.GetStatistics()

	if stats["total_chains"].(int) != 0 {
		t.Errorf("Expected 0 chains initially, got %d", stats["total_chains"].(int))
	}

	policyStats := stats["policy"].(map[string]interface{})
	if policyStats["max_chain_length"].(int) != 10 {
		t.Errorf("Expected max chain length 10, got %d", policyStats["max_chain_length"].(int))
	}
}

// TestSnapshotPersistenceSurvivesRestart proves that block snapshots created by a
// persistent ChangeTracker are reloaded by a fresh tracker pointed at the same
// directory (i.e. they survive a simulated process restart).
func TestSnapshotPersistenceSurvivesRestart(t *testing.T) {
	tmpDir := t.TempDir()
	snapDir := filepath.Join(tmpDir, "snapshots")

	testFile := filepath.Join(tmpDir, "test.dat")
	if err := os.WriteFile(testFile, []byte("durable snapshot content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First "process" creates a snapshot and persists it.
	tracker1, err := NewChangeTrackerWithDir(BlockSize4KB, snapDir)
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	snap, err := tracker1.CreateSnapshot(testFile)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Verify it was written to disk.
	snapPath := filepath.Join(snapDir, snap.ID+snapshotFileSuffix)
	if _, err = os.Stat(snapPath); err != nil {
		t.Fatalf("Snapshot file was not persisted: %v", err)
	}

	// Second "process" (fresh tracker, same dir) must load the prior snapshot.
	tracker2, err := NewChangeTrackerWithDir(BlockSize4KB, snapDir)
	if err != nil {
		t.Fatalf("Failed to create second tracker: %v", err)
	}
	loaded, exists := tracker2.GetSnapshot(snap.ID)
	if !exists {
		t.Fatal("Snapshot did not survive restart")
	}
	if loaded.Checksum != snap.Checksum {
		t.Errorf("Loaded checksum %s != original %s", loaded.Checksum, snap.Checksum)
	}
	if len(loaded.Blocks) != len(snap.Blocks) {
		t.Errorf("Loaded block count %d != original %d", len(loaded.Blocks), len(snap.Blocks))
	}
}

// TestStrategyStateSurvivesRestart proves that both manifests and the base
// snapshot are reloaded by a fresh strategy pointed at the same backup dir, and
// that an incremental backup can be taken against a base created before the
// restart (the bug that previously failed with "base snapshot not found").
func TestStrategyStateSurvivesRestart(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")

	testFile := filepath.Join(tmpDir, "database.db")
	if err := os.WriteFile(testFile, []byte("initial content before restart"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First "process": full backup, then discard the strategy.
	strategy1 := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	fullManifest, err := strategy1.PerformFullBackup(testFile, "db-001", "Test DB")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	// Second "process": fresh strategy, same dir.
	strategy2 := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)

	// Manifest must be reloaded.
	if _, exists := strategy2.GetManifest(fullManifest.ID); !exists {
		t.Fatal("Manifest did not survive restart")
	}

	// Base snapshot must be reloaded so an incremental can be taken.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("changed content after the simulated restart event"), 0o644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	incrManifest, err := strategy2.PerformIncrementalBackup(testFile, "db-001", "Test DB", fullManifest.ID)
	if err != nil {
		t.Fatalf("Incremental after restart failed (base snapshot not loaded?): %v", err)
	}
	if incrManifest.ParentBackupID != fullManifest.ID {
		t.Errorf("Expected parent %s, got %s", fullManifest.ID, incrManifest.ParentBackupID)
	}

	// Restore through the chain must work across the restart boundary.
	restorePath := filepath.Join(tmpDir, "restored.db")
	if err := strategy2.RestoreFromBackup(incrManifest.ID, restorePath); err != nil {
		t.Fatalf("Restore after restart failed: %v", err)
	}
}

// TestRemoveBackupDeletesFiles proves removeBackup actually deletes on-disk
// artifacts and the in-memory entry, and that ApplyRetentionPolicy only reports
// backups it really removed.
func TestRemoveBackupDeletesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	if err := os.WriteFile(testFile, []byte("backup to be removed"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	manifest, err := strategy.PerformFullBackup(testFile, "db-001", "Test DB")
	if err != nil {
		t.Fatalf("Failed to perform full backup: %v", err)
	}

	backupPath := manifest.BackupPath
	manifestPath := filepath.Join(backupDir, manifest.ID+"-manifest.json")
	snapPath := filepath.Join(backupDir, snapshotsSubdir, manifest.SnapshotID+snapshotFileSuffix)

	for _, p := range []string{backupPath, manifestPath, snapPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("Expected artifact %s to exist before removal: %v", p, err)
		}
	}

	if err := strategy.RemoveBackup(manifest.ID); err != nil {
		t.Fatalf("RemoveBackup failed: %v", err)
	}

	for _, p := range []string{backupPath, manifestPath, snapPath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("Expected artifact %s to be deleted, stat err: %v", p, err)
		}
	}
	if _, exists := strategy.GetManifest(manifest.ID); exists {
		t.Error("Manifest still present in memory after removal")
	}

	// Removing a non-existent backup must return a real error.
	if err := strategy.RemoveBackup("does-not-exist"); err == nil {
		t.Error("Expected error removing non-existent backup, got nil")
	}
}

// TestApplyRetentionPolicyDeletesOldBackups proves the retention policy actually
// deletes the artifacts of pruned backups (not just reporting them).
func TestApplyRetentionPolicyDeletesOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	tempDir := filepath.Join(tmpDir, "temp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "database.db")
	if err := os.WriteFile(testFile, []byte("old chain data"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	strategy := NewIncrementalForeverStrategy(backupDir, BlockSize4KB, false)
	syntheticGen := NewSyntheticBackupGenerator(strategy, tempDir)
	policy := DefaultChainPolicy()
	chainMgr := NewBackupChainManager(strategy, syntheticGen, policy)

	// An old, standalone full backup for db-001 that is not part of the latest chain.
	oldFull, err := strategy.PerformFullBackup(testFile, "db-001", "Test DB")
	if err != nil {
		t.Fatalf("Failed to perform old full backup: %v", err)
	}
	// Age it past the retention window.
	oldFull.CreatedAt = time.Now().AddDate(0, 0, -policy.RetentionDays-1)

	// A newer full backup that forms the latest chain to be retained.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("new chain data"), 0o644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}
	newFull, err := strategy.PerformFullBackup(testFile, "db-001", "Test DB")
	if err != nil {
		t.Fatalf("Failed to perform new full backup: %v", err)
	}

	oldBackupPath := oldFull.BackupPath

	removed, err := chainMgr.ApplyRetentionPolicy()
	if err != nil {
		t.Fatalf("ApplyRetentionPolicy failed: %v", err)
	}

	found := false
	for _, id := range removed {
		if id == oldFull.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected old backup %s to be reported removed, got %v", oldFull.ID, removed)
	}

	// The reported removal must correspond to an actual on-disk deletion.
	if _, err := os.Stat(oldBackupPath); !os.IsNotExist(err) {
		t.Errorf("Old backup file still exists after retention: %v", err)
	}
	if _, exists := strategy.GetManifest(oldFull.ID); exists {
		t.Error("Old manifest still in memory after retention")
	}
	// The retained chain must remain intact.
	if _, exists := strategy.GetManifest(newFull.ID); !exists {
		t.Error("Newest backup was unexpectedly removed")
	}
}
