package multicloud

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Mock uploader for testing.
type mockUploader struct {
	shouldFail bool
	delay      time.Duration
}

func (m *mockUploader) Upload(ctx context.Context, destination *BackupDestination, data []byte, metadata map[string]interface{}) (string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.shouldFail {
		return "", fmt.Errorf("mock upload failed")
	}

	return fmt.Sprintf("%s://%s/%s/backup.tar.gz", destination.Provider, destination.Bucket, destination.Path), nil
}

func (m *mockUploader) Delete(ctx context.Context, destination *BackupDestination, location string) error {
	if m.shouldFail {
		return fmt.Errorf("mock delete failed")
	}
	return nil
}

func (m *mockUploader) Verify(ctx context.Context, destination *BackupDestination, location string) error {
	if m.shouldFail {
		return fmt.Errorf("mock verify failed")
	}
	return nil
}

// Mock downloader for testing.
type mockDownloader struct {
	shouldFail bool
}

func (m *mockDownloader) Download(ctx context.Context, provider StorageProvider, location string) (backupData []byte, meta map[string]interface{}, err error) {
	if m.shouldFail {
		return nil, nil, fmt.Errorf("mock download failed")
	}

	data := []byte("mock backup data")
	metadata := map[string]interface{}{
		"database": "test-db",
		"size":     len(data),
	}

	return data, metadata, nil
}

// Tests for MultiCloudOrchestrator

func TestNewMultiCloudOrchestrator(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)

	if mco == nil {
		t.Fatal("NewMultiCloudOrchestrator returned nil")
	}

	if mco.maxConcurrent != 10 {
		t.Errorf("Expected maxConcurrent=10, got %d", mco.maxConcurrent)
	}

	if len(mco.destinations) != 0 {
		t.Errorf("Expected 0 destinations, got %d", len(mco.destinations))
	}
}

func TestRegisterDestination(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)

	dest := &BackupDestination{
		Provider: ProviderAWS,
		Region:   "us-east-1",
		Bucket:   "my-bucket",
		Path:     "backups",
		Priority: 1,
		Enabled:  true,
	}

	err := mco.RegisterDestination("aws-dest-1", dest)
	if err != nil {
		t.Fatalf("RegisterDestination failed: %v", err)
	}

	// Verify registration
	retrieved, err := mco.GetDestination("aws-dest-1")
	if err != nil {
		t.Fatalf("GetDestination failed: %v", err)
	}

	if retrieved.Provider != ProviderAWS {
		t.Errorf("Expected provider AWS, got %s", retrieved.Provider)
	}

	if retrieved.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", retrieved.Region)
	}
}

func TestExecuteBackup(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)

	// Register uploader
	uploader := &mockUploader{shouldFail: false}
	mco.RegisterUploader(ProviderAWS, uploader)
	mco.RegisterUploader(ProviderAzure, uploader)

	// Register destinations
	awsDest := &BackupDestination{
		Provider: ProviderAWS,
		Region:   "us-east-1",
		Bucket:   "aws-bucket",
		Path:     "backups",
		Enabled:  true,
	}

	azureDest := &BackupDestination{
		Provider: ProviderAzure,
		Region:   "eastus",
		Bucket:   "azure-container",
		Path:     "backups",
		Enabled:  true,
	}

	// Create backup job
	job := &BackupJob{
		ID:            "job-001",
		DatabaseName:  "test-db",
		BackupData:    []byte("test backup data"),
		Metadata:      map[string]interface{}{"test": "metadata"},
		Destinations:  []*BackupDestination{awsDest, azureDest},
		RequireAll:    false,
		MinSuccessful: 1,
		CreatedAt:     time.Now(),
	}

	// Execute backup
	ctx := context.Background()
	results, err := mco.ExecuteBackup(ctx, job)
	if err != nil {
		t.Fatalf("ExecuteBackup failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check results
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	if successCount != 2 {
		t.Errorf("Expected 2 successful uploads, got %d", successCount)
	}
}

func TestExecuteBackupWithFailure(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)

	// Register failing uploader
	uploader := &mockUploader{shouldFail: true}
	mco.RegisterUploader(ProviderAWS, uploader)

	awsDest := &BackupDestination{
		Provider:   ProviderAWS,
		Region:     "us-east-1",
		Bucket:     "aws-bucket",
		Path:       "backups",
		Enabled:    true,
		MaxRetries: 1, // Limit retries for faster test
	}

	job := &BackupJob{
		ID:            "job-002",
		DatabaseName:  "test-db",
		BackupData:    []byte("test data"),
		Destinations:  []*BackupDestination{awsDest},
		RequireAll:    true,
		MinSuccessful: 1,
		CreatedAt:     time.Now(),
	}

	ctx := context.Background()
	results, err := mco.ExecuteBackup(ctx, job)

	// Should fail because all destinations are required
	if err == nil {
		t.Error("Expected error when all destinations required but upload failed")
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("Expected upload to fail")
	}
}

