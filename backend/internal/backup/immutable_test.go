package backup

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	stor "github.com/sanskarpan/db-backup/internal/storage"
)

// errNotImplemented is a sentinel returned by fake providers for methods that
// are not exercised by these tests.
var errNotImplemented = errors.New("not implemented")

// fakeImmutableProvider implements storage.Provider + storage.ImmutableProvider
// and records the object-lock calls it receives so tests can assert on them.
type fakeImmutableProvider struct {
	// recorded retention / legal-hold state keyed by remote path
	retUntil map[string]time.Time
	retMode  map[string]string
	holds    map[string]bool

	setRetentionCalls int
	setLegalHoldCalls int
}

func newFakeImmutableProvider() *fakeImmutableProvider {
	return &fakeImmutableProvider{
		retUntil: make(map[string]time.Time),
		retMode:  make(map[string]string),
		holds:    make(map[string]bool),
	}
}

// --- storage.Provider ---

func (f *fakeImmutableProvider) Upload(ctx context.Context, localPath, remotePath string, opts *stor.UploadOptions) error {
	return nil
}

func (f *fakeImmutableProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *stor.UploadOptions) error {
	return nil
}

func (f *fakeImmutableProvider) Download(ctx context.Context, remotePath, localPath string) error {
	return nil
}

