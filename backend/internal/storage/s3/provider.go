// Package s3 provides AWS S3 storage
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// S3Provider implements AWS S3 storage
type S3Provider struct {
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	config     *storage.S3Config
}

// NewS3Provider creates a new S3 storage provider
func NewS3Provider(cfg *storage.S3Config) (*S3Provider, error) {
	ctx := context.Background()

	// Load AWS config
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, pkgErrors.ErrStorageUpload(err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Create uploader and downloader
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 64 * 1024 * 1024 // 64MB parts
		u.Concurrency = 10
	})

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 64 * 1024 * 1024
		d.Concurrency = 10
	})

	return &S3Provider{
		client:     client,
		uploader:   uploader,
		downloader: downloader,
		config:     cfg,
	}, nil
}

// Upload uploads a file to S3
func (p *S3Provider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}
	defer file.Close()

	return p.UploadStream(ctx, file, remotePath, opts)
}

// UploadStream uploads data from a reader to S3
func (p *S3Provider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
		Body:   reader,
	}

	if opts != nil {
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
		}

		if opts.ServerSideEncryption {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		}

		if len(opts.Metadata) > 0 {
			input.Metadata = opts.Metadata
		}

		if opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(opts.StorageClass)
		}
	}

	// Upload with multipart support
	_, err := p.uploader.Upload(ctx, input)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err).WithMetadata("bucket", p.config.Bucket).WithMetadata("key", remotePath)
	}

	return nil
}

// Download downloads a file from S3
func (p *S3Provider) Download(ctx context.Context, remotePath, localPath string) error {
	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}
	defer file.Close()

	// Download from S3
	_, err = p.downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		return pkgErrors.ErrStorageDownload(err).WithMetadata("bucket", p.config.Bucket).WithMetadata("key", remotePath)
	}

	return nil
}

// DownloadStream downloads data to a reader
func (p *S3Provider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		return nil, pkgErrors.ErrStorageDownload(err).WithMetadata("bucket", p.config.Bucket).WithMetadata("key", remotePath)
	}

	return result.Body, nil
}

// Delete deletes a file from S3
func (p *S3Provider) Delete(ctx context.Context, remotePath string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		return pkgErrors.ErrStorageUpload(err).WithMetadata("operation", "delete").WithMetadata("bucket", p.config.Bucket).WithMetadata("key", remotePath)
	}

	return nil
}

// Exists checks if a file exists in S3
func (p *S3Provider) Exists(ctx context.Context, remotePath string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		// Check if it's a "not found" error
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetMetadata retrieves file metadata
func (p *S3Provider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	result, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})

	if err != nil {
		return nil, pkgErrors.ErrStorageDownload(err)
	}

	metadata := &storage.FileMetadata{
		Path:         remotePath,
		Size:         aws.ToInt64(result.ContentLength),
		LastModified: aws.ToTime(result.LastModified),
		ContentType:  aws.ToString(result.ContentType),
		Checksum:     aws.ToString(result.ETag),
		Metadata:     result.Metadata,
	}

	return metadata, nil
}

// List lists files with a given prefix
func (p *S3Provider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	var files []*storage.FileMetadata

	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.config.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, pkgErrors.ErrStorageDownload(err)
		}

		for _, obj := range page.Contents {
			metadata := &storage.FileMetadata{
				Path:         aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: aws.ToTime(obj.LastModified),
				Checksum:     aws.ToString(obj.ETag),
			}

			files = append(files, metadata)
		}
	}

	return files, nil
}

// GetType returns the provider type
func (p *S3Provider) GetType() storage.ProviderType {
	return storage.ProviderTypeS3
}

// ValidateConfig validates the provider configuration
func (p *S3Provider) ValidateConfig() error {
	if p.config.Bucket == "" {
		return fmt.Errorf("S3 bucket is required")
	}

	if p.config.Region == "" {
		return fmt.Errorf("S3 region is required")
	}

	if p.config.AccessKey == "" || p.config.SecretKey == "" {
		return fmt.Errorf("S3 credentials are required")
	}

	return nil
}
