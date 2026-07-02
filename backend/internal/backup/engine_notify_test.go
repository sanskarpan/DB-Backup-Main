package backup

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/notification"
)

// fakeNotifier records every notification it is asked to send.
type fakeNotifier struct {
	returnErr error
	calls     []*notification.Notification
	mu        sync.Mutex
}

func (f *fakeNotifier) Send(_ context.Context, notif *notification.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, notif)
	return f.returnErr
}

func (f *fakeNotifier) last() *notification.Notification {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return nil
	}
	return f.calls[len(f.calls)-1]
}

func (f *fakeNotifier) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func TestNotifyResultSuccess(t *testing.T) {
	fake := &fakeNotifier{}
	engine := NewEngine(&Config{Notifier: fake})

	meta := &models.BackupMetadata{
		ID:       "bkp-1",
		Database: "orders",
		Status:   database.BackupStatusSuccess,
		Size:     2048,
	}
	engine.notifyResult(context.Background(), meta, nil)

	if fake.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", fake.count())
	}
	got := fake.last()
	if got.Level != notification.LevelSuccess {
		t.Errorf("expected success level, got %s", got.Level)
	}
	if !hasTag(got.Tags, "success") {
		t.Errorf("expected success tag, got %v", got.Tags)
	}
	if got.Metadata["backup_id"] != "bkp-1" {
		t.Errorf("expected backup_id metadata, got %v", got.Metadata["backup_id"])
	}
}

func TestNotifyResultFailure(t *testing.T) {
	fake := &fakeNotifier{}
	engine := NewEngine(&Config{Notifier: fake})

	meta := &models.BackupMetadata{
		ID:       "bkp-2",
		Database: "orders",
		Status:   database.BackupStatusFailed,
	}
	backupErr := errors.New("connection refused")
	engine.notifyResult(context.Background(), meta, backupErr)

	if fake.count() != 1 {
		t.Fatalf("expected 1 notification, got %d", fake.count())
	}
	got := fake.last()
	if got.Level != notification.LevelError {
		t.Errorf("expected error level, got %s", got.Level)
	}
	if !hasTag(got.Tags, "failure") {
		t.Errorf("expected failure tag, got %v", got.Tags)
	}
	if got.Metadata["error"] != "connection refused" {
		t.Errorf("expected error metadata, got %v", got.Metadata["error"])
	}
}

func TestCreateBackupFailurePathNotifies(t *testing.T) {
	fake := &fakeNotifier{}
	engine := NewEngine(&Config{Notifier: fake, TempDirectory: t.TempDir()})

	// An unregistered database type makes createBackup fail early, exercising
	// the failure path of the public CreateBackup wrapper.
	opts := &CreateOptions{
		DatabaseType: database.DatabaseType("bogus-driver"),
		Database:     "orders",
	}
	meta, err := engine.CreateBackup(context.Background(), opts)
	if err == nil {
		t.Fatal("expected CreateBackup to fail for an unregistered driver")
	}
	if meta == nil {
		t.Fatal("expected non-nil metadata even on failure")
	}
	if fake.count() != 1 {
		t.Fatalf("expected 1 failure notification, got %d", fake.count())
	}
	if got := fake.last(); got.Level != notification.LevelError {
		t.Errorf("expected error level, got %s", got.Level)
	}
}

func TestCreateBackupNotifyErrorIsSwallowed(t *testing.T) {
	fake := &fakeNotifier{returnErr: errors.New("slack down")}
	engine := NewEngine(&Config{Notifier: fake, TempDirectory: t.TempDir()})

	opts := &CreateOptions{
		DatabaseType: database.DatabaseType("bogus-driver"),
		Database:     "orders",
	}
	// The backup itself fails, but the notify error must not replace or mask it.
	_, err := engine.CreateBackup(context.Background(), opts)
	if err == nil {
		t.Fatal("expected backup error to be returned")
	}
	if fake.count() != 1 {
		t.Fatalf("expected notifier to have been called once, got %d", fake.count())
	}
}

func TestNilNotifierIsSafe(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})

	// Direct helper call must not panic with a nil notifier.
	engine.notifyResult(context.Background(), &models.BackupMetadata{ID: "x"}, nil)

	// Full failure path via the wrapper must not panic either.
	opts := &CreateOptions{
		DatabaseType: database.DatabaseType("bogus-driver"),
		Database:     "orders",
	}
	if _, err := engine.CreateBackup(context.Background(), opts); err == nil {
		t.Fatal("expected CreateBackup to fail for an unregistered driver")
	}
}
