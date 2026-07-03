package replication

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// fakeProvider is an in-memory storage.Provider test double.
type fakeProvider struct {
	mu       sync.Mutex
	typ      storage.ProviderType
	objects  map[string][]byte
	checksum map[string]string

	// failUploads makes the next failUploads UploadStream calls fail, to
	// exercise transient-failure retry logic.
	failUploads int
	// downloadErr, when set, is returned by DownloadStream.
	downloadErr error
	// uploadErr, when set, is returned by UploadStream after failUploads
	// is exhausted (persistent failure).
	uploadErr error
}

func newFakeProvider(typ storage.ProviderType) *fakeProvider {
	return &fakeProvider{
		typ:      typ,
		objects:  make(map[string][]byte),
		checksum: make(map[string]string),
	}
}

func (f *fakeProvider) put(remotePath string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	buf := make([]byte, len(data))
	copy(buf, data)
	f.objects[remotePath] = buf
	sum := sha256.Sum256(buf)
	f.checksum[remotePath] = hex.EncodeToString(sum[:])
}

func (f *fakeProvider) Upload(_ context.Context, _, _ string, _ *storage.UploadOptions) error {
	return errors.New("not implemented")
}

func (f *fakeProvider) UploadStream(_ context.Context, reader io.Reader, remotePath string, _ *storage.UploadOptions) error {
	f.mu.Lock()
	if f.failUploads > 0 {
		f.failUploads--
		f.mu.Unlock()
		return errors.New("transient upload failure")
	}
	if f.uploadErr != nil {
		f.mu.Unlock()
		return f.uploadErr
	}
	f.mu.Unlock()

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	f.put(remotePath, data)
	return nil
}

func (f *fakeProvider) Download(_ context.Context, _, _ string) error {
	return errors.New("not implemented")
}

func (f *fakeProvider) DownloadStream(_ context.Context, remotePath string) (io.ReadCloser, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[remotePath]
	if !ok {
		return nil, errors.New("object not found")
	}
	buf := make([]byte, len(data))
	copy(buf, data)
	return io.NopCloser(bytes.NewReader(buf)), nil
}

func (f *fakeProvider) Delete(_ context.Context, remotePath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, remotePath)
	delete(f.checksum, remotePath)
	return nil
}

func (f *fakeProvider) Exists(_ context.Context, remotePath string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[remotePath]
	return ok, nil
}

func (f *fakeProvider) GetMetadata(_ context.Context, remotePath string) (*storage.FileMetadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.objects[remotePath]
	if !ok {
		return nil, errors.New("object not found")
	}
	return &storage.FileMetadata{
		Path:     remotePath,
		Size:     int64(len(data)),
		Checksum: f.checksum[remotePath],
	}, nil
}

func (f *fakeProvider) List(_ context.Context, prefix string) ([]*storage.FileMetadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*storage.FileMetadata
	for p, data := range f.objects {
		if prefix == "" || (len(p) >= len(prefix) && p[:len(prefix)] == prefix) {
			out = append(out, &storage.FileMetadata{Path: p, Size: int64(len(data))})
		}
	}
	return out, nil
}

func (f *fakeProvider) GetType() storage.ProviderType { return f.typ }

func (f *fakeProvider) ValidateConfig() error { return nil }

func TestReplicate_CopiesBytesAndVerifies(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)

	payload := []byte("hello replication world")
	src.put("backups/db.dump", payload)

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "backups/db.dump", Options{VerifyAfterCopy: true})
	if err != nil {
		t.Fatalf("Replicate returned error: %v", err)
	}
	if res.Status != StatusCopied {
		t.Fatalf("expected status %q, got %q", StatusCopied, res.Status)
	}
	if res.BytesCopied != int64(len(payload)) {
		t.Fatalf("expected %d bytes copied, got %d", len(payload), res.BytesCopied)
	}

	sum := sha256.Sum256(payload)
	if res.Checksum != hex.EncodeToString(sum[:]) {
		t.Fatalf("checksum mismatch: got %s", res.Checksum)
	}
	if res.SourceType != storage.ProviderTypeS3 || res.DestType != storage.ProviderTypeGCS {
		t.Fatalf("unexpected provider types: %s -> %s", res.SourceType, res.DestType)
	}

	got, err := dst.DownloadStream(ctx, "backups/db.dump")
	if err != nil {
		t.Fatalf("destination download failed: %v", err)
	}
	defer func() { _ = got.Close() }()
	data, err := io.ReadAll(got)
	if err != nil {
		t.Fatalf("reading destination object: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("destination bytes differ from source")
	}
}

func TestReplicate_MissingSource(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "nope", Options{})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
	if res.Status != StatusFailed {
		t.Fatalf("expected StatusFailed, got %q", res.Status)
	}
}

func TestReplicate_SkipIfExists(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)

	src.put("obj", []byte("new-content"))
	dst.put("obj", []byte("existing-content"))

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "obj", Options{Overwrite: false})
	if err != nil {
		t.Fatalf("Replicate returned error: %v", err)
	}
	if res.Status != StatusSkipped {
		t.Fatalf("expected StatusSkipped, got %q", res.Status)
	}

	// Destination must be untouched.
	rc, _ := dst.DownloadStream(ctx, "obj")
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "existing-content" {
		t.Fatalf("destination was overwritten: %q", data)
	}
}

