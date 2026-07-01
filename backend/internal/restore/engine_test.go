package restore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/sanskarpan/db-backup/internal/models"
	stor "github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/local"
)

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func newLocalProvider(t *testing.T, dir string) stor.Provider {
	t.Helper()
	p, err := local.NewLocalProvider(&stor.LocalConfig{Path: dir})
	if err != nil {
		t.Fatalf("new local provider: %v", err)
	}
	return p
}

// uploadArtifact writes content to a temp source file and uploads it to the
// provider at remotePath, returning the checksum.
func uploadArtifact(t *testing.T, provider stor.Provider, remotePath string, content []byte) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "src")
	if err := os.WriteFile(tmp, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := provider.Upload(context.Background(), tmp, remotePath, nil); err != nil {
		t.Fatalf("upload: %v", err)
	}
	return sha256Hex(content)
}

// TestDownloadBackup_RemoteRoundTrip proves an uploaded artifact can be
// downloaded and its checksum verifies.
func TestDownloadBackup_RemoteRoundTrip(t *testing.T) {
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "temp")
	provider := newLocalProvider(t, storeDir)

	content := []byte("BACKUP-CONTENT-round-trip")
	remotePath := "backups/abc123/abc123.sql"
	checksum := uploadArtifact(t, provider, remotePath, content)

	meta := &models.BackupMetadata{
		ID:              "abc123",
		BackupPath:      filepath.Join(tempDir, "abc123.sql"),
		StorageLocation: "local://" + remotePath,
		Checksum:        checksum,
	}

	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	dest := filepath.Join(root, "downloaded.sql")
	if err := engine.DownloadBackup(context.Background(), meta, dest); err != nil {
		t.Fatalf("DownloadBackup: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q", string(got))
	}
}

// TestDownloadBackup_ChecksumMismatch ensures a corrupted/incorrect checksum
// causes the download verification to fail.
func TestDownloadBackup_ChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "temp")
	provider := newLocalProvider(t, storeDir)

	remotePath := "backups/xyz/xyz.sql"
	uploadArtifact(t, provider, remotePath, []byte("real content"))

	meta := &models.BackupMetadata{
		ID:              "xyz",
		StorageLocation: "local://" + remotePath,
		Checksum:        "0000000000000000000000000000000000000000000000000000000000000000",
	}
	engine := NewEngine(&Config{TempDirectory: tempDir, StorageProvider: provider})

	if err := engine.DownloadBackup(context.Background(), meta, filepath.Join(root, "out.sql")); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
}

// TestValidateBackup_LocalChecksumMismatch checks local validation fails on
// checksum mismatch.
func TestValidateBackup_LocalChecksumMismatch(t *testing.T) {
	root := t.TempDir()
	localPath := filepath.Join(root, "b.sql")
	if err := os.WriteFile(localPath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	meta := &models.BackupMetadata{Checksum: "wrongsum"}
	engine := NewEngine(&Config{TempDirectory: root})

	if err := engine.validateBackup(context.Background(), meta, localPath); err == nil {
		t.Fatalf("expected checksum mismatch error")
	}

	// Correct checksum should pass.
	meta.Checksum = sha256Hex([]byte("data"))
	if err := engine.validateBackup(context.Background(), meta, localPath); err != nil {
		t.Errorf("expected validation to pass, got %v", err)
	}
}

// TestValidateBackup_RemoteMissing ensures validation fails when a remote
// artifact does not exist.
func TestValidateBackup_RemoteMissing(t *testing.T) {
	root := t.TempDir()
	provider := newLocalProvider(t, filepath.Join(root, "store"))
	engine := NewEngine(&Config{TempDirectory: root, StorageProvider: provider})

	meta := &models.BackupMetadata{StorageLocation: "local://backups/missing/missing.sql"}
	if err := engine.validateBackup(context.Background(), meta, ""); err == nil {
		t.Fatalf("expected error for missing remote artifact")
	}
}

// TestDownloadBackup_LocalNilProvider verifies local backups are copied without
// a provider configured.
func TestDownloadBackup_LocalNilProvider(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "local.sql")
	content := []byte("local-only-backup")
	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}
	meta := &models.BackupMetadata{
		BackupPath:      src,
		StorageLocation: src,
		Checksum:        sha256Hex(content),
	}
	engine := NewEngine(&Config{TempDirectory: root})

	dest := filepath.Join(root, "copy.sql")
	if err := engine.DownloadBackup(context.Background(), meta, dest); err != nil {
		t.Fatalf("DownloadBackup: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != string(content) {
		t.Errorf("content mismatch")
	}
}
