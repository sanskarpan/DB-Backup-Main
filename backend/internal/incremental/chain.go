package incremental

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ChainPolicy defines policies for backup chain management.
type ChainPolicy struct {
	MaxChainLength          int           // Maximum number of incrementals before synthetic
	MaxChainAge             time.Duration // Maximum age of base backup before synthetic
	RetentionDays           int           // Number of days to retain backups
	MinFullBackupInterval   time.Duration // Minimum time between full backups
	AutoSyntheticEnabled    bool          // Automatically create synthetic fulls
	AutoCleanupEnabled      bool          // Automatically cleanup old backups
	ChainHealthCheckEnabled bool          // Enable chain health monitoring
}

// DefaultChainPolicy returns default chain management policies.
func DefaultChainPolicy() *ChainPolicy {
	return &ChainPolicy{
		MaxChainLength:          10,
		MaxChainAge:             7 * 24 * time.Hour,
		RetentionDays:           30,
		MinFullBackupInterval:   24 * time.Hour,
		AutoSyntheticEnabled:    true,
		AutoCleanupEnabled:      true,
		ChainHealthCheckEnabled: true,
	}
}

// ChainHealth represents the health status of a backup chain.
type ChainHealth struct {
	ChainID         string
	Status          string // "healthy", "warning", "critical"
	ChainLength     int
	ChainAge        time.Duration
	Issues          []string
	Recommendations []string
	LastChecked     time.Time
}

// BackupChainManager manages backup chains.
type BackupChainManager struct {
	mu              sync.RWMutex
	strategy        *IncrementalForeverStrategy
	syntheticGen    *SyntheticBackupGenerator
	policy          *ChainPolicy
	chainHealth     map[string]*ChainHealth
	lastFullBackups map[string]time.Time // Track last full backup per database
}

// NewBackupChainManager creates a new backup chain manager.
func NewBackupChainManager(strategy *IncrementalForeverStrategy, syntheticGen *SyntheticBackupGenerator, policy *ChainPolicy) *BackupChainManager {
	if policy == nil {
		policy = DefaultChainPolicy()
	}

	return &BackupChainManager{
		strategy:        strategy,
		syntheticGen:    syntheticGen,
		policy:          policy,
		chainHealth:     make(map[string]*ChainHealth),
		lastFullBackups: make(map[string]time.Time),
	}
}

// DetermineBackupType determines what type of backup should be performed.
func (bcm *BackupChainManager) DetermineBackupType(databaseID string) (BackupType, string, error) {
	bcm.mu.RLock()
	defer bcm.mu.RUnlock()

	// Get all backups for this database
	manifests := bcm.getBackupsForDatabase(databaseID)

	// If no backups exist, must do full backup
	if len(manifests) == 0 {
		return BackupTypeFull, "No existing backups", nil
	}

	// Get latest backup
	latest := manifests[0]

	// Check if enough time has passed for a full backup
	if lastFull, exists := bcm.lastFullBackups[databaseID]; exists {
		if time.Since(lastFull) < bcm.policy.MinFullBackupInterval {
			// Get the chain
			chain, err := bcm.strategy.GetBackupChain(latest.ID)
			if err == nil && len(chain) > 0 {
				baseBackup := chain[0]

				// Check chain length
				incrementalCount := 0
				for _, m := range chain {
					if m.Type == BackupTypeIncremental {
						incrementalCount++
					}
				}

				// Check if synthetic is needed
				if incrementalCount >= bcm.policy.MaxChainLength {
					return BackupTypeSynthetic, fmt.Sprintf("Chain length (%d) exceeds max (%d)", incrementalCount, bcm.policy.MaxChainLength), nil
				}

				// Check chain age
				if time.Since(baseBackup.CreatedAt) > bcm.policy.MaxChainAge {
					return BackupTypeSynthetic, fmt.Sprintf("Base backup age (%v) exceeds max (%v)", time.Since(baseBackup.CreatedAt), bcm.policy.MaxChainAge), nil
				}

				return BackupTypeIncremental, "Normal incremental backup", nil
			}
		}
	}

	// Default to incremental if we have a base
	return BackupTypeIncremental, "Normal incremental backup", nil
}

