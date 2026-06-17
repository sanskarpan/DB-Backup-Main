// Package backblaze provides Backblaze B2 storage provider implementation
package backblaze

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kurin/blazer/b2"
	"github.com/sanskarpan/db-backup/internal/storage"
)

// BackblazeB2Provider implements the storage.Provider interface for Backblaze B2
type BackblazeB2Provider struct {
	client *b2.Client
	bucket *b2.Bucket
	config *storage.BackblazeB2Config
}

// NewBackblazeB2Provider creates a new Backblaze B2 storage provider
func NewBackblazeB2Provider(cfg *storage.BackblazeB2Config) (*BackblazeB2Provider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Create B2 client
	client, err := b2.NewClient(ctx, cfg.AccountID, cfg.ApplicationKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create B2 client: %w", err)
	}

	// Get or create bucket
	bucket, err := client.Bucket(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &BackblazeB2Provider{
		client: client,
		bucket: bucket,
		config: cfg,
	}, nil
}

// validateConfig validates the Backblaze B2 configuration
func validateConfig(cfg *storage.BackblazeB2Config) error {
	if cfg == nil {
		return fmt.Errorf("Backblaze B2 config is required")
	}
	if cfg.AccountID == "" {
		return fmt.Errorf("Backblaze B2 account ID is required")
	}
	if cfg.ApplicationKey == "" {
		return fmt.Errorf("Backblaze B2 application key is required")
	}
	if cfg.Bucket == "" {
		return fmt.Errorf("Backblaze B2 bucket is required")
	}
	return nil
}

// Upload uploads a file to Backblaze B2
func (p *BackblazeB2Provider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	return p.uploadStream(ctx, file, remotePath, info.Size(), opts)
}

// UploadStream uploads data from a reader to Backblaze B2
func (p *BackblazeB2Provider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	// For stream uploads, we need to buffer the data to get the size
	// In production, you might want to use a temporary file or require size in opts
	return p.uploadStream(ctx, reader, remotePath, -1, opts)
}

// uploadStream is the internal upload method
func (p *BackblazeB2Provider) uploadStream(ctx context.Context, reader io.Reader, remotePath string, size int64, opts *storage.UploadOptions) error {
	if opts == nil {
		opts = &storage.UploadOptions{}
	}

	// Create object writer
	obj := p.bucket.Object(remotePath)
	writer := obj.NewWriter(ctx)

	// Build upload attributes
	attrs := &b2.Attrs{}
	if opts.ContentType != "" {
		attrs.ContentType = opts.ContentType
	}
	if len(opts.Metadata) > 0 {
		info := make(map[string]string)
		for k, v := range opts.Metadata {
			info[k] = v
		}
		attrs.Info = info
	}
	writer.WithAttrs(attrs)

	// Copy data to writer
	_, err := io.Copy(writer, reader)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload to Backblaze B2: %w", err)
	}

	// Close writer to complete upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to complete upload to Backblaze B2: %w", err)
	}

	return nil
}

// Download downloads a file from Backblaze B2
func (p *BackblazeB2Provider) Download(ctx context.Context, remotePath, localPath string) error {
	// Create local directory if it doesn't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Download from B2
	obj := p.bucket.Object(remotePath)
	reader := obj.NewReader(ctx)
	defer reader.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to download from Backblaze B2: %w", err)
	}

	return nil
}

// DownloadStream downloads data to a reader
func (p *BackblazeB2Provider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	obj := p.bucket.Object(remotePath)
	reader := obj.NewReader(ctx)
	return reader, nil
}

// Delete deletes a file from Backblaze B2
func (p *BackblazeB2Provider) Delete(ctx context.Context, remotePath string) error {
	obj := p.bucket.Object(remotePath)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete from Backblaze B2: %w", err)
	}

	return nil
}

// Exists checks if a file exists in Backblaze B2
func (p *BackblazeB2Provider) Exists(ctx context.Context, remotePath string) (bool, error) {
	obj := p.bucket.Object(remotePath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return false, nil
		}
		return false, err
	}

	return attrs != nil, nil
}

