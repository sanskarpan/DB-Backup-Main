package bac

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// VersionManager manages configuration versions.
type VersionManager struct {
	mu          sync.RWMutex
	versions    map[string][]*ConfigVersion // configName -> versions
	versionDir  string
	maxVersions int
	autoVersion bool
}

// ConfigVersion represents a versioned configuration.
type ConfigVersion struct {
	Version       string        `json:"version"`
	VersionNum    int           `json:"versionNum"`
	Config        *BackupConfig `json:"config"`
	CreatedAt     time.Time     `json:"createdAt"`
	CreatedBy     string        `json:"createdBy,omitempty"`
	ChangeNote    string        `json:"changeNote,omitempty"`
	Tags          []string      `json:"tags,omitempty"`
	Checksum      string        `json:"checksum"`
	ParentVersion string        `json:"parentVersion,omitempty"`
}

// VersionDiff represents differences between two versions.
type VersionDiff struct {
	FromVersion string          `json:"fromVersion"`
	ToVersion   string          `json:"toVersion"`
	Changes     []VersionChange `json:"changes"`
	Summary     string          `json:"summary"`
	CreatedAt   time.Time       `json:"createdAt"`
}

// VersionChange represents a single change between versions.
type VersionChange struct {
	Type     string      `json:"type"` // added, removed, modified
	Path     string      `json:"path"`
	OldValue interface{} `json:"oldValue,omitempty"`
	NewValue interface{} `json:"newValue,omitempty"`
}

