package bac

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// BackupConfig represents the declarative backup configuration.
type BackupConfig struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion"`
	Kind       string         `yaml:"kind" json:"kind"`
	Metadata   ConfigMetadata `yaml:"metadata" json:"metadata"`
	Spec       BackupSpec     `yaml:"spec" json:"spec"`
	Status     *ConfigStatus  `yaml:"status,omitempty" json:"status,omitempty"`
}

// ConfigMetadata contains metadata about the configuration.
type ConfigMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Version     string            `yaml:"version" json:"version"`
	CreatedAt   time.Time         `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time         `yaml:"updatedAt" json:"updatedAt"`
}

// BackupSpec defines the desired state of backup configuration.
type BackupSpec struct {
	Databases  []DatabaseConfig `yaml:"databases" json:"databases"`
	Schedules  []ScheduleConfig `yaml:"schedules" json:"schedules"`
	Policies   []PolicyConfig   `yaml:"policies" json:"policies"`
	Retention  RetentionConfig  `yaml:"retention" json:"retention"`
	Storage    StorageConfig    `yaml:"storage" json:"storage"`
	Compliance ComplianceConfig `yaml:"compliance,omitempty" json:"compliance,omitempty"`
	GitOps     *GitOpsConfig    `yaml:"gitOps,omitempty" json:"gitOps,omitempty"`
}

// DatabaseConfig defines a database to backup.
type DatabaseConfig struct {
	ID         string            `yaml:"id" json:"id"`
	Name       string            `yaml:"name" json:"name"`
	Type       string            `yaml:"type" json:"type"` // postgres, mysql, mongodb, etc.
	Connection ConnectionConfig  `yaml:"connection" json:"connection"`
	Enabled    bool              `yaml:"enabled" json:"enabled"`
	Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Metadata   map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// ConnectionConfig defines database connection details.
type ConnectionConfig struct {
	Host        string            `yaml:"host" json:"host"`
	Port        int               `yaml:"port" json:"port"`
	Database    string            `yaml:"database,omitempty" json:"database,omitempty"`
	Username    string            `yaml:"username" json:"username"`
	PasswordRef string            `yaml:"passwordRef" json:"passwordRef"` // Reference to secret
	SSLMode     string            `yaml:"sslMode,omitempty" json:"sslMode,omitempty"`
	SSLCertRef  string            `yaml:"sslCertRef,omitempty" json:"sslCertRef,omitempty"`
	Options     map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// ScheduleConfig defines backup schedules.
type ScheduleConfig struct {
	Name           string   `yaml:"name" json:"name"`
	Type           string   `yaml:"type" json:"type"` // full, incremental, differential
	CronExpression string   `yaml:"cron" json:"cron"`
	Databases      []string `yaml:"databases,omitempty" json:"databases,omitempty"` // Database IDs or "*" for all
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	Window         *Window  `yaml:"window,omitempty" json:"window,omitempty"`
}

// Window defines a time window for backups.
type Window struct {
	Start    string `yaml:"start" json:"start"` // HH:MM format
	End      string `yaml:"end" json:"end"`     // HH:MM format
	Timezone string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

// PolicyConfig defines backup policies.
type PolicyConfig struct {
	Name               string               `yaml:"name" json:"name"`
	Compression        string               `yaml:"compression,omitempty" json:"compression,omitempty"` // gzip, zstd, lz4
	Encryption         *EncryptionConfig    `yaml:"encryption,omitempty" json:"encryption,omitempty"`
	Verification       *VerificationConfig  `yaml:"verification,omitempty" json:"verification,omitempty"`
	Parallelism        int                  `yaml:"parallelism,omitempty" json:"parallelism,omitempty"`
	Timeout            string               `yaml:"timeout,omitempty" json:"timeout,omitempty"` // Duration string like "2h"
	RetryPolicy        *RetryPolicy         `yaml:"retryPolicy,omitempty" json:"retryPolicy,omitempty"`
	Notifications      []NotificationConfig `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	PreBackupScript    string               `yaml:"preBackupScript,omitempty" json:"preBackupScript,omitempty"`
	PostBackupScript   string               `yaml:"postBackupScript,omitempty" json:"postBackupScript,omitempty"`
	IncrementalForever bool                 `yaml:"incrementalForever,omitempty" json:"incrementalForever,omitempty"`
	SyntheticFull      *SyntheticFullConfig `yaml:"syntheticFull,omitempty" json:"syntheticFull,omitempty"`
}

