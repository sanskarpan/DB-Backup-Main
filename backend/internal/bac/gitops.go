package bac

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// syncStatusFailure marks a sync record as failed.
const syncStatusFailure = "failure"

// GitOpsController manages GitOps workflow for backup configurations.
type GitOpsController struct {
	mu              sync.RWMutex
	configManager   *ConfigManager
	validator       *Validator
	gitConfig       *GitOpsConfig
	localRepoPath   string
	lastSync        time.Time
	syncInterval    time.Duration
	reconcileQueue  chan *ReconcileRequest
	stopChan        chan struct{}
	running         bool
	syncHistory     []*SyncRecord
	conflictHandler ConflictHandler
}

// ReconcileRequest represents a request to reconcile configuration.
type ReconcileRequest struct {
	ConfigName string
	Source     string // "git" or "local"
	Timestamp  time.Time
}

// SyncRecord tracks synchronization history.
type SyncRecord struct {
	Timestamp      time.Time     `json:"timestamp"`
	CommitHash     string        `json:"commitHash"`
	Status         string        `json:"status"` // success, failure, conflict
	ConfigsAdded   int           `json:"configsAdded"`
	ConfigsUpdated int           `json:"configsUpdated"`
	ConfigsDeleted int           `json:"configsDeleted"`
	Errors         []string      `json:"errors,omitempty"`
	Duration       time.Duration `json:"duration"`
}

// ConflictHandler handles conflicts between Git and local configurations.
type ConflictHandler interface {
	ResolveConflict(gitConfig, localConfig *BackupConfig) (*BackupConfig, error)
}

// GitSourcePriorityHandler always prefers Git source.
type GitSourcePriorityHandler struct{}

func (h *GitSourcePriorityHandler) ResolveConflict(gitConfig, localConfig *BackupConfig) (*BackupConfig, error) {
	return gitConfig, nil
}

// LocalSourcePriorityHandler always prefers local source.
type LocalSourcePriorityHandler struct{}

func (h *LocalSourcePriorityHandler) ResolveConflict(gitConfig, localConfig *BackupConfig) (*BackupConfig, error) {
	return localConfig, nil
}

// NewestWinsHandler prefers the most recently updated configuration.
type NewestWinsHandler struct{}

func (h *NewestWinsHandler) ResolveConflict(gitConfig, localConfig *BackupConfig) (*BackupConfig, error) {
	if gitConfig.Metadata.UpdatedAt.After(localConfig.Metadata.UpdatedAt) {
		return gitConfig, nil
	}
	return localConfig, nil
}

// NewGitOpsController creates a new GitOps controller.
func NewGitOpsController(
	configManager *ConfigManager,
	validator *Validator,
	gitConfig *GitOpsConfig,
	localRepoPath string,
) *GitOpsController {
	syncInterval, err := time.ParseDuration(gitConfig.SyncInterval)
	if err != nil || syncInterval == 0 {
		syncInterval = 5 * time.Minute // Default 5 minutes
	}

	return &GitOpsController{
		configManager:   configManager,
		validator:       validator,
		gitConfig:       gitConfig,
		localRepoPath:   localRepoPath,
		syncInterval:    syncInterval,
		reconcileQueue:  make(chan *ReconcileRequest, 100),
		stopChan:        make(chan struct{}),
		syncHistory:     make([]*SyncRecord, 0),
		conflictHandler: &GitSourcePriorityHandler{}, // Default handler
	}
}

// SetConflictHandler sets the conflict resolution handler.
func (gc *GitOpsController) SetConflictHandler(handler ConflictHandler) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.conflictHandler = handler
}

// Start starts the GitOps controller.
func (gc *GitOpsController) Start() error {
	gc.mu.Lock()
	if gc.running {
		gc.mu.Unlock()
		return fmt.Errorf("GitOps controller already running")
	}
	gc.running = true
	gc.mu.Unlock()

	// Clone or pull repository
	if err := gc.initializeRepository(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Start reconciliation worker
	go gc.reconcileWorker()

	// Start sync loop
	go gc.syncLoop()

	return nil
}

// Stop stops the GitOps controller.
func (gc *GitOpsController) Stop() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if !gc.running {
		return fmt.Errorf("GitOps controller not running")
	}

	close(gc.stopChan)
	gc.running = false

	return nil
}

