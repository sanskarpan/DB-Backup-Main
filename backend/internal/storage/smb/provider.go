// Package smb provides SMB/CIFS storage support
package smb

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

// SMBProvider implements SMB/CIFS storage
type SMBProvider struct {
	config      *SMBConfig
	mountPoint  string
	isMounted   bool
	autoCleanup bool
}

// SMBConfig holds SMB/CIFS configuration
type SMBConfig struct {
	Server       string            // SMB server address (e.g., "//192.168.1.100/share")
	Share        string            // Share name (e.g., "backups")
	Username     string            // SMB username
	Password     string            // SMB password
	Domain       string            // Windows domain (optional)
	MountPoint   string            // Local mount point (e.g., "/mnt/smb-backups")
	Version      string            // SMB version ("1.0", "2.0", "2.1", "3.0", "3.1.1")
	Options      map[string]string // Mount options
	Timeout      time.Duration     // Mount timeout
	AutoMount    bool              // Automatically mount on initialization
	AutoUnmount  bool              // Automatically unmount on close
	UseKerberos  bool              // Use Kerberos authentication
	DirectIO     bool              // Use direct I/O (no caching)
	FileMode     string            // File permissions (e.g., "0644")
	DirMode      string            // Directory permissions (e.g., "0755")
}

// NewSMBProvider creates a new SMB/CIFS storage provider
func NewSMBProvider(config *SMBConfig) (*SMBProvider, error) {
	if config == nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeConfiguration, "SMB config is required")
	}

	// Set defaults
	if config.Version == "" {
		config.Version = "3.0"
	}

	if config.MountPoint == "" {
		// Create temporary mount point
		config.MountPoint = filepath.Join("/tmp", fmt.Sprintf("smb-mount-%d", time.Now().Unix()))
		config.AutoUnmount = true
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.FileMode == "" {
		config.FileMode = "0644"
	}

	if config.DirMode == "" {
		config.DirMode = "0755"
	}

	provider := &SMBProvider{
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

// Mount mounts the SMB share
func (p *SMBProvider) Mount(ctx context.Context) error {
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

	// Build SMB path
	var smbPath string
	if strings.HasPrefix(p.config.Server, "//") {
		smbPath = fmt.Sprintf("%s/%s", p.config.Server, p.config.Share)
	} else {
		smbPath = fmt.Sprintf("//%s/%s", p.config.Server, p.config.Share)
	}

	// Build mount command arguments
	args := p.buildMountArgs(smbPath)

	// Execute mount command
	cmd := exec.CommandContext(ctx, "mount", args...)

	// Set environment for credentials if not using credentials file
	if p.config.Username != "" && p.config.Password != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("PASSWD=%s", p.config.Password))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to mount SMB: %v, output: %s", err, string(output)))
	}

	p.isMounted = true
	return nil
}

// Unmount unmounts the SMB share
func (p *SMBProvider) Unmount(ctx context.Context) error {
	if !p.isMounted {
		return nil // Not mounted
	}

	cmd := exec.CommandContext(ctx, "umount", p.mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to unmount SMB: %v, output: %s", err, string(output)))
	}

	p.isMounted = false

	// Remove temporary mount point if auto-cleanup is enabled
	if p.autoCleanup && strings.HasPrefix(p.mountPoint, "/tmp/smb-mount-") {
		os.Remove(p.mountPoint)
	}

	return nil
}

// Upload uploads a file to SMB storage
func (p *SMBProvider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
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

	// Sync to ensure data is written
	if err := destFile.Sync(); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	return nil
}

// UploadStream uploads data from a reader to SMB storage
func (p *SMBProvider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
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

// Download downloads a file from SMB storage
func (p *SMBProvider) Download(ctx context.Context, remotePath, localPath string) error {
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
func (p *SMBProvider) DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error) {
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

// Delete deletes a file from SMB storage
func (p *SMBProvider) Delete(ctx context.Context, remotePath string) error {
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

// Exists checks if a file exists in SMB storage
func (p *SMBProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
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
func (p *SMBProvider) GetMetadata(ctx context.Context, remotePath string) (*storage.FileMetadata, error) {
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
func (p *SMBProvider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
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
func (p *SMBProvider) GetType() storage.ProviderType {
	return "smb"
}

// ValidateConfig validates the provider configuration
func (p *SMBProvider) ValidateConfig() error {
	if p.config.Server == "" {
		return fmt.Errorf("SMB server is required")
	}

	if p.config.Share == "" {
		return fmt.Errorf("SMB share name is required")
	}

	// Check if mount command is available
	if _, err := exec.LookPath("mount"); err != nil {
		return fmt.Errorf("mount command not found - SMB support requires mount.cifs utility")
	}

	return nil
}

// Close closes the provider and unmounts if configured
func (p *SMBProvider) Close() error {
	if p.autoCleanup && p.isMounted {
		return p.Unmount(context.Background())
	}
	return nil
}

// Helper methods

func (p *SMBProvider) buildMountArgs(smbPath string) []string {
	args := []string{
		"-t", "cifs",
	}

	// Build options
	opts := []string{
		fmt.Sprintf("vers=%s", p.config.Version),
	}

	// Add authentication
	if p.config.Username != "" {
		opts = append(opts, fmt.Sprintf("username=%s", p.config.Username))

		// Use credentials option if password is provided
		if p.config.Password != "" {
			// Create credentials file for security
			credsFile, err := p.createCredentialsFile()
			if err == nil {
				opts = append(opts, fmt.Sprintf("credentials=%s", credsFile))
			} else {
				// Fallback to password in options (less secure)
				opts = append(opts, fmt.Sprintf("password=%s", p.config.Password))
			}
		}
	}

	// Add domain if specified
	if p.config.Domain != "" {
		opts = append(opts, fmt.Sprintf("domain=%s", p.config.Domain))
	}

	// Add file/dir modes
	if p.config.FileMode != "" {
		opts = append(opts, fmt.Sprintf("file_mode=%s", p.config.FileMode))
	}
	if p.config.DirMode != "" {
		opts = append(opts, fmt.Sprintf("dir_mode=%s", p.config.DirMode))
	}

	// Add Kerberos if enabled
	if p.config.UseKerberos {
		opts = append(opts, "sec=krb5")
	}

	// Add direct I/O if enabled
	if p.config.DirectIO {
		opts = append(opts, "directio")
	}

	// Add custom options
	for key, value := range p.config.Options {
		if value != "" {
			opts = append(opts, fmt.Sprintf("%s=%s", key, value))
		} else {
			opts = append(opts, key)
		}
	}

	if len(opts) > 0 {
		args = append(args, "-o", strings.Join(opts, ","))
	}

	// Add SMB path and mount point
	args = append(args, smbPath, p.mountPoint)

	return args
}

func (p *SMBProvider) createCredentialsFile() (string, error) {
	// Create temporary credentials file
	tmpFile, err := os.CreateTemp("", "smb-creds-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write credentials
	content := fmt.Sprintf("username=%s\npassword=%s\n", p.config.Username, p.config.Password)
	if p.config.Domain != "" {
		content += fmt.Sprintf("domain=%s\n", p.config.Domain)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	// Set restrictive permissions
	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func (p *SMBProvider) checkMounted() (bool, error) {
	// Read /proc/mounts to check if already mounted
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		// /proc/mounts might not exist on non-Linux systems
		// Try using mount command
		cmd := exec.Command("mount")
		output, err := cmd.Output()
		if err != nil {
			return false, nil
		}
		data = output
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
