package nfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
)

func TestNewNFSProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    *NFSConfig
		expectErr bool
	}{
		{
			name:      "nil config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "valid config with defaults",
			config: &NFSConfig{
				Server: "192.168.1.100",
				Export: "/export/backups",
			},
			expectErr: false,
		},
		{
			name: "valid config with custom mount point",
			config: &NFSConfig{
				Server:     "192.168.1.100",
				Export:     "/export/backups",
				MountPoint: "/mnt/custom-nfs",
				Version:    "4.1",
				AutoMount:  false,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewNFSProvider(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if provider == nil {
				t.Fatal("Provider should not be nil")
			}

			// Check defaults
			if tt.config.Version == "" && provider.config.Version != "4" {
				t.Errorf("Expected default version 4, got %s", provider.config.Version)
			}

			if tt.config.Timeout == 0 && provider.config.Timeout != 30*time.Second {
				t.Errorf("Expected default timeout 30s, got %v", provider.config.Timeout)
			}

			if tt.config.MountPoint == "" && provider.mountPoint == "" {
				t.Error("Mount point should be set")
			}
		})
	}
}

func TestNFSProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *NFSConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &NFSConfig{
				Server: "192.168.1.100",
				Export: "/export/backups",
			},
			expectErr: false,
		},
		{
			name: "missing server",
			config: &NFSConfig{
				Export: "/export/backups",
			},
			expectErr: true,
			errMsg:    "NFS server is required",
		},
		{
			name: "missing export",
			config: &NFSConfig{
				Server: "192.168.1.100",
			},
			expectErr: true,
			errMsg:    "NFS export path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &NFSProvider{
				config: tt.config,
			}

			err := provider.ValidateConfig()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNFSProvider_GetType(t *testing.T) {
	provider := &NFSProvider{}

	providerType := provider.GetType()
	if providerType != "nfs" {
		t.Errorf("Expected provider type 'nfs', got '%s'", providerType)
	}
}

func TestBuildMountArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   *NFSConfig
		nfsPath  string
		expected []string
	}{
		{
			name: "basic mount",
			config: &NFSConfig{
				Server:     "192.168.1.100",
				Export:     "/export/backups",
				Version:    "4",
				MountPoint: "/mnt/nfs",
			},
			nfsPath: "192.168.1.100:/export/backups",
			expected: []string{
				"-t", "nfs",
				"-o", "vers=4,rw",
				"192.168.1.100:/export/backups",
				"/mnt/nfs",
			},
		},
		{
			name: "with custom options",
			config: &NFSConfig{
				Server:     "192.168.1.100",
				Export:     "/export/backups",
				Version:    "4.1",
				MountPoint: "/mnt/nfs",
				Options: map[string]string{
					"hard":  "",
					"sync":  "",
					"proto": "tcp",
				},
			},
			nfsPath: "192.168.1.100:/export/backups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &NFSProvider{
				config:     tt.config,
				mountPoint: tt.config.MountPoint,
			}

			args := provider.buildMountArgs(tt.nfsPath)

			// Check that basic elements are present
			if args[0] != "-t" || args[1] != "nfs" {
				t.Error("Expected -t nfs in mount args")
			}

			// Find -o index
			oIndex := -1
			for i, arg := range args {
				if arg == "-o" {
					oIndex = i
					break
				}
			}

			if oIndex == -1 {
				t.Error("Expected -o option in mount args")
			}

			// Check that NFS path and mount point are at the end
			if args[len(args)-2] != tt.nfsPath {
				t.Errorf("Expected NFS path %s, got %s", tt.nfsPath, args[len(args)-2])
			}

			if args[len(args)-1] != tt.config.MountPoint {
				t.Errorf("Expected mount point %s, got %s", tt.config.MountPoint, args[len(args)-1])
			}
		})
	}
}

