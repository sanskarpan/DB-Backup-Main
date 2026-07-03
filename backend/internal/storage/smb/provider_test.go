package smb

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/storage"
)

func TestNewSMBProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    *SMBConfig
		expectErr bool
	}{
		{
			name:      "nil config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "valid config with defaults",
			config: &SMBConfig{
				Server: "//192.168.1.100/share",
				Share:  "backups",
			},
			expectErr: false,
		},
		{
			name: "valid config with custom settings",
			config: &SMBConfig{
				Server:     "192.168.1.100",
				Share:      "backups",
				Username:   "testuser",
				Password:   "testpass",
				Domain:     "TESTDOMAIN",
				MountPoint: "/mnt/custom-smb",
				Version:    "3.1.1",
				AutoMount:  false,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSMBProvider(tt.config)

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

			verifyProviderDefaults(t, tt.config, provider)
		})
	}
}

// verifyProviderDefaults asserts that unset config fields were populated with
// the expected default values.
func verifyProviderDefaults(t *testing.T, config *SMBConfig, provider *SMBProvider) {
	t.Helper()

	if config.Version == "" && provider.config.Version != "3.0" {
		t.Errorf("Expected default version 3.0, got %s", provider.config.Version)
	}

	if config.Timeout == 0 && provider.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", provider.config.Timeout)
	}

	if config.FileMode == "" && provider.config.FileMode != "0644" {
		t.Errorf("Expected default file mode 0644, got %s", provider.config.FileMode)
	}

	if config.DirMode == "" && provider.config.DirMode != "0755" {
		t.Errorf("Expected default dir mode 0755, got %s", provider.config.DirMode)
	}

	if config.MountPoint == "" && provider.mountPoint == "" {
		t.Error("Mount point should be set")
	}
}

func TestSMBProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *SMBConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &SMBConfig{
				Server: "//192.168.1.100/share",
				Share:  "backups",
			},
			expectErr: false,
		},
		{
			name: "missing server",
			config: &SMBConfig{
				Share: "backups",
			},
			expectErr: true,
			errMsg:    "SMB server is required",
		},
		{
			name: "missing share",
			config: &SMBConfig{
				Server: "//192.168.1.100/share",
			},
			expectErr: true,
			errMsg:    "SMB share name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &SMBProvider{
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
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSMBProvider_GetType(t *testing.T) {
	provider := &SMBProvider{}

	providerType := provider.GetType()
	if providerType != "smb" {
		t.Errorf("Expected provider type 'smb', got '%s'", providerType)
	}
}

func TestBuildMountArgs(t *testing.T) {
	tests := []struct {
		name    string
		config  *SMBConfig
		smbPath string
	}{
		{
			name: "basic mount",
			config: &SMBConfig{
				Server:     "//192.168.1.100/share",
				Share:      "backups",
				MountPoint: "/mnt/smb",
				Version:    "3.0",
			},
			smbPath: "//192.168.1.100/share/backups",
		},
		{
			name: "with credentials",
			config: &SMBConfig{
				Server:     "//192.168.1.100/share",
				Share:      "backups",
				Username:   "testuser",
				Password:   "testpass",
				MountPoint: "/mnt/smb",
				Version:    "3.0",
			},
			smbPath: "//192.168.1.100/share/backups",
		},
		{
			name: "with domain",
			config: &SMBConfig{
				Server:     "//192.168.1.100/share",
				Share:      "backups",
				Username:   "testuser",
				Password:   "testpass",
				Domain:     "WORKGROUP",
				MountPoint: "/mnt/smb",
				Version:    "3.0",
			},
			smbPath: "//192.168.1.100/share/backups",
		},
		{
			name: "with Kerberos",
			config: &SMBConfig{
				Server:      "//192.168.1.100/share",
				Share:       "backups",
				MountPoint:  "/mnt/smb",
				Version:     "3.0",
				UseKerberos: true,
			},
			smbPath: "//192.168.1.100/share/backups",
		},
		{
			name: "with DirectIO",
			config: &SMBConfig{
				Server:     "//192.168.1.100/share",
				Share:      "backups",
				MountPoint: "/mnt/smb",
				Version:    "3.0",
				DirectIO:   true,
			},
			smbPath: "//192.168.1.100/share/backups",
		},
		{
			name: "with custom file and dir modes",
			config: &SMBConfig{
				Server:     "//192.168.1.100/share",
				Share:      "backups",
				MountPoint: "/mnt/smb",
				Version:    "3.0",
				FileMode:   "0600",
				DirMode:    "0700",
			},
			smbPath: "//192.168.1.100/share/backups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &SMBProvider{
				config:     tt.config,
				mountPoint: tt.config.MountPoint,
			}

			args := provider.buildMountArgs(tt.smbPath)
			verifyMountArgs(t, tt.config, tt.smbPath, args)
		})
	}
}

