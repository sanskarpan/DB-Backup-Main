// Package wasabi provides Wasabi storage provider implementation
package wasabi

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// WasabiProvider implements the storage.Provider interface for Wasabi.
type WasabiProvider struct { //nolint:revive // keeps public name stable across packages
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
	config     *storage.WasabiConfig
	bucket     string
}

// Regional endpoints for Wasabi.
var wasabiEndpoints = map[string]string{
	"us-east-1":      "s3.wasabisys.com",
	"us-east-2":      "s3.us-east-2.wasabisys.com",
	"us-west-1":      "s3.us-west-1.wasabisys.com",
	"eu-central-1":   "s3.eu-central-1.wasabisys.com",
	"eu-west-1":      "s3.eu-west-1.wasabisys.com",
	"eu-west-2":      "s3.eu-west-2.wasabisys.com",
	"ap-northeast-1": "s3.ap-northeast-1.wasabisys.com",
	"ap-northeast-2": "s3.ap-northeast-2.wasabisys.com",
	"ap-southeast-1": "s3.ap-southeast-1.wasabisys.com",
	"ap-southeast-2": "s3.ap-southeast-2.wasabisys.com",
}

// NewWasabiProvider creates a new Wasabi storage provider.
func NewWasabiProvider(cfg *storage.WasabiConfig) (*WasabiProvider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Get endpoint for region
	endpoint := cfg.Endpoint
	if endpoint == "" {
		if ep, ok := wasabiEndpoints[cfg.Region]; ok {
			endpoint = ep
		} else {
			endpoint = wasabiEndpoints["us-east-1"] // Default to us-east-1
		}
	}

	// Create custom resolver for Wasabi endpoint.
	//nolint:staticcheck // SA1019: aws-sdk-go-v2 global EndpointResolver is deprecated but still required to point the S3 client at Wasabi's custom endpoint; migrating to per-service BaseEndpoint is tracked separately.
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			scheme := "http"
			if cfg.UseSSL {
				scheme = "https"
			}
			return aws.Endpoint{ //nolint:staticcheck // SA1019: deprecated type required for Wasabi custom endpoint resolution.
				URL:               fmt.Sprintf("%s://%s", scheme, endpoint),
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{} //nolint:staticcheck // SA1019: deprecated type required for Wasabi custom endpoint resolution.
	})

	// Build AWS config for Wasabi
	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Wasabi config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	// Create uploader and downloader
	uploader := manager.NewUploader(client)
	downloader := manager.NewDownloader(client)

	return &WasabiProvider{
		client:     client,
		uploader:   uploader,
		downloader: downloader,
		config:     cfg,
		bucket:     cfg.Bucket,
	}, nil
}

// validateConfig validates the Wasabi configuration.
func validateConfig(cfg *storage.WasabiConfig) error {
	if cfg == nil {
		return fmt.Errorf("wasabi config is required")
	}
	if cfg.AccessKey == "" {
		return fmt.Errorf("wasabi access key is required")
	}
	if cfg.SecretKey == "" {
		return fmt.Errorf("wasabi secret key is required")
	}
	if cfg.Bucket == "" {
		return fmt.Errorf("wasabi bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1" // Default region
	}
	return nil
}

// Upload uploads a file to Wasabi.
func (p *WasabiProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.UploadStream(ctx, file, remotePath, opts)
}

// UploadStream uploads data from a reader to Wasabi.
func (p *WasabiProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
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

	// Enable object lock if immutability is enabled
	if p.config.Immutable && p.config.RetentionDays > 0 {
		retainUntil := time.Now().AddDate(0, 0, p.config.RetentionDays)
		input.ObjectLockMode = types.ObjectLockModeCompliance
		input.ObjectLockRetainUntilDate = &retainUntil
	}

	// Upload using uploader (supports multipart for large files)
	_, err := p.uploader.Upload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to Wasabi: %w", err)
	}

	return nil
}

