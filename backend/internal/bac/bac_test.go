package bac

import (
	"path/filepath"
	"testing"
	"time"
)

// Test ConfigManager

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager("/tmp/test-configs")
	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}

	if cm.configDir != "/tmp/test-configs" {
		t.Errorf("Expected configDir '/tmp/test-configs', got '%s'", cm.configDir)
	}
}

func TestConfigManager_RegisterAndGetConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/test-configs")

	config := &BackupConfig{
		APIVersion: "v1",
		Kind:       "BackupConfig",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			Version:   "1.0.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{
				{
					ID:      "db-001",
					Name:    "test-db",
					Type:    "postgres",
					Enabled: true,
					Connection: ConnectionConfig{
						Host:        "localhost",
						Port:        5432,
						Username:    "postgres",
						PasswordRef: "secret-001",
					},
				},
			},
			Schedules: []ScheduleConfig{
				{
					Name:           "daily-full",
					Type:           "full",
					CronExpression: "0 2 * * *",
					Databases:      []string{"*"},
					Enabled:        true,
				},
			},
			Policies: []PolicyConfig{},
			Retention: RetentionConfig{
				Daily:   7,
				Weekly:  4,
				Monthly: 12,
				Yearly:  5,
			},
			Storage: StorageConfig{
				Type:     "s3",
				Location: "s3://backup-bucket/backups",
			},
		},
	}

	// Register config
	err := cm.RegisterConfig(config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Get config
	retrieved, exists := cm.GetConfig("test-config")
	if !exists {
		t.Fatal("Config not found after registration")
	}

	if retrieved.Metadata.Name != "test-config" {
		t.Errorf("Expected name 'test-config', got '%s'", retrieved.Metadata.Name)
	}
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/test-configs")

	config := &BackupConfig{
		APIVersion: "v1",
		Kind:       "BackupConfig",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			Version:   "1.0.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 7},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	err := cm.RegisterConfig(config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Update config
	config.Spec.Retention.Daily = 14

	err = cm.UpdateConfig(config)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify update
	retrieved, _ := cm.GetConfig("test-config")
	if retrieved.Spec.Retention.Daily != 14 {
		t.Errorf("Expected retention.daily 14, got %d", retrieved.Spec.Retention.Daily)
	}
}

func TestConfigManager_UnregisterConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/test-configs")

	config := &BackupConfig{
		APIVersion: "v1",
		Kind:       "BackupConfig",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	err := cm.RegisterConfig(config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	// Unregister
	err = cm.UnregisterConfig("test-config")
	if err != nil {
		t.Fatalf("Failed to unregister config: %v", err)
	}

	// Verify removal
	_, exists := cm.GetConfig("test-config")
	if exists {
		t.Error("Config still exists after unregistration")
	}
}

func TestConfigManager_LoadAndSaveConfig(t *testing.T) {
	cm := NewConfigManager("/tmp/test-configs")

	config := &BackupConfig{
		APIVersion: "v1",
		Kind:       "BackupConfig",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			Version:   "1.0.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{
				{
					ID:      "db-001",
					Name:    "test-db",
					Type:    "postgres",
					Enabled: true,
					Connection: ConnectionConfig{
						Host:        "localhost",
						Port:        5432,
						Username:    "postgres",
						PasswordRef: "secret-001",
					},
				},
			},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 7},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Save config
	err := cm.SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loaded, err := cm.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.Metadata.Name != "test-config" {
		t.Errorf("Expected name 'test-config', got '%s'", loaded.Metadata.Name)
	}

	if len(loaded.Spec.Databases) != 1 {
		t.Errorf("Expected 1 database, got %d", len(loaded.Spec.Databases))
	}
}

// Test Validator

func TestValidator_ValidateAPIVersion(t *testing.T) {
	validator := NewValidator(false)

	tests := []struct {
		name      string
		config    *BackupConfig
		shouldErr bool
	}{
		{
			name: "valid v1",
			config: &BackupConfig{
				APIVersion: "v1",
				Metadata:   ConfigMetadata{Name: "test"},
				Spec: BackupSpec{
					Databases: []DatabaseConfig{
						{
							ID:   "db-001",
							Name: "test",
							Type: "postgres",
							Connection: ConnectionConfig{
								Host:        "localhost",
								Port:        5432,
								Username:    "user",
								PasswordRef: "secret",
							},
						},
					},
					Schedules: []ScheduleConfig{},
					Policies:  []PolicyConfig{},
					Retention: RetentionConfig{},
					Storage:   StorageConfig{Type: "local", Location: "/tmp"},
				},
			},
			shouldErr: false,
		},
		{
			name: "valid v1alpha1",
			config: &BackupConfig{
				APIVersion: "v1alpha1",
				Metadata:   ConfigMetadata{Name: "test"},
				Spec: BackupSpec{
					Databases: []DatabaseConfig{
						{
							ID:   "db-001",
							Name: "test",
							Type: "postgres",
							Connection: ConnectionConfig{
								Host:        "localhost",
								Port:        5432,
								Username:    "user",
								PasswordRef: "secret",
							},
						},
					},
					Schedules: []ScheduleConfig{},
					Policies:  []PolicyConfig{},
					Retention: RetentionConfig{},
					Storage:   StorageConfig{Type: "local", Location: "/tmp"},
				},
			},
			shouldErr: false,
		},
		{
			name: "invalid version",
			config: &BackupConfig{
				APIVersion: "invalid",
				Metadata:   ConfigMetadata{Name: "test"},
				Spec: BackupSpec{
					Databases: []DatabaseConfig{},
					Schedules: []ScheduleConfig{},
					Policies:  []PolicyConfig{},
					Retention: RetentionConfig{},
					Storage:   StorageConfig{Type: "local", Location: "/tmp"},
				},
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.config)
			hasError := !result.Valid

			if hasError != tt.shouldErr {
				t.Errorf("Expected error=%v, got error=%v", tt.shouldErr, hasError)
			}
		})
	}
}

