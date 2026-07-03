// Package ceph provides Ceph storage backend integration via RADOS Gateway (S3 API)
package ceph

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// CephProvider implements storage provider for Ceph via RADOS Gateway.
//
//nolint:revive // keeps public name stable; used by other packages
type CephProvider struct {
	config     *storage.CephConfig
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

// NewCephProvider creates a new Ceph storage provider using RADOS Gateway S3 API.
func NewCephProvider(cfg *storage.CephConfig) (*CephProvider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "invalid configuration",
			Err:     err,
		}
	}

	// Create AWS config with Ceph RADOS Gateway endpoint
	awsConfig, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to load AWS config",
			Err:     err,
		}
	}

	// Create S3 client with custom endpoint for Ceph RADOS Gateway
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true // Ceph RGW typically uses path-style addressing
	})

	// Create uploader and downloader
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 64 * 1024 * 1024 // 64MB parts for large files
		u.Concurrency = 5
	})

	downloader := manager.NewDownloader(client, func(d *manager.Downloader) {
		d.PartSize = 64 * 1024 * 1024
		d.Concurrency = 5
	})

	return &CephProvider{
		config:     cfg,
		client:     client,
		uploader:   uploader,
		downloader: downloader,
	}, nil
}

// Upload uploads a file to Ceph storage.
func (p *CephProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to open file for upload: %s", localPath),
			Err:     err,
		}
	}
	defer file.Close()

	return p.UploadStream(ctx, file, remotePath, opts)
}

// UploadStream uploads data from a reader to Ceph storage.
func (p *CephProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
		Body:   reader,
	}

	if opts != nil {
		if opts.ContentType != "" {
			input.ContentType = aws.String(opts.ContentType)
		}
		if opts.Metadata != nil {
			input.Metadata = opts.Metadata
		}
		if opts.StorageClass != "" {
			input.StorageClass = types.StorageClass(opts.StorageClass)
		}
		if opts.ServerSideEncryption {
			input.ServerSideEncryption = types.ServerSideEncryptionAes256
		}
	}

	_, err := p.uploader.Upload(ctx, input)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to upload to Ceph: %s", remotePath),
			Err:     err,
		}
	}

	return nil
}

// Download downloads a file from Ceph storage.
func (p *CephProvider) Download(ctx context.Context, remotePath, localPath string) error {
	file, err := createFile(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = p.downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to download from Ceph: %s", remotePath),
			Err:     err,
		}
	}

	return nil
}

// DownloadStream downloads data to a writer.
func (p *CephProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to download stream from Ceph: %s", remotePath),
			Err:     err,
		}
	}

	return result.Body, nil
}

// Delete deletes a file from Ceph storage.
func (p *CephProvider) Delete(ctx context.Context, remotePath string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to delete from Ceph: %s", remotePath),
			Err:     err,
		}
	}

	return nil
}

// Exists checks if a file exists in Ceph storage.
func (p *CephProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		// Check if error is NotFound
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to check existence in Ceph: %s", remotePath),
			Err:     err,
		}
	}

	return true, nil
}

// GetMetadata retrieves file metadata.
func (p *CephProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	result, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to get metadata from Ceph: %s", remotePath),
			Err:     err,
		}
	}

	metadata := &storage.FileMetadata{
		Path:         remotePath,
		Size:         *result.ContentLength,
		LastModified: *result.LastModified,
		ContentType:  aws.ToString(result.ContentType),
		Checksum:     aws.ToString(result.ETag),
		StorageClass: string(result.StorageClass),
		Metadata:     result.Metadata,
	}

	return metadata, nil
}

// List lists files with a given prefix.
func (p *CephProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	result, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.config.Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: fmt.Sprintf("failed to list objects in Ceph: %s", prefix),
			Err:     err,
		}
	}

	files := make([]*storage.FileMetadata, 0, len(result.Contents))
	for _, obj := range result.Contents {
		files = append(files, &storage.FileMetadata{
			Path:         *obj.Key,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
			Checksum:     *obj.ETag,
			StorageClass: string(obj.StorageClass),
		})
	}

	return files, nil
}

// GetType returns the provider type.
func (p *CephProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeCeph
}

// ValidateConfig validates the provider configuration.
func (p *CephProvider) ValidateConfig() error {
	return validateConfig(p.config)
}

// CreateBucket creates a bucket in Ceph.
func (p *CephProvider) CreateBucket(ctx context.Context) error {
	_, err := p.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		// Check if bucket already exists
		var bucketExists *types.BucketAlreadyOwnedByYou
		if errors.As(err, &bucketExists) {
			return nil // Bucket already exists, not an error
		}
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to create bucket in Ceph",
			Err:     err,
		}
	}

	return nil
}

// DeleteBucket deletes a bucket from Ceph.
func (p *CephProvider) DeleteBucket(ctx context.Context) error {
	_, err := p.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to delete bucket from Ceph",
			Err:     err,
		}
	}

	return nil
}

// SetReplication sets replication rules for Ceph bucket.
func (p *CephProvider) SetReplication(ctx context.Context, rules []types.ReplicationRule) error {
	_, err := p.client.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
		Bucket: aws.String(p.config.Bucket),
		ReplicationConfiguration: &types.ReplicationConfiguration{
			Role:  aws.String(""),
			Rules: rules,
		},
	})
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to set replication in Ceph",
			Err:     err,
		}
	}

	return nil
}

// GetReplication gets replication configuration.
func (p *CephProvider) GetReplication(ctx context.Context) (*types.ReplicationConfiguration, error) {
	result, err := p.client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to get replication from Ceph",
			Err:     err,
		}
	}

	return result.ReplicationConfiguration, nil
}

// SetLifecyclePolicy sets lifecycle policy for Ceph bucket.
func (p *CephProvider) SetLifecyclePolicy(ctx context.Context, rules []types.LifecycleRule) error {
	_, err := p.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(p.config.Bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: rules,
		},
	})
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to set lifecycle policy in Ceph",
			Err:     err,
		}
	}

	return nil
}

// GetLifecyclePolicy gets lifecycle policy.
func (p *CephProvider) GetLifecyclePolicy(ctx context.Context) ([]types.LifecycleRule, error) {
	result, err := p.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(p.config.Bucket),
	})
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeCeph,
			Message: "failed to get lifecycle policy from Ceph",
			Err:     err,
		}
	}

	return result.Rules, nil
}

// Helper functions

func validateConfig(cfg *storage.CephConfig) error {
	if cfg == nil {
		return errors.New("ceph config is required")
	}
	if cfg.Endpoint == "" {
		return errors.New("ceph RADOS Gateway endpoint is required")
	}
	if cfg.AccessKey == "" {
		return errors.New("ceph access key is required")
	}
	if cfg.SecretKey == "" {
		return errors.New("ceph secret key is required")
	}
	if cfg.Bucket == "" {
		return errors.New("ceph bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "default" // Ceph uses "default" region
	}
	return nil
}

func createFile(path string) (*os.File, error) {
	return os.Create(path)
}