// Download downloads a file from Wasabi.
func (p *WasabiProvider) Download(ctx context.Context, remotePath, localPath string) error {
	// Create local directory if it doesn't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
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
		return fmt.Errorf("failed to download from Wasabi: %w", err)
	}

	return nil
}

// DownloadStream downloads data to a reader.
func (p *WasabiProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	output, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from Wasabi: %w", err)
	}

	return output.Body, nil
}

// Delete deletes a file from Wasabi.
func (p *WasabiProvider) Delete(ctx context.Context, remotePath string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from Wasabi: %w", err)
	}

	return nil
}

// Exists checks if a file exists in Wasabi.
func (p *WasabiProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
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

// GetMetadata retrieves file metadata from Wasabi.
func (p *WasabiProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	output, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from Wasabi: %w", err)
	}

	metadata := &storage.FileMetadata{
		Path:        remotePath,
		Size:        *output.ContentLength,
		ContentType: aws.ToString(output.ContentType),
		Metadata:    output.Metadata,
	}

	if output.LastModified != nil {
		metadata.LastModified = *output.LastModified
	}

	if output.ETag != nil {
		metadata.Checksum = strings.Trim(*output.ETag, "\"")
	}

	return metadata, nil
}

// List lists files with a given prefix in Wasabi.
func (p *WasabiProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	var files []*storage.FileMetadata

	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from Wasabi: %w", err)
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

// GetType returns the provider type.
func (p *WasabiProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeWasabi
}

// ValidateConfig validates the provider configuration.
func (p *WasabiProvider) ValidateConfig() error {
	return validateConfig(p.config)
}

// CreateBucket creates a bucket in Wasabi.
func (p *WasabiProvider) CreateBucket(ctx context.Context) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(p.bucket),
	}

	// Enable object lock if immutability is required
	if p.config.Immutable {
		input.ObjectLockEnabledForBucket = aws.Bool(true)
	}

	_, err := p.client.CreateBucket(ctx, input)
	if err != nil {
		// Check if bucket already exists
		if strings.Contains(err.Error(), "BucketAlreadyExists") || strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Set object lock configuration if immutability is enabled
	if p.config.Immutable && p.config.RetentionDays > 0 {
		if err := p.setObjectLockConfiguration(ctx); err != nil {
			return err
		}
	}

	return nil
}

// setObjectLockConfiguration sets the default object lock configuration.
func (p *WasabiProvider) setObjectLockConfiguration(ctx context.Context) error {
	// Clamp retention days to the int32 range to avoid overflow.
	retentionDays := p.config.RetentionDays
	if retentionDays < 0 {
		retentionDays = 0
	}
	if retentionDays > math.MaxInt32 {
		retentionDays = math.MaxInt32
	}

	_, err := p.client.PutObjectLockConfiguration(ctx, &s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(p.bucket),
		ObjectLockConfiguration: &types.ObjectLockConfiguration{
			ObjectLockEnabled: types.ObjectLockEnabledEnabled,
			Rule: &types.ObjectLockRule{
				DefaultRetention: &types.DefaultRetention{
					Mode: types.ObjectLockRetentionModeCompliance,
					Days: aws.Int32(int32(retentionDays)), //nolint:gosec // G115: retentionDays is clamped to [0, math.MaxInt32] above.
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set object lock configuration: %w", err)
	}

	return nil
}

// GetObjectLockConfiguration retrieves the object lock configuration.
func (p *WasabiProvider) GetObjectLockConfiguration(ctx context.Context) (*types.ObjectLockConfiguration, error) {
	output, err := p.client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object lock configuration: %w", err)
	}

	return output.ObjectLockConfiguration, nil
}

// SetLifecyclePolicy sets a lifecycle policy for the bucket.
func (p *WasabiProvider) SetLifecyclePolicy(ctx context.Context, rules []types.LifecycleRule) error {
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

// GetLifecyclePolicy returns the lifecycle policy of the bucket.
func (p *WasabiProvider) GetLifecyclePolicy(ctx context.Context) ([]types.LifecycleRule, error) {
	output, err := p.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(p.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get lifecycle policy: %w", err)
	}

	return output.Rules, nil
}
