package consistency

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Test Hook Manager

func TestHookManager_RegisterHook(t *testing.T) {
	hm := NewHookManager(nil)

	hook := &Hook{
		Type:    HookTypePreBackup,
		Command: "/bin/echo",
		Args:    []string{"test"},
		Enabled: true,
	}

	err := hm.RegisterHook(hook)
	if err != nil {
		t.Fatalf("Failed to register hook: %v", err)
	}

	if hook.ID == "" {
		t.Error("Hook ID should be auto-generated")
	}

	if hook.Timeout == 0 {
		t.Error("Hook timeout should have default value")
	}
}

func TestHookManager_RegisterInvalidHook(t *testing.T) {
	hm := NewHookManager(nil)

	// Test nil hook
	err := hm.RegisterHook(nil)
	if err == nil {
		t.Error("Should reject nil hook")
	}

	// Test empty command
	hook := &Hook{
		Type:    HookTypePreBackup,
		Command: "",
		Enabled: true,
	}
	err = hm.RegisterHook(hook)
	if err == nil {
		t.Error("Should reject hook with empty command")
	}
}

func TestHookManager_ExecuteSequential(t *testing.T) {
	hm := NewHookManager(&HookConfig{
		ExecutionMode: ExecutionModeSequential,
		GlobalTimeout: 1 * time.Minute,
		MaxConcurrent: 5,
	})

	// Register test hooks
	for i := 0; i < 3; i++ {
		hook := &Hook{
			Type:    HookTypePreBackup,
			Command: "/bin/echo",
			Args:    []string{fmt.Sprintf("test-%d", i)},
			Enabled: true,
			Timeout: 5 * time.Second,
		}
		if err := hm.RegisterHook(hook); err != nil {
			t.Fatalf("Failed to register hook: %v", err)
		}
	}

	ctx := context.Background()
	results, err := hm.ExecuteHooks(ctx, HookTypePreBackup, nil)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, result := range results {
		if !result.Success {
			t.Errorf("Hook %s failed: %v", result.HookID, result.Error)
		}
	}
}

func TestHookManager_ExecuteParallel(t *testing.T) {
	hm := NewHookManager(&HookConfig{
		ExecutionMode: ExecutionModeParallel,
		GlobalTimeout: 1 * time.Minute,
		MaxConcurrent: 3,
	})

	for i := 0; i < 5; i++ {
		hook := &Hook{
			Type:    HookTypePreBackup,
			Command: "/bin/sleep",
			Args:    []string{"0.1"},
			Enabled: true,
			Timeout: 5 * time.Second,
		}
		if err := hm.RegisterHook(hook); err != nil {
			t.Fatalf("Failed to register hook: %v", err)
		}
	}

	ctx := context.Background()
	start := time.Now()
	results, err := hm.ExecuteHooks(ctx, HookTypePreBackup, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// With max concurrency of 3, 5 hooks should complete faster than 5 * 0.1s = 0.5s
	// but slower than 2 * 0.1s = 0.2s
	if duration > 1*time.Second {
		t.Errorf("Parallel execution took too long: %v", duration)
	}
}

func TestHookManager_ExecuteWithConditions(t *testing.T) {
	hm := NewHookManager(nil)

	// Hook with conditions
	hook := &Hook{
		Type:    HookTypePreBackup,
		Command: "/bin/echo",
		Args:    []string{"conditional"},
		Enabled: true,
		Conditions: map[string]string{
			"database": "postgres",
		},
	}
	if err := hm.RegisterHook(hook); err != nil {
		t.Fatalf("Failed to register hook: %v", err)
	}

	ctx := context.Background()

	// Execute with matching conditions
	metadata := map[string]string{"database": "postgres"}
	results, err := hm.ExecuteHooks(ctx, HookTypePreBackup, metadata)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result with matching conditions, got %d", len(results))
	}

	// Execute with non-matching conditions
	metadata = map[string]string{"database": "mysql"}
	results, err = hm.ExecuteHooks(ctx, HookTypePreBackup, metadata)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results with non-matching conditions, got %d", len(results))
	}
}

