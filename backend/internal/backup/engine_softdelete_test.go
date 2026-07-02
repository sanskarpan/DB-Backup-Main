package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/models"
	stor "github.com/sanskarpan/db-backup/internal/storage"
)

// writeBackup persists a metadata entry plus a local artifact file, returning the
// artifact path. It gives soft-delete tests a live backup to operate on without
// running a full CreateBackup.
func writeBackup(t *testing.T, e *Engine, id string) string {
	t.Helper()
	artifactPath := filepath.Join(e.config.TempDirectory, id+".sql")
	if err := os.MkdirAll(e.config.TempDirectory, 0o700); err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("dump-"+id), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	meta := &models.BackupMetadata{
		ID:              id,
		Name:            id,
		BackupPath:      artifactPath,
		StorageLocation: artifactPath,
	}
	if err := e.saveMetadata(meta); err != nil {
		t.Fatalf("saveMetadata: %v", err)
	}
	return artifactPath
}

func ids(backups []*models.BackupMetadata) map[string]bool {
	set := make(map[string]bool, len(backups))
	for _, b := range backups {
		set[b.ID] = true
	}
	return set
}

// TestDeleteBackup_SoftDeleteHidesFromList proves a soft-deleted backup is hidden
// from ListBackups but surfaces in ListDeletedBackups, and its artifact/metadata
// are retained.
func TestDeleteBackup_SoftDeleteHidesFromList(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	artifact := writeBackup(t, engine, "b1")
	writeBackup(t, engine, "b2")

	if err := engine.DeleteBackup(ctx, "b1"); err != nil {
		t.Fatalf("DeleteBackup: %v", err)
	}

	live, err := engine.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if liveIDs := ids(live); liveIDs["b1"] || !liveIDs["b2"] {
		t.Fatalf("ListBackups should exclude b1 and include b2, got %v", liveIDs)
	}

	deleted, err := engine.ListDeletedBackups(ctx)
	if err != nil {
		t.Fatalf("ListDeletedBackups: %v", err)
	}
	if delIDs := ids(deleted); !delIDs["b1"] || delIDs["b2"] {
		t.Fatalf("ListDeletedBackups should include only b1, got %v", delIDs)
	}

	// Soft delete must retain the artifact and metadata.
	if _, err := os.Stat(artifact); err != nil {
		t.Errorf("artifact should be retained after soft delete: %v", err)
	}
	meta, err := engine.GetBackup(ctx, "b1")
	if err != nil {
		t.Fatalf("GetBackup after soft delete: %v", err)
	}
	if meta.DeletedAt == nil {
		t.Errorf("DeletedAt should be set after soft delete")
	}
}

// TestRestoreDeletedBackup brings a soft-deleted backup back to the live list.
func TestRestoreDeletedBackup(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	writeBackup(t, engine, "b1")
	if err := engine.DeleteBackup(ctx, "b1"); err != nil {
		t.Fatalf("DeleteBackup: %v", err)
	}
	if err := engine.RestoreDeletedBackup(ctx, "b1"); err != nil {
		t.Fatalf("RestoreDeletedBackup: %v", err)
	}

	live, err := engine.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if !ids(live)["b1"] {
		t.Errorf("restored backup should appear in ListBackups")
	}

	meta, err := engine.GetBackup(ctx, "b1")
	if err != nil {
		t.Fatalf("GetBackup: %v", err)
	}
	if meta.DeletedAt != nil {
		t.Errorf("DeletedAt should be cleared after restore")
	}
}

// TestRestoreDeletedBackup_NotDeleted errors when the backup is live.
func TestRestoreDeletedBackup_NotDeleted(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	writeBackup(t, engine, "b1")
	if err := engine.RestoreDeletedBackup(ctx, "b1"); err == nil {
		t.Fatalf("expected error restoring a backup that is not in the recycle bin")
	}
}

