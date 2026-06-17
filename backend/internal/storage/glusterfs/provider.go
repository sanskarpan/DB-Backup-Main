// Package glusterfs provides GlusterFS distributed file system storage backend
package glusterfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sanskarpan/db-backup/internal/storage"
)

// GlusterFSProvider implements storage provider for GlusterFS
type GlusterFSProvider struct {
	config *storage.GlusterFSConfig
}

// NewGlusterFSProvider creates a new GlusterFS storage provider
func NewGlusterFSProvider(cfg *storage.GlusterFSConfig) (*GlusterFSProvider, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: "invalid configuration",
			Err:     err,
		}
	}

	// Verify mount path exists
	if _, err := os.Stat(cfg.MountPath); os.IsNotExist(err) {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("GlusterFS mount path does not exist: %s", cfg.MountPath),
			Err:     err,
		}
	}

	return &GlusterFSProvider{
		config: cfg,
	}, nil
}

// Upload uploads a file to GlusterFS
func (p *GlusterFSProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	sourceFile, err := os.Open(localPath)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to open source file: %s", localPath),
			Err:     err,
		}
	}
	defer sourceFile.Close()

	return p.UploadStream(ctx, sourceFile, remotePath, opts)
}

// UploadStream uploads data from a reader to GlusterFS
func (p *GlusterFSProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	destPath := p.getFullPath(remotePath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to create directory: %s", filepath.Dir(destPath)),
			Err:     err,
		}
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to create destination file: %s", destPath),
			Err:     err,
		}
	}
	defer destFile.Close()

	// Copy data
	if _, err := io.Copy(destFile, reader); err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to copy data to GlusterFS: %s", destPath),
			Err:     err,
		}
	}

	// Set extended attributes if metadata is provided
	if opts != nil && opts.Metadata != nil {
		if err := p.setXattrs(destPath, opts.Metadata); err != nil {
			// Log error but don't fail the upload
			fmt.Printf("Warning: failed to set extended attributes: %v\n", err)
		}
	}

	return nil
}

// Download downloads a file from GlusterFS
func (p *GlusterFSProvider) Download(ctx context.Context, remotePath, localPath string) error {
	sourcePath := p.getFullPath(remotePath)

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to open source file in GlusterFS: %s", sourcePath),
			Err:     err,
		}
	}
	defer sourceFile.Close()

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to create local directory: %s", filepath.Dir(localPath)),
			Err:     err,
		}
	}

	destFile, err := os.Create(localPath)
	if err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to create destination file: %s", localPath),
			Err:     err,
		}
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to copy data from GlusterFS: %s", sourcePath),
			Err:     err,
		}
	}

	return nil
}

// DownloadStream downloads data to a reader
func (p *GlusterFSProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	sourcePath := p.getFullPath(remotePath)

	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to open file in GlusterFS: %s", sourcePath),
			Err:     err,
		}
	}

	return file, nil
}

// Delete deletes a file from GlusterFS
func (p *GlusterFSProvider) Delete(ctx context.Context, remotePath string) error {
	fullPath := p.getFullPath(remotePath)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to delete file from GlusterFS: %s", fullPath),
			Err:     err,
		}
	}

	return nil
}

// Exists checks if a file exists in GlusterFS
func (p *GlusterFSProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	fullPath := p.getFullPath(remotePath)

	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to check file existence in GlusterFS: %s", fullPath),
			Err:     err,
		}
	}

	return true, nil
}

// GetMetadata retrieves file metadata
func (p *GlusterFSProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	fullPath := p.getFullPath(remotePath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to get metadata from GlusterFS: %s", fullPath),
			Err:     err,
		}
	}

	// Get extended attributes
	xattrs, err := p.getXattrs(fullPath)
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to get extended attributes: %v\n", err)
	}

	metadata := &storage.FileMetadata{
		Path:         remotePath,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		Metadata:     xattrs,
	}

	return metadata, nil
}

// List lists files with a given prefix
func (p *GlusterFSProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	fullPath := p.getFullPath(prefix)

	var files []*storage.FileMetadata

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// Get relative path
			relPath, err := filepath.Rel(p.getBasePath(), path)
			if err != nil {
				return err
			}

			files = append(files, &storage.FileMetadata{
				Path:         relPath,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, &storage.ProviderError{
			Type:    storage.ProviderTypeGlusterFS,
			Message: fmt.Sprintf("failed to list files in GlusterFS: %s", fullPath),
			Err:     err,
		}
	}

	return files, nil
}

// GetType returns the provider type
func (p *GlusterFSProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeGlusterFS
}

// ValidateConfig validates the provider configuration
func (p *GlusterFSProvider) ValidateConfig() error {
	return validateConfig(p.config)
}

// GetVolumeInfo returns information about the GlusterFS volume
func (p *GlusterFSProvider) GetVolumeInfo(ctx context.Context) (*VolumeInfo, error) {
	// This would typically call gluster volume info command
	// For now, return basic info based on config
	return &VolumeInfo{
		Name:        p.config.VolumeName,
		Host:        p.config.VolumeHost,
		MountPath:   p.config.MountPath,
		Replication: p.config.Replication,
	}, nil
}

// GetBrickStatus returns the status of GlusterFS bricks
func (p *GlusterFSProvider) GetBrickStatus(ctx context.Context) ([]BrickStatus, error) {
	// This would typically call gluster volume status command
	// Placeholder implementation
	return []BrickStatus{
		{
			Host:   p.config.VolumeHost,
			Status: "online",
		},
	}, nil
}

// Helper functions

func (p *GlusterFSProvider) getBasePath() string {
	return filepath.Join(p.config.MountPath, p.config.SubPath)
}

func (p *GlusterFSProvider) getFullPath(remotePath string) string {
	return filepath.Join(p.getBasePath(), remotePath)
}

func (p *GlusterFSProvider) setXattrs(path string, xattrs map[string]string) error {
	// Placeholder - implement extended attributes support
	// This would use syscall or golang.org/x/sys/unix for xattr support
	return nil
}

func (p *GlusterFSProvider) getXattrs(path string) (map[string]string, error) {
	// Placeholder - implement extended attributes retrieval
	return make(map[string]string), nil
}

func validateConfig(cfg *storage.GlusterFSConfig) error {
	if cfg == nil {
		return fmt.Errorf("GlusterFS config is required")
	}
	if cfg.VolumeHost == "" {
		return fmt.Errorf("GlusterFS volume host is required")
	}
	if cfg.VolumeName == "" {
		return fmt.Errorf("GlusterFS volume name is required")
	}
	if cfg.MountPath == "" {
		return fmt.Errorf("GlusterFS mount path is required")
	}
	if cfg.Replication < 1 {
		cfg.Replication = 3 // Default replication factor
	}
	return nil
}

// VolumeInfo contains information about a GlusterFS volume
type VolumeInfo struct {
	Name        string
	Host        string
	MountPath   string
	Replication int
}

// BrickStatus contains status information about a GlusterFS brick
type BrickStatus struct {
	Host   string
	Path   string
	Status string
}
