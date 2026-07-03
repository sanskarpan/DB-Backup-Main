package multicloud

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// fakeHealthChecker is a test double implementing HealthChecker.
type fakeHealthChecker struct {
	err error
}

func (f *fakeHealthChecker) Probe(ctx context.Context) error {
	return f.err
}

// fakeStorageProvider is a minimal storage.Provider test double whose Exists
// behavior is controlled by existsErr.
type fakeStorageProvider struct {
	existsErr error
	exists    bool
}

func (f *fakeStorageProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	return nil
}

func (f *fakeStorageProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	return nil
}

func (f *fakeStorageProvider) Download(ctx context.Context, remotePath, localPath string) error {
	return nil
}

func (f *fakeStorageProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (f *fakeStorageProvider) Delete(ctx context.Context, remotePath string) error {
	return nil
}

func (f *fakeStorageProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	return f.exists, f.existsErr
}

func (f *fakeStorageProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	return &storage.FileMetadata{}, nil
}

func (f *fakeStorageProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	return nil, nil
}

func (f *fakeStorageProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeLocal
}

func (f *fakeStorageProvider) ValidateConfig() error {
	return nil
}

func TestStorageHealthCheckerHealthy(t *testing.T) {
	checker := NewStorageHealthChecker(&fakeStorageProvider{exists: false}, "")
	if err := checker.Probe(context.Background()); err != nil {
		t.Fatalf("expected healthy probe, got error: %v", err)
	}
}

func TestStorageHealthCheckerUnhealthy(t *testing.T) {
	wantErr := errors.New("connection refused")
	checker := NewStorageHealthChecker(&fakeStorageProvider{existsErr: wantErr}, "probe")
	err := checker.Probe(context.Background())
	if err == nil {
		t.Fatal("expected unhealthy probe to return an error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped transport error, got %v", err)
	}
}

func TestStorageHealthCheckerNilProvider(t *testing.T) {
	checker := NewStorageHealthChecker(nil, "probe")
	if err := checker.Probe(context.Background()); err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestPerformHealthCheckHealthy(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	config := &HealthCheckConfig{
		Provider:         ProviderAWS,
		Checker:          NewStorageHealthChecker(&fakeStorageProvider{exists: false}, "probe"),
		FailureThreshold: 1,
	}
	if err := fm.RegisterHealthCheck(config); err != nil {
		t.Fatalf("RegisterHealthCheck failed: %v", err)
	}

	fm.performHealthCheck(context.Background(), config)

	if !config.Healthy {
		t.Errorf("expected provider to remain healthy, got unhealthy (err=%v)", config.LastError)
	}
	if config.LastError != nil {
		t.Errorf("expected no last error, got %v", config.LastError)
	}
	if config.LastCheck.IsZero() {
		t.Error("expected LastCheck to be recorded")
	}
}

func TestPerformHealthCheckDetectsUnhealthy(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	probeErr := errors.New("backend unreachable")
	config := &HealthCheckConfig{
		Provider:         ProviderAWS,
		Checker:          &fakeHealthChecker{err: probeErr},
		FailureThreshold: 1,
	}
	if err := fm.RegisterHealthCheck(config); err != nil {
		t.Fatalf("RegisterHealthCheck failed: %v", err)
	}
	fm.SetActiveProvider(ProviderAWS)

	fm.performHealthCheck(context.Background(), config)

	if config.Healthy {
		t.Error("expected provider to be detected as unhealthy")
	}
	if !errors.Is(config.LastError, probeErr) {
		t.Errorf("expected recorded probe error, got %v", config.LastError)
	}
}

func TestTriggerFailoverSwitchesActiveProvider(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	// AWS is the active/primary and has gone unhealthy.
	awsConfig := &HealthCheckConfig{Provider: ProviderAWS, Checker: &fakeHealthChecker{err: errors.New("down")}}
	azureConfig := &HealthCheckConfig{Provider: ProviderAzure, Checker: &fakeHealthChecker{}}
	if err := fm.RegisterHealthCheck(awsConfig); err != nil {
		t.Fatalf("RegisterHealthCheck failed: %v", err)
	}
	if err := fm.RegisterHealthCheck(azureConfig); err != nil {
		t.Fatalf("RegisterHealthCheck failed: %v", err)
	}
	awsConfig.Healthy = false
	fm.SetActiveProvider(ProviderAWS)

	rule := &FailoverRule{
		ID:              "rule-aws-azure",
		TriggerProvider: ProviderAWS,
		TargetProviders: []StorageProvider{ProviderAzure},
		Enabled:         true,
	}
	if err := fm.AddFailoverRule(rule); err != nil {
		t.Fatalf("AddFailoverRule failed: %v", err)
	}

	event, err := fm.triggerFailover(context.Background(), ProviderAWS, "primary down")
	if err != nil {
		t.Fatalf("triggerFailover returned error: %v", err)
	}
	if !event.Success {
		t.Error("expected failover event to be successful")
	}
	if event.TargetProvider != ProviderAzure {
		t.Errorf("expected failover target Azure, got %s", event.TargetProvider)
	}
	if event.RuleID != "rule-aws-azure" {
		t.Errorf("expected rule id recorded, got %q", event.RuleID)
	}
	if got := fm.GetActiveProvider(); got != ProviderAzure {
		t.Errorf("expected active provider re-routed to Azure, got %s", got)
	}

	events := fm.GetFailoverEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 recorded failover event, got %d", len(events))
	}
}

func TestTriggerFailoverFallbackNoRule(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	awsConfig := &HealthCheckConfig{Provider: ProviderAWS, Checker: &fakeHealthChecker{err: errors.New("down")}}
	gcpConfig := &HealthCheckConfig{Provider: ProviderGCP, Checker: &fakeHealthChecker{}}
	_ = fm.RegisterHealthCheck(awsConfig)
	_ = fm.RegisterHealthCheck(gcpConfig)
	awsConfig.Healthy = false
	fm.SetActiveProvider(ProviderAWS)

	event, err := fm.triggerFailover(context.Background(), ProviderAWS, "no rule failover")
	if err != nil {
		t.Fatalf("expected fallback failover to succeed, got %v", err)
	}
	if event.TargetProvider != ProviderGCP {
		t.Errorf("expected fallback target GCP, got %s", event.TargetProvider)
	}
	if got := fm.GetActiveProvider(); got != ProviderGCP {
		t.Errorf("expected active provider GCP, got %s", got)
	}
}

func TestTriggerFailoverNoHealthyTarget(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	awsConfig := &HealthCheckConfig{Provider: ProviderAWS}
	azureConfig := &HealthCheckConfig{Provider: ProviderAzure}
	_ = fm.RegisterHealthCheck(awsConfig)
	_ = fm.RegisterHealthCheck(azureConfig)
	// Both destinations are unhealthy.
	awsConfig.Healthy = false
	azureConfig.Healthy = false
	fm.SetActiveProvider(ProviderAWS)

	event, err := fm.triggerFailover(context.Background(), ProviderAWS, "everything down")
	if err == nil {
		t.Fatal("expected an honest error when no healthy target exists")
	}
	if event.Success {
		t.Error("expected failover event to be marked unsuccessful")
	}
	if event.Error == nil {
		t.Error("expected event error to be recorded")
	}
	if got := fm.GetActiveProvider(); got != ProviderAWS {
		t.Errorf("expected active provider to remain AWS, got %s", got)
	}
}

func TestSelectHealthyTargetSkipsFailingProbe(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())

	// Marked healthy by flag, but its on-demand probe now fails.
	staleConfig := &HealthCheckConfig{Provider: ProviderAzure, Checker: &fakeHealthChecker{err: errors.New("just failed")}}
	goodConfig := &HealthCheckConfig{Provider: ProviderGCP, Checker: &fakeHealthChecker{}}
	_ = fm.RegisterHealthCheck(staleConfig)
	_ = fm.RegisterHealthCheck(goodConfig)
	fm.SetActiveProvider(ProviderAWS)

	event, err := fm.triggerFailover(context.Background(), ProviderAWS, "primary down")
	if err != nil {
		t.Fatalf("expected failover to succeed via probe-verified target, got %v", err)
	}
	if event.TargetProvider != ProviderGCP {
		t.Errorf("expected probe-verified target GCP, got %s", event.TargetProvider)
	}
}

func TestPerformHealthCheckLatencyRecorded(t *testing.T) {
	fm := NewFailoverManager(NewMultiCloudOrchestrator(1), NewCloudSelector())
	config := &HealthCheckConfig{
		Provider: ProviderAWS,
		Checker:  &fakeHealthChecker{},
		Timeout:  time.Second,
	}
	_ = fm.RegisterHealthCheck(config)

	fm.performHealthCheck(context.Background(), config)
	if config.LastLatency < 0 {
		t.Errorf("expected non-negative latency, got %v", config.LastLatency)
	}
}