// initializeRepository clones or pulls the Git repository.
func (gc *GitOpsController) initializeRepository() error {
	// Check if repository exists
	if _, err := os.Stat(filepath.Join(gc.localRepoPath, ".git")); os.IsNotExist(err) {
		// Clone repository
		return gc.cloneRepository()
	}

	// Pull latest changes
	return gc.pullRepository()
}

// cloneRepository clones the Git repository.
func (gc *GitOpsController) cloneRepository() error {
	args := []string{"clone", "--branch", gc.gitConfig.Branch}

	// Add SSH key if provided
	if gc.gitConfig.SSHKeyRef != "" {
		// Set GIT_SSH_COMMAND to use specific SSH key
		os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", gc.gitConfig.SSHKeyRef))
	}

	args = append(args, gc.gitConfig.Repository, gc.localRepoPath)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s - %w", string(output), err)
	}

	return nil
}

// pullRepository pulls latest changes from Git.
func (gc *GitOpsController) pullRepository() error {
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd := exec.Command("git", "-C", gc.localRepoPath, "pull", "origin", gc.gitConfig.Branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s - %w", string(output), err)
	}

	return nil
}

// syncLoop periodically syncs with Git repository.
func (gc *GitOpsController) syncLoop() {
	ticker := time.NewTicker(gc.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := gc.Sync(); err != nil {
				// Log error but continue
				fmt.Printf("GitOps sync error: %v\n", err)
			}
		case <-gc.stopChan:
			return
		}
	}
}

// Sync synchronizes configurations from Git repository.
func (gc *GitOpsController) Sync() error {
	startTime := time.Now()
	record := &SyncRecord{
		Timestamp: startTime,
		Status:    "in_progress",
		Errors:    make([]string, 0),
	}

	// Pull latest changes
	if err := gc.pullRepository(); err != nil {
		record.Status = syncStatusFailure
		record.Errors = append(record.Errors, err.Error())
		gc.addSyncRecord(record)
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	// Get current commit hash
	commitHash, err := gc.getCurrentCommitHash()
	if err != nil {
		record.Status = syncStatusFailure
		record.Errors = append(record.Errors, err.Error())
		gc.addSyncRecord(record)
		return fmt.Errorf("failed to get commit hash: %w", err)
	}
	record.CommitHash = commitHash

	// Load configurations from Git
	configPath := filepath.Join(gc.localRepoPath, gc.gitConfig.Path)
	if gc.gitConfig.Path == "" {
		configPath = gc.localRepoPath
	}

	gitConfigs := make(map[string]*BackupConfig)
	if err := gc.loadGitConfigs(configPath, gitConfigs); err != nil {
		record.Status = syncStatusFailure
		record.Errors = append(record.Errors, err.Error())
		gc.addSyncRecord(record)
		return fmt.Errorf("failed to load Git configs: %w", err)
	}

	// Get current local configs
	localConfigs := make(map[string]*BackupConfig)
	for _, config := range gc.configManager.ListConfigs() {
		localConfigs[config.Metadata.Name] = config
	}

	// Reconcile configurations
	added, updated, deleted, errors := gc.reconcileConfigs(gitConfigs, localConfigs)

	record.ConfigsAdded = added
	record.ConfigsUpdated = updated
	record.ConfigsDeleted = deleted
	record.Errors = errors
	record.Duration = time.Since(startTime)

	if len(errors) > 0 {
		record.Status = "partial_success"
	} else {
		record.Status = "success"
	}

	gc.addSyncRecord(record)
	gc.lastSync = time.Now()

	return nil
}

// loadGitConfigs loads configurations from Git repository.
func (gc *GitOpsController) loadGitConfigs(path string, configs map[string]*BackupConfig) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Recursively load from subdirectories
			if err := gc.loadGitConfigs(fullPath, configs); err != nil {
				return err
			}
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		config, err := gc.configManager.LoadConfig(fullPath)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}

		configs[config.Metadata.Name] = config
	}

	return nil
}