// mountOptionsString returns the value following the "-o" flag in mount args.
func mountOptionsString(t *testing.T, args []string) string {
	t.Helper()

	oIndex := -1
	for i, arg := range args {
		if arg == "-o" {
			oIndex = i
			break
		}
	}

	if oIndex == -1 {
		t.Error("Expected -o option in mount args")
		return ""
	}

	return args[oIndex+1]
}

// assertMountOption fails the test if the expected option is not present.
func assertMountOption(t *testing.T, optionsStr, expected string) {
	t.Helper()
	if !containsSMBOption(optionsStr, expected) {
		t.Errorf("Expected options to contain '%s', got: %s", expected, optionsStr)
	}
}

// verifyMountArgs asserts that the generated mount arguments reflect the config.
func verifyMountArgs(t *testing.T, config *SMBConfig, smbPath string, args []string) {
	t.Helper()

	// Check that basic elements are present
	if args[0] != "-t" || args[1] != "cifs" {
		t.Error("Expected -t cifs in mount args")
	}

	optionsStr := mountOptionsString(t, args)

	assertMountOption(t, optionsStr, "vers="+config.Version)

	if config.Username != "" {
		assertMountOption(t, optionsStr, "username="+config.Username)
	}
	if config.Domain != "" {
		assertMountOption(t, optionsStr, "domain="+config.Domain)
	}
	if config.UseKerberos {
		assertMountOption(t, optionsStr, "sec=krb5")
	}
	if config.DirectIO {
		assertMountOption(t, optionsStr, "directio")
	}
	if config.FileMode != "" {
		assertMountOption(t, optionsStr, "file_mode="+config.FileMode)
	}
	if config.DirMode != "" {
		assertMountOption(t, optionsStr, "dir_mode="+config.DirMode)
	}

	// Check that SMB path and mount point are at the end
	if args[len(args)-2] != smbPath {
		t.Errorf("Expected SMB path %s, got %s", smbPath, args[len(args)-2])
	}

	if args[len(args)-1] != config.MountPoint {
		t.Errorf("Expected mount point %s, got %s", config.MountPoint, args[len(args)-1])
	}
}

func TestSMBProvider_UploadDownload(t *testing.T) {
	// Create a temporary directory to simulate mount point
	tempDir, err := os.MkdirTemp("", "smb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider := &SMBProvider{
		config: &SMBConfig{
			Server:     "//192.168.1.100/share",
			Share:      "backups",
			MountPoint: tempDir,
		},
		mountPoint: tempDir,
		isMounted:  true, // Simulate mounted state
	}

	ctx := context.Background()

	t.Run("Upload", func(t *testing.T) { testUpload(t, ctx, provider, tempDir) })
	t.Run("Download", func(t *testing.T) { testDownload(t, ctx, provider, tempDir) })
	t.Run("Exists", func(t *testing.T) { testExists(t, ctx, provider) })
	t.Run("Delete", func(t *testing.T) { testDelete(t, ctx, provider, tempDir) })
	t.Run("GetMetadata", func(t *testing.T) { testGetMetadata(t, ctx, provider, tempDir) })
	t.Run("List", func(t *testing.T) { testList(t, ctx, provider, tempDir) })
}