// TestPurgeBackup permanently removes a soft-deleted backup's artifact + metadata.
func TestPurgeBackup(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	artifact := writeBackup(t, engine, "b1")
	if err := engine.DeleteBackup(ctx, "b1"); err != nil {
		t.Fatalf("DeleteBackup: %v", err)
	}
	if err := engine.PurgeBackup(ctx, "b1"); err != nil {
		t.Fatalf("PurgeBackup: %v", err)
	}

	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Errorf("artifact should be removed after purge, stat err=%v", err)
	}
	if _, err := engine.GetBackup(ctx, "b1"); err == nil {
		t.Errorf("metadata should be gone after purge")
	}
	deleted, err := engine.ListDeletedBackups(ctx)
	if err != nil {
		t.Fatalf("ListDeletedBackups: %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("recycle bin should be empty after purge, got %d", len(deleted))
	}
}

// TestPurgeBackup_LiveRejected refuses to purge a backup that was not soft-deleted.
func TestPurgeBackup_LiveRejected(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	artifact := writeBackup(t, engine, "b1")
	if err := engine.PurgeBackup(ctx, "b1"); err == nil {
		t.Fatalf("expected error purging a live (non-soft-deleted) backup")
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Errorf("artifact should be retained when purge is rejected: %v", err)
	}
}

// TestPurgeExpired purges only recycle-bin backups older than the retention window.
func TestPurgeExpired(t *testing.T) {
	engine := NewEngine(&Config{TempDirectory: t.TempDir()})
	ctx := context.Background()

	// old: soft-deleted 48h ago. recent: soft-deleted just now. live: never deleted.
	writeBackup(t, engine, "old")
	writeBackup(t, engine, "recent")
	writeBackup(t, engine, "live")

	oldMeta, err := engine.GetBackup(ctx, "old")
	if err != nil {
		t.Fatalf("GetBackup old: %v", err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	oldMeta.DeletedAt = &oldTime
	if err := engine.saveMetadata(oldMeta); err != nil {
		t.Fatalf("saveMetadata old: %v", err)
	}
	if err := engine.DeleteBackup(ctx, "recent"); err != nil {
		t.Fatalf("DeleteBackup recent: %v", err)
	}

	purged, err := engine.PurgeExpired(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("PurgeExpired: %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 purged, got %d", purged)
	}

	if _, err := engine.GetBackup(ctx, "old"); err == nil {
		t.Errorf("old backup should be purged")
	}
	if _, err := engine.GetBackup(ctx, "recent"); err != nil {
		t.Errorf("recent backup should survive: %v", err)
	}
	if _, err := engine.GetBackup(ctx, "live"); err != nil {
		t.Errorf("live backup should be untouched: %v", err)
	}
}

// recordingProvider records Delete calls so purge tests can assert the remote
// artifact is removed.
type recordingProvider struct {
	stor.Provider
	deleted []string
}

func (p *recordingProvider) Delete(_ context.Context, remotePath string) error {
	p.deleted = append(p.deleted, remotePath)
	return nil
}

func (p *recordingProvider) GetType() stor.ProviderType { return stor.ProviderTypeLocal }

// TestPurgeBackup_RemovesRemoteArtifact verifies purge deletes the remote object
// (and its metadata.json) through the storage provider.
func TestPurgeBackup_RemovesRemoteArtifact(t *testing.T) {
	provider := &recordingProvider{}
	engine := NewEngine(&Config{TempDirectory: t.TempDir(), StorageProvider: provider})
	ctx := context.Background()

	artifact := writeBackup(t, engine, "b1")
	meta, err := engine.GetBackup(ctx, "b1")
	if err != nil {
		t.Fatalf("GetBackup: %v", err)
	}
	meta.StorageLocation = "local://backups/b1/b1.sql"
	if err := engine.saveMetadata(meta); err != nil {
		t.Fatalf("saveMetadata: %v", err)
	}

	if err := engine.DeleteBackup(ctx, "b1"); err != nil {
		t.Fatalf("DeleteBackup: %v", err)
	}
	if err := engine.PurgeBackup(ctx, "b1"); err != nil {
		t.Fatalf("PurgeBackup: %v", err)
	}

	if len(provider.deleted) != 2 {
		t.Fatalf("expected 2 remote deletes (artifact + metadata.json), got %v", provider.deleted)
	}
	if provider.deleted[0] != "backups/b1/b1.sql" {
		t.Errorf("unexpected remote artifact delete path: %q", provider.deleted[0])
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Errorf("local temp copy should also be removed, stat err=%v", err)
	}
}