func TestValidator_ValidateDatabases(t *testing.T) {
	validator := NewValidator(false)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata:   ConfigMetadata{Name: "test"},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{
				{
					ID:   "db-001",
					Name: "test-db",
					Type: "postgres",
					Connection: ConnectionConfig{
						Host:        "localhost",
						Port:        5432,
						Username:    "postgres",
						PasswordRef: "secret-001",
					},
				},
			},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	result := validator.Validate(config)
	if !result.Valid {
		t.Errorf("Valid database config failed validation: %v", result.Errors)
	}
}

func TestValidator_ValidateSchedules(t *testing.T) {
	validator := NewValidator(false)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata:   ConfigMetadata{Name: "test"},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{
				{
					ID:   "db-001",
					Name: "test-db",
					Type: "postgres",
					Connection: ConnectionConfig{
						Host:        "localhost",
						Port:        5432,
						Username:    "postgres",
						PasswordRef: "secret-001",
					},
				},
			},
			Schedules: []ScheduleConfig{
				{
					Name:           "daily-full",
					Type:           "full",
					CronExpression: "0 2 * * *",
					Databases:      []string{"db-001"},
					Enabled:        true,
				},
			},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	result := validator.Validate(config)
	if !result.Valid {
		t.Errorf("Valid schedule config failed validation: %v", result.Errors)
	}
}

func TestValidator_ValidateInvalidCron(t *testing.T) {
	validator := NewValidator(false)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata:   ConfigMetadata{Name: "test"},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{
				{
					ID:   "db-001",
					Name: "test",
					Type: "postgres",
					Connection: ConnectionConfig{
						Host:        "localhost",
						Port:        5432,
						Username:    "user",
						PasswordRef: "secret",
					},
				},
			},
			Schedules: []ScheduleConfig{
				{
					Name:           "invalid-cron",
					Type:           "full",
					CronExpression: "invalid cron",
					Databases:      []string{"db-001"},
					Enabled:        true,
				},
			},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	result := validator.Validate(config)
	if result.Valid {
		t.Error("Expected validation to fail for invalid cron expression")
	}
}

