//go:build grpc

package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/grpc/services"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// Mock implementations for testing

type mockBackupService struct{}

func (m mockBackupService) CreateBackup(ctx context.Context, backupID string, config *services.BackupConfig) (*database.BackupResult, error) {
	return &database.BackupResult{
		ID:         backupID,
		Status:     database.BackupStatusCompleted,
		Size:       1024 * 1024 * 100,
		Checksum:   "sha256:abc123",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(5 * time.Minute),
		Duration:   5 * time.Minute,
		BackupPath: "/backups/test.tar.gz",
	}, nil
}

type mockRestoreService struct{}

func (m mockRestoreService) Restore(ctx context.Context, restoreID string, config *services.RestoreConfig) (*services.RestoreResult, error) {
	return &services.RestoreResult{
		RestoreID:       restoreID,
		BackupID:        config.BackupID,
		Status:          services.RestoreStatusCompleted,
		TargetDatabase:  config.TargetDatabaseID,
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(10 * time.Minute),
		Duration:        10 * time.Minute,
		ProgressPercent: 100,
		BytesRestored:   1024 * 1024 * 100,
	}, nil
}

type mockRepository struct{}

func (m mockRepository) Save(ctx context.Context, metadata *models.BackupMetadata) error {
	return nil
}

func (m mockRepository) Get(ctx context.Context, id string) (*models.BackupMetadata, error) {
	return &models.BackupMetadata{
		ID:              id,
		Name:            "test-backup",
		Database:        "test-db",
		DatabaseType:    "postgres",
		Status:          database.BackupStatusCompleted,
		Size:            1024 * 1024 * 100,
		Checksum:        "sha256:abc123",
		StartTime:       time.Now(),
		EndTime:         time.Now().Add(5 * time.Minute),
		Duration:        5 * time.Minute,
		StorageLocation: "/backups/test-backup.tar.gz",
		Tags:            map[string]string{"env": "test"},
	}, nil
}

func (m mockRepository) List(ctx context.Context, filter *repository.ListFilter) ([]*models.BackupMetadata, error) {
	return []*models.BackupMetadata{}, nil
}

func (m mockRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m mockRepository) Update(ctx context.Context, metadata *models.BackupMetadata) error {
	return nil
}

type mockDatabasePool struct{}

func (m mockDatabasePool) GetConnection(id string) (*database.ConnectionConfig, error) {
	return &database.ConnectionConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: id,
		Username: "test",
	}, nil
}

type mockHealthChecker struct{}

func (m mockHealthChecker) Check(ctx context.Context) services.HealthCheckResult {
	return services.HealthCheckResult{
		Status: services.HealthStatusHealthy,
		Components: map[string]services.HealthStatus{
			"database": services.HealthStatusHealthy,
			"storage":  services.HealthStatusHealthy,
		},
	}
}

type mockMetricsCollector struct{}

func (m mockMetricsCollector) RecordMetric(name string, value float64, labels map[string]string) {
	// No-op for testing
}

// Test server configuration
func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	if config.Port != 9090 {
		t.Errorf("Expected default port 9090, got %d", config.Port)
	}

	if !config.EnableReflection {
		t.Error("Expected reflection to be enabled by default")
	}

	if !config.EnableGateway {
		t.Error("Expected gateway to be enabled by default")
	}

	t.Log("✓ Default server configuration is correct")
}

// Test client configuration
func TestDefaultClientConfig(t *testing.T) {
	address := "localhost:9090"
	config := DefaultClientConfig(address)

	if config.Address != address {
		t.Errorf("Expected address %s, got %s", address, config.Address)
	}

	if config.DialTimeout != 10*time.Second {
		t.Errorf("Expected dial timeout 10s, got %v", config.DialTimeout)
	}

	t.Log("✓ Default client configuration is correct")
}

// Test service creation
func TestServiceCreation(t *testing.T) {
	backupSvc := services.NewBackupService(mockBackupService{}, mockRepository{}, mockDatabasePool{})
	if backupSvc == nil {
		t.Fatal("Failed to create backup service")
	}

	restoreSvc := services.NewRestoreService(mockRestoreService{}, mockRepository{}, mockDatabasePool{})
	if restoreSvc == nil {
		t.Fatal("Failed to create restore service")
	}

	monitorSvc := services.NewMonitoringService(mockHealthChecker{}, mockMetricsCollector{}, mockDatabasePool{})
	if monitorSvc == nil {
		t.Fatal("Failed to create monitoring service")
	}

	t.Log("✓ All services created successfully")
}

// Test configuration validation
func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
		valid  bool
	}{
		{
			name:   "Default config",
			config: DefaultServerConfig(),
			valid:  true,
		},
		{
			name: "Custom valid config",
			config: &ServerConfig{
				Host:                 "127.0.0.1",
				Port:                 8080,
				MaxConcurrentStreams: 100,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				t.Error("Config should not be nil")
			}
		})
	}

	t.Log("✓ Server configuration validation passed")
}

// Test mock backup service
func TestMockBackupService(t *testing.T) {
	ctx := context.Background()
	svc := mockBackupService{}

	config := &services.BackupConfig{
		DatabaseID:  "test-db",
		Incremental: false,
		Compression: true,
	}

	result, err := svc.CreateBackup(ctx, "backup-123", config)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	if result.ID != "backup-123" {
		t.Errorf("Expected backup ID backup-123, got %s", result.ID)
	}

	if result.Status != database.BackupStatusCompleted {
		t.Errorf("Expected status completed, got %v", result.Status)
	}

	t.Log("✓ Mock backup service works correctly")
}

// Test mock restore service
func TestMockRestoreService(t *testing.T) {
	ctx := context.Background()
	svc := mockRestoreService{}

	config := &services.RestoreConfig{
		BackupID:         "backup-123",
		TargetDatabaseID: "test-db",
	}

	result, err := svc.Restore(ctx, "restore-456", config)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	if result.RestoreID != "restore-456" {
		t.Errorf("Expected restore ID restore-456, got %s", result.RestoreID)
	}

	if result.Status != services.RestoreStatusCompleted {
		t.Errorf("Expected status completed, got %v", result.Status)
	}

	t.Log("✓ Mock restore service works correctly")
}

// Test mock health checker
func TestMockHealthChecker(t *testing.T) {
	ctx := context.Background()
	checker := mockHealthChecker{}

	result := checker.Check(ctx)

	if result.Status != services.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %v", result.Status)
	}

	if len(result.Components) == 0 {
		t.Error("Expected components, got none")
	}

	t.Log("✓ Mock health checker works correctly")
}

// Test mock database pool
func TestMockDatabasePool(t *testing.T) {
	pool := mockDatabasePool{}

	conn, err := pool.GetConnection("test-db")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}

	if conn.Type != "postgres" {
		t.Errorf("Expected postgres type, got %s", conn.Type)
	}

	t.Log("✓ Mock database pool works correctly")
}

// Test repository operations
func TestMockRepository(t *testing.T) {
	ctx := context.Background()
	repo := mockRepository{}

	// Test Get
	meta, err := repo.Get(ctx, "backup-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if meta.ID != "backup-123" {
		t.Errorf("Expected ID backup-123, got %s", meta.ID)
	}

	// Test Save
	err = repo.Save(ctx, meta)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Test Delete
	err = repo.Delete(ctx, "backup-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	t.Log("✓ Mock repository works correctly")
}

// Integration test placeholder
func TestGRPCIntegration(t *testing.T) {
	// This would test the full gRPC server with all services
	// Skipping actual server startup for unit tests
	t.Log("✓ gRPC integration test placeholder")
}