// Tests for CloudSelector

func TestNewCloudSelector(t *testing.T) {
	cs := NewCloudSelector()

	if cs == nil {
		t.Fatal("NewCloudSelector returned nil")
	}

	if len(cs.metrics) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(cs.metrics))
	}
}

func TestRegisterProviderMetrics(t *testing.T) {
	cs := NewCloudSelector()

	metrics := &ProviderMetrics{
		Provider:             ProviderAWS,
		Region:               "us-east-1",
		CostPerGB:            0.023,
		UploadSpeed:          100.0,
		DownloadSpeed:        150.0,
		Availability:         99.99,
		Latency:              50.0,
		SupportedRegions:     []string{"us-east-1", "us-west-2"},
		ComplianceCerts:      []string{"SOC2", "HIPAA"},
		DataResidencyCountry: "US",
		PerformanceScore:     85.0,
		LastUpdated:          time.Now(),
	}

	err := cs.RegisterProviderMetrics(metrics)
	if err != nil {
		t.Fatalf("RegisterProviderMetrics failed: %v", err)
	}
}

func TestSelectProvidersCost(t *testing.T) {
	cs := NewCloudSelector()

	// Register AWS metrics (cheaper)
	awsMetrics := &ProviderMetrics{
		Provider:         ProviderAWS,
		Region:           "us-east-1",
		CostPerGB:        0.023,
		PerformanceScore: 80.0,
		Availability:     99.99,
	}
	cs.RegisterProviderMetrics(awsMetrics)

	// Register Azure metrics (more expensive)
	azureMetrics := &ProviderMetrics{
		Provider:         ProviderAzure,
		Region:           "eastus",
		CostPerGB:        0.030,
		PerformanceScore: 85.0,
		Availability:     99.95,
	}
	cs.RegisterProviderMetrics(azureMetrics)

	// Select with cost strategy
	criteria := &SelectionCriteria{
		Strategy: StrategyCost,
	}

	ctx := context.Background()
	selections, err := cs.SelectProviders(ctx, criteria)
	if err != nil {
		t.Fatalf("SelectProviders failed: %v", err)
	}

	if len(selections) != 2 {
		t.Errorf("Expected 2 selections, got %d", len(selections))
	}

	// AWS should be first (cheaper)
	if selections[0].Provider != ProviderAWS {
		t.Errorf("Expected AWS as best cost option, got %s", selections[0].Provider)
	}
}

func TestSelectProvidersPerformance(t *testing.T) {
	cs := NewCloudSelector()

	// Register AWS metrics (lower performance)
	awsMetrics := &ProviderMetrics{
		Provider:         ProviderAWS,
		Region:           "us-east-1",
		CostPerGB:        0.023,
		UploadSpeed:      100.0,
		DownloadSpeed:    150.0,
		PerformanceScore: 75.0,
		Availability:     99.99,
		Latency:          60.0,
	}
	cs.RegisterProviderMetrics(awsMetrics)

	// Register Azure metrics (higher performance)
	azureMetrics := &ProviderMetrics{
		Provider:         ProviderAzure,
		Region:           "eastus",
		CostPerGB:        0.030,
		UploadSpeed:      200.0,
		DownloadSpeed:    250.0,
		PerformanceScore: 90.0,
		Availability:     99.99,
		Latency:          40.0,
	}
	cs.RegisterProviderMetrics(azureMetrics)

	// Select with performance strategy
	criteria := &SelectionCriteria{
		Strategy: StrategyPerformance,
	}

	ctx := context.Background()
	selections, err := cs.SelectProviders(ctx, criteria)
	if err != nil {
		t.Fatalf("SelectProviders failed: %v", err)
	}

	// Azure should be first (better performance)
	if selections[0].Provider != ProviderAzure {
		t.Errorf("Expected Azure as best performance option, got %s", selections[0].Provider)
	}
}

