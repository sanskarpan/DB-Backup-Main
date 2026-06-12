package consistency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConsistencyLevel defines the level of consistency required
type ConsistencyLevel string

const (
	ConsistencyNone       ConsistencyLevel = "none"       // No consistency guarantees
	ConsistencyDatabase   ConsistencyLevel = "database"   // Single database consistency
	ConsistencyLevelGroup ConsistencyLevel = "group"      // Consistency group
	ConsistencyGlobal     ConsistencyLevel = "global"     // Global consistency across all databases
	ConsistencyCrashSafe  ConsistencyLevel = "crash-safe" // Crash-consistent (file system level)
	ConsistencyTransactional ConsistencyLevel = "transactional" // Transaction-consistent
)

// DatabaseDependency represents a dependency between databases
type DatabaseDependency struct {
	SourceDatabase string
	TargetDatabase string
	DependencyType string // "foreign_key", "replication", "application"
	Description    string
}

// ConsistencyGroup represents a group of databases that must be backed up together
type ConsistencyGroup struct {
	ID           string
	Name         string
	Databases    []string
	Dependencies []*DatabaseDependency
	Level        ConsistencyLevel
	Priority     int
	Timeout      time.Duration
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// BackupOrder represents the order in which databases should be backed up
type BackupOrder struct {
	Sequence   [][]string // Each inner slice represents databases that can be backed up in parallel
	TotalSteps int
	Metadata   map[string]interface{}
}

// TransactionLogEntry represents a transaction log entry
type TransactionLogEntry struct {
	ID           string
	DatabaseID   string
	DatabaseType string
	Timestamp    time.Time
	LSN          string // Log Sequence Number (PostgreSQL)
	BinlogFile   string // Binlog file (MySQL)
	BinlogPos    int64  // Binlog position (MySQL)
	OplogTime    int64  // Oplog timestamp (MongoDB)
	TransactionID string
	Metadata     map[string]interface{}
}

// ConsistencyPoint represents a point in time for consistency
type ConsistencyPoint struct {
	ID        string
	Timestamp time.Time
	GroupID   string
	Databases map[string]*TransactionLogEntry
	Metadata  map[string]interface{}
}

// ConsistencyCoordinator coordinates application-consistent backups
type ConsistencyCoordinator struct {
	mu                sync.RWMutex
	groups            map[string]*ConsistencyGroup
	consistencyPoints map[string]*ConsistencyPoint
	dependencies      []*DatabaseDependency
	hookManager       *HookManager
	quiesceManager    *QuiesceManager
}

// NewConsistencyCoordinator creates a new consistency coordinator
func NewConsistencyCoordinator(hookManager *HookManager, quiesceManager *QuiesceManager) *ConsistencyCoordinator {
	return &ConsistencyCoordinator{
		groups:            make(map[string]*ConsistencyGroup),
		consistencyPoints: make(map[string]*ConsistencyPoint),
		dependencies:      make([]*DatabaseDependency, 0),
		hookManager:       hookManager,
		quiesceManager:    quiesceManager,
	}
}

// CreateConsistencyGroup creates a new consistency group
func (cc *ConsistencyCoordinator) CreateConsistencyGroup(group *ConsistencyGroup) error {
	if group == nil {
		return fmt.Errorf("consistency group cannot be nil")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Generate unique ID if not provided
	if group.ID == "" {
		for {
			group.ID = fmt.Sprintf("cg-%d", time.Now().UnixNano())
			if _, exists := cc.groups[group.ID]; !exists {
				break
			}
			time.Sleep(1 * time.Microsecond)
		}
	}

	if len(group.Databases) == 0 {
		return fmt.Errorf("consistency group must contain at least one database")
	}

	if group.Timeout == 0 {
		group.Timeout = 30 * time.Minute
	}

	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	cc.groups[group.ID] = group
	return nil
}

// AddDependency adds a database dependency
func (cc *ConsistencyCoordinator) AddDependency(dep *DatabaseDependency) error {
	if dep == nil {
		return fmt.Errorf("dependency cannot be nil")
	}

	if dep.SourceDatabase == "" || dep.TargetDatabase == "" {
		return fmt.Errorf("source and target databases must be specified")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.dependencies = append(cc.dependencies, dep)
	return nil
}

// CalculateBackupOrder calculates the optimal backup order based on dependencies
func (cc *ConsistencyCoordinator) CalculateBackupOrder(databases []string) (*BackupOrder, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize graph nodes
	for _, db := range databases {
		graph[db] = make([]string, 0)
		inDegree[db] = 0
	}

	// Build edges from dependencies
	for _, dep := range cc.dependencies {
		// Only include dependencies for databases in the backup set
		if contains(databases, dep.SourceDatabase) && contains(databases, dep.TargetDatabase) {
			graph[dep.SourceDatabase] = append(graph[dep.SourceDatabase], dep.TargetDatabase)
			inDegree[dep.TargetDatabase]++
		}
	}

	// Perform topological sort using Kahn's algorithm
	sequence := make([][]string, 0)
	remaining := len(databases)

	for remaining > 0 {
		// Find all nodes with in-degree 0 (can be backed up in parallel)
		level := make([]string, 0)
		for db := range inDegree {
			if inDegree[db] == 0 {
				level = append(level, db)
			}
		}

		if len(level) == 0 {
			// Circular dependency detected
			return nil, fmt.Errorf("circular dependency detected in backup order")
		}

		sequence = append(sequence, level)

		// Remove these nodes from the graph
		for _, db := range level {
			delete(inDegree, db)
			for _, target := range graph[db] {
				inDegree[target]--
			}
			remaining--
		}
	}

	order := &BackupOrder{
		Sequence:   sequence,
		TotalSteps: len(sequence),
		Metadata: map[string]interface{}{
			"total_databases":  len(databases),
			"parallel_batches": len(sequence),
			"dependencies":     len(cc.dependencies),
		},
	}

	return order, nil
}

// CreateConsistencyPoint creates a consistency point across multiple databases
func (cc *ConsistencyCoordinator) CreateConsistencyPoint(ctx context.Context, groupID string) (*ConsistencyPoint, error) {
	cc.mu.RLock()
	group, exists := cc.groups[groupID]
	cc.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("consistency group %s not found", groupID)
	}

	// Generate unique ID with Lock held to prevent collisions
	cc.mu.Lock()

	// Generate unique ID - keep trying until we get a unique one
	var pointID string
	for {
		pointID = fmt.Sprintf("cp-%d", time.Now().UnixNano())
		if _, exists := cc.consistencyPoints[pointID]; !exists {
			break
		}
		time.Sleep(1 * time.Microsecond)
	}

	point := &ConsistencyPoint{
		ID:        pointID,
		Timestamp: time.Now(),
		GroupID:   groupID,
		Databases: make(map[string]*TransactionLogEntry),
		Metadata:  make(map[string]interface{}),
	}

	point.Metadata["group_name"] = group.Name
	point.Metadata["consistency_level"] = group.Level
	point.Metadata["database_count"] = len(group.Databases)

	// Store consistency point
	cc.consistencyPoints[point.ID] = point
	cc.mu.Unlock()

	return point, nil
}

// RecordTransactionLog records transaction log information for a database
func (cc *ConsistencyCoordinator) RecordTransactionLog(pointID string, entry *TransactionLogEntry) error {
	if entry == nil {
		return fmt.Errorf("transaction log entry cannot be nil")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	point, exists := cc.consistencyPoints[pointID]
	if !exists {
		return fmt.Errorf("consistency point %s not found", pointID)
	}

	entry.ID = fmt.Sprintf("txlog-%s-%d", entry.DatabaseID, time.Now().UnixNano())
	entry.Timestamp = time.Now()

	point.Databases[entry.DatabaseID] = entry
	return nil
}

// ExecuteConsistentBackup performs an application-consistent backup for a consistency group
func (cc *ConsistencyCoordinator) ExecuteConsistentBackup(ctx context.Context, groupID string, backupFunc func(ctx context.Context, database string) error) error {
	cc.mu.RLock()
	group, exists := cc.groups[groupID]
	cc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("consistency group %s not found", groupID)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, group.Timeout)
	defer cancel()

	// Execute pre-backup hooks
	if cc.hookManager != nil {
		metadata := map[string]string{
			"group_id":   groupID,
			"group_name": group.Name,
			"level":      string(group.Level),
		}
		_, err := cc.hookManager.ExecuteHooks(timeoutCtx, HookTypePreBackup, metadata)
		if err != nil {
			return fmt.Errorf("pre-backup hooks failed: %w", err)
		}
	}

	// Create consistency point
	point, err := cc.CreateConsistencyPoint(timeoutCtx, groupID)
	if err != nil {
		return fmt.Errorf("failed to create consistency point: %w", err)
	}

	// Calculate backup order
	order, err := cc.CalculateBackupOrder(group.Databases)
	if err != nil {
		return fmt.Errorf("failed to calculate backup order: %w", err)
	}

	// Execute backups in order
	var backupErr error
	for stepNum, databases := range order.Sequence {
		// Backup databases in this step (can be done in parallel)
		var wg sync.WaitGroup
		errChan := make(chan error, len(databases))

		for _, dbID := range databases {
			wg.Add(1)
			go func(database string) {
				defer wg.Done()

				// Execute pre-quiesce hooks
				if cc.hookManager != nil {
					metadata := map[string]string{
						"database": database,
						"group_id": groupID,
						"step":     fmt.Sprintf("%d", stepNum),
					}
					_, _ = cc.hookManager.ExecuteHooks(timeoutCtx, HookTypePreQuiesce, metadata)
				}

				// Perform backup
				err := backupFunc(timeoutCtx, database)
				if err != nil {
					errChan <- fmt.Errorf("backup failed for %s: %w", database, err)
					return
				}

				// Execute post-quiesce hooks
				if cc.hookManager != nil {
					metadata := map[string]string{
						"database": database,
						"group_id": groupID,
						"step":     fmt.Sprintf("%d", stepNum),
					}
					_, _ = cc.hookManager.ExecuteHooks(timeoutCtx, HookTypePostQuiesce, metadata)
				}

			}(dbID)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			if backupErr == nil {
				backupErr = err
			}
		}

		if backupErr != nil {
			break
		}
	}

	// Execute post-backup hooks
	if cc.hookManager != nil {
		metadata := map[string]string{
			"group_id":         groupID,
			"group_name":       group.Name,
			"consistency_point": point.ID,
		}

		if backupErr == nil {
			_, _ = cc.hookManager.ExecuteHooks(timeoutCtx, HookTypeOnSuccess, metadata)
		} else {
			metadata["error"] = backupErr.Error()
			_, _ = cc.hookManager.ExecuteHooks(timeoutCtx, HookTypeOnFailure, metadata)
		}

		_, err := cc.hookManager.ExecuteHooks(timeoutCtx, HookTypePostBackup, metadata)
		if err != nil && backupErr == nil {
			backupErr = fmt.Errorf("post-backup hooks failed: %w", err)
		}
	}

	return backupErr
}

// GetConsistencyGroup returns a consistency group by ID
func (cc *ConsistencyCoordinator) GetConsistencyGroup(groupID string) (*ConsistencyGroup, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	group, exists := cc.groups[groupID]
	return group, exists
}

// ListConsistencyGroups returns all consistency groups
func (cc *ConsistencyCoordinator) ListConsistencyGroups() []*ConsistencyGroup {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	groups := make([]*ConsistencyGroup, 0, len(cc.groups))
	for _, group := range cc.groups {
		groups = append(groups, group)
	}
	return groups
}

// GetConsistencyPoint returns a consistency point by ID
func (cc *ConsistencyCoordinator) GetConsistencyPoint(pointID string) (*ConsistencyPoint, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	point, exists := cc.consistencyPoints[pointID]
	return point, exists
}

// ListConsistencyPoints returns all consistency points for a group
func (cc *ConsistencyCoordinator) ListConsistencyPoints(groupID string) []*ConsistencyPoint {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	points := make([]*ConsistencyPoint, 0)
	for _, point := range cc.consistencyPoints {
		if point.GroupID == groupID {
			points = append(points, point)
		}
	}
	return points
}

// DeleteConsistencyGroup deletes a consistency group
func (cc *ConsistencyCoordinator) DeleteConsistencyGroup(groupID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, exists := cc.groups[groupID]; !exists {
		return fmt.Errorf("consistency group %s not found", groupID)
	}

	delete(cc.groups, groupID)
	return nil
}

// UpdateConsistencyGroup updates a consistency group
func (cc *ConsistencyCoordinator) UpdateConsistencyGroup(group *ConsistencyGroup) error {
	if group == nil || group.ID == "" {
		return fmt.Errorf("invalid consistency group")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, exists := cc.groups[group.ID]; !exists {
		return fmt.Errorf("consistency group %s not found", group.ID)
	}

	group.UpdatedAt = time.Now()
	cc.groups[group.ID] = group
	return nil
}

// ValidateConsistencyGroup validates a consistency group configuration
func (cc *ConsistencyCoordinator) ValidateConsistencyGroup(group *ConsistencyGroup) error {
	if group == nil {
		return fmt.Errorf("consistency group cannot be nil")
	}

	if group.Name == "" {
		return fmt.Errorf("consistency group name is required")
	}

	if len(group.Databases) == 0 {
		return fmt.Errorf("consistency group must contain at least one database")
	}

	// Check for circular dependencies
	_, err := cc.CalculateBackupOrder(group.Databases)
	if err != nil {
		return fmt.Errorf("invalid dependency graph: %w", err)
	}

	return nil
}

// GetDependencies returns all dependencies for a database
func (cc *ConsistencyCoordinator) GetDependencies(database string) []*DatabaseDependency {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	deps := make([]*DatabaseDependency, 0)
	for _, dep := range cc.dependencies {
		if dep.SourceDatabase == database || dep.TargetDatabase == database {
			deps = append(deps, dep)
		}
	}
	return deps
}

// Helper function to check if a slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// CleanupOldConsistencyPoints removes consistency points older than the specified duration
func (cc *ConsistencyCoordinator) CleanupOldConsistencyPoints(maxAge time.Duration) int {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, point := range cc.consistencyPoints {
		if point.Timestamp.Before(cutoff) {
			delete(cc.consistencyPoints, id)
			removed++
		}
	}

	return removed
}

// GetStatistics returns statistics about the consistency coordinator
func (cc *ConsistencyCoordinator) GetStatistics() map[string]interface{} {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	stats := map[string]interface{}{
		"total_groups":            len(cc.groups),
		"total_dependencies":      len(cc.dependencies),
		"total_consistency_points": len(cc.consistencyPoints),
	}

	// Count databases across all groups
	databaseSet := make(map[string]bool)
	for _, group := range cc.groups {
		for _, db := range group.Databases {
			databaseSet[db] = true
		}
	}
	stats["total_databases"] = len(databaseSet)

	// Count by consistency level
	levelCounts := make(map[ConsistencyLevel]int)
	for _, group := range cc.groups {
		levelCounts[group.Level]++
	}
	stats["by_level"] = levelCounts

	return stats
}