// ValidateChain validates the integrity of a backup chain.
func (bcm *BackupChainManager) ValidateChain(backupID string) (*ChainHealth, error) {
	chain, err := bcm.strategy.GetBackupChain(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup chain: %w", err)
	}

	health := &ChainHealth{
		ChainID:         backupID,
		Status:          "healthy",
		ChainLength:     len(chain),
		Issues:          make([]string, 0),
		Recommendations: make([]string, 0),
		LastChecked:     time.Now(),
	}

	if len(chain) == 0 {
		health.Status = "critical"
		health.Issues = append(health.Issues, "Empty backup chain")
		return health, nil
	}

	// Check base backup
	baseBackup := chain[0]
	if baseBackup.Type != BackupTypeFull && baseBackup.Type != BackupTypeSynthetic {
		health.Status = "critical"
		health.Issues = append(health.Issues, "First backup in chain is not a full or synthetic backup")
	}

	// Check backup continuity
	for i := 1; i < len(chain); i++ {
		current := chain[i]
		previous := chain[i-1]

		if current.ParentBackupID != previous.ID {
			health.Status = "critical"
			health.Issues = append(health.Issues, fmt.Sprintf("Broken chain: backup %s does not reference previous backup %s", current.ID, previous.ID))
		}

		if current.CreatedAt.Before(previous.CreatedAt) {
			health.Status = "critical"
			health.Issues = append(health.Issues, fmt.Sprintf("Backup timestamps out of order at position %d", i))
		}
	}

	// Check if all backup files exist
	for _, manifest := range chain {
		err := bcm.strategy.VerifyBackup(manifest.ID)
		if err != nil {
			health.Status = "critical"
			health.Issues = append(health.Issues, fmt.Sprintf("Backup %s verification failed: %v", manifest.ID, err))
		}
	}

	// Calculate chain age
	health.ChainAge = time.Since(baseBackup.CreatedAt)

	// Check chain length and age for warnings
	incrementalCount := 0
	for _, m := range chain {
		if m.Type == BackupTypeIncremental {
			incrementalCount++
		}
	}

	if incrementalCount >= bcm.policy.MaxChainLength {
		if health.Status == "healthy" {
			health.Status = "warning"
		}
		health.Issues = append(health.Issues, fmt.Sprintf("Chain has %d incrementals (max recommended: %d)", incrementalCount, bcm.policy.MaxChainLength))
		health.Recommendations = append(health.Recommendations, "Consider creating a synthetic full backup")
	}

	if health.ChainAge > bcm.policy.MaxChainAge {
		if health.Status == "healthy" {
			health.Status = "warning"
		}
		health.Issues = append(health.Issues, fmt.Sprintf("Base backup is %v old (max recommended: %v)", health.ChainAge, bcm.policy.MaxChainAge))
		health.Recommendations = append(health.Recommendations, "Consider creating a synthetic full backup or new full backup")
	}

	// Store health status
	bcm.mu.Lock()
	bcm.chainHealth[backupID] = health
	bcm.mu.Unlock()

	return health, nil
}

// MonitorChains monitors all backup chains and reports health.
func (bcm *BackupChainManager) MonitorChains() ([]*ChainHealth, error) {
	manifests := bcm.strategy.ListBackups()

	// Group by database and get latest backup for each
	latestByDatabase := make(map[string]*BackupManifest)
	for _, manifest := range manifests {
		if latest, exists := latestByDatabase[manifest.DatabaseID]; !exists || manifest.CreatedAt.After(latest.CreatedAt) {
			latestByDatabase[manifest.DatabaseID] = manifest
		}
	}

	// Validate each chain
	healthReports := make([]*ChainHealth, 0)
	for _, latest := range latestByDatabase {
		health, err := bcm.ValidateChain(latest.ID)
		if err != nil {
			continue
		}
		healthReports = append(healthReports, health)
	}

	return healthReports, nil
}