// EncryptionConfig defines encryption settings.
type EncryptionConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Algorithm   string `yaml:"algorithm" json:"algorithm"`                         // aes-256-gcm, chacha20-poly1305
	KeyRef      string `yaml:"keyRef" json:"keyRef"`                               // Reference to encryption key
	KMSProvider string `yaml:"kmsProvider,omitempty" json:"kmsProvider,omitempty"` // aws-kms, azure-kv, gcp-kms
}

// VerificationConfig defines backup verification settings.
type VerificationConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	Checksum        bool   `yaml:"checksum" json:"checksum"`
	RestoreTest     bool   `yaml:"restoreTest,omitempty" json:"restoreTest,omitempty"`
	RestoreTestFreq string `yaml:"restoreTestFreq,omitempty" json:"restoreTestFreq,omitempty"` // daily, weekly, monthly
}

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	MaxAttempts int    `yaml:"maxAttempts" json:"maxAttempts"`
	Delay       string `yaml:"delay" json:"delay"`             // Duration string like "5m"
	BackoffType string `yaml:"backoffType" json:"backoffType"` // linear, exponential
}

// NotificationConfig defines notification settings.
type NotificationConfig struct {
	Type     string   `yaml:"type" json:"type"` // email, slack, webhook, pagerduty
	Endpoint string   `yaml:"endpoint" json:"endpoint"`
	Events   []string `yaml:"events" json:"events"` // success, failure, warning
	Template string   `yaml:"template,omitempty" json:"template,omitempty"`
}

// SyntheticFullConfig defines synthetic full backup settings.
type SyntheticFullConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	MaxChainLength int    `yaml:"maxChainLength" json:"maxChainLength"`
	MaxChainAge    string `yaml:"maxChainAge" json:"maxChainAge"` // Duration string like "7d"
	AutoCreate     bool   `yaml:"autoCreate" json:"autoCreate"`
	Schedule       string `yaml:"schedule,omitempty" json:"schedule,omitempty"` // Cron expression
}

// RetentionConfig defines retention policies.
type RetentionConfig struct {
	Daily          int `yaml:"daily" json:"daily"`                                       // Keep daily backups for N days
	Weekly         int `yaml:"weekly" json:"weekly"`                                     // Keep weekly backups for N weeks
	Monthly        int `yaml:"monthly" json:"monthly"`                                   // Keep monthly backups for N months
	Yearly         int `yaml:"yearly" json:"yearly"`                                     // Keep yearly backups for N years
	MinimumBackups int `yaml:"minimumBackups,omitempty" json:"minimumBackups,omitempty"` // Always keep at least N backups
}

// StorageConfig defines storage backend configuration.
type StorageConfig struct {
	Type       string                 `yaml:"type" json:"type"` // s3, gcs, azure, local, nfs
	Location   string                 `yaml:"location" json:"location"`
	Encryption bool                   `yaml:"encryption,omitempty" json:"encryption,omitempty"`
	Options    map[string]interface{} `yaml:"options,omitempty" json:"options,omitempty"`
	Redundancy *RedundancyConfig      `yaml:"redundancy,omitempty" json:"redundancy,omitempty"`
}

// RedundancyConfig defines storage redundancy settings.
type RedundancyConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	Destinations []string `yaml:"destinations" json:"destinations"`
	RequireAll   bool     `yaml:"requireAll" json:"requireAll"` // Require all destinations to succeed
}

// ComplianceConfig defines compliance requirements.
type ComplianceConfig struct {
	Standards    []string            `yaml:"standards,omitempty" json:"standards,omitempty"` // GDPR, HIPAA, SOC2, etc.
	Immutability *ImmutabilityConfig `yaml:"immutability,omitempty" json:"immutability,omitempty"`
	Audit        *AuditConfig        `yaml:"audit,omitempty" json:"audit,omitempty"`
	DataMasking  []DataMaskingRule   `yaml:"dataMasking,omitempty" json:"dataMasking,omitempty"`
}