func TestHookManager_LoadFromDirectory(t *testing.T) {
	// Create temporary directory with hook scripts
	tmpDir := t.TempDir()

	// Create test hook scripts
	scripts := []string{
		"pre-backup-test.sh",
		"post-backup-test.sh",
		"pre-quiesce-test.sh",
	}

	for _, script := range scripts {
		path := filepath.Join(tmpDir, script)
		content := "#!/bin/bash\necho 'test'\n"
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}
	}

	hm := NewHookManager(nil)
	err := hm.LoadHooksFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load hooks from directory: %v", err)
	}

	// Verify hooks were loaded
	hm.mu.RLock()
	preBackupHooks := len(hm.hooks[HookTypePreBackup])
	postBackupHooks := len(hm.hooks[HookTypePostBackup])
	preQuiesceHooks := len(hm.hooks[HookTypePreQuiesce])
	hm.mu.RUnlock()

	if preBackupHooks != 1 {
		t.Errorf("Expected 1 pre-backup hook, got %d", preBackupHooks)
	}
	if postBackupHooks != 1 {
		t.Errorf("Expected 1 post-backup hook, got %d", postBackupHooks)
	}
	if preQuiesceHooks != 1 {
		t.Errorf("Expected 1 pre-quiesce hook, got %d", preQuiesceHooks)
	}
}

func TestHookManager_GetResults(t *testing.T) {
	hm := NewHookManager(nil)

	hook := &Hook{
		Type:    HookTypePreBackup,
		Command: "/bin/echo",
		Args:    []string{"test"},
		Enabled: true,
	}
	if err := hm.RegisterHook(hook); err != nil {
		t.Fatalf("Failed to register hook: %v", err)
	}

	ctx := context.Background()
	_, err := hm.ExecuteHooks(ctx, HookTypePreBackup, nil)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	results := hm.GetResults()
	if len(results) == 0 {
		t.Error("Expected to retrieve hook results")
	}

	resultsByType := hm.GetResultsByType(HookTypePreBackup)
	if len(resultsByType) == 0 {
		t.Error("Expected to retrieve hook results by type")
	}
}

// Test Quiesce Manager

func TestQuiesceManager_Operations(t *testing.T) {
	qm := NewQuiesceManager()

	// Test basic operation tracking
	op := &QuiesceOperation{
		DatabaseType: "postgresql",
		DatabaseName: "testdb",
		State:        StateQuiesced,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	dbID := "test-db-1"
	qm.mu.Lock()
	qm.operations[dbID] = op
	qm.mu.Unlock()

	// Test GetOperation
	retrieved, exists := qm.GetOperation(dbID)
	if !exists {
		t.Error("Operation should exist")
	}
	if retrieved.DatabaseType != "postgresql" {
		t.Errorf("Expected postgresql, got %s", retrieved.DatabaseType)
	}

	// Test GetAllOperations
	all := qm.GetAllOperations()
	if len(all) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(all))
	}

	// Test ClearOperation
	qm.ClearOperation(dbID)
	_, exists = qm.GetOperation(dbID)
	if exists {
		t.Error("Operation should be cleared")
	}
}

// Test Consistency Coordinator

func TestConsistencyCoordinator_CreateConsistencyGroup(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	group := &ConsistencyGroup{
		Name:      "test-group",
		Databases: []string{"db1", "db2", "db3"},
		Level:     ConsistencyLevelGroup,
		Priority:  1,
	}

	err := cc.CreateConsistencyGroup(group)
	if err != nil {
		t.Fatalf("Failed to create consistency group: %v", err)
	}

	if group.ID == "" {
		t.Error("Group ID should be auto-generated")
	}

	// Verify group was stored
	retrieved, exists := cc.GetConsistencyGroup(group.ID)
	if !exists {
		t.Error("Group should exist")
	}
	if retrieved.Name != "test-group" {
		t.Errorf("Expected test-group, got %s", retrieved.Name)
	}
}

func TestConsistencyCoordinator_AddDependency(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	dep := &DatabaseDependency{
		SourceDatabase: "app-db",
		TargetDatabase: "reporting-db",
		DependencyType: "replication",
		Description:    "Reporting DB is a replica of App DB",
	}

	err := cc.AddDependency(dep)
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	deps := cc.GetDependencies("app-db")
	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}
}

