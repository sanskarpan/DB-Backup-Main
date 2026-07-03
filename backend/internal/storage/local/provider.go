// Package local provides local filesystem storage
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// LocalProvider implements local filesystem storage.
//
//nolint:revive // keeps public name stable; renaming would break other packages
type LocalProvider struct {
	config *storage.LocalConfig
}

// NewLocalProvider creates a new local storage provider.
func NewLocalProvider(config *storage.LocalConfig) (*LocalProvider, error) {
	// Ensure storage directory exists
	if err := os.MkdirAll(config.Path, 0o755); err != nil {
		return nil, pkgErrors.ErrStorageUpload(err)
	}

	return &LocalProvider{
		config: config,
	}, nil
}

// Upload uploads a file to local storage.
func (p *LocalProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	// Open source file
	srcFile, err := os.Open(localPath)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}
	defer srcFile.Close()

	// Get file info for progress callback
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	// Create destination path
	destPath := filepath.Join(p.config.Path, remotePath)

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err = os.MkdirAll(destDir, 0o755); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}
	defer destFile.Close()

	// Copy with progress tracking
	if opts != nil && opts.ProgressCallback != nil {
		reader := &progressReader{
			reader:   srcFile,
			total:    fileInfo.Size(),
			callback: opts.ProgressCallback,
		}
		_, err = io.Copy(destFile, reader)
	} else {
		_, err = io.Copy(destFile, srcFile)
	}

	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	return nil
}

// UploadStream uploads data from a reader to local storage.
func (p *LocalProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	destPath := filepath.Join(p.config.Path, remotePath)

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}
	defer destFile.Close()

	// Copy data
	_, err = io.Copy(destFile, reader)
	if err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	return nil
}

// Download downloads a file from local storage.
func (p *LocalProvider) Download(ctx context.Context, remotePath, localPath string) error {
	srcPath := filepath.Join(p.config.Path, remotePath)

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}
	defer srcFile.Close()

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err = os.MkdirAll(localDir, 0o755); err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}

	// Create destination file
	destFile, err := os.Create(localPath)
	if err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}
	defer destFile.Close()

	// Copy data
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}

	return nil
}

// DownloadStream downloads data to a reader.
func (p *LocalProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	srcPath := filepath.Join(p.config.Path, remotePath)

	file, err := os.Open(srcPath)
	if err != nil {
		return nil, pkgErrors.ErrStorageDownload(err)
	}

	return file, nil
}

// Delete deletes a file from local storage.
func (p *LocalProvider) Delete(ctx context.Context, remotePath string) error {
	filePath := filepath.Join(p.config.Path, remotePath)

	if err := os.Remove(filePath); err != nil {
		return pkgErrors.ErrStorageUpload(err).WithMetadata("operation", "delete")
	}

	return nil
}

// Exists checks if a file exists in local storage.
func (p *LocalProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	filePath := filepath.Join(p.config.Path, remotePath)

	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// GetMetadata retrieves file metadata.
func (p *LocalProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	filePath := filepath.Join(p.config.Path, remotePath)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, pkgErrors.ErrStorageDownload(err)
	}

	metadata := &storage.FileMetadata{
		Path:         remotePath,
		Size:         fileInfo.Size(),
		LastModified: fileInfo.ModTime(),
		ContentType:  "application/octet-stream",
		Metadata:     make(map[string]string),
	}

	return metadata, nil
}

// List lists files with a given prefix.
func (p *LocalProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	searchPath := filepath.Join(p.config.Path, prefix)

	var files []*storage.FileMetadata

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(p.config.Path, path)
		if err != nil {
			return err
		}

		metadata := &storage.FileMetadata{
			Path:         relPath,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			ContentType:  "application/octet-stream",
		}

		files = append(files, metadata)
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return files, nil
}

// GetType returns the provider type.
func (p *LocalProvider) GetType() storage.ProviderType {
	return storage.ProviderTypeLocal
}

// ValidateConfig validates the provider configuration.
func (p *LocalProvider) ValidateConfig() error {
	if p.config.Path == "" {
		return fmt.Errorf("local storage path is required")
	}

	// Try to create directory if it doesn't exist
	if err := os.MkdirAll(p.config.Path, 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
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

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)

	if r.callback != nil {
		r.callback(r.read, r.total)
	}

	return n, err
}