// ImmutabilityConfig defines immutability settings.
type ImmutabilityConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	LockPeriod string `yaml:"lockPeriod" json:"lockPeriod"` // Duration string like "90d"
	LegalHold  bool   `yaml:"legalHold,omitempty" json:"legalHold,omitempty"`
}

// AuditConfig defines audit logging settings.
type AuditConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	LogLevel     string   `yaml:"logLevel" json:"logLevel"`         // info, verbose, debug
	Destinations []string `yaml:"destinations" json:"destinations"` // file, syslog, cloudwatch
}

// DataMaskingRule defines data masking rules.
type DataMaskingRule struct {
	Table   string   `yaml:"table" json:"table"`
	Columns []string `yaml:"columns" json:"columns"`
	Method  string   `yaml:"method" json:"method"` // hash, redact, random, preserve-format
}

// GitOpsConfig defines GitOps integration settings.
type GitOpsConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Repository     string `yaml:"repository" json:"repository"`
	Branch         string `yaml:"branch" json:"branch"`
	Path           string `yaml:"path,omitempty" json:"path,omitempty"`
	SyncInterval   string `yaml:"syncInterval" json:"syncInterval"` // Duration string like "5m"
	AutoApply      bool   `yaml:"autoApply" json:"autoApply"`
	DryRun         bool   `yaml:"dryRun,omitempty" json:"dryRun,omitempty"`
	SSHKeyRef      string `yaml:"sshKeyRef,omitempty" json:"sshKeyRef,omitempty"`
	WebhookEnabled bool   `yaml:"webhookEnabled,omitempty" json:"webhookEnabled,omitempty"`
}

// ConfigStatus represents the current status of the configuration.
type ConfigStatus struct {
	Phase         string      `yaml:"phase" json:"phase"` // Pending, Active, Failed, Updating
	LastApplied   time.Time   `yaml:"lastApplied" json:"lastApplied"`
	LastSynced    time.Time   `yaml:"lastSynced,omitempty" json:"lastSynced,omitempty"`
	Conditions    []Condition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	ActiveBackups int         `yaml:"activeBackups" json:"activeBackups"`
	FailedBackups int         `yaml:"failedBackups" json:"failedBackups"`
	NextScheduled *time.Time  `yaml:"nextScheduled,omitempty" json:"nextScheduled,omitempty"`
	ErrorMessage  string      `yaml:"errorMessage,omitempty" json:"errorMessage,omitempty"`
}

// Condition represents a status condition.
type Condition struct {
	Type               string    `yaml:"type" json:"type"`     // Ready, Validated, Synced
	Status             string    `yaml:"status" json:"status"` // True, False, Unknown
	LastTransitionTime time.Time `yaml:"lastTransitionTime" json:"lastTransitionTime"`
	Reason             string    `yaml:"reason" json:"reason"`
	Message            string    `yaml:"message,omitempty" json:"message,omitempty"`
}

// ConfigManager manages backup configurations.
type ConfigManager struct {
	mu           sync.RWMutex
	configs      map[string]*BackupConfig
	configDir    string
	watchEnabled bool
	watchers     []ConfigWatcher
}