func testUpload(t *testing.T, ctx context.Context, provider *SMBProvider, tempDir string) {
	t.Helper()

	// Create a test file
	testFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content for SMB upload")
	if err := os.WriteFile(testFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	remotePath := "backups/test-upload.txt"
	if err := provider.Upload(ctx, testFile, remotePath, nil); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Verify file was uploaded
	uploadedPath := filepath.Join(tempDir, remotePath)
	if _, statErr := os.Stat(uploadedPath); os.IsNotExist(statErr) {
		t.Error("Uploaded file does not exist")
	}

	// Verify content
	content, err := os.ReadFile(uploadedPath)
	if err != nil {
		t.Fatalf("Failed to read uploaded file: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
	}
}

func testDownload(t *testing.T, ctx context.Context, provider *SMBProvider, tempDir string) {
	t.Helper()

	// Create a remote file
	remotePath := "backups/test-download.txt"
	remoteFile := filepath.Join(tempDir, remotePath)
	testContent := []byte("test content for SMB download")

	if err := os.MkdirAll(filepath.Dir(remoteFile), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(remoteFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create remote file: %v", err)
	}

	// Download to local path
	localPath := filepath.Join(tempDir, "downloaded.txt")
	if err := provider.Download(ctx, remotePath, localPath); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify download
	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
	}
}

func testExists(t *testing.T, ctx context.Context, provider *SMBProvider) {
	t.Helper()

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
}

func testDelete(t *testing.T, ctx context.Context, provider *SMBProvider, tempDir string) {
	t.Helper()

	remotePath := "backups/test-delete.txt"
	remoteFile := filepath.Join(tempDir, remotePath)
	testContent := []byte("test content for deletion")

	if err := os.WriteFile(remoteFile, testContent, 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	if err := provider.Delete(ctx, remotePath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(remoteFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}
}

func testGetMetadata(t *testing.T, ctx context.Context, provider *SMBProvider, tempDir string) {
	t.Helper()

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
}

func testList(t *testing.T, ctx context.Context, provider *SMBProvider, tempDir string) {
	t.Helper()

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
}

func TestSMBProvider_UploadWithProgress(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider := &SMBProvider{
		config: &SMBConfig{
			Server:     "//192.168.1.100/share",
			Share:      "backups",
			MountPoint: tempDir,
		},
		mountPoint: tempDir,
		isMounted:  true,
	}

	ctx := context.Background()

	// Create a test file
	testFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content with progress tracking")
	if err = os.WriteFile(testFile, testContent, 0o644); err != nil {
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

func TestSMBProvider_Close(t *testing.T) {
	t.Run("with auto cleanup", func(t *testing.T) {
		provider := &SMBProvider{
			config: &SMBConfig{
				Server:      "//192.168.1.100/share",
				Share:       "backups",
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
		provider := &SMBProvider{
			config: &SMBConfig{
				Server:      "//192.168.1.100/share",
				Share:       "backups",
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

func TestSMBVersions(t *testing.T) {
	versions := []string{"1.0", "2.0", "2.1", "3.0", "3.1.1"}

	for _, version := range versions {
		t.Run("SMB v"+version, func(t *testing.T) {
			config := &SMBConfig{
				Server:  "//192.168.1.100/share",
				Share:   "backups",
				Version: version,
			}

			provider, err := NewSMBProvider(config)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			if provider.config.Version != version {
				t.Errorf("Expected version %s, got %s", version, provider.config.Version)
			}

			args := provider.buildMountArgs("//192.168.1.100/share/backups")

			// Find the options string
			var optionsStr string
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					optionsStr = args[i+1]
					break
				}
			}

			expectedVersion := "vers=" + version
			if !containsSMBOption(optionsStr, expectedVersion) {
				t.Errorf("Expected options to contain '%s', got: %s", expectedVersion, optionsStr)
			}
		})
	}
}

func TestSMBProvider_CredentialsFile(t *testing.T) {
	provider := &SMBProvider{
		config: &SMBConfig{
			Server:   "//192.168.1.100/share",
			Share:    "backups",
			Username: "testuser",
			Password: "testpass",
			Domain:   "WORKGROUP",
		},
	}

	credsFile, err := provider.createCredentialsFile()
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}
	defer os.Remove(credsFile)

	// Verify file exists
	if _, statErr := os.Stat(credsFile); os.IsNotExist(statErr) {
		t.Error("Credentials file should exist")
	}

	// Verify file permissions (should be 0600)
	info, err := os.Stat(credsFile)
	if err != nil {
		t.Fatalf("Failed to stat credentials file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("Expected file mode 0600, got %04o", mode)
	}

	// Verify file content
	content, err := os.ReadFile(credsFile)
	if err != nil {
		t.Fatalf("Failed to read credentials file: %v", err)
	}

	contentStr := string(content)
	expectedUsername := "username=testuser\n"
	expectedPassword := "password=testpass\n"
	expectedDomain := "domain=WORKGROUP\n"

	if !containsLine(contentStr, expectedUsername) {
		t.Error("Credentials file should contain username")
	}

	if !containsLine(contentStr, expectedPassword) {
		t.Error("Credentials file should contain password")
	}

	if !containsLine(contentStr, expectedDomain) {
		t.Error("Credentials file should contain domain")
	}
}

func TestSMBPathFormatting(t *testing.T) {
	tests := []struct {
		name     string
		server   string
		share    string
		expected string
	}{
		{
			name:     "server with //",
			server:   "//192.168.1.100/share",
			share:    "backups",
			expected: "//192.168.1.100/share/backups",
		},
		{
			name:     "server without //",
			server:   "192.168.1.100",
			share:    "backups",
			expected: "//192.168.1.100/backups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &SMBProvider{
				config: &SMBConfig{
					Server: tt.server,
					Share:  tt.share,
				},
			}

			args := provider.buildMountArgs(tt.expected)

			// The SMB path should be second to last argument
			smbPath := args[len(args)-2]

			if smbPath != tt.expected {
				t.Errorf("Expected SMB path %s, got %s", tt.expected, smbPath)
			}
		})
	}
}

// Helper functions

func containsSMBOption(optionsStr, option string) bool {
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

func containsLine(content, line string) bool {
	for i := 0; i <= len(content)-len(line); i++ {
		if content[i:i+len(line)] == line {
			return true
		}
	}
	return false
}