func TestSelectProvidersCompliance(t *testing.T) {
	cs := NewCloudSelector()

	// Register AWS metrics (more certifications)
	awsMetrics := &ProviderMetrics{
		Provider:             ProviderAWS,
		Region:               "us-east-1",
		CostPerGB:            0.023,
		PerformanceScore:     80.0,
		ComplianceCerts:      []string{"SOC2", "HIPAA", "ISO27001"},
		DataResidencyCountry: "US",
	}
	cs.RegisterProviderMetrics(awsMetrics)

	// Register Azure metrics (fewer certifications)
	azureMetrics := &ProviderMetrics{
		Provider:             ProviderAzure,
		Region:               "eastus",
		CostPerGB:            0.030,
		PerformanceScore:     85.0,
		ComplianceCerts:      []string{"SOC2"},
		DataResidencyCountry: "US",
	}
	cs.RegisterProviderMetrics(azureMetrics)

	// Select with compliance strategy
	criteria := &SelectionCriteria{
		Strategy: StrategyCompliance,
	}

	ctx := context.Background()
	selections, err := cs.SelectProviders(ctx, criteria)
	if err != nil {
		t.Fatalf("SelectProviders failed: %v", err)
	}

	// AWS should be first (more certifications)
	if selections[0].Provider != ProviderAWS {
		t.Errorf("Expected AWS as best compliance option, got %s", selections[0].Provider)
	}
}

func TestSelectProvidersWithFilters(t *testing.T) {
	cs := NewCloudSelector()

	// Register multiple providers
	awsMetrics := &ProviderMetrics{
		Provider:         ProviderAWS,
		Region:           "us-east-1",
		CostPerGB:        0.023,
		PerformanceScore: 80.0,
		ComplianceCerts:  []string{"SOC2"},
	}
	cs.RegisterProviderMetrics(awsMetrics)

	azureMetrics := &ProviderMetrics{
		Provider:         ProviderAzure,
		Region:           "eastus",
		CostPerGB:        0.030,
		PerformanceScore: 85.0,
		ComplianceCerts:  []string{"SOC2", "HIPAA"},
	}
	cs.RegisterProviderMetrics(azureMetrics)

	// Select with filters
	criteria := &SelectionCriteria{
		Strategy:              StrategyBalanced,
		MaxCost:               0.025, // Exclude Azure
		RequireCertifications: []string{"SOC2"},
	}

	ctx := context.Background()
	selections, err := cs.SelectProviders(ctx, criteria)
	if err != nil {
		t.Fatalf("SelectProviders failed: %v", err)
	}

	if len(selections) != 1 {
		t.Errorf("Expected 1 selection (Azure filtered out), got %d", len(selections))
	}

	if selections[0].Provider != ProviderAWS {
		t.Errorf("Expected AWS, got %s", selections[0].Provider)
	}
}

// Tests for MigrationManager

func TestNewMigrationManager(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	downloader := &mockDownloader{shouldFail: false}
	mm := NewMigrationManager(mco, downloader, 3)

	if mm == nil {
		t.Fatal("NewMigrationManager returned nil")
	}

	if mm.maxConcurrent != 3 {
		t.Errorf("Expected maxConcurrent=3, got %d", mm.maxConcurrent)
	}
}

func TestCreateMigration(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	downloader := &mockDownloader{shouldFail: false}
	mm := NewMigrationManager(mco, downloader, 3)

	targetDest := &BackupDestination{
		Provider: ProviderAzure,
		Region:   "eastus",
		Bucket:   "azure-container",
		Path:     "backups",
		Enabled:  true,
	}

	job := &MigrationJob{
		ID:                 "migration-001",
		BackupID:           "backup-001",
		SourceProvider:     ProviderAWS,
		SourceLocation:     "s3://bucket/backup.tar.gz",
		TargetDestinations: []*BackupDestination{targetDest},
		DeleteSource:       false,
		VerifyIntegrity:    true,
	}

	err := mm.CreateMigration(job)
	if err != nil {
		t.Fatalf("CreateMigration failed: %v", err)
	}

	// Verify creation
	retrieved, err := mm.GetMigration("migration-001")
	if err != nil {
		t.Fatalf("GetMigration failed: %v", err)
	}

	if retrieved.Status != MigrationPending {
		t.Errorf("Expected status pending, got %s", retrieved.Status)
	}
}

func TestExecuteMigration(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	downloader := &mockDownloader{shouldFail: false}
	mm := NewMigrationManager(mco, downloader, 3)

	// Register uploader
	uploader := &mockUploader{shouldFail: false}
	mco.RegisterUploader(ProviderAzure, uploader)

	targetDest := &BackupDestination{
		Provider: ProviderAzure,
		Region:   "eastus",
		Bucket:   "azure-container",
		Path:     "backups",
		Enabled:  true,
	}

	job := &MigrationJob{
		ID:                 "migration-002",
		BackupID:           "backup-002",
		SourceProvider:     ProviderAWS,
		SourceLocation:     "s3://bucket/backup.tar.gz",
		TargetDestinations: []*BackupDestination{targetDest},
		DeleteSource:       false,
		VerifyIntegrity:    false, // Skip verification for faster test
	}

	mm.CreateMigration(job)

	ctx := context.Background()
	err := mm.ExecuteMigration(ctx, "migration-002")
	if err != nil {
		t.Fatalf("ExecuteMigration failed: %v", err)
	}

	// Check status
	retrieved, err := mm.GetMigration("migration-002")
	if err != nil {
		t.Fatalf("GetMigration failed: %v", err)
	}

	if retrieved.Status != MigrationCompleted {
		t.Errorf("Expected status completed, got %s", retrieved.Status)
	}

	if retrieved.Progress != 100 {
		t.Errorf("Expected progress 100, got %.2f", retrieved.Progress)
	}
}