func TestConsistencyCoordinator_CalculateBackupOrder(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Create dependencies: db1 -> db2 -> db3
	deps := []*DatabaseDependency{
		{SourceDatabase: "db1", TargetDatabase: "db2"},
		{SourceDatabase: "db2", TargetDatabase: "db3"},
	}

	for _, dep := range deps {
		if err := cc.AddDependency(dep); err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}
	}

	databases := []string{"db1", "db2", "db3"}
	order, err := cc.CalculateBackupOrder(databases)
	if err != nil {
		t.Fatalf("Failed to calculate backup order: %v", err)
	}

	if order.TotalSteps != 3 {
		t.Errorf("Expected 3 steps, got %d", order.TotalSteps)
	}

	// Verify order: db1 first, db2 second, db3 third
	if len(order.Sequence) < 3 {
		t.Fatalf("Expected at least 3 sequence steps")
	}

	if order.Sequence[0][0] != "db1" {
		t.Errorf("Expected db1 first, got %s", order.Sequence[0][0])
	}
}

func TestConsistencyCoordinator_CalculateParallelBackupOrder(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Create parallel structure: db1 -> db3, db2 -> db3
	// db1 and db2 can run in parallel
	deps := []*DatabaseDependency{
		{SourceDatabase: "db1", TargetDatabase: "db3"},
		{SourceDatabase: "db2", TargetDatabase: "db3"},
	}

	for _, dep := range deps {
		if err := cc.AddDependency(dep); err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}
	}

	databases := []string{"db1", "db2", "db3"}
	order, err := cc.CalculateBackupOrder(databases)
	if err != nil {
		t.Fatalf("Failed to calculate backup order: %v", err)
	}

	// Should have 2 steps: [db1, db2] in parallel, then db3
	if order.TotalSteps != 2 {
		t.Errorf("Expected 2 steps for parallel execution, got %d", order.TotalSteps)
	}

	// First step should have 2 databases (db1 and db2)
	if len(order.Sequence[0]) != 2 {
		t.Errorf("Expected 2 databases in first step, got %d", len(order.Sequence[0]))
	}

	// Second step should have 1 database (db3)
	if len(order.Sequence[1]) != 1 {
		t.Errorf("Expected 1 database in second step, got %d", len(order.Sequence[1]))
	}
}

func TestConsistencyCoordinator_DetectCircularDependency(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Create circular dependency: db1 -> db2 -> db3 -> db1
	deps := []*DatabaseDependency{
		{SourceDatabase: "db1", TargetDatabase: "db2"},
		{SourceDatabase: "db2", TargetDatabase: "db3"},
		{SourceDatabase: "db3", TargetDatabase: "db1"},
	}

	for _, dep := range deps {
		if err := cc.AddDependency(dep); err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}
	}

	databases := []string{"db1", "db2", "db3"}
	_, err := cc.CalculateBackupOrder(databases)
	if err == nil {
		t.Error("Should detect circular dependency")
	}
}

