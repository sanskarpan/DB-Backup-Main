// Package nfs provides NFS (Network File System) storage support
package nfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// NFSProvider implements NFS storage
type NFSProvider struct {
	config      *NFSConfig
	mountPoint  string
	isMounted   bool
	autoCleanup bool
}

// NFSConfig holds NFS configuration
type NFSConfig struct {
	Server      string            // NFS server address (e.g., "192.168.1.100")
	Export      string            // NFS export path (e.g., "/export/backups")
	MountPoint  string            // Local mount point (e.g., "/mnt/nfs-backups")
	Version     string            // NFS version ("3", "4", "4.1", "4.2")
	Options     map[string]string // Mount options (e.g., "rw", "sync", "hard")
	Timeout     time.Duration     // Mount timeout
	AutoMount   bool              // Automatically mount on initialization
	AutoUnmount bool              // Automatically unmount on close
}

// NewNFSProvider creates a new NFS storage provider
func NewNFSProvider(config *NFSConfig) (*NFSProvider, error) {
	if config == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "NFS config is required")
	}

	// Set defaults
	if config.Version == "" {
		config.Version = "4"
	}

	if config.MountPoint == "" {
		// Create temporary mount point
		config.MountPoint = filepath.Join("/tmp", fmt.Sprintf("nfs-mount-%d", time.Now().Unix()))
		config.AutoUnmount = true
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	provider := &NFSProvider{
		config:      config,
		mountPoint:  config.MountPoint,
		autoCleanup: config.AutoUnmount,
	}

	// Auto-mount if configured
	if config.AutoMount {
		if err := provider.Mount(context.Background()); err != nil {
			return nil, err
		}
	}

	return provider, nil
}

// Mount mounts the NFS share
func (p *NFSProvider) Mount(ctx context.Context) error {
	if p.isMounted {
		return nil // Already mounted
	}

	// Create mount point if it doesn't exist
	if err := os.MkdirAll(p.mountPoint, 0755); err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage, fmt.Sprintf("failed to create mount point: %v", err))
	}

	// Check if already mounted
	if mounted, err := p.checkMounted(); err != nil {
		return err
	} else if mounted {
		p.isMounted = true
		return nil
	}

	// Build mount command
	nfsPath := fmt.Sprintf("%s:%s", p.config.Server, p.config.Export)
	args := p.buildMountArgs(nfsPath)

	// Execute mount command
	cmd := exec.CommandContext(ctx, "mount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to mount NFS: %v, output: %s", err, string(output)))
	}

	p.isMounted = true
	return nil
}

// Unmount unmounts the NFS share
func (p *NFSProvider) Unmount(ctx context.Context) error {
	if !p.isMounted {
		return nil // Not mounted
	}

	cmd := exec.CommandContext(ctx, "umount", p.mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to unmount NFS: %v, output: %s", err, string(output)))
	}

	p.isMounted = false

	// Remove temporary mount point if auto-cleanup is enabled
	if p.autoCleanup && strings.HasPrefix(p.mountPoint, "/tmp/nfs-mount-") {
		os.Remove(p.mountPoint)
	}

	return nil
}

// Upload uploads a file to NFS storage
func (p *NFSProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return err
		}
	}

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
	destPath := filepath.Join(p.mountPoint, remotePath)

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
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

	// Sync to ensure data is written (important for NFS)
	if err := destFile.Sync(); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	return nil
}

// UploadStream uploads data from a reader to NFS storage
func (p *NFSProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return err
		}
	}

	destPath := filepath.Join(p.mountPoint, remotePath)

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
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

	// Sync to ensure data is written
	if err := destFile.Sync(); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	return nil
}

// Download downloads a file from NFS storage
func (p *NFSProvider) Download(ctx context.Context, remotePath, localPath string) error {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return err
		}
	}

	srcPath := filepath.Join(p.mountPoint, remotePath)

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return pkgErrors.ErrStorageDownload(err)
	}
	defer srcFile.Close()

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
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

// DownloadStream downloads data to a reader
func (p *NFSProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return nil, err
		}
	}

	srcPath := filepath.Join(p.mountPoint, remotePath)

	file, err := os.Open(srcPath)
	if err != nil {
		return nil, pkgErrors.ErrStorageDownload(err)
	}

	return file, nil
}

// Delete deletes a file from NFS storage
func (p *NFSProvider) Delete(ctx context.Context, remotePath string) error {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return err
		}
	}

	filePath := filepath.Join(p.mountPoint, remotePath)

	if err := os.Remove(filePath); err != nil {
		return pkgErrors.ErrStorageUpload(err).WithMetadata("operation", "delete")
	}

	return nil
}

// Exists checks if a file exists in NFS storage
func (p *NFSProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return false, err
		}
	}

	filePath := filepath.Join(p.mountPoint, remotePath)

	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// GetMetadata retrieves file metadata
func (p *NFSProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return nil, err
		}
	}

	filePath := filepath.Join(p.mountPoint, remotePath)

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

// List lists files with a given prefix
func (p *NFSProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
	if !p.isMounted {
		if err := p.Mount(ctx); err != nil {
			return nil, err
		}
	}

	searchPath := filepath.Join(p.mountPoint, prefix)

	var files []*storage.FileMetadata

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(p.mountPoint, path)
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

// GetType returns the provider type
func (p *NFSProvider) GetType() storage.ProviderType {
	return "nfs"
}

// ValidateConfig validates the provider configuration
func (p *NFSProvider) ValidateConfig() error {
	if p.config.Server == "" {
		return fmt.Errorf("NFS server is required")
	}

	if p.config.Export == "" {
		return fmt.Errorf("NFS export path is required")
	}

	// Check if mount command is available
	if _, err := exec.LookPath("mount"); err != nil {
		return fmt.Errorf("mount command not found - NFS support requires mount utility")
	}

	return nil
}

// Close closes the provider and unmounts if configured
func (p *NFSProvider) Close() error {
	if p.autoCleanup && p.isMounted {
		return p.Unmount(context.Background())
	}
	return nil
}

// Helper methods

func (p *NFSProvider) buildMountArgs(nfsPath string) []string {
	args := []string{
		"-t", "nfs",
	}

	// Build options
	opts := []string{
		fmt.Sprintf("vers=%s", p.config.Version),
	}

	// Add custom options
	for key, value := range p.config.Options {
		if value != "" {
			opts = append(opts, fmt.Sprintf("%s=%s", key, value))
		} else {
			opts = append(opts, key)
		}
	}

	// Add default options if not specified
	if _, hasRW := p.config.Options["rw"]; !hasRW {
		opts = append(opts, "rw")
	}

	if len(opts) > 0 {
		args = append(args, "-o", strings.Join(opts, ","))
	}

	// Add NFS path and mount point
	args = append(args, nfsPath, p.mountPoint)

	return args
}

func (p *NFSProvider) checkMounted() (bool, error) {
	// Read /proc/mounts to check if already mounted
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		// /proc/mounts might not exist on non-Linux systems
		return false, nil
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == p.mountPoint {
			return true, nil
		}
	}

	return false, nil
}

// progressReader wraps a reader to track progress
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