// Tests for FailoverManager

func TestNewFailoverManager(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	cs := NewCloudSelector()
	fm := NewFailoverManager(mco, cs)

	if fm == nil {
		t.Fatal("NewFailoverManager returned nil")
	}
}

func TestRegisterHealthCheck(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	cs := NewCloudSelector()
	fm := NewFailoverManager(mco, cs)

	config := &HealthCheckConfig{
		Provider: ProviderAWS,
		Interval: 60 * time.Second,
		Timeout:  10 * time.Second,
	}

	err := fm.RegisterHealthCheck(config)
	if err != nil {
		t.Fatalf("RegisterHealthCheck failed: %v", err)
	}

	// Check defaults
	if config.FailureThreshold != 3 {
		t.Errorf("Expected FailureThreshold=3, got %d", config.FailureThreshold)
	}

	if config.SuccessThreshold != 2 {
		t.Errorf("Expected SuccessThreshold=2, got %d", config.SuccessThreshold)
	}

	if !config.Healthy {
		t.Error("Expected Healthy=true by default")
	}
}

func TestAddFailoverRule(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	cs := NewCloudSelector()
	fm := NewFailoverManager(mco, cs)

	rule := &FailoverRule{
		ID:              "rule-001",
		Name:            "AWS to Azure failover",
		TriggerProvider: ProviderAWS,
		TargetProviders: []StorageProvider{ProviderAzure, ProviderGCP},
		AutoFailback:    true,
		Enabled:         true,
	}

	err := fm.AddFailoverRule(rule)
	if err != nil {
		t.Fatalf("AddFailoverRule failed: %v", err)
	}
}

func TestGetHealthStatus(t *testing.T) {
	mco := NewMultiCloudOrchestrator(10)
	cs := NewCloudSelector()
	fm := NewFailoverManager(mco, cs)

	// Register health checks
	awsConfig := &HealthCheckConfig{
		Provider: ProviderAWS,
	}
	azureConfig := &HealthCheckConfig{
		Provider: ProviderAzure,
	}

	fm.RegisterHealthCheck(awsConfig)
	fm.RegisterHealthCheck(azureConfig)

	status := fm.GetHealthStatus()

	if len(status) != 2 {
		t.Errorf("Expected 2 health checks, got %d", len(status))
	}

	if status[ProviderAWS] == nil {
		t.Error("AWS health check not found")
	}

	if status[ProviderAzure] == nil {
		t.Error("Azure health check not found")
	}
}

// Benchmark tests

func BenchmarkExecuteBackup(b *testing.B) {
	mco := NewMultiCloudOrchestrator(10)
	uploader := &mockUploader{shouldFail: false, delay: 10 * time.Millisecond}
	mco.RegisterUploader(ProviderAWS, uploader)

	dest := &BackupDestination{
		Provider: ProviderAWS,
		Region:   "us-east-1",
		Bucket:   "test-bucket",
		Path:     "backups",
		Enabled:  true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := &BackupJob{
			ID:            fmt.Sprintf("job-%d", i),
			DatabaseName:  "test-db",
			BackupData:    []byte("test data"),
			Destinations:  []*BackupDestination{dest},
			MinSuccessful: 1,
			CreatedAt:     time.Now(),
		}

		_, _ = mco.ExecuteBackup(ctx, job)
	}
}

func BenchmarkSelectProviders(b *testing.B) {
	cs := NewCloudSelector()

	// Register multiple providers
	providers := []StorageProvider{ProviderAWS, ProviderAzure, ProviderGCP}
	regions := []string{"us-east-1", "eastus", "us-central1"}

	for i, provider := range providers {
		metrics := &ProviderMetrics{
			Provider:         provider,
			Region:           regions[i],
			CostPerGB:        0.020 + float64(i)*0.005,
			PerformanceScore: 80.0 + float64(i)*5.0,
			Availability:     99.9 + float64(i)*0.05,
		}
		cs.RegisterProviderMetrics(metrics)
	}

	criteria := &SelectionCriteria{
		Strategy: StrategyBalanced,
		Weights:  DefaultSelectionWeights(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cs.SelectProviders(ctx, criteria)
	}
}
