// Package minio provides MinIO storage provider implementation
package minio

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sanskarpan/db-backup/internal/storage"
)

// MinIOProvider implements the storage.Provider interface for MinIO
type MinIOProvider struct {
	client       *s3.Client
	uploader     *manager.Uploader
	downloader   *manager.Downloader
	config       *storage.MinIOConfig
	bucket       string
}

// NewMinIOProvider creates a new MinIO storage provider
func NewMinIOProvider(cfg *storage.MinIOConfig) (*MinIOProvider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Create custom resolver for MinIO endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               buildEndpointURL(cfg),
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Build AWS config for MinIO
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO config: %w", err)
	}

	// Create S3 client with path-style addressing (required for MinIO)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Create uploader and downloader
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &MinIOProvider{
		client:     client,
		uploader:   uploader,
		downloader: downloader,
		config:     cfg,
		bucket:     cfg.Bucket,
	}, nil
}

// buildEndpointURL builds the MinIO endpoint URL
func buildEndpointURL(cfg *storage.MinIOConfig) string {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, cfg.Endpoint)
}

// validateConfig validates the MinIO configuration
func validateConfig(cfg *storage.MinIOConfig) error {
	if cfg == nil {
		return fmt.Errorf("MinIO config is required")
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("MinIO endpoint is required")
	}
	if cfg.AccessKey == "" {
		return fmt.Errorf("MinIO access key is required")
	}
	if cfg.SecretKey == "" {
		return fmt.Errorf("MinIO secret key is required")
	}
	if cfg.Bucket == "" {
		return fmt.Errorf("MinIO bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1" // Default region
	}
	return nil
}

// Upload uploads a file to MinIO
func (p *MinIOProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.UploadStream(ctx, file, remotePath, opts)
}

// UploadStream uploads data from a reader to MinIO
func (p *MinIOProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	if opts == nil {
		opts = &storage.UploadOptions{}
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
		Body:   reader,
	}

	// Set content type if provided
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}

	// Set storage class if provided
	if opts.StorageClass != "" {
		input.StorageClass = types.StorageClass(opts.StorageClass)
	}

	// Set metadata if provided
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	// Set ACL if provided
	if opts.ACL != "" {
		input.ACL = types.ObjectCannedACL(opts.ACL)
	}

	// Upload using uploader (supports multipart for large files)
	_, err := p.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	return nil
}

// Download downloads a file from MinIO
func (p *MinIOProvider) Download(ctx context.Context, remotePath, localPath string) error {
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

	// Download using downloader
	_, err = p.downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return fmt.Errorf("failed to download from MinIO: %w", err)
	}

	return nil
}

// DownloadStream downloads data to a reader
func (p *MinIOProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from MinIO: %w", err)
	}

	return output.Body, nil
}

// Delete deletes a file from MinIO
func (p *MinIOProvider) Delete(ctx context.Context, remotePath string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from MinIO: %w", err)
	}

	return nil
}

// Exists checks if a file exists in MinIO
func (p *MinIOProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		// Check if error is NotFound
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetMetadata retrieves file metadata from MinIO
func (p *MinIOProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	output, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from MinIO: %w", err)
	}

	metadata := &storage.FileMetadata{
		Path:         remotePath,
		Size:         *output.ContentLength,
		ContentType:  aws.ToString(output.ContentType),
		Metadata:     output.Metadata,
	}

	if output.LastModified != nil {
		metadata.LastModified = *output.LastModified
	}

	if output.ETag != nil {
		metadata.Checksum = strings.Trim(*output.ETag, "\"")
	}

	return metadata, nil
}

// List lists files with a given prefix in MinIO
func (p *MinIOProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	var files []*storage.FileMetadata

	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from MinIO: %w", err)
		}

		for _, obj := range page.Contents {
			metadata := &storage.FileMetadata{
				Path:         *obj.Key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			}

			if obj.ETag != nil {
				metadata.Checksum = strings.Trim(*obj.ETag, "\"")
			}

			if obj.StorageClass != "" {
				metadata.StorageClass = string(obj.StorageClass)
			}

			files = append(files, metadata)
		}
	}

	return files, nil
}

// GetType returns the provider type
func (p *MinIOProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeMinIO
}

// ValidateConfig validates the provider configuration
func (p *MinIOProvider) ValidateConfig() error {
	return validateConfig(p.config)
}

// CreateBucket creates a bucket in MinIO
func (p *MinIOProvider) CreateBucket(ctx context.Context) error {
	_, err := p.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		// Check if bucket already exists
		if strings.Contains(err.Error(), "BucketAlreadyExists") || strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// DeleteBucket deletes a bucket from MinIO
func (p *MinIOProvider) DeleteBucket(ctx context.Context) error {
	_, err := p.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// SetVersioning enables or disables versioning for the bucket
func (p *MinIOProvider) SetVersioning(ctx context.Context, enabled bool) error {
	status := types.BucketVersioningStatusSuspended
	if enabled {
		status = types.BucketVersioningStatusEnabled
	}

	_, err := p.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(p.bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: status,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set versioning: %w", err)
	}

	return nil
}

// GetVersioning returns the versioning status of the bucket
func (p *MinIOProvider) GetVersioning(ctx context.Context) (bool, error) {
	output, err := p.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return false, fmt.Errorf("failed to get versioning: %w", err)
	}

	return output.Status == types.BucketVersioningStatusEnabled, nil
}

// SetReplication configures replication for the bucket
func (p *MinIOProvider) SetReplication(ctx context.Context, destinationBucket, destinationRegion string) error {
	// MinIO uses the same replication config as S3
	replicationConfig := &types.ReplicationConfiguration{
		Role: aws.String("arn:aws:iam::minioadmin:role/replication"),
		Rules: []types.ReplicationRule{
			{
				ID:       aws.String("replication-rule-1"),
				Priority: aws.Int32(1),
				Status:   types.ReplicationRuleStatusEnabled,
				Filter:   &types.ReplicationRuleFilterMemberPrefix{Value: ""},
				Destination: &types.Destination{
					Bucket: aws.String(fmt.Sprintf("arn:aws:s3:::%s", destinationBucket)),
				},
			},
		},
	}

	_, err := p.client.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
		Bucket:                   aws.String(p.bucket),
		ReplicationConfiguration: replicationConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to set replication: %w", err)
	}

	return nil
}

// GetReplication returns the replication configuration of the bucket
func (p *MinIOProvider) GetReplication(ctx context.Context) (*types.ReplicationConfiguration, error) {
	output, err := p.client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get replication: %w", err)
	}

	return output.ReplicationConfiguration, nil
}

// SetLifecyclePolicy sets a lifecycle policy for the bucket
func (p *MinIOProvider) SetLifecyclePolicy(ctx context.Context, rules []types.LifecycleRule) error {
	_, err := p.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(p.bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: rules,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set lifecycle policy: %w", err)
	}

	return nil
}

// GetLifecyclePolicy returns the lifecycle policy of the bucket
func (p *MinIOProvider) GetLifecyclePolicy(ctx context.Context) ([]types.LifecycleRule, error) {
	output, err := p.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get lifecycle policy: %w", err)
	}

	return output.Rules, nil
}