// Test VersionManager

func TestVersionManager_CreateVersion(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	version, err := vm.CreateVersion(config, "test-user", "Initial version")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	if version.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", version.Version)
	}

	if version.VersionNum != 1 {
		t.Errorf("Expected version number 1, got %d", version.VersionNum)
	}

	if version.CreatedBy != "test-user" {
		t.Errorf("Expected created by 'test-user', got '%s'", version.CreatedBy)
	}
}

func TestVersionManager_GetVersion(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	created, err := vm.CreateVersion(config, "test-user", "Initial version")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	retrieved, err := vm.GetVersion("test-config", created.Version)
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	if retrieved.Version != created.Version {
		t.Errorf("Expected version '%s', got '%s'", created.Version, retrieved.Version)
	}
}

func TestVersionManager_ListVersions(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	// Create multiple versions
	for i := 0; i < 3; i++ {
		_, err := vm.CreateVersion(config, "test-user", "Version update")
		if err != nil {
			t.Fatalf("Failed to create version %d: %v", i+1, err)
		}
	}

	versions, err := vm.ListVersions("test-config")
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
}

func TestVersionManager_DiffVersions(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config1 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 7},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	v1, err := vm.CreateVersion(config1, "test-user", "Version 1")
	if err != nil {
		t.Fatalf("Failed to create version 1: %v", err)
	}

	config2 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 14}, // Changed
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	v2, err := vm.CreateVersion(config2, "test-user", "Version 2")
	if err != nil {
		t.Fatalf("Failed to create version 2: %v", err)
	}

	diff, err := vm.DiffVersions("test-config", v1.Version, v2.Version)
	if err != nil {
		t.Fatalf("Failed to diff versions: %v", err)
	}

	if len(diff.Changes) == 0 {
		t.Error("Expected changes between versions")
	}
}

func TestVersionManager_CreateRollbackPlan(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config1 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 7},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	v1, _ := vm.CreateVersion(config1, "test-user", "Version 1")

	config2 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 14},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	vm.CreateVersion(config2, "test-user", "Version 2")

	plan, err := vm.CreateRollbackPlan("test-config", v1.Version)
	if err != nil {
		t.Fatalf("Failed to create rollback plan: %v", err)
	}

	if plan.TargetVersion != v1.Version {
		t.Errorf("Expected target version '%s', got '%s'", v1.Version, plan.TargetVersion)
	}

	if plan.ImpactAnalysis == nil {
		t.Error("Expected impact analysis to be present")
	}
}

func TestVersionManager_ExecuteRollback(t *testing.T) {
	tempDir := t.TempDir()
	vm := NewVersionManager(tempDir, 10, true)

	config1 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 7},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	v1, _ := vm.CreateVersion(config1, "test-user", "Version 1")

	config2 := &BackupConfig{
		APIVersion: "v1",
		Metadata: ConfigMetadata{
			Name:      "test-config",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Spec: BackupSpec{
			Databases: []DatabaseConfig{},
			Schedules: []ScheduleConfig{},
			Policies:  []PolicyConfig{},
			Retention: RetentionConfig{Daily: 14},
			Storage:   StorageConfig{Type: "local", Location: "/tmp"},
		},
	}

	vm.CreateVersion(config2, "test-user", "Version 2")

	plan, _ := vm.CreateRollbackPlan("test-config", v1.Version)

	rolledBack, err := vm.ExecuteRollback(plan)
	if err != nil {
		t.Fatalf("Failed to execute rollback: %v", err)
	}

	if rolledBack.Spec.Retention.Daily != 7 {
		t.Errorf("Expected retention.daily 7 after rollback, got %d", rolledBack.Spec.Retention.Daily)
	}

	// Check rollback annotation
	if rolledBack.Metadata.Annotations["rollback.to"] != v1.Version {
		t.Error("Expected rollback annotation to be set")
	}
}