// reconcileConfigs reconciles Git and local configurations.
func (gc *GitOpsController) reconcileConfigs(
	gitConfigs map[string]*BackupConfig,
	localConfigs map[string]*BackupConfig,
) (added, updated, deleted int, errors []string) {
	errors = make([]string, 0)

	// Add or update configs from Git
	for name, gitConfig := range gitConfigs {
		a, u, errs := gc.reconcileGitConfig(name, gitConfig, localConfigs)
		added += a
		updated += u
		errors = append(errors, errs...)
	}

	// Delete configs removed from Git
	for name := range localConfigs {
		if _, exists := gitConfigs[name]; exists {
			continue
		}
		d, errs := gc.reconcileDeletion(name)
		deleted += d
		errors = append(errors, errs...)
	}

	return added, updated, deleted, errors
}

// reconcileGitConfig reconciles a single config sourced from Git.
func (gc *GitOpsController) reconcileGitConfig(
	name string,
	gitConfig *BackupConfig,
	localConfigs map[string]*BackupConfig,
) (added, updated int, errors []string) {
	// Validate configuration
	validationResult := gc.validator.Validate(gitConfig)
	if !validationResult.Valid {
		errors = append(errors, fmt.Sprintf("Config %s validation failed: %d errors", name, len(validationResult.Errors)))
		return added, updated, errors
	}

	localConfig, exists := localConfigs[name]
	if !exists {
		// New configuration from Git
		if gc.gitConfig.DryRun {
			fmt.Printf("DRY RUN: Would add config: %s\n", name)
			return added, updated, errors
		}
		if gc.gitConfig.AutoApply {
			if err := gc.configManager.RegisterConfig(gitConfig); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to register %s: %v", name, err))
			} else {
				added++
			}
		}
		return added, updated, errors
	}

	// Existing config: check if it changed
	if !gc.configChanged(gitConfig, localConfig) {
		return added, updated, errors
	}

	// Resolve conflict
	resolvedConfig, err := gc.conflictHandler.ResolveConflict(gitConfig, localConfig)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to resolve conflict for %s: %v", name, err))
		return added, updated, errors
	}

	if gc.gitConfig.DryRun {
		fmt.Printf("DRY RUN: Would update config: %s\n", name)
		return added, updated, errors
	}
	if gc.gitConfig.AutoApply {
		if err := gc.configManager.UpdateConfig(resolvedConfig); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to update %s: %v", name, err))
		} else {
			updated++
		}
	}

	return added, updated, errors
}

// reconcileDeletion removes a local config that no longer exists in Git.
func (gc *GitOpsController) reconcileDeletion(name string) (deleted int, errors []string) {
	if gc.gitConfig.DryRun {
		fmt.Printf("DRY RUN: Would delete config: %s\n", name)
		return deleted, errors
	}
	if gc.gitConfig.AutoApply {
		if err := gc.configManager.UnregisterConfig(name); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to unregister %s: %v", name, err))
		} else {
			deleted++
		}
	}
	return deleted, errors
}

// configChanged checks if a configuration has changed.
func (gc *GitOpsController) configChanged(gitConfig, localConfig *BackupConfig) bool {
	// Compare checksums of serialized configs
	gitHash := gc.computeConfigHash(gitConfig)
	localHash := gc.computeConfigHash(localConfig)

	return gitHash != localHash
}