// RollbackPlan represents a plan for rolling back to a previous version.
type RollbackPlan struct {
	ConfigName     string          `json:"configName"`
	CurrentVersion string          `json:"currentVersion"`
	TargetVersion  string          `json:"targetVersion"`
	Changes        []VersionChange `json:"changes"`
	ImpactAnalysis *ImpactAnalysis `json:"impactAnalysis"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// ImpactAnalysis analyzes the impact of a rollback.
type ImpactAnalysis struct {
	AffectedDatabases []string `json:"affectedDatabases"`
	AffectedSchedules []string `json:"affectedSchedules"`
	DisabledFeatures  []string `json:"disabledFeatures,omitempty"`
	EnabledFeatures   []string `json:"enabledFeatures,omitempty"`
	RiskLevel         string   `json:"riskLevel"` // low, medium, high
	Warnings          []string `json:"warnings,omitempty"`
}

// NewVersionManager creates a new version manager.
func NewVersionManager(versionDir string, maxVersions int, autoVersion bool) *VersionManager {
	return &VersionManager{
		versions:    make(map[string][]*ConfigVersion),
		versionDir:  versionDir,
		maxVersions: maxVersions,
		autoVersion: autoVersion,
	}
}

// CreateVersion creates a new version of a configuration.
func (vm *VersionManager) CreateVersion(config *BackupConfig, createdBy, changeNote string) (*ConfigVersion, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	configName := config.Metadata.Name

	// Get existing versions
	versions, exists := vm.versions[configName]
	if !exists {
		versions = make([]*ConfigVersion, 0)
	}

	// Calculate next version number
	versionNum := 1
	var parentVersion string
	if len(versions) > 0 {
		lastVersion := versions[len(versions)-1]
		versionNum = lastVersion.VersionNum + 1
		parentVersion = lastVersion.Version
	}

	// Create version
	version := &ConfigVersion{
		Version:       fmt.Sprintf("v%d", versionNum),
		VersionNum:    versionNum,
		Config:        config,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
		ChangeNote:    changeNote,
		Tags:          make([]string, 0),
		ParentVersion: parentVersion,
	}

	// Calculate checksum
	checksum, err := vm.calculateChecksum(config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	version.Checksum = checksum

	// Add version
	versions = append(versions, version)
	vm.versions[configName] = versions

	// Save version to disk
	if err := vm.saveVersion(version, configName); err != nil {
		return nil, fmt.Errorf("failed to save version: %w", err)
	}

	// Prune old versions if exceeding max
	if vm.maxVersions > 0 && len(versions) > vm.maxVersions {
		vm.pruneVersions(configName)
	}

	return version, nil
}

// GetVersion retrieves a specific version of a configuration.
func (vm *VersionManager) GetVersion(configName, version string) (*ConfigVersion, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	versions, exists := vm.versions[configName]
	if !exists {
		return nil, fmt.Errorf("no versions found for config: %s", configName)
	}

	for _, v := range versions {
		if v.Version == version {
			return v, nil
		}
	}

	return nil, fmt.Errorf("version %s not found for config %s", version, configName)
}

// ListVersions returns all versions of a configuration.
func (vm *VersionManager) ListVersions(configName string) ([]*ConfigVersion, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	versions, exists := vm.versions[configName]
	if !exists {
		return nil, fmt.Errorf("no versions found for config: %s", configName)
	}

	// Return a copy
	result := make([]*ConfigVersion, len(versions))
	copy(result, versions)

	return result, nil
}

// GetLatestVersion returns the latest version of a configuration.
func (vm *VersionManager) GetLatestVersion(configName string) (*ConfigVersion, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	versions, exists := vm.versions[configName]
	if !exists || len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for config: %s", configName)
	}

	return versions[len(versions)-1], nil
}

// DiffVersions computes the difference between two versions.
func (vm *VersionManager) DiffVersions(configName, fromVersion, toVersion string) (*VersionDiff, error) {
	from, err := vm.GetVersion(configName, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get from version: %w", err)
	}

	to, err := vm.GetVersion(configName, toVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get to version: %w", err)
	}

	diff := &VersionDiff{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Changes:     make([]VersionChange, 0),
		CreatedAt:   time.Now(),
	}

	// Compare configurations
	diff.Changes = vm.compareConfigs(from.Config, to.Config)

	// Generate summary
	diff.Summary = vm.generateDiffSummary(diff.Changes)

	return diff, nil
}

// compareConfigs compares two configurations and returns changes.
func (vm *VersionManager) compareConfigs(from, to *BackupConfig) []VersionChange {
	changes := make([]VersionChange, 0)

	// Compare metadata
	if from.Metadata.Name != to.Metadata.Name {
		changes = append(changes, VersionChange{
			Type:     "modified",
			Path:     "metadata.name",
			OldValue: from.Metadata.Name,
			NewValue: to.Metadata.Name,
		})
	}

	// Compare databases
	changes = append(changes, vm.compareDatabases(from.Spec.Databases, to.Spec.Databases)...)

	// Compare schedules
	changes = append(changes, vm.compareSchedules(from.Spec.Schedules, to.Spec.Schedules)...)

	// Compare policies
	changes = append(changes, vm.comparePolicies(from.Spec.Policies, to.Spec.Policies)...)

	// Compare retention
	changes = append(changes, vm.compareRetention(&from.Spec.Retention, &to.Spec.Retention)...)

	return changes
}

// compareDatabases compares database configurations.
func (vm *VersionManager) compareDatabases(from, to []DatabaseConfig) []VersionChange {
	changes := make([]VersionChange, 0)

	fromMap := make(map[string]DatabaseConfig)
	toMap := make(map[string]DatabaseConfig)

	for _, db := range from {
		fromMap[db.ID] = db
	}
	for _, db := range to {
		toMap[db.ID] = db
	}

	// Check for removed databases
	for id := range fromMap {
		if _, exists := toMap[id]; !exists {
			changes = append(changes, VersionChange{
				Type:     "removed",
				Path:     fmt.Sprintf("spec.databases[%s]", id),
				OldValue: fromMap[id].Name,
			})
		}
	}

	// Check for added databases
	for id := range toMap {
		if _, exists := fromMap[id]; !exists {
			changes = append(changes, VersionChange{
				Type:     "added",
				Path:     fmt.Sprintf("spec.databases[%s]", id),
				NewValue: toMap[id].Name,
			})
		}
	}

	// Check for modified databases
	for id, fromDB := range fromMap {
		if toDB, exists := toMap[id]; exists {
			if fromDB.Enabled != toDB.Enabled {
				changes = append(changes, VersionChange{
					Type:     "modified",
					Path:     fmt.Sprintf("spec.databases[%s].enabled", id),
					OldValue: fromDB.Enabled,
					NewValue: toDB.Enabled,
				})
			}
		}
	}

	return changes
}

// compareSchedules compares schedule configurations.
func (vm *VersionManager) compareSchedules(from, to []ScheduleConfig) []VersionChange {
	changes := make([]VersionChange, 0)

	fromMap := make(map[string]ScheduleConfig)
	toMap := make(map[string]ScheduleConfig)

	for _, sched := range from {
		fromMap[sched.Name] = sched
	}
	for _, sched := range to {
		toMap[sched.Name] = sched
	}

	// Check for removed schedules
	for name := range fromMap {
		if _, exists := toMap[name]; !exists {
			changes = append(changes, VersionChange{
				Type:     "removed",
				Path:     fmt.Sprintf("spec.schedules[%s]", name),
				OldValue: fromMap[name].CronExpression,
			})
		}
	}

	// Check for added schedules
	for name := range toMap {
		if _, exists := fromMap[name]; !exists {
			changes = append(changes, VersionChange{
				Type:     "added",
				Path:     fmt.Sprintf("spec.schedules[%s]", name),
				NewValue: toMap[name].CronExpression,
			})
		}
	}

	// Check for modified schedules
	for name, fromSched := range fromMap {
		if toSched, exists := toMap[name]; exists {
			if fromSched.CronExpression != toSched.CronExpression {
				changes = append(changes, VersionChange{
					Type:     "modified",
					Path:     fmt.Sprintf("spec.schedules[%s].cron", name),
					OldValue: fromSched.CronExpression,
					NewValue: toSched.CronExpression,
				})
			}
			if fromSched.Enabled != toSched.Enabled {
				changes = append(changes, VersionChange{
					Type:     "modified",
					Path:     fmt.Sprintf("spec.schedules[%s].enabled", name),
					OldValue: fromSched.Enabled,
					NewValue: toSched.Enabled,
				})
			}
		}
	}

	return changes
}

// comparePolicies compares policy configurations.
func (vm *VersionManager) comparePolicies(from, to []PolicyConfig) []VersionChange {
	changes := make([]VersionChange, 0)

	fromMap := make(map[string]PolicyConfig)
	toMap := make(map[string]PolicyConfig)

	for _, policy := range from {
		fromMap[policy.Name] = policy
	}
	for _, policy := range to {
		toMap[policy.Name] = policy
	}

	// Check for removed policies
	for name := range fromMap {
		if _, exists := toMap[name]; !exists {
			changes = append(changes, VersionChange{
				Type: "removed",
				Path: fmt.Sprintf("spec.policies[%s]", name),
			})
		}
	}

	// Check for added policies
	for name := range toMap {
		if _, exists := fromMap[name]; !exists {
			changes = append(changes, VersionChange{
				Type: "added",
				Path: fmt.Sprintf("spec.policies[%s]", name),
			})
		}
	}

	return changes
}

// compareRetention compares retention configurations.
func (vm *VersionManager) compareRetention(from, to *RetentionConfig) []VersionChange {
	changes := make([]VersionChange, 0)

	if from.Daily != to.Daily {
		changes = append(changes, VersionChange{
			Type:     "modified",
			Path:     "spec.retention.daily",
			OldValue: from.Daily,
			NewValue: to.Daily,
		})
	}

	if from.Weekly != to.Weekly {
		changes = append(changes, VersionChange{
			Type:     "modified",
			Path:     "spec.retention.weekly",
			OldValue: from.Weekly,
			NewValue: to.Weekly,
		})
	}

	if from.Monthly != to.Monthly {
		changes = append(changes, VersionChange{
			Type:     "modified",
			Path:     "spec.retention.monthly",
			OldValue: from.Monthly,
			NewValue: to.Monthly,
		})
	}

	return changes
}

// generateDiffSummary generates a summary of changes.
func (vm *VersionManager) generateDiffSummary(changes []VersionChange) string {
	if len(changes) == 0 {
		return "No changes"
	}

	added := 0
	removed := 0
	modified := 0

	for _, change := range changes {
		switch change.Type {
		case "added":
			added++
		case "removed":
			removed++
		case "modified":
			modified++
		}
	}

	return fmt.Sprintf("%d added, %d removed, %d modified", added, removed, modified)
}

// CreateRollbackPlan creates a plan for rolling back to a previous version.
func (vm *VersionManager) CreateRollbackPlan(configName, targetVersion string) (*RollbackPlan, error) {
	current, err := vm.GetLatestVersion(configName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	target, err := vm.GetVersion(configName, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}

	// Compare versions
	changes := vm.compareConfigs(current.Config, target.Config)

	// Analyze impact
	impact := vm.analyzeImpact(current.Config, target.Config)

	plan := &RollbackPlan{
		ConfigName:     configName,
		CurrentVersion: current.Version,
		TargetVersion:  targetVersion,
		Changes:        changes,
		ImpactAnalysis: impact,
		CreatedAt:      time.Now(),
	}

	return plan, nil
}

// analyzeImpact analyzes the impact of rolling back.
func (vm *VersionManager) analyzeImpact(current, target *BackupConfig) *ImpactAnalysis {
	impact := &ImpactAnalysis{
		AffectedDatabases: make([]string, 0),
		AffectedSchedules: make([]string, 0),
		DisabledFeatures:  make([]string, 0),
		EnabledFeatures:   make([]string, 0),
		Warnings:          make([]string, 0),
		RiskLevel:         "low",
	}

	// Check affected databases
	currentDBs := make(map[string]bool)
	for _, db := range current.Spec.Databases {
		currentDBs[db.ID] = true
	}

	targetDBs := make(map[string]bool)
	for _, db := range target.Spec.Databases {
		targetDBs[db.ID] = true
	}

	// Databases that will be removed
	for dbID := range currentDBs {
		if !targetDBs[dbID] {
			impact.AffectedDatabases = append(impact.AffectedDatabases, dbID)
			impact.RiskLevel = "medium"
			impact.Warnings = append(impact.Warnings, fmt.Sprintf("Database %s will be removed from backup", dbID))
		}
	}

	// Check affected schedules
	if len(current.Spec.Schedules) != len(target.Spec.Schedules) {
		impact.RiskLevel = "medium"
		impact.Warnings = append(impact.Warnings, "Number of schedules will change")
	}

	// Check for disabled features
	if current.Spec.Compliance.Audit != nil && current.Spec.Compliance.Audit.Enabled {
		if target.Spec.Compliance.Audit == nil || !target.Spec.Compliance.Audit.Enabled {
			impact.DisabledFeatures = append(impact.DisabledFeatures, "Audit logging")
		}
	}

	if len(impact.DisabledFeatures) > 0 || len(impact.AffectedDatabases) > 2 {
		impact.RiskLevel = "high"
	}

	return impact
}

// ExecuteRollback executes a rollback plan.
func (vm *VersionManager) ExecuteRollback(plan *RollbackPlan) (*BackupConfig, error) {
	target, err := vm.GetVersion(plan.ConfigName, plan.TargetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}

	// Create a new version with rollback annotation
	rolledBackConfig := target.Config
	if rolledBackConfig.Metadata.Annotations == nil {
		rolledBackConfig.Metadata.Annotations = make(map[string]string)
	}
	rolledBackConfig.Metadata.Annotations["rollback.from"] = plan.CurrentVersion
	rolledBackConfig.Metadata.Annotations["rollback.to"] = plan.TargetVersion
	rolledBackConfig.Metadata.Annotations["rollback.at"] = time.Now().Format(time.RFC3339)

	return rolledBackConfig, nil
}

// TagVersion adds tags to a version.
func (vm *VersionManager) TagVersion(configName, version string, tags []string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	v, err := vm.GetVersion(configName, version)
	if err != nil {
		return err
	}

	v.Tags = append(v.Tags, tags...)

	return vm.saveVersion(v, configName)
}

// GetVersionsByTag returns versions with a specific tag.
func (vm *VersionManager) GetVersionsByTag(configName, tag string) ([]*ConfigVersion, error) {
	versions, err := vm.ListVersions(configName)
	if err != nil {
		return nil, err
	}

	tagged := make([]*ConfigVersion, 0)
	for _, v := range versions {
		for _, t := range v.Tags {
			if t == tag {
				tagged = append(tagged, v)
				break
			}
		}
	}

	return tagged, nil
}

// calculateChecksum calculates a checksum for a configuration.
func (vm *VersionManager) calculateChecksum(config *BackupConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", data)
	return hash[:16], nil // Return first 16 chars
}

// saveVersion saves a version to disk.
func (vm *VersionManager) saveVersion(version *ConfigVersion, configName string) error {
	versionPath := filepath.Join(vm.versionDir, configName)

	// Create directory if not exists
	if err := os.MkdirAll(versionPath, 0o755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Save version
	filePath := filepath.Join(versionPath, fmt.Sprintf("%s.json", version.Version))
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// loadVersions loads versions from disk.
func (vm *VersionManager) loadVersions(configName string) error {
	versionPath := filepath.Join(vm.versionDir, configName)

	entries, err := os.ReadDir(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No versions yet
		}
		return fmt.Errorf("failed to read version directory: %w", err)
	}

	versions := make([]*ConfigVersion, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(versionPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var version ConfigVersion
		if err := json.Unmarshal(data, &version); err != nil {
			continue
		}

		versions = append(versions, &version)
	}

	// Sort by version number
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].VersionNum < versions[j].VersionNum
	})

	vm.versions[configName] = versions

	return nil
}

// pruneVersions removes old versions exceeding max limit.
func (vm *VersionManager) pruneVersions(configName string) {
	versions := vm.versions[configName]

	if len(versions) <= vm.maxVersions {
		return
	}

	// Keep only the latest maxVersions
	toRemove := versions[:len(versions)-vm.maxVersions]

	for _, version := range toRemove {
		// Remove file
		filePath := filepath.Join(vm.versionDir, configName, fmt.Sprintf("%s.json", version.Version))
		os.Remove(filePath)
	}

	// Update in-memory versions
	vm.versions[configName] = versions[len(versions)-vm.maxVersions:]
}

// GetVersionHistory returns version history with metadata.
func (vm *VersionManager) GetVersionHistory(configName string) ([]map[string]interface{}, error) {
	versions, err := vm.ListVersions(configName)
	if err != nil {
		return nil, err
	}

	history := make([]map[string]interface{}, 0)

	for _, v := range versions {
		entry := map[string]interface{}{
			"version":     v.Version,
			"created_at":  v.CreatedAt,
			"created_by":  v.CreatedBy,
			"change_note": v.ChangeNote,
			"tags":        v.Tags,
			"checksum":    v.Checksum,
		}
		history = append(history, entry)
	}

	return history, nil
}