func TestConsistencyCoordinator_CreateConsistencyPoint(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	group := &ConsistencyGroup{
		Name:      "test-group",
		Databases: []string{"db1", "db2"},
		Level:     ConsistencyTransactional,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	ctx := context.Background()
	point, err := cc.CreateConsistencyPoint(ctx, group.ID)
	if err != nil {
		t.Fatalf("Failed to create consistency point: %v", err)
	}

	if point.ID == "" {
		t.Error("Consistency point ID should be generated")
	}

	if point.GroupID != group.ID {
		t.Errorf("Expected group ID %s, got %s", group.ID, point.GroupID)
	}

	// Verify point was stored
	retrieved, exists := cc.GetConsistencyPoint(point.ID)
	if !exists {
		t.Error("Consistency point should exist")
	}
	if retrieved.GroupID != group.ID {
		t.Errorf("Expected group ID %s, got %s", group.ID, retrieved.GroupID)
	}
}

func TestConsistencyCoordinator_RecordTransactionLog(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	group := &ConsistencyGroup{
		Name:      "test-group",
		Databases: []string{"db1"},
		Level:     ConsistencyTransactional,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	ctx := context.Background()
	point, err := cc.CreateConsistencyPoint(ctx, group.ID)
	if err != nil {
		t.Fatalf("Failed to create consistency point: %v", err)
	}

	entry := &TransactionLogEntry{
		DatabaseID:   "db1",
		DatabaseType: "postgresql",
		LSN:          "0/12345678",
		Metadata:     make(map[string]interface{}),
	}

	err = cc.RecordTransactionLog(point.ID, entry)
	if err != nil {
		t.Fatalf("Failed to record transaction log: %v", err)
	}

	// Verify entry was stored
	retrieved, _ := cc.GetConsistencyPoint(point.ID)
	if len(retrieved.Databases) != 1 {
		t.Errorf("Expected 1 database entry, got %d", len(retrieved.Databases))
	}

	dbEntry, exists := retrieved.Databases["db1"]
	if !exists {
		t.Error("Database entry should exist")
	}
	if dbEntry.LSN != "0/12345678" {
		t.Errorf("Expected LSN 0/12345678, got %s", dbEntry.LSN)
	}
}

func TestConsistencyCoordinator_ValidateConsistencyGroup(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Valid group
	validGroup := &ConsistencyGroup{
		Name:      "valid-group",
		Databases: []string{"db1", "db2"},
		Level:     ConsistencyLevelGroup,
	}
	err := cc.ValidateConsistencyGroup(validGroup)
	if err != nil {
		t.Errorf("Valid group should pass validation: %v", err)
	}

	// Invalid: nil group
	err = cc.ValidateConsistencyGroup(nil)
	if err == nil {
		t.Error("Nil group should fail validation")
	}

	// Invalid: empty name
	invalidGroup := &ConsistencyGroup{
		Name:      "",
		Databases: []string{"db1"},
	}
	err = cc.ValidateConsistencyGroup(invalidGroup)
	if err == nil {
		t.Error("Group with empty name should fail validation")
	}

	// Invalid: no databases
	invalidGroup = &ConsistencyGroup{
		Name:      "no-databases",
		Databases: []string{},
	}
	err = cc.ValidateConsistencyGroup(invalidGroup)
	if err == nil {
		t.Error("Group with no databases should fail validation")
	}
}

func TestConsistencyCoordinator_ListConsistencyPoints(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	group := &ConsistencyGroup{
		Name:      "test-group",
		Databases: []string{"db1"},
		Level:     ConsistencyLevelGroup,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	ctx := context.Background()

	// Create multiple consistency points
	for i := 0; i < 3; i++ {
		_, err := cc.CreateConsistencyPoint(ctx, group.ID)
		if err != nil {
			t.Fatalf("Failed to create consistency point: %v", err)
		}
	}

	points := cc.ListConsistencyPoints(group.ID)
	if len(points) != 3 {
		t.Errorf("Expected 3 consistency points, got %d", len(points))
	}
}

func TestConsistencyCoordinator_CleanupOldConsistencyPoints(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Create old consistency point
	oldPoint := &ConsistencyPoint{
		ID:        "old-point",
		Timestamp: time.Now().Add(-2 * time.Hour),
		GroupID:   "test-group",
		Databases: make(map[string]*TransactionLogEntry),
		Metadata:  make(map[string]interface{}),
	}

	// Create new consistency point
	newPoint := &ConsistencyPoint{
		ID:        "new-point",
		Timestamp: time.Now(),
		GroupID:   "test-group",
		Databases: make(map[string]*TransactionLogEntry),
		Metadata:  make(map[string]interface{}),
	}

	cc.mu.Lock()
	cc.consistencyPoints[oldPoint.ID] = oldPoint
	cc.consistencyPoints[newPoint.ID] = newPoint
	cc.mu.Unlock()

	// Cleanup points older than 1 hour
	removed := cc.CleanupOldConsistencyPoints(1 * time.Hour)
	if removed != 1 {
		t.Errorf("Expected to remove 1 old point, removed %d", removed)
	}

	// Verify old point was removed
	_, exists := cc.GetConsistencyPoint("old-point")
	if exists {
		t.Error("Old consistency point should be removed")
	}

	// Verify new point still exists
	_, exists = cc.GetConsistencyPoint("new-point")
	if !exists {
		t.Error("New consistency point should still exist")
	}
}

func TestConsistencyCoordinator_GetStatistics(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	// Create test data
	group1 := &ConsistencyGroup{
		Name:      "group1",
		Databases: []string{"db1", "db2"},
		Level:     ConsistencyDatabase,
	}
	group2 := &ConsistencyGroup{
		Name:      "group2",
		Databases: []string{"db3"},
		Level:     ConsistencyLevelGroup,
	}

	cc.CreateConsistencyGroup(group1)
	cc.CreateConsistencyGroup(group2)

	dep := &DatabaseDependency{
		SourceDatabase: "db1",
		TargetDatabase: "db2",
	}
	cc.AddDependency(dep)

	stats := cc.GetStatistics()

	if stats["total_groups"] != 2 {
		t.Errorf("Expected 2 groups, got %v", stats["total_groups"])
	}

	if stats["total_dependencies"] != 1 {
		t.Errorf("Expected 1 dependency, got %v", stats["total_dependencies"])
	}

	if stats["total_databases"] != 3 {
		t.Errorf("Expected 3 unique databases, got %v", stats["total_databases"])
	}
}

// fakeQuiescer is an in-memory Quiescer used to prove that the consistent
// backup flow freezes the database, records its log position, and always
// resumes it.
type fakeQuiescer struct {
	quiesceErr error
	resumeErr  error
	metadata   map[string]interface{}
	dbType     string
	quiesceN   int32
	resumeN    int32
	closeN     int32
}

func (f *fakeQuiescer) Quiesce(ctx context.Context) (*QuiesceOperation, error) {
	atomic.AddInt32(&f.quiesceN, 1)
	if f.quiesceErr != nil {
		return &QuiesceOperation{State: StateFailed, Error: f.quiesceErr}, f.quiesceErr
	}
	md := make(map[string]interface{})
	for k, v := range f.metadata {
		md[k] = v
	}
	return &QuiesceOperation{
		DatabaseType: f.dbType,
		State:        StateQuiesced,
		StartTime:    time.Now(),
		QuiesceTime:  time.Now(),
		Metadata:     md,
	}, nil
}

func (f *fakeQuiescer) Resume(ctx context.Context) error {
	atomic.AddInt32(&f.resumeN, 1)
	return f.resumeErr
}

func (f *fakeQuiescer) Close() error {
	atomic.AddInt32(&f.closeN, 1)
	return nil
}

func TestExecuteConsistentBackup_QuiescesAndRecordsLog(t *testing.T) {
	qm := NewQuiesceManager()
	cc := NewConsistencyCoordinator(nil, qm)

	fq := &fakeQuiescer{
		dbType:   "postgresql",
		metadata: map[string]interface{}{"checkpoint_lsn": "0/ABCDEF12"},
	}
	cc.RegisterQuiescer("db1", fq)

	group := &ConsistencyGroup{
		Name:      "g",
		Databases: []string{"db1"},
		Level:     ConsistencyTransactional,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	backupCalled := int32(0)
	err := cc.ExecuteConsistentBackup(context.Background(), group.ID, func(ctx context.Context, db string) error {
		atomic.AddInt32(&backupCalled, 1)
		// While the backup runs the database must be frozen.
		if atomic.LoadInt32(&fq.quiesceN) != 1 {
			t.Errorf("database not quiesced before backup ran")
		}
		if atomic.LoadInt32(&fq.resumeN) != 0 {
			t.Errorf("database resumed before backup completed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteConsistentBackup failed: %v", err)
	}

	if atomic.LoadInt32(&backupCalled) != 1 {
		t.Errorf("backupFunc should be called once, got %d", backupCalled)
	}
	if got := atomic.LoadInt32(&fq.quiesceN); got != 1 {
		t.Errorf("QuiesceDatabase should be invoked once, got %d", got)
	}
	if got := atomic.LoadInt32(&fq.resumeN); got != 1 {
		t.Errorf("ResumeDatabase should be invoked once, got %d", got)
	}

	// The consistency point must capture the real LSN, not an empty map.
	points := cc.ListConsistencyPoints(group.ID)
	if len(points) != 1 {
		t.Fatalf("expected 1 consistency point, got %d", len(points))
	}
	entry, ok := points[0].Databases["db1"]
	if !ok {
		t.Fatalf("transaction log entry for db1 not recorded")
	}
	if entry.LSN != "0/ABCDEF12" {
		t.Errorf("expected LSN 0/ABCDEF12 recorded, got %q", entry.LSN)
	}
}

func TestExecuteConsistentBackup_ResumesOnBackupError(t *testing.T) {
	qm := NewQuiesceManager()
	cc := NewConsistencyCoordinator(nil, qm)

	fq := &fakeQuiescer{
		dbType:   "mysql",
		metadata: map[string]interface{}{"binlog_file": "bin.000001", "binlog_position": int64(120)},
	}
	cc.RegisterQuiescer("db1", fq)

	group := &ConsistencyGroup{
		Name:      "g",
		Databases: []string{"db1"},
		Level:     ConsistencyTransactional,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	err := cc.ExecuteConsistentBackup(context.Background(), group.ID, func(ctx context.Context, db string) error {
		return fmt.Errorf("backup boom")
	})
	if err == nil {
		t.Fatal("expected backup error to propagate")
	}

	// Even though the backup failed, the database MUST be unfrozen.
	if got := atomic.LoadInt32(&fq.quiesceN); got != 1 {
		t.Errorf("QuiesceDatabase should be invoked once, got %d", got)
	}
	if got := atomic.LoadInt32(&fq.resumeN); got != 1 {
		t.Errorf("ResumeDatabase must run even on backup error, got %d", got)
	}
}

func TestExecuteConsistentBackup_QuiesceFailureAbortsBackup(t *testing.T) {
	qm := NewQuiesceManager()
	cc := NewConsistencyCoordinator(nil, qm)

	fq := &fakeQuiescer{
		dbType:     "postgresql",
		quiesceErr: fmt.Errorf("freeze failed"),
	}
	cc.RegisterQuiescer("db1", fq)

	group := &ConsistencyGroup{
		Name:      "g",
		Databases: []string{"db1"},
		Level:     ConsistencyTransactional,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	backupCalled := int32(0)
	err := cc.ExecuteConsistentBackup(context.Background(), group.ID, func(ctx context.Context, db string) error {
		atomic.AddInt32(&backupCalled, 1)
		return nil
	})
	if err == nil {
		t.Fatal("expected error when quiesce fails")
	}
	if atomic.LoadInt32(&backupCalled) != 0 {
		t.Errorf("backup must not run when quiesce fails")
	}

	// No consistency point should report a captured database.
	points := cc.ListConsistencyPoints(group.ID)
	if len(points) == 1 && len(points[0].Databases) != 0 {
		t.Errorf("no transaction log should be recorded on quiesce failure, got %d", len(points[0].Databases))
	}
}

func TestConsistencyCoordinator_UpdateDeleteGroup(t *testing.T) {
	cc := NewConsistencyCoordinator(nil, nil)

	group := &ConsistencyGroup{
		Name:      "test-group",
		Databases: []string{"db1"},
		Level:     ConsistencyLevelGroup,
	}
	if err := cc.CreateConsistencyGroup(group); err != nil {
		t.Fatalf("Failed to create group: %v", err)
	}

	groupID := group.ID

	// Update group
	group.Name = "updated-group"
	group.Databases = []string{"db1", "db2"}
	err := cc.UpdateConsistencyGroup(group)
	if err != nil {
		t.Fatalf("Failed to update group: %v", err)
	}

	// Verify update
	retrieved, _ := cc.GetConsistencyGroup(groupID)
	if retrieved.Name != "updated-group" {
		t.Errorf("Expected updated-group, got %s", retrieved.Name)
	}
	if len(retrieved.Databases) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(retrieved.Databases))
	}

	// Delete group
	err = cc.DeleteConsistencyGroup(groupID)
	if err != nil {
		t.Fatalf("Failed to delete group: %v", err)
	}

	// Verify deletion
	_, exists := cc.GetConsistencyGroup(groupID)
	if exists {
		t.Error("Group should be deleted")
	}
}