// computeConfigHash computes a hash of the configuration.
func (gc *GitOpsController) computeConfigHash(config *BackupConfig) string {
	// Create a copy without status and metadata timestamps
	configCopy := *config
	configCopy.Status = nil
	configCopy.Metadata.CreatedAt = time.Time{}
	configCopy.Metadata.UpdatedAt = time.Time{}

	// Serialize to YAML
	data := marshalYAML(&configCopy)

	// Compute SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// marshalYAML is a helper to marshal configuration.
func marshalYAML(config *BackupConfig) []byte {
	return []byte(fmt.Sprintf("%+v", config))
}

// getCurrentCommitHash gets the current Git commit hash.
func (gc *GitOpsController) getCurrentCommitHash() (string, error) {
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd := exec.Command("git", "-C", gc.localRepoPath, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return string(bytes.TrimSpace(output)), nil
}

// reconcileWorker processes reconciliation requests.
func (gc *GitOpsController) reconcileWorker() {
	for {
		select {
		case req := <-gc.reconcileQueue:
			gc.processReconcileRequest(req)
		case <-gc.stopChan:
			return
		}
	}
}

// processReconcileRequest processes a reconciliation request.
func (gc *GitOpsController) processReconcileRequest(req *ReconcileRequest) {
	// Implementation for processing reconciliation requests
	fmt.Printf("Processing reconcile request for %s from %s\n", req.ConfigName, req.Source)
}

// addSyncRecord adds a sync record to history.
func (gc *GitOpsController) addSyncRecord(record *SyncRecord) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.syncHistory = append(gc.syncHistory, record)

	// Keep only last 100 records
	if len(gc.syncHistory) > 100 {
		gc.syncHistory = gc.syncHistory[1:]
	}
}

// GetSyncHistory returns sync history.
func (gc *GitOpsController) GetSyncHistory() []*SyncRecord {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	history := make([]*SyncRecord, len(gc.syncHistory))
	copy(history, gc.syncHistory)

	return history
}

// GetLastSync returns the timestamp of the last sync.
func (gc *GitOpsController) GetLastSync() time.Time {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	return gc.lastSync
}

// TriggerSync manually triggers a sync.
func (gc *GitOpsController) TriggerSync() error {
	return gc.Sync()
}

// PushConfig pushes a local configuration to Git.
func (gc *GitOpsController) PushConfig(config *BackupConfig, commitMessage string) error {
	// Save config to repository
	configPath := filepath.Join(gc.localRepoPath, gc.gitConfig.Path, config.Metadata.Name+".yaml")

	if err := gc.configManager.SaveConfig(config, configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Git add
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd := exec.Command("git", "-C", gc.localRepoPath, "add", configPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s - %w", string(output), err)
	}

	// Git commit
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd = exec.Command("git", "-C", gc.localRepoPath, "commit", "-m", commitMessage)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s - %w", string(output), err)
	}

	// Git push
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd = exec.Command("git", "-C", gc.localRepoPath, "push", "origin", gc.gitConfig.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s - %w", string(output), err)
	}

	return nil
}

// ValidateRepository validates the Git repository configuration.
func (gc *GitOpsController) ValidateRepository() error {
	// Check if Git is installed
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}

	// Check repository access
	if err := gc.testRepositoryAccess(); err != nil {
		return fmt.Errorf("repository access failed: %w", err)
	}

	return nil
}

// testRepositoryAccess tests if the repository is accessible.
func (gc *GitOpsController) testRepositoryAccess() error {
	//nolint:gosec // G204: git args come from trusted GitOps config, not user input
	cmd := exec.Command("git", "ls-remote", gc.gitConfig.Repository)

	if gc.gitConfig.SSHKeyRef != "" {
		os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", gc.gitConfig.SSHKeyRef))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to access repository: %s - %w", string(output), err)
	}

	return nil
}

// GetStatus returns the current GitOps status.
func (gc *GitOpsController) GetStatus() map[string]interface{} {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	status := map[string]interface{}{
		"running":       gc.running,
		"last_sync":     gc.lastSync,
		"sync_interval": gc.syncInterval.String(),
		"repository":    gc.gitConfig.Repository,
		"branch":        gc.gitConfig.Branch,
		"auto_apply":    gc.gitConfig.AutoApply,
		"dry_run":       gc.gitConfig.DryRun,
	}

	// Add last sync record
	if len(gc.syncHistory) > 0 {
		lastRecord := gc.syncHistory[len(gc.syncHistory)-1]
		status["last_sync_status"] = lastRecord.Status
		status["last_sync_duration"] = lastRecord.Duration.String()
		status["last_commit_hash"] = lastRecord.CommitHash
	}

	return status
}

// ExportConfig exports a configuration to a file.
func (gc *GitOpsController) ExportConfig(config *BackupConfig, outputPath string) error {
	return gc.configManager.SaveConfig(config, outputPath)
}

// ImportConfig imports a configuration from a file.
func (gc *GitOpsController) ImportConfig(inputPath string) (*BackupConfig, error) {
	config, err := gc.configManager.LoadConfig(inputPath)
	if err != nil {
		return nil, err
	}

	// Validate
	validationResult := gc.validator.Validate(config)
	if !validationResult.Valid {
		return nil, fmt.Errorf("validation failed: %d errors", len(validationResult.Errors))
	}

	return config, nil
}
