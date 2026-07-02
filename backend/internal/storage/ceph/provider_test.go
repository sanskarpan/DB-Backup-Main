package ceph

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// TestUploadWritesFileContents verifies that Upload opens the local file and
// streams its contents to Ceph via the S3-compatible RADOS Gateway API.
func TestUploadWritesFileContents(t *testing.T) {
	const (
		wantBody = "hello ceph upload contents\n"
		bucket   = "test-bucket"
		key      = "backups/db.dump"
	)

	var (
		mu       sync.Mutex
		gotBody  string
		gotPath  string
		gotType  string
		gotEnc   string
		gotClass string
	)

	// Fake Ceph RADOS Gateway (S3-compatible) endpoint capturing the PutObject.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodPut {
			mu.Lock()
			gotBody = string(body)
			gotPath = r.URL.Path
			gotType = r.Header.Get("Content-Type")
			gotEnc = r.Header.Get("X-Amz-Server-Side-Encryption")
			gotClass = r.Header.Get("X-Amz-Storage-Class")
			mu.Unlock()
		}
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	provider, err := NewCephProvider(&storage.CephConfig{
		Endpoint:  srv.URL,
		Region:    "default",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Bucket:    bucket,
	})
	if err != nil {
		t.Fatalf("NewCephProvider: %v", err)
	}

	// Write a real local file to be uploaded.
	localPath := filepath.Join(t.TempDir(), "db.dump")
	if err := os.WriteFile(localPath, []byte(wantBody), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	opts := &storage.UploadOptions{
		ContentType:          "application/octet-stream",
		StorageClass:         "STANDARD",
		ServerSideEncryption: true,
	}
	if err := provider.Upload(context.Background(), localPath, key, opts); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotBody != wantBody {
		t.Errorf("uploaded body = %q, want %q", gotBody, wantBody)
	}
	if want := "/" + bucket + "/" + key; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
	if gotType != opts.ContentType {
		t.Errorf("Content-Type = %q, want %q", gotType, opts.ContentType)
	}
	if !strings.EqualFold(gotEnc, "AES256") {
		t.Errorf("server-side encryption header = %q, want AES256", gotEnc)
	}
	if gotClass != opts.StorageClass {
		t.Errorf("storage class = %q, want %q", gotClass, opts.StorageClass)
	}
}

// TestUploadMissingFile ensures Upload surfaces a provider error when the local
// file cannot be opened, rather than silently succeeding.
func TestUploadMissingFile(t *testing.T) {
	provider, err := NewCephProvider(&storage.CephConfig{
		Endpoint:  "http://127.0.0.1:0",
		Region:    "default",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Bucket:    "test-bucket",
	})
	if err != nil {
		t.Fatalf("NewCephProvider: %v", err)
	}

	err = provider.Upload(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), "remote/key", nil)
	if err == nil {
		t.Fatal("expected error uploading non-existent file, got nil")
	}

	var provErr *storage.ProviderError
	if !errors.As(err, &provErr) {
		t.Fatalf("expected *storage.ProviderError, got %T: %v", err, err)
	}
}