func (f *fakeImmutableProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (f *fakeImmutableProvider) Delete(ctx context.Context, remotePath string) error { return nil }

func (f *fakeImmutableProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	return true, nil
}

func (f *fakeImmutableProvider) GetMetadata(ctx context.Context, remotePath string) (*stor.FileMetadata, error) {
	return &stor.FileMetadata{Path: remotePath}, nil
}

func (f *fakeImmutableProvider) List(ctx context.Context, prefix string) ([]*stor.FileMetadata, error) {
	return nil, nil
}

func (f *fakeImmutableProvider) GetType() stor.ProviderType { return stor.ProviderType("fake") }

func (f *fakeImmutableProvider) ValidateConfig() error { return nil }

// --- storage.ImmutableProvider ---

func (f *fakeImmutableProvider) SetRetention(ctx context.Context, remotePath string, until time.Time, mode string) error {
	f.setRetentionCalls++
	f.retUntil[remotePath] = until
	f.retMode[remotePath] = mode
	return nil
}

func (f *fakeImmutableProvider) GetRetention(ctx context.Context, remotePath string) (time.Time, string, error) {
	until, ok := f.retUntil[remotePath]
	if !ok {
		return time.Time{}, "", stor.ErrNoRetention
	}
	return until, f.retMode[remotePath], nil
}

func (f *fakeImmutableProvider) SetLegalHold(ctx context.Context, remotePath string, on bool) error {
	f.setLegalHoldCalls++
	f.holds[remotePath] = on
	return nil
}

func (f *fakeImmutableProvider) GetLegalHold(ctx context.Context, remotePath string) (bool, error) {
	return f.holds[remotePath], nil
}

// TestCreateBackup_ImmutableAppliesRetentionAndLegalHold proves that requesting
// immutability applies SetRetention (and SetLegalHold) with the right args and
// records the protection on the metadata.
func TestCreateBackup_ImmutableAppliesRetentionAndLegalHold(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	provider := newFakeImmutableProvider()
	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	until := time.Now().Add(72 * time.Hour)
	metadata, err := engine.CreateBackup(context.Background(), &CreateOptions{
		DatabaseType:   database.DatabaseTypeSQLite,
		Database:       dbPath,
		Immutable:      true,
		RetentionUntil: until,
		LockMode:       stor.LockModeCompliance,
		LegalHold:      true,
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if metadata.Status != database.BackupStatusSuccess {
		t.Fatalf("expected success, got %s", metadata.Status)
	}

	// Object-lock calls must have happened exactly once each.
	if provider.setRetentionCalls != 1 {
		t.Errorf("expected 1 SetRetention call, got %d", provider.setRetentionCalls)
	}
	if provider.setLegalHoldCalls != 1 {
		t.Errorf("expected 1 SetLegalHold call, got %d", provider.setLegalHoldCalls)
	}

	remotePath, ok := parseRemotePath(metadata.StorageLocation)
	if !ok {
		t.Fatalf("expected remote storage location, got %q", metadata.StorageLocation)
	}
	if got := provider.retUntil[remotePath]; !got.Equal(until) {
		t.Errorf("retention until = %v, want %v", got, until)
	}
	if got := provider.retMode[remotePath]; got != stor.LockModeCompliance {
		t.Errorf("retention mode = %q, want %q", got, stor.LockModeCompliance)
	}
	if !provider.holds[remotePath] {
		t.Errorf("expected legal hold to be on for %q", remotePath)
	}

	// Protection must be recorded on the metadata.
	if !metadata.Immutable {
		t.Errorf("metadata.Immutable = false, want true")
	}
	if metadata.ImmutableUntil == nil || !metadata.ImmutableUntil.Equal(until) {
		t.Errorf("metadata.ImmutableUntil = %v, want %v", metadata.ImmutableUntil, until)
	}
	if metadata.LockMode != stor.LockModeCompliance {
		t.Errorf("metadata.LockMode = %q, want %q", metadata.LockMode, stor.LockModeCompliance)
	}
	if !metadata.LegalHold {
		t.Errorf("metadata.LegalHold = false, want true")
	}
}

// TestCreateBackup_ImmutableOnNonImmutableProviderFails proves the backup fails
// closed when immutability is requested against a provider that does not
// support object lock.
func TestCreateBackup_ImmutableOnNonImmutableProviderFails(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	storeDir := filepath.Join(root, "store")
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	// The local provider implements storage.Provider but NOT ImmutableProvider.
	provider := newLocalProvider(t, storeDir)
	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	metadata, err := engine.CreateBackup(context.Background(), &CreateOptions{
		DatabaseType:   database.DatabaseTypeSQLite,
		Database:       dbPath,
		Immutable:      true,
		RetentionUntil: time.Now().Add(24 * time.Hour),
	})
	if err == nil {
		t.Fatalf("expected error when immutability requested against non-immutable provider")
	}
	if metadata.Status == database.BackupStatusSuccess {
		t.Errorf("backup must not be marked successful when immutability cannot be applied")
	}
}

// TestCreateBackup_ImmutableNoProviderFails proves immutability fails closed
// when no storage provider is configured (local-only).
func TestCreateBackup_ImmutableNoProviderFails(t *testing.T) {
	root := t.TempDir()
	tempDir := filepath.Join(root, "temp")
	dbPath := filepath.Join(root, "source.db")
	seedSQLiteDB(t, dbPath)

	engine := NewEngine(&Config{TempDirectory: tempDir})

	_, err := engine.CreateBackup(context.Background(), &CreateOptions{
		DatabaseType:   database.DatabaseTypeSQLite,
		Database:       dbPath,
		Immutable:      true,
		RetentionUntil: time.Now().Add(24 * time.Hour),
	})
	if err == nil {
		t.Fatalf("expected error when immutability requested with no provider")
	}
}

// TestGetBackupImmutabilityAndApplyLegalHold proves GetBackupImmutability reads
// live provider state and ApplyLegalHold toggles and records the hold.
func TestGetBackupImmutabilityAndApplyLegalHold(t *testing.T) {
	tempDir := t.TempDir()
	provider := newFakeImmutableProvider()
	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	remotePath := "backups/b1/b1.sql"
	until := time.Now().Add(48 * time.Hour).UTC()
	if err := provider.SetRetention(context.Background(), remotePath, until, stor.LockModeGovernance); err != nil {
		t.Fatalf("SetRetention: %v", err)
	}

	metadata := &models.BackupMetadata{
		ID:              "b1",
		StorageLocation: "fake://" + remotePath,
	}

	gotUntil, gotMode, gotHold, err := engine.GetBackupImmutability(context.Background(), metadata)
	if err != nil {
		t.Fatalf("GetBackupImmutability: %v", err)
	}
	if !gotUntil.Equal(until) {
		t.Errorf("until = %v, want %v", gotUntil, until)
	}
	if gotMode != stor.LockModeGovernance {
		t.Errorf("mode = %q, want %q", gotMode, stor.LockModeGovernance)
	}
	if gotHold {
		t.Errorf("legal hold = true, want false")
	}

	// Enabling a legal hold must call the provider and persist on metadata.
	if holdErr := engine.ApplyLegalHold(context.Background(), metadata, true); holdErr != nil {
		t.Fatalf("ApplyLegalHold: %v", holdErr)
	}
	if !metadata.LegalHold {
		t.Errorf("metadata.LegalHold = false, want true after ApplyLegalHold")
	}
	if !provider.holds[remotePath] {
		t.Errorf("expected provider legal hold on for %q", remotePath)
	}

	_, _, gotHold, err = engine.GetBackupImmutability(context.Background(), metadata)
	if err != nil {
		t.Fatalf("GetBackupImmutability after hold: %v", err)
	}
	if !gotHold {
		t.Errorf("legal hold = false, want true after ApplyLegalHold")
	}
}