func TestReplicate_OverwriteReplaces(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)

	src.put("obj", []byte("new-content"))
	dst.put("obj", []byte("old"))

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "obj", Options{Overwrite: true, VerifyAfterCopy: true})
	if err != nil {
		t.Fatalf("Replicate returned error: %v", err)
	}
	if res.Status != StatusCopied {
		t.Fatalf("expected StatusCopied, got %q", res.Status)
	}
	rc, _ := dst.DownloadStream(ctx, "obj")
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(rc)
	if string(data) != "new-content" {
		t.Fatalf("destination not overwritten: %q", data)
	}
}

func TestReplicate_DestPathPrefix(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)
	src.put("db.dump", []byte("x"))

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "db.dump", Options{DestPathPrefix: "replicas/2026"})
	if err != nil {
		t.Fatalf("Replicate returned error: %v", err)
	}
	if res.DestPath != "replicas/2026/db.dump" {
		t.Fatalf("unexpected dest path: %q", res.DestPath)
	}
	if ok, _ := dst.Exists(ctx, "replicas/2026/db.dump"); !ok {
		t.Fatalf("object not stored at prefixed path")
	}
}

func TestReplicate_RetryOnTransientError(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)
	src.put("obj", []byte("retry-me"))
	dst.failUploads = 2 // fail twice, succeed on third attempt

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "obj", Options{
		MaxRetries:   3,
		RetryBackoff: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Replicate returned error: %v", err)
	}
	if res.Status != StatusCopied {
		t.Fatalf("expected StatusCopied after retries, got %q (err=%v)", res.Status, res.Err)
	}
	if dst.failUploads != 0 {
		t.Fatalf("expected all transient failures consumed, %d remain", dst.failUploads)
	}
}

func TestReplicate_RetryExhausted(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	dst := newFakeProvider(storage.ProviderTypeGCS)
	src.put("obj", []byte("data"))
	dst.failUploads = 5 // more failures than retries allow

	r := NewReplicator()
	res, err := r.Replicate(ctx, src, dst, "obj", Options{MaxRetries: 1, RetryBackoff: time.Millisecond})
	if err == nil {
		t.Fatalf("expected error when retries exhausted")
	}
	if res.Status != StatusFailed {
		t.Fatalf("expected StatusFailed, got %q", res.Status)
	}
}

func TestReplicateToMany_AllSucceed(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	src.put("obj", []byte("fan-out"))

	dsts := []storage.Provider{
		newFakeProvider(storage.ProviderTypeGCS),
		newFakeProvider(storage.ProviderTypeAzure),
		newFakeProvider(storage.ProviderTypeWasabi),
	}

	r := NewReplicator()
	results, err := r.ReplicateToMany(ctx, src, dsts, "obj", Options{VerifyAfterCopy: true})
	if err != nil {
		t.Fatalf("ReplicateToMany returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, res := range results {
		if res.Status != StatusCopied {
			t.Fatalf("result %d: expected StatusCopied, got %q", i, res.Status)
		}
	}
}

func TestReplicateToMany_MinSuccessfulMet(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	src.put("obj", []byte("payload"))

	good := newFakeProvider(storage.ProviderTypeGCS)
	bad := newFakeProvider(storage.ProviderTypeAzure)
	bad.uploadErr = errors.New("permanent failure")

	dsts := []storage.Provider{good, bad}

	r := NewReplicator()
	results, err := r.ReplicateToMany(ctx, src, dsts, "obj", Options{MinSuccessful: 1})
	if err != nil {
		t.Fatalf("expected success with min=1, got %v", err)
	}
	if results[0].Status != StatusCopied {
		t.Fatalf("expected first destination copied, got %q", results[0].Status)
	}
	if results[1].Status != StatusFailed {
		t.Fatalf("expected second destination failed, got %q", results[1].Status)
	}
}

func TestReplicateToMany_MinSuccessfulNotMet(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	src.put("obj", []byte("payload"))

	good := newFakeProvider(storage.ProviderTypeGCS)
	bad := newFakeProvider(storage.ProviderTypeAzure)
	bad.uploadErr = errors.New("permanent failure")

	dsts := []storage.Provider{good, bad}

	r := NewReplicator()
	results, err := r.ReplicateToMany(ctx, src, dsts, "obj", Options{MinSuccessful: 2})
	if !errors.Is(err, ErrMinSuccessfulNotMet) {
		t.Fatalf("expected ErrMinSuccessfulNotMet, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results even on failure, got %d", len(results))
	}
}

func TestReplicateToMany_NoDestinations(t *testing.T) {
	ctx := context.Background()
	src := newFakeProvider(storage.ProviderTypeS3)
	r := NewReplicator()
	if _, err := r.ReplicateToMany(ctx, src, nil, "obj", Options{}); err == nil {
		t.Fatalf("expected error for empty destinations")
	}
}
