// Package gcs provides Google Cloud Storage
package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	st "github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// GCSProvider implements Google Cloud Storage
type GCSProvider struct {
	client *storage.Client
	bucket *storage.BucketHandle
	config *st.GCSConfig
}

// NewGCSProvider creates a new GCS storage provider
func NewGCSProvider(config *st.GCSConfig) (*GCSProvider, error) {
	if config.Bucket == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "GCS bucket is required")
	}

	ctx := context.Background()

	var opts []option.ClientOption
	if config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create GCS client")
	}

	bucket := client.Bucket(config.Bucket)

	return &GCSProvider{
		client: client,
		bucket: bucket,
		config: config,
	}, nil
}

// Upload uploads a file to GCS
func (p *GCSProvider) Upload(ctx context.Context, localPath, remotePath string, opts *st.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to open local file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to stat local file")
	}

	return p.UploadStream(ctx, &progressReader{
		reader:   file,
		total:    stat.Size(),
		callback: opts.ProgressCallback,
	}, remotePath, opts)
}

// UploadStream uploads data from a reader to GCS
func (p *GCSProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *st.UploadOptions) error {
	obj := p.bucket.Object(remotePath)
	writer := obj.NewWriter(ctx)

	// Set metadata
	if opts != nil {
		if opts.ContentType != "" {
			writer.ContentType = opts.ContentType
		}
		if opts.Metadata != nil {
			writer.Metadata = opts.Metadata
		}
		if opts.StorageClass != "" {
			writer.StorageClass = opts.StorageClass
		}
	}

	// Copy data
	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to upload to GCS")
	}

	if err := writer.Close(); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to close GCS writer")
	}

	return nil
}

// Download downloads a file from GCS
func (p *GCSProvider) Download(ctx context.Context, remotePath, localPath string) error {
	// Create directory if needed
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create local directory")
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create local file")
	}
	defer file.Close()

	// Download from GCS
	reader, err := p.DownloadStream(ctx, remotePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to write to local file")
	}

	return nil
}

// DownloadStream downloads data to a reader
func (p *GCSProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	obj := p.bucket.Object(remotePath)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, "object not found in GCS")
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create GCS reader")
	}

	return reader, nil
}

// Delete deletes a file from GCS
func (p *GCSProvider) Delete(ctx context.Context, remotePath string) error {
	obj := p.bucket.Object(remotePath)
	if err := obj.Delete(ctx); err != nil {
		if err == storage.ErrObjectNotExist {
			return pkgErrors.New(pkgErrors.ErrorTypeNotFound, "object not found in GCS")
		}
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to delete from GCS")
	}

	return nil
}

// Exists checks if a file exists in GCS
func (p *GCSProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	obj := p.bucket.Object(remotePath)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to check GCS object")
	}

	return true, nil
}

// GetMetadata retrieves file metadata
func (p *GCSProvider) GetMetadata(ctx context.Context, remotePath string) (*st.FileMetadata, error) {
	obj := p.bucket.Object(remotePath)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, "object not found in GCS")
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to get GCS metadata")
	}

	metadata := &st.FileMetadata{
		Path:         remotePath,
		Size:         attrs.Size,
		LastModified: attrs.Updated,
		ContentType:  attrs.ContentType,
		StorageClass: attrs.StorageClass,
		Metadata:     attrs.Metadata,
	}

	// Use CRC32C checksum (Google's recommended integrity check) instead of MD5
	// CRC32C is faster and GCS provides it automatically for all objects
	if attrs.CRC32C != 0 {
		metadata.Checksum = fmt.Sprintf("crc32c:%08x", attrs.CRC32C)
	} else if attrs.MD5 != nil {
		// Fallback to MD5 only if CRC32C is not available (rare case)
		metadata.Checksum = fmt.Sprintf("md5:%x", attrs.MD5)
	}

	return metadata, nil
}

// List lists files with a given prefix
func (p *GCSProvider) List(ctx context.Context, prefix string) ([]*st.FileMetadata, error) {
	var files []*st.FileMetadata

	query := &storage.Query{Prefix: prefix}
	it := p.bucket.Objects(ctx, query)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to list GCS objects")
		}

		// Skip directories
		if attrs.Name == "" || attrs.Name[len(attrs.Name)-1] == '/' {
			continue
		}

		metadata := &st.FileMetadata{
			Path:         attrs.Name,
			Size:         attrs.Size,
			LastModified: attrs.Updated,
			ContentType:  attrs.ContentType,
			StorageClass: attrs.StorageClass,
			Metadata:     attrs.Metadata,
		}

		// Use CRC32C checksum (Google's recommended integrity check) instead of MD5
		if attrs.CRC32C != 0 {
			metadata.Checksum = fmt.Sprintf("crc32c:%08x", attrs.CRC32C)
		} else if attrs.MD5 != nil {
			// Fallback to MD5 only if CRC32C is not available (rare case)
			metadata.Checksum = fmt.Sprintf("md5:%x", attrs.MD5)
		}

		files = append(files, metadata)
	}

	return files, nil
}

// GetType returns the provider type
func (p *GCSProvider) GetType() st.ProviderType {
	return st.ProviderTypeGCS
}

// ValidateConfig validates the provider configuration
func (p *GCSProvider) ValidateConfig() error {
	if p.config.Bucket == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "GCS bucket is required")
	}
	return nil
}

// Close closes the GCS client
func (p *GCSProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// progressReader wraps a reader to track progress
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback func(uploaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)

	if pr.callback != nil {
		pr.callback(pr.read, pr.total)
	}

	return n, err
}
