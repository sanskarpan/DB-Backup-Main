// Package azure provides Azure Blob Storage
package azure

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	st "github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// AzureProvider implements Azure Blob Storage.
//
//nolint:revive // AzureProvider is a public type used by other packages; renaming would break them
type AzureProvider struct {
	client    *azblob.Client
	container *container.Client
	config    *st.AzureConfig
}

// NewAzureProvider creates a new Azure storage provider.
func NewAzureProvider(config *st.AzureConfig) (*AzureProvider, error) {
	if config.Container == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure container is required")
	}
	if config.AccountName == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure account name is required")
	}
	if config.AccountKey == "" {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure account key is required")
	}

	// Build connection string
	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
		config.AccountName,
		config.AccountKey,
	)

	// Create client
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create Azure client")
	}

	// Get container client
	containerClient := client.ServiceClient().NewContainerClient(config.Container)

	return &AzureProvider{
		client:    client,
		container: containerClient,
		config:    config,
	}, nil
}

// Upload uploads a file to Azure.
func (p *AzureProvider) Upload(ctx context.Context, localPath, remotePath string, opts *st.UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to open local file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to stat local file")
	}

	var reader io.Reader = file
	if opts != nil && opts.ProgressCallback != nil {
		reader = &progressReader{
			reader:   file,
			total:    stat.Size(),
			callback: opts.ProgressCallback,
		}
	}

	return p.UploadStream(ctx, reader, remotePath, opts)
}

// UploadStream uploads data from a reader to Azure.
func (p *AzureProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *st.UploadOptions) error {
	blobClient := p.container.NewBlockBlobClient(remotePath)

	uploadOptions := &blockblob.UploadStreamOptions{}

	// Set metadata (convert map[string]string to map[string]*string)
	if opts != nil && opts.Metadata != nil {
		metadata := make(map[string]*string)
		for k, v := range opts.Metadata {
			val := v
			metadata[k] = &val
		}
		uploadOptions.Metadata = metadata
	}

	// Set content type
	if opts != nil && opts.ContentType != "" {
		uploadOptions.HTTPHeaders = &blob.HTTPHeaders{
			BlobContentType: &opts.ContentType,
		}
	}

	// Upload the blob
	_, err := blobClient.UploadStream(ctx, reader, uploadOptions)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to upload to Azure")
	}

	return nil
}

// Download downloads a file from Azure.
func (p *AzureProvider) Download(ctx context.Context, remotePath, localPath string) error {
	// Create directory if needed
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create local directory")
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create local file")
	}
	defer file.Close()

	// Download from Azure
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

// DownloadStream downloads data to a reader.
func (p *AzureProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	blobClient := p.container.NewBlobClient(remotePath)

	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, "blob not found in Azure")
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to download from Azure")
	}

	return resp.Body, nil
}

// Delete deletes a file from Azure.
func (p *AzureProvider) Delete(ctx context.Context, remotePath string) error {
	blobClient := p.container.NewBlobClient(remotePath)

	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return pkgErrors.New(pkgErrors.ErrorTypeNotFound, "blob not found in Azure")
		}
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to delete from Azure")
	}

	return nil
}

// Exists checks if a file exists in Azure.
func (p *AzureProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	blobClient := p.container.NewBlobClient(remotePath)

	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return false, nil
		}
		return false, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to check Azure blob")
	}

	return true, nil
}

// GetMetadata retrieves file metadata.
func (p *AzureProvider) GetMetadata(ctx context.Context, remotePath string) (*st.FileMetadata, error) {
	blobClient := p.container.NewBlobClient(remotePath)

	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, "blob not found in Azure")
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to get Azure metadata")
	}

	// Convert metadata from map[string]*string to map[string]string
	stringMetadata := make(map[string]string)
	for k, v := range props.Metadata {
		if v != nil {
			stringMetadata[k] = *v
		}
	}

	metadata := &st.FileMetadata{
		Path:         remotePath,
		Size:         *props.ContentLength,
		LastModified: *props.LastModified,
		Metadata:     stringMetadata,
	}

	if props.ContentType != nil {
		metadata.ContentType = *props.ContentType
	}

	// Azure provides MD5 for integrity checking (not cryptographic security)
	// Note: Azure Blob Storage doesn't natively support SHA-256 checksums
	// For production use, consider computing and storing SHA-256 in blob metadata
	if props.ContentMD5 != nil {
		metadata.Checksum = fmt.Sprintf("md5:%x", props.ContentMD5)
	}

	return metadata, nil
}

// List lists files with a given prefix.
func (p *AzureProvider) List(ctx context.Context, prefix string) ([]*st.FileMetadata, error) {
	var files []*st.FileMetadata

	pager := p.container.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to list Azure blobs")
		}

		for _, item := range resp.Segment.BlobItems {
			// Skip directories
			if item.Name == nil || (*item.Name)[len(*item.Name)-1] == '/' {
				continue
			}

			files = append(files, blobItemToMetadata(item))
		}
	}

	return files, nil
}

// blobItemToMetadata converts an Azure blob list item into FileMetadata.
func blobItemToMetadata(item *container.BlobItem) *st.FileMetadata {
	metadata := &st.FileMetadata{
		Path: *item.Name,
	}

	if item.Properties != nil {
		if item.Properties.ContentLength != nil {
			metadata.Size = *item.Properties.ContentLength
		}
		if item.Properties.LastModified != nil {
			metadata.LastModified = *item.Properties.LastModified
		}
		if item.Properties.ContentType != nil {
			metadata.ContentType = *item.Properties.ContentType
		}
		// Azure provides MD5 for integrity checking (not cryptographic security)
		if item.Properties.ContentMD5 != nil {
			metadata.Checksum = fmt.Sprintf("md5:%x", item.Properties.ContentMD5)
		}
	}

	// Convert metadata from map[string]*string to map[string]string
	if item.Metadata != nil {
		stringMetadata := make(map[string]string)
		for k, v := range item.Metadata {
			if v != nil {
				stringMetadata[k] = *v
			}
		}
		metadata.Metadata = stringMetadata
	}

	return metadata
}

// GetType returns the provider type.
func (p *AzureProvider) GetType() st.ProviderType {
	return st.ProviderTypeAzure
}

// ValidateConfig validates the provider configuration.
func (p *AzureProvider) ValidateConfig() error {
	if p.config.Container == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure container is required")
	}
	if p.config.AccountName == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure account name is required")
	}
	if p.config.AccountKey == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "Azure account key is required")
	}
	return nil
}

// progressReader wraps a reader to track progress.
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