// ApplyRetentionPolicy applies retention policy to backups.
func (bcm *BackupChainManager) ApplyRetentionPolicy() ([]string, error) {
	if !bcm.policy.AutoCleanupEnabled {
		return nil, fmt.Errorf("auto cleanup is disabled")
	}

	manifests := bcm.strategy.ListBackups()

	cutoffTime := time.Now().AddDate(0, 0, -bcm.policy.RetentionDays)
	removed := make([]string, 0)

	// Group backups by database
	backupsByDB := make(map[string][]*BackupManifest)
	for _, manifest := range manifests {
		backupsByDB[manifest.DatabaseID] = append(backupsByDB[manifest.DatabaseID], manifest)
	}

	// For each database, keep at least one full backup chain
	for databaseID, dbBackups := range backupsByDB {
		// Sort by creation time (newest first)
		sort.Slice(dbBackups, func(i, j int) bool {
			return dbBackups[i].CreatedAt.After(dbBackups[j].CreatedAt)
		})

		// Keep the latest chain
		if len(dbBackups) == 0 {
			continue
		}

		latestBackup := dbBackups[0]
		chain, err := bcm.strategy.GetBackupChain(latestBackup.ID)
		if err != nil {
			continue
		}

		chainIDs := make(map[string]bool)
		for _, manifest := range chain {
			chainIDs[manifest.ID] = true
		}

		// Remove old backups that are not part of the latest chain
		for _, manifest := range dbBackups {
			// Skip if part of latest chain
			if chainIDs[manifest.ID] {
				continue
			}

			// Skip if newer than retention period
			if manifest.CreatedAt.After(cutoffTime) {
				continue
			}

			// Remove this backup
			err := bcm.removeBackup(manifest)
			if err != nil {
				return removed, fmt.Errorf("failed to remove backup %s: %w", manifest.ID, err)
			}

			removed = append(removed, manifest.ID)
		}

		_ = databaseID // Use the variable
	}

	return removed, nil
}

// OptimizeChain optimizes a backup chain by creating synthetic fulls if needed.
func (bcm *BackupChainManager) OptimizeChain(backupID string) (*BackupManifest, error) {
	if !bcm.policy.AutoSyntheticEnabled {
		return nil, fmt.Errorf("auto synthetic generation is disabled")
	}

	// Check if synthetic is recommended
	recommendation, err := bcm.syntheticGen.GetSyntheticBackupRecommendation(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recommendation: %w", err)
	}

	recommended, ok := recommendation["recommended"].(bool)
	if !ok || !recommended {
		return nil, fmt.Errorf("synthetic backup not recommended")
	}

	// Generate synthetic full
	synthetic, err := bcm.syntheticGen.GenerateSyntheticFull(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate synthetic: %w", err)
	}

	// Convert to base for future incrementals
	err = bcm.syntheticGen.ConvertToBase(synthetic.ID)
	if err != nil {
		return synthetic, fmt.Errorf("failed to convert to base: %w", err)
	}

	// Optionally cleanup old chain
	if bcm.policy.AutoCleanupEnabled {
		_, err = bcm.syntheticGen.CleanupOldChain(synthetic.ID, 3) // Keep last 3 incrementals
		if err != nil {
			return synthetic, fmt.Errorf("warning: cleanup failed: %w", err)
		}
	}

	return synthetic, nil
}