func TestNFSProvider_UploadDownload(t *testing.T) {
	// Create a temporary directory to simulate mount point
	tempDir, err := os.MkdirTemp("", "nfs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider := &NFSProvider{
		config: &NFSConfig{
			Server:     "192.168.1.100",
			Export:     "/export/backups",
			MountPoint: tempDir,
		},
		mountPoint: tempDir,
		isMounted:  true, // Simulate mounted state
	}

	ctx := context.Background()

	// Test Upload
	t.Run("Upload", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "source.txt")
		testContent := []byte("test content for NFS upload")
		if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		remotePath := "backups/test-upload.txt"
		err := provider.Upload(ctx, testFile, remotePath, nil)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		// Verify file was uploaded
		uploadedPath := filepath.Join(tempDir, remotePath)
		if _, err := os.Stat(uploadedPath); os.IsNotExist(err) {
			t.Error("Uploaded file does not exist")
		}

		// Verify content
		content, err := os.ReadFile(uploadedPath)
		if err != nil {
			t.Fatalf("Failed to read uploaded file: %v", err)
		}

		if string(content) != string(testContent) {
			t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
		}
	})

	// Test Download
	t.Run("Download", func(t *testing.T) {
		// Create a remote file
		remotePath := "backups/test-download.txt"
		remoteFile := filepath.Join(tempDir, remotePath)
		testContent := []byte("test content for NFS download")

		if err := os.MkdirAll(filepath.Dir(remoteFile), 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		if err := os.WriteFile(remoteFile, testContent, 0o644); err != nil {
			t.Fatalf("Failed to create remote file: %v", err)
		}

		// Download to local path
		localPath := filepath.Join(tempDir, "downloaded.txt")
		err := provider.Download(ctx, remotePath, localPath)
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}

		// Verify download
		content, err := os.ReadFile(localPath)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		if string(content) != string(testContent) {
			t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
		}
	})

	// Test Exists
	t.Run("Exists", func(t *testing.T) {
		remotePath := "backups/test-download.txt"

		exists, err := provider.Exists(ctx, remotePath)
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}

		if !exists {
			t.Error("File should exist")
		}

		nonExistentPath := "nonexistent/file.txt"
		exists, err = provider.Exists(ctx, nonExistentPath)
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}

		if exists {
			t.Error("File should not exist")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		remotePath := "backups/test-delete.txt"
		remoteFile := filepath.Join(tempDir, remotePath)
		testContent := []byte("test content for deletion")

		if err := os.WriteFile(remoteFile, testContent, 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err := provider.Delete(ctx, remotePath)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify file was deleted
		if _, err := os.Stat(remoteFile); !os.IsNotExist(err) {
			t.Error("File should have been deleted")
		}
	})

	// Test GetMetadata
	t.Run("GetMetadata", func(t *testing.T) {
		remotePath := "backups/test-metadata.txt"
		remoteFile := filepath.Join(tempDir, remotePath)
		testContent := []byte("test content for metadata")

		if err := os.WriteFile(remoteFile, testContent, 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		metadata, err := provider.GetMetadata(ctx, remotePath)
		if err != nil {
			t.Fatalf("GetMetadata failed: %v", err)
		}

		if metadata.Path != remotePath {
			t.Errorf("Expected path %s, got %s", remotePath, metadata.Path)
		}

		if metadata.Size != int64(len(testContent)) {
			t.Errorf("Expected size %d, got %d", len(testContent), metadata.Size)
		}

		if metadata.ContentType != "application/octet-stream" {
			t.Errorf("Expected content type 'application/octet-stream', got %s", metadata.ContentType)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		// Create multiple files
		prefix := "backups/list-test"
		testFiles := []string{
			"file1.txt",
			"file2.txt",
			"subdir/file3.txt",
		}

		for _, file := range testFiles {
			fullPath := filepath.Join(tempDir, prefix, file)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
		}

		files, err := provider.List(ctx, prefix)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(files) != len(testFiles) {
			t.Errorf("Expected %d files, got %d", len(testFiles), len(files))
		}
	})
}

func TestNFSProvider_UploadWithProgress(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nfs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider := &NFSProvider{
		config: &NFSConfig{
			Server:     "192.168.1.100",
			Export:     "/export/backups",
			MountPoint: tempDir,
		},
		mountPoint: tempDir,
		isMounted:  true,
	}

	ctx := context.Background()

	// Create a test file
	testFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content with progress tracking")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var progressCalled bool
	var uploadedBytes int64

	opts := &storage.UploadOptions{
		ProgressCallback: func(uploaded, total int64) {
			progressCalled = true
			uploadedBytes = uploaded
		},
	}

	remotePath := "backups/test-progress.txt"
	err = provider.Upload(ctx, testFile, remotePath, opts)
	if err != nil {
		t.Fatalf("Upload with progress failed: %v", err)
	}

	if !progressCalled {
		t.Error("Progress callback should have been called")
	}

	if uploadedBytes != int64(len(testContent)) {
		t.Errorf("Expected %d bytes uploaded, got %d", len(testContent), uploadedBytes)
	}
}

func TestNFSProvider_Close(t *testing.T) {
	t.Run("with auto cleanup", func(t *testing.T) {
		provider := &NFSProvider{
			config: &NFSConfig{
				Server:      "192.168.1.100",
				Export:      "/export/backups",
				AutoUnmount: true,
			},
			autoCleanup: true,
			isMounted:   false,
		}

		err := provider.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("without auto cleanup", func(t *testing.T) {
		provider := &NFSProvider{
			config: &NFSConfig{
				Server:      "192.168.1.100",
				Export:      "/export/backups",
				AutoUnmount: false,
			},
			autoCleanup: false,
			isMounted:   true,
		}

		err := provider.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}

		// Mount should still be active
		if !provider.isMounted {
			t.Error("Mount should still be active when auto cleanup is disabled")
		}
	})
}

func TestNFSVersions(t *testing.T) {
	versions := []string{"3", "4", "4.1", "4.2"}

	for _, version := range versions {
		t.Run("NFS v"+version, func(t *testing.T) {
			config := &NFSConfig{
				Server:  "192.168.1.100",
				Export:  "/export/backups",
				Version: version,
			}

			provider, err := NewNFSProvider(config)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			if provider.config.Version != version {
				t.Errorf("Expected version %s, got %s", version, provider.config.Version)
			}

			args := provider.buildMountArgs("192.168.1.100:/export/backups")

			// Find the options string
			var optionsStr string
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					optionsStr = args[i+1]
					break
				}
			}

			expectedVersion := "vers=" + version
			if !containsOption(optionsStr, expectedVersion) {
				t.Errorf("Expected options to contain '%s', got: %s", expectedVersion, optionsStr)
			}
		})
	}
}

func containsOption(optionsStr, option string) bool {
	if optionsStr == option {
		return true
	}
	// Check if it's in a comma-separated list
	for i := 0; i <= len(optionsStr)-len(option); i++ {
		if i+len(option) <= len(optionsStr) && optionsStr[i:i+len(option)] == option {
			// Check if it's a complete option (not a substring)
			if (i == 0 || optionsStr[i-1] == ',') &&
				(i+len(option) == len(optionsStr) || optionsStr[i+len(option)] == ',') {
				return true
			}
		}
	}
	return false
}
