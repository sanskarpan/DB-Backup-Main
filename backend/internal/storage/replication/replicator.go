// Package replication provides storage-to-storage replication of backup
// objects, streaming an object from a source provider to one or more
// destination providers for redundancy and cross-cloud portability.
package replication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"path"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// Status describes the outcome of replicating an object to one destination.
type Status string

const (
	// StatusCopied indicates the object was streamed to the destination.
	StatusCopied Status = "copied"

	// StatusSkipped indicates the object already existed and was not overwritten.
	StatusSkipped Status = "skipped"

	// StatusFailed indicates replication to the destination failed.
	StatusFailed Status = "failed"
)

// ErrObjectNotFound is returned when the source object does not exist.
var ErrObjectNotFound = errors.New("source object does not exist")

// ErrMinSuccessfulNotMet is returned by ReplicateToMany when fewer
// destinations succeeded than the configured minimum.
var ErrMinSuccessfulNotMet = errors.New("minimum successful replications not met")

// ErrVerificationFailed is returned when a post-copy verification mismatch
// is detected between the source object and the destination object.
var ErrVerificationFailed = errors.New("post-copy verification failed")

// Options controls how a replication is performed.
type Options struct {
	// Overwrite replaces an existing destination object. When false, an
	// existing object is left untouched and reported as StatusSkipped.
	Overwrite bool

	// VerifyAfterCopy compares destination metadata (size and, when
	// available, checksum) against the copied stream after the upload.
	VerifyAfterCopy bool

	// MaxRetries is the number of additional attempts after the first on a
	// transient failure. Zero means a single attempt.
	MaxRetries int

	// RetryBackoff is the base backoff between retry attempts. The delay
	// grows quadratically with the attempt number. Defaults to one second.
	RetryBackoff time.Duration

	// DestPathPrefix is joined in front of the source remote path to form
	// the destination path. Ignored when DestPathRename is set.
	DestPathPrefix string

	// DestPathRename, when non-empty, is used as the destination path
	// verbatim instead of the source remote path.
	DestPathRename string

	// UploadOptions is passed through to the destination provider's
	// UploadStream call. May be nil.
	UploadOptions *storage.UploadOptions

	// MinSuccessful is the minimum number of destinations that must succeed
	// for ReplicateToMany to report overall success. When zero or negative
	// it defaults to the number of destinations (all must succeed).
	MinSuccessful int

	// MaxConcurrent bounds the number of destinations replicated to in
	// parallel by ReplicateToMany. Defaults to the number of destinations.
	MaxConcurrent int
}

// ReplicationResult captures the outcome of replicating a single object to a
// single destination provider.
//
//nolint:revive // public name kept stable for API consumers
type ReplicationResult struct {
	// SourceType is the type of the source provider.
	SourceType storage.ProviderType

	// DestType is the type of the destination provider.
	DestType storage.ProviderType

	// RemotePath is the source object path.
	RemotePath string

	// DestPath is the resolved destination object path.
	DestPath string

	// BytesCopied is the number of bytes streamed to the destination.
	BytesCopied int64

	// Checksum is the SHA-256 checksum of the streamed bytes (hex encoded).
	Checksum string

	// DurationMs is the wall-clock duration of the replication in milliseconds.
	DurationMs int64

	// Status is the terminal status of the replication.
	Status Status

	// Err holds the error when Status is StatusFailed.
	Err error
}

// Replicator copies backup objects between storage providers by streaming
// from a source provider to one or more destination providers.
type Replicator struct{}

// NewReplicator creates a new Replicator.
func NewReplicator() *Replicator {
	return &Replicator{}
}

// resolveDestPath computes the destination path from the source path and options.
func resolveDestPath(remotePath string, opts *Options) string {
	if opts.DestPathRename != "" {
		return opts.DestPathRename
	}
	if opts.DestPathPrefix != "" {
		return path.Join(opts.DestPathPrefix, remotePath)
	}
	return remotePath
}

// Replicate streams a single object from src to dst and returns the outcome.
// It never returns a nil result: on failure the result carries StatusFailed
// and Err, which is also returned as the error.
//
//nolint:gocritic // hugeParam: value semantics intended for this options struct
func (r *Replicator) Replicate(ctx context.Context, src, dst storage.Provider, remotePath string, opts Options) (*ReplicationResult, error) {
	start := time.Now()
	destPath := resolveDestPath(remotePath, &opts)
	result := &ReplicationResult{
		SourceType: src.GetType(),
		DestType:   dst.GetType(),
		RemotePath: remotePath,
		DestPath:   destPath,
		Status:     StatusFailed,
	}

	// Confirm the source object exists before doing any work.
	exists, err := src.Exists(ctx, remotePath)
	if err != nil {
		result.Err = fmt.Errorf("checking source object: %w", err)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, result.Err
	}
	if !exists {
		result.Err = fmt.Errorf("%w: %s", ErrObjectNotFound, remotePath)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, result.Err
	}

	// Honor skip-if-exists.
	if !opts.Overwrite {
		destExists, existErr := dst.Exists(ctx, destPath)
		if existErr != nil {
			result.Err = fmt.Errorf("checking destination object: %w", existErr)
			result.DurationMs = time.Since(start).Milliseconds()
			return result, result.Err
		}
		if destExists {
			result.Status = StatusSkipped
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}
	}

	if err := r.copyWithRetry(ctx, src, dst, remotePath, destPath, &opts, result); err != nil {
		result.Err = err
		result.DurationMs = time.Since(start).Milliseconds()
		return result, err
	}

	if opts.VerifyAfterCopy {
		if err := verify(ctx, src, dst, remotePath, destPath, result); err != nil {
			result.Err = err
			result.DurationMs = time.Since(start).Milliseconds()
			return result, err
		}
	}

	result.Status = StatusCopied
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

// copyWithRetry performs the streaming copy, retrying transient failures with
// a quadratic backoff. On success it records BytesCopied and Checksum.
func (r *Replicator) copyWithRetry(ctx context.Context, src, dst storage.Provider, remotePath, destPath string, opts *Options, result *ReplicationResult) error {
	backoff := opts.RetryBackoff
	if backoff <= 0 {
		backoff = time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt*attempt) * backoff
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("replication canceled during backoff: %w", ctx.Err())
			}
		}

		bytesCopied, checksum, err := streamCopy(ctx, src, dst, remotePath, destPath, opts.UploadOptions)
		if err == nil {
			result.BytesCopied = bytesCopied
			result.Checksum = checksum
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("replication failed after %d attempt(s): %w", opts.MaxRetries+1, lastErr)
}

