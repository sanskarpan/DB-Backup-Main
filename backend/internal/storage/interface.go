// Package storage provides storage provider interfaces and implementations
package storage

import (
	"context"
	"io"
	"time"
)

// Provider defines the interface for storage providers.
type Provider interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, localPath, remotePath string, opts *UploadOptions) error

	// UploadStream uploads data from a reader to storage
	UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *UploadOptions) error

	// Download downloads a file from storage
	Download(ctx context.Context, remotePath, localPath string) error

	// DownloadStream downloads data to a writer
	DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, remotePath string) error

	// Exists checks if a file exists in storage
	Exists(ctx context.Context, remotePath string) (bool, error)

	// GetMetadata retrieves file metadata
	GetMetadata(ctx context.Context, remotePath string) (*FileMetadata, error)

	// List lists files with a given prefix
	List(ctx context.Context, prefix string) ([]*FileMetadata, error)

	// GetType returns the provider type
	GetType() ProviderType

	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// ProviderType represents the type of storage provider.
type ProviderType string

const (
	ProviderTypeLocal       ProviderType = "local"
	ProviderTypeS3          ProviderType = "s3"
	ProviderTypeGCS         ProviderType = "gcs"
	ProviderTypeAzure       ProviderType = "azure"
	ProviderTypeMinIO       ProviderType = "minio"
	ProviderTypeWasabi      ProviderType = "wasabi"
	ProviderTypeBackblazeB2 ProviderType = "b2"
	ProviderTypeCeph        ProviderType = "ceph"
	ProviderTypeGlusterFS   ProviderType = "glusterfs"
)

// UploadOptions holds options for upload operations.
type UploadOptions struct {
	ContentType          string
	Metadata             map[string]string
	StorageClass         string
	ServerSideEncryption bool
	ACL                  string
	ChunkSize            int64
	Parallel             int
	Checksum             string
	ProgressCallback     func(uploaded, total int64)
}

// FileMetadata contains metadata about a file.
type FileMetadata struct {
	Path         string
	Size         int64
	LastModified time.Time
	ContentType  string
	Checksum     string
	StorageClass string
	Metadata     map[string]string
}

// NewProvider creates a new storage provider based on type
// This is a placeholder - use specific provider constructors from sub-packages.
func NewProvider(providerType ProviderType, config interface{}) (Provider, error) {
	return nil, &ProviderError{
		Type:    providerType,
		Message: "use specific provider constructor from sub-package (local.NewLocalProvider, s3.NewS3Provider, etc.)",
	}
}

// ProviderError represents a provider-specific error.
type ProviderError struct {
	Type    ProviderType
	Message string
	Err     error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return string(e.Type) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Type) + ": " + e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// LocalConfig holds local storage configuration.
type LocalConfig struct {
	Path string
}

// S3Config holds AWS S3 configuration.
type S3Config struct {
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	Endpoint     string
	UsePathStyle bool
}

// GCSConfig holds Google Cloud Storage configuration.
type GCSConfig struct {
	Project         string
	Bucket          string
	CredentialsFile string
}

// AzureConfig holds Azure Blob Storage configuration.
type AzureConfig struct {
	AccountName string
	AccountKey  string
	Container   string
}

// MinIOConfig holds MinIO storage configuration.
type MinIOConfig struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	UseSSL       bool
	Region       string
	UsePathStyle bool
}

// WasabiConfig holds Wasabi storage configuration.
type WasabiConfig struct {
	Endpoint      string // Wasabi endpoint (e.g., s3.wasabisys.com)
	Region        string
	AccessKey     string
	SecretKey     string
	Bucket        string
	UseSSL        bool
	Immutable     bool // Enable object lock for immutability
	RetentionDays int  // Days to retain immutable objects
}

// BackblazeB2Config holds Backblaze B2 storage configuration.
type BackblazeB2Config struct {
	AccountID      string
	ApplicationKey string
	Bucket         string
	BucketID       string
	Endpoint       string // Auto-detected or custom
	Immutable      bool   // Enable file lock for immutability
	RetentionDays  int    // Days to retain immutable files
}

// CephConfig holds Ceph storage configuration via RADOS Gateway.
type CephConfig struct {
	Endpoint  string // Ceph RADOS Gateway endpoint
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	Pool      string // Optional: Ceph pool name
}

// GlusterFSConfig holds GlusterFS storage configuration.
type GlusterFSConfig struct {
	VolumeHost  string // GlusterFS volume host
	VolumeName  string // GlusterFS volume name
	MountPath   string // Mount path for GlusterFS volume
	SubPath     string // Sub-path within the volume for backups
	Replication int    // Replication factor
}