// getBackupsForDatabase returns all backups for a database sorted by creation time (newest first).
func (bcm *BackupChainManager) getBackupsForDatabase(databaseID string) []*BackupManifest {
	allBackups := bcm.strategy.ListBackups()

	dbBackups := make([]*BackupManifest, 0)
	for _, manifest := range allBackups {
		if manifest.DatabaseID == databaseID {
			dbBackups = append(dbBackups, manifest)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(dbBackups, func(i, j int) bool {
		return dbBackups[i].CreatedAt.After(dbBackups[j].CreatedAt)
	})

	return dbBackups
}

// removeBackup removes a backup and its persisted artifacts (data file, manifest
// file and snapshot) via the strategy. It returns a real error on failure so the
// retention policy never counts a backup as removed unless it actually was.
func (bcm *BackupChainManager) removeBackup(manifest *BackupManifest) error {
	if manifest == nil {
		return fmt.Errorf("cannot remove nil manifest")
	}
	return bcm.strategy.RemoveBackup(manifest.ID)
}

// RecordFullBackup records when a full backup was performed.
func (bcm *BackupChainManager) RecordFullBackup(databaseID string, timestamp time.Time) {
	bcm.mu.Lock()
	defer bcm.mu.Unlock()

	bcm.lastFullBackups[databaseID] = timestamp
}

// GetChainHealth returns the health status of a chain.
func (bcm *BackupChainManager) GetChainHealth(chainID string) (*ChainHealth, bool) {
	bcm.mu.RLock()
	defer bcm.mu.RUnlock()

	health, exists := bcm.chainHealth[chainID]
	return health, exists
}

// GetPolicy returns the current chain policy.
func (bcm *BackupChainManager) GetPolicy() *ChainPolicy {
	return bcm.policy
}

// UpdatePolicy updates the chain management policy.
func (bcm *BackupChainManager) UpdatePolicy(policy *ChainPolicy) {
	bcm.mu.Lock()
	defer bcm.mu.Unlock()

	bcm.policy = policy
}

// GetStatistics returns statistics about chain management.
func (bcm *BackupChainManager) GetStatistics() map[string]interface{} {
	bcm.mu.RLock()
	defer bcm.mu.RUnlock()

	healthyChains := 0
	warningChains := 0
	criticalChains := 0

	for _, health := range bcm.chainHealth {
		switch health.Status {
		case "healthy":
			healthyChains++
		case "warning":
			warningChains++
		case "critical":
			criticalChains++
		}
	}

	stats := map[string]interface{}{
		"total_chains":    len(bcm.chainHealth),
		"healthy_chains":  healthyChains,
		"warning_chains":  warningChains,
		"critical_chains": criticalChains,
		"policy": map[string]interface{}{
			"max_chain_length":       bcm.policy.MaxChainLength,
			"max_chain_age":          bcm.policy.MaxChainAge,
			"retention_days":         bcm.policy.RetentionDays,
			"auto_synthetic_enabled": bcm.policy.AutoSyntheticEnabled,
			"auto_cleanup_enabled":   bcm.policy.AutoCleanupEnabled,
		},
	}

	return stats
}

// PerformHealthCheck performs a comprehensive health check on all chains.
func (bcm *BackupChainManager) PerformHealthCheck() (map[string]interface{}, error) {
	if !bcm.policy.ChainHealthCheckEnabled {
		return nil, fmt.Errorf("health check is disabled")
	}

	healthReports, err := bcm.MonitorChains()
	if err != nil {
		return nil, fmt.Errorf("failed to monitor chains: %w", err)
	}

	result := map[string]interface{}{
		"timestamp":    time.Now(),
		"total_chains": len(healthReports),
		"healthy":      0,
		"warning":      0,
		"critical":     0,
		"chains":       healthReports,
	}

	for _, health := range healthReports {
		switch health.Status {
		case "healthy":
			result["healthy"] = result["healthy"].(int) + 1
		case "warning":
			result["warning"] = result["warning"].(int) + 1
		case "critical":
			result["critical"] = result["critical"].(int) + 1
		}
	}

	return result, nil
}

// GetOptimizationRecommendations returns optimization recommendations for all chains.
func (bcm *BackupChainManager) GetOptimizationRecommendations() ([]map[string]interface{}, error) {
	manifests := bcm.strategy.ListBackups()

	// Group by database and get latest backup for each
	latestByDatabase := make(map[string]*BackupManifest)
	for _, manifest := range manifests {
		if latest, exists := latestByDatabase[manifest.DatabaseID]; !exists || manifest.CreatedAt.After(latest.CreatedAt) {
			latestByDatabase[manifest.DatabaseID] = manifest
		}
	}

	recommendations := make([]map[string]interface{}, 0)

	for databaseID, latest := range latestByDatabase {
		recommendation, err := bcm.syntheticGen.GetSyntheticBackupRecommendation(latest.ID)
		if err != nil {
			continue
		}

		recommendation["database_id"] = databaseID
		recommendation["latest_backup_id"] = latest.ID

		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}