// hashingReader wraps a reader, computing a running SHA-256 checksum and byte
// count over everything read through it.
type hashingReader struct {
	reader io.Reader
	hash   hash.Hash
	count  int64
}

func newHashingReader(reader io.Reader) *hashingReader {
	return &hashingReader{reader: reader, hash: sha256.New()}
}

func (h *hashingReader) Read(p []byte) (int, error) {
	n, err := h.reader.Read(p)
	if n > 0 {
		h.count += int64(n)
		// hash.Write never returns an error per the hash.Hash contract.
		_, _ = h.hash.Write(p[:n])
	}
	return n, err
}

func (h *hashingReader) checksum() string {
	return hex.EncodeToString(h.hash.Sum(nil))
}

// streamCopy streams one object from src to dst without buffering the whole
// payload, returning the byte count and SHA-256 checksum of the stream.
func streamCopy(ctx context.Context, src, dst storage.Provider, remotePath, destPath string, uploadOpts *storage.UploadOptions) (bytesCopied int64, checksum string, err error) {
	reader, err := src.DownloadStream(ctx, remotePath)
	if err != nil {
		return 0, "", fmt.Errorf("opening source stream: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing source stream: %w", closeErr)
		}
	}()

	hr := newHashingReader(reader)
	if uploadErr := dst.UploadStream(ctx, hr, destPath, uploadOpts); uploadErr != nil {
		return 0, "", fmt.Errorf("uploading to destination: %w", uploadErr)
	}

	return hr.count, hr.checksum(), nil
}

// verify compares the destination object metadata against the source. Size is
// always checked; checksum is checked only when both sides expose one.
func verify(ctx context.Context, src, dst storage.Provider, remotePath, destPath string, result *ReplicationResult) error {
	destMeta, err := dst.GetMetadata(ctx, destPath)
	if err != nil {
		return fmt.Errorf("reading destination metadata: %w", err)
	}

	if destMeta.Size != result.BytesCopied {
		return fmt.Errorf("%w: size mismatch (copied %d, destination reports %d)",
			ErrVerificationFailed, result.BytesCopied, destMeta.Size)
	}

	// Prefer comparing the destination checksum against the checksum we
	// computed over the stream, falling back to the source checksum.
	expected := result.Checksum
	if srcMeta, srcErr := src.GetMetadata(ctx, remotePath); srcErr == nil && srcMeta.Checksum != "" {
		expected = srcMeta.Checksum
	}

	if destMeta.Checksum != "" && expected != "" && destMeta.Checksum != expected {
		return fmt.Errorf("%w: checksum mismatch (expected %s, destination reports %s)",
			ErrVerificationFailed, expected, destMeta.Checksum)
	}

	return nil
}

// ReplicateToMany fans out replication of a single object to multiple
// destinations concurrently. It returns the per-destination results in the
// same order as dsts, and a non-nil aggregate error only when fewer than the
// configured minimum number of destinations succeeded.
//
//nolint:gocritic // hugeParam: value semantics intended for this options struct
func (r *Replicator) ReplicateToMany(ctx context.Context, src storage.Provider, dsts []storage.Provider, remotePath string, opts Options) ([]*ReplicationResult, error) {
	if len(dsts) == 0 {
		return nil, errors.New("no destination providers specified")
	}

	minSuccessful := opts.MinSuccessful
	if minSuccessful <= 0 {
		minSuccessful = len(dsts)
	}

	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = len(dsts)
	}
	semaphore := make(chan struct{}, maxConcurrent)

	results := make([]*ReplicationResult, len(dsts))
	var wg sync.WaitGroup
	for i, dst := range dsts {
		wg.Add(1)
		go func(idx int, provider storage.Provider) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results[idx] = &ReplicationResult{
					SourceType: src.GetType(),
					DestType:   provider.GetType(),
					RemotePath: remotePath,
					DestPath:   resolveDestPath(remotePath, &opts),
					Status:     StatusFailed,
					Err:        ctx.Err(),
				}
				return
			}

			// Replicate always returns a non-nil result whose Err mirrors the
			// returned error; record it either way.
			res, err := r.Replicate(ctx, src, provider, remotePath, opts)
			if err != nil && res.Err == nil {
				res.Err = err
			}
			results[idx] = res
		}(i, dst)
	}
	wg.Wait()

	successCount := 0
	for _, res := range results {
		if res != nil && (res.Status == StatusCopied || res.Status == StatusSkipped) {
			successCount++
		}
	}

	if successCount < minSuccessful {
		return results, fmt.Errorf("%w: %d/%d succeeded", ErrMinSuccessfulNotMet, successCount, minSuccessful)
	}

	return results, nil
}