// GetMetadata retrieves file metadata from Backblaze B2
func (p *BackblazeB2Provider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	obj := p.bucket.Object(remotePath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from Backblaze B2: %w", err)
	}

	metadata := &storage.FileMetadata{
		Path:        remotePath,
		Size:        attrs.Size,
		ContentType: attrs.ContentType,
	}

	if !attrs.UploadTimestamp.IsZero() {
		metadata.LastModified = attrs.UploadTimestamp
	}

	if attrs.SHA1 != "" {
		metadata.Checksum = attrs.SHA1
	}

	// Convert B2 metadata to standard metadata
	if len(attrs.Info) > 0 {
		metadata.Metadata = attrs.Info
	}

	return metadata, nil
}

// List lists files with a given prefix in Backblaze B2
func (p *BackblazeB2Provider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	var files []*storage.FileMetadata

	// List objects with prefix
	iter := p.bucket.List(ctx, b2.ListPrefix(prefix))

	for iter.Next() {
		obj := iter.Object()
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			continue
		}

		metadata := &storage.FileMetadata{
			Path:        obj.Name(),
			Size:        attrs.Size,
			ContentType: attrs.ContentType,
		}

		if !attrs.UploadTimestamp.IsZero() {
			metadata.LastModified = attrs.UploadTimestamp
		}

		if attrs.SHA1 != "" {
			metadata.Checksum = attrs.SHA1
		}

		files = append(files, metadata)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list objects from Backblaze B2: %w", err)
	}

	return files, nil
}

// GetType returns the provider type
func (p *BackblazeB2Provider) GetType() storage.ProviderType {
	return storage.ProviderTypeBackblazeB2
}

// ValidateConfig validates the provider configuration
func (p *BackblazeB2Provider) ValidateConfig() error {
	return validateConfig(p.config)
}

// CreateBucket creates a bucket in Backblaze B2
func (p *BackblazeB2Provider) CreateBucket(ctx context.Context, bucketName string, public bool) error {
	bucketType := b2.BucketType(b2.Private)
	if public {
		bucketType = b2.BucketType(b2.Public)
	}

	bucket, err := p.client.NewBucket(ctx, bucketName, &b2.BucketAttrs{
		Type: bucketType,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	p.bucket = bucket
	return nil
}

// DeleteBucket deletes a bucket from Backblaze B2
func (p *BackblazeB2Provider) DeleteBucket(ctx context.Context) error {
	if err := p.bucket.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// SetFileLock sets file lock/retention for a file.
// Note: file lock is not supported by the current blazer library version.
func (p *BackblazeB2Provider) SetFileLock(ctx context.Context, remotePath string, retentionDays int) error {
	return fmt.Errorf("file lock/retention not supported by the current Backblaze B2 library version")
}

// GetFileLock retrieves file lock/retention information for a file.
// Note: file lock is not supported by the current blazer library version.
func (p *BackblazeB2Provider) GetFileLock(ctx context.Context, remotePath string) (string, time.Time, error) {
	return "", time.Time{}, fmt.Errorf("file lock/retention not supported by the current Backblaze B2 library version")
}

// SetLifecycleRules sets lifecycle rules for the bucket
func (p *BackblazeB2Provider) SetLifecycleRules(ctx context.Context, rules []b2.LifecycleRule) error {
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket attributes: %w", err)
	}

	attrs.LifecycleRules = rules

	if err := p.bucket.Update(ctx, attrs); err != nil {
		return fmt.Errorf("failed to set lifecycle rules: %w", err)
	}

	return nil
}

// GetLifecycleRules retrieves lifecycle rules for the bucket
func (p *BackblazeB2Provider) GetLifecycleRules(ctx context.Context) ([]b2.LifecycleRule, error) {
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get lifecycle rules: %w", err)
	}

	return attrs.LifecycleRules, nil
}

// GetBucketInfo retrieves bucket information
func (p *BackblazeB2Provider) GetBucketInfo(ctx context.Context) (*b2.BucketAttrs, error) {
	attrs, err := p.bucket.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket info: %w", err)
	}

	return attrs, nil
}