// ConfigWatcher is an interface for watching configuration changes.
type ConfigWatcher interface {
	OnConfigChange(config *BackupConfig, changeType string) error
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(configDir string) *ConfigManager {
	return &ConfigManager{
		configs:      make(map[string]*BackupConfig),
		configDir:    configDir,
		watchEnabled: false,
		watchers:     make([]ConfigWatcher, 0),
	}
}

// LoadConfig loads a backup configuration from file.
func (cm *ConfigManager) LoadConfig(filePath string) (*BackupConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config BackupConfig

	// Determine format based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = json.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set default values
	if config.Status == nil {
		config.Status = &ConfigStatus{
			Phase: "Pending",
		}
	}

	return &config, nil
}

// SaveConfig saves a backup configuration to file.
func (cm *ConfigManager) SaveConfig(config *BackupConfig, filePath string) error {
	var data []byte
	var err error

	// Update timestamp
	config.Metadata.UpdatedAt = time.Now()

	// Determine format based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filePath, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// RegisterConfig registers a configuration.
func (cm *ConfigManager) RegisterConfig(config *BackupConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if config.Metadata.Name == "" {
		return fmt.Errorf("config name is required")
	}

	cm.configs[config.Metadata.Name] = config

	// Notify watchers
	for _, watcher := range cm.watchers {
		if err := watcher.OnConfigChange(config, "register"); err != nil {
			return fmt.Errorf("watcher error: %w", err)
		}
	}

	return nil
}

// UnregisterConfig unregisters a configuration.
func (cm *ConfigManager) UnregisterConfig(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("config %s not found", name)
	}

	delete(cm.configs, name)

	// Notify watchers
	for _, watcher := range cm.watchers {
		if err := watcher.OnConfigChange(config, "unregister"); err != nil {
			return fmt.Errorf("watcher error: %w", err)
		}
	}

	return nil
}

// GetConfig retrieves a configuration by name.
func (cm *ConfigManager) GetConfig(name string) (*BackupConfig, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, exists := cm.configs[name]
	return config, exists
}

// ListConfigs returns all registered configurations.
func (cm *ConfigManager) ListConfigs() []*BackupConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*BackupConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configs = append(configs, config)
	}

	return configs
}

// UpdateConfig updates an existing configuration.
func (cm *ConfigManager) UpdateConfig(config *BackupConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if config.Metadata.Name == "" {
		return fmt.Errorf("config name is required")
	}

	if _, exists := cm.configs[config.Metadata.Name]; !exists {
		return fmt.Errorf("config %s not found", config.Metadata.Name)
	}

	config.Metadata.UpdatedAt = time.Now()
	cm.configs[config.Metadata.Name] = config

	// Notify watchers
	for _, watcher := range cm.watchers {
		if err := watcher.OnConfigChange(config, "update"); err != nil {
			return fmt.Errorf("watcher error: %w", err)
		}
	}

	return nil
}

// AddWatcher adds a configuration watcher.
func (cm *ConfigManager) AddWatcher(watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.watchers = append(cm.watchers, watcher)
}

// LoadConfigsFromDirectory loads all configurations from a directory.
func (cm *ConfigManager) LoadConfigsFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		config, err := cm.LoadConfig(filePath)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}

		err = cm.RegisterConfig(config)
		if err != nil {
			return fmt.Errorf("failed to register %s: %w", entry.Name(), err)
		}

		loadedCount++
	}

	return nil
}

// GetConfigsByDatabase returns configurations that include a specific database.
func (cm *ConfigManager) GetConfigsByDatabase(databaseID string) []*BackupConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*BackupConfig, 0)
	for _, config := range cm.configs {
		for _, db := range config.Spec.Databases {
			if db.ID == databaseID {
				configs = append(configs, config)
				break
			}
		}
	}

	return configs
}

// GetConfigsByLabel returns configurations matching label selectors.
func (cm *ConfigManager) GetConfigsByLabel(labelSelector map[string]string) []*BackupConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configs := make([]*BackupConfig, 0)
	for _, config := range cm.configs {
		if matchesLabels(config.Metadata.Labels, labelSelector) {
			configs = append(configs, config)
		}
	}

	return configs
}

// matchesLabels checks if labels match the selector.
func matchesLabels(labels, selector map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// UpdateConfigStatus updates the status of a configuration.
func (cm *ConfigManager) UpdateConfigStatus(name string, status *ConfigStatus) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("config %s not found", name)
	}

	config.Status = status
	return nil
}

// GetStatistics returns statistics about managed configurations.
func (cm *ConfigManager) GetStatistics() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalDatabases := 0
	totalSchedules := 0
	activeConfigs := 0
	failedConfigs := 0

	for _, config := range cm.configs {
		totalDatabases += len(config.Spec.Databases)
		totalSchedules += len(config.Spec.Schedules)

		if config.Status != nil {
			switch config.Status.Phase {
			case "Active":
				activeConfigs++
			case "Failed":
				failedConfigs++
			}
		}
	}

	return map[string]interface{}{
		"total_configs":   len(cm.configs),
		"active_configs":  activeConfigs,
		"failed_configs":  failedConfigs,
		"total_databases": totalDatabases,
		"total_schedules": totalSchedules,
	}
}
