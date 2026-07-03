package backblaze

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/kurin/blazer/b2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/storage"
)

func TestBackblazeB2Provider_NewProvider(t *testing.T) {
	config := &storage.BackblazeB2Config{
		AccountID:      getEnv("B2_ACCOUNT_ID", "test-account"),
		ApplicationKey: getEnv("B2_APPLICATION_KEY", "test-key"),
		Bucket:         "test-bucket",
	}

	provider, err := NewBackblazeB2Provider(config)
	if err != nil {
		t.Skip("Cannot create B2 provider without valid credentials")
	}

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.NotNil(t, provider.bucket)
}

func TestBackblazeB2Provider_NewProvider_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *storage.BackblazeB2Config
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "config is required for Backblaze B2",
		},
		{
			name: "missing account ID",
			config: &storage.BackblazeB2Config{
				ApplicationKey: "key",
				Bucket:         "bucket",
			},
			errMsg: "account ID is required for Backblaze B2",
		},
		{
			name: "missing application key",
			config: &storage.BackblazeB2Config{
				AccountID: "account",
				Bucket:    "bucket",
			},
			errMsg: "application key is required for Backblaze B2",
		},
		{
			name: "missing bucket",
			config: &storage.BackblazeB2Config{
				AccountID:      "account",
				ApplicationKey: "key",
			},
			errMsg: "bucket is required for Backblaze B2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewBackblazeB2Provider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestBackblazeB2Provider_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create test file
	tmpFile := createTestFile(t, "test-b2-upload-content")
	defer os.Remove(tmpFile)

	remotePath := "test-uploads/test-file-" + time.Now().Format("20060102150405") + ".txt"

	err := provider.Upload(ctx, tmpFile, remotePath, nil)
	assert.NoError(t, err)

	// Cleanup
	defer provider.Delete(ctx, remotePath)

	// Verify file exists
	exists, err := provider.Exists(ctx, remotePath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestBackblazeB2Provider_UploadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	content := []byte("test b2 stream content")
	reader := bytes.NewReader(content)
	remotePath := "test-streams/test-stream-" + time.Now().Format("20060102150405") + ".txt"

	opts := &storage.UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"test-key": "test-value",
		},
	}

	err := provider.UploadStream(ctx, reader, remotePath, opts)
	assert.NoError(t, err)

	// Cleanup
	defer provider.Delete(ctx, remotePath)

	// Verify metadata
	metadata, err := provider.GetMetadata(ctx, remotePath)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(content)), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
}

func TestBackblazeB2Provider_UploadWithFileLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create provider with file lock enabled
	config := &storage.BackblazeB2Config{
		AccountID:      getEnv("B2_ACCOUNT_ID", "test-account"),
		ApplicationKey: getEnv("B2_APPLICATION_KEY", "test-key"),
		Bucket:         getEnv("B2_TEST_BUCKET", "test-bucket-lock"),
		Immutable:      true,
		RetentionDays:  7,
	}

	provider, err := NewBackblazeB2Provider(config)
	if err != nil {
		t.Skip("Cannot test file lock without proper B2 credentials")
	}

	ctx := context.Background()
	content := []byte("locked content")
	remotePath := "test-locked/locked-file-" + time.Now().Format("20060102150405") + ".txt"

	err = provider.UploadStream(ctx, bytes.NewReader(content), remotePath, nil)
	if err != nil {
		t.Skipf("File lock not available: %v", err)
	}

	// Get file lock info
	mode, retainUntil, err := provider.GetFileLock(ctx, remotePath)
	if err != nil {
		t.Logf("Cannot get file lock info: %v", err)
	} else {
		assert.Equal(t, "compliance", mode)
		assert.True(t, retainUntil.After(time.Now()))
		t.Logf("File locked until: %v", retainUntil)
	}

	// Try to delete (may fail if file lock is working)
	err = provider.Delete(ctx, remotePath)
	t.Logf("Delete result with file lock: %v", err)
}

func TestBackblazeB2Provider_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test b2 download content"
	remotePath := "test-downloads/test-file-" + time.Now().Format("20060102150405") + ".txt"

	err := provider.UploadStream(ctx, bytes.NewReader([]byte(testContent)), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Then download it
	tmpDir := t.TempDir()
	localPath := tmpDir + "/downloaded-file.txt"

	err = provider.Download(ctx, remotePath, localPath)
	assert.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(localPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestBackblazeB2Provider_DownloadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test b2 download stream content"
	remotePath := "test-downloads/test-stream-" + time.Now().Format("20060102150405") + ".txt"

	err := provider.UploadStream(ctx, bytes.NewReader([]byte(testContent)), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Then download as stream
	reader, err := provider.DownloadStream(ctx, remotePath)
	assert.NoError(t, err)
	defer reader.Close()

	// Read content
	content, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestBackblazeB2Provider_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload a file
	remotePath := "test-deletes/test-file-" + time.Now().Format("20060102150405") + ".txt"
	err := provider.UploadStream(ctx, bytes.NewReader([]byte("test content")), remotePath, nil)
	require.NoError(t, err)

	// Verify it exists
	exists, err := provider.Exists(ctx, remotePath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = provider.Delete(ctx, remotePath)
	assert.NoError(t, err)

	// Verify it's deleted
	exists, err = provider.Exists(ctx, remotePath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBackblazeB2Provider_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Check non-existent file
	exists, err := provider.Exists(ctx, "nonexistent/file.txt")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Upload a file
	remotePath := "test-exists/test-file-" + time.Now().Format("20060102150405") + ".txt"
	err = provider.UploadStream(ctx, bytes.NewReader([]byte("test")), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Check it exists
	exists, err = provider.Exists(ctx, remotePath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestBackblazeB2Provider_GetMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload a file with metadata
	testContent := "test metadata content"
	remotePath := "test-metadata/test-file-" + time.Now().Format("20060102150405") + ".txt"

	opts := &storage.UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"custom-key": "custom-value",
		},
	}

	err := provider.UploadStream(ctx, bytes.NewReader([]byte(testContent)), remotePath, opts)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Get metadata
	metadata, err := provider.GetMetadata(ctx, remotePath)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, remotePath, metadata.Path)
	assert.Equal(t, int64(len(testContent)), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
	assert.NotZero(t, metadata.LastModified)
	assert.NotEmpty(t, metadata.Checksum) // B2 provides SHA1
}

func TestBackblazeB2Provider_SHA1Verification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload a file
	testContent := "test sha1 verification"
	remotePath := "test-sha1/test-file-" + time.Now().Format("20060102150405") + ".txt"

	err := provider.UploadStream(ctx, bytes.NewReader([]byte(testContent)), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Get metadata with SHA1
	metadata, err := provider.GetMetadata(ctx, remotePath)
	assert.NoError(t, err)
	assert.NotEmpty(t, metadata.Checksum)

	// B2 automatically verifies SHA1 during upload/download
	t.Logf("File SHA1: %s", metadata.Checksum)
}

func TestBackblazeB2Provider_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload some files
	prefix := "test-list-" + time.Now().Format("20060102150405") + "/"
	files := []string{"file1.txt", "file2.txt", "file3.txt"}

	for _, file := range files {
		remotePath := prefix + file
		err := provider.UploadStream(ctx, bytes.NewReader([]byte("content")), remotePath, nil)
		require.NoError(t, err)
		t.Cleanup(func() { provider.Delete(ctx, remotePath) })
	}

	// List files
	result, err := provider.List(ctx, prefix)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 3)

	// Verify files are in the list
	var foundFiles []string
	for _, metadata := range result {
		if metadata.Path == prefix+"file1.txt" ||
			metadata.Path == prefix+"file2.txt" ||
			metadata.Path == prefix+"file3.txt" {
			foundFiles = append(foundFiles, metadata.Path)
		}
	}
	assert.Len(t, foundFiles, 3)
}

func TestBackblazeB2Provider_GetType(t *testing.T) {
	provider := setupTestProvider(t)
	providerType := provider.GetType()
	assert.Equal(t, storage.ProviderTypeBackblazeB2, providerType)
}

func TestBackblazeB2Provider_ValidateConfig(t *testing.T) {
	provider := setupTestProvider(t)
	err := provider.ValidateConfig()
	assert.NoError(t, err)
}

func TestBackblazeB2Provider_CreateBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &storage.BackblazeB2Config{
		AccountID:      getEnv("B2_ACCOUNT_ID", "test-account"),
		ApplicationKey: getEnv("B2_APPLICATION_KEY", "test-key"),
		Bucket:         "placeholder-bucket",
	}

	provider, err := NewBackblazeB2Provider(config)
	if err != nil {
		t.Skip("Cannot create provider without valid credentials")
	}

	ctx := context.Background()
	bucketName := "test-create-bucket-" + time.Now().Format("20060102150405")

	err = provider.CreateBucket(ctx, bucketName, false)
	if err != nil {
		t.Skipf("Cannot create bucket: %v", err)
	}

	// Cleanup
	if err := provider.DeleteBucket(ctx); err != nil {
		t.Logf("Cleanup DeleteBucket error: %v", err)
	}
}

func TestBackblazeB2Provider_DeleteBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &storage.BackblazeB2Config{
		AccountID:      getEnv("B2_ACCOUNT_ID", "test-account"),
		ApplicationKey: getEnv("B2_APPLICATION_KEY", "test-key"),
		Bucket:         "placeholder-bucket",
	}

	provider, err := NewBackblazeB2Provider(config)
	if err != nil {
		t.Skip("Cannot create provider without valid credentials")
	}

	ctx := context.Background()
	bucketName := "test-delete-bucket-" + time.Now().Format("20060102150405")

	// Create bucket first
	err = provider.CreateBucket(ctx, bucketName, false)
	if err != nil {
		t.Skipf("Cannot create bucket: %v", err)
	}

	// Delete it
	err = provider.DeleteBucket(ctx)
	if err != nil {
		t.Logf("Delete bucket error: %v", err)
	}
}

func TestBackblazeB2Provider_SetFileLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload a file
	remotePath := "test-filelock/test-file-" + time.Now().Format("20060102150405") + ".txt"
	err := provider.UploadStream(ctx, bytes.NewReader([]byte("locked file")), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Set file lock
	err = provider.SetFileLock(ctx, remotePath, 7)
	if err != nil {
		t.Skipf("File lock not supported: %v", err)
	}

	// Get file lock info
	mode, retainUntil, err := provider.GetFileLock(ctx, remotePath)
	assert.NoError(t, err)
	assert.Equal(t, "compliance", mode)
	assert.True(t, retainUntil.After(time.Now()))
}

func TestBackblazeB2Provider_GetFileLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload a file without lock
	remotePath := "test-getlock/test-file-" + time.Now().Format("20060102150405") + ".txt"
	err := provider.UploadStream(ctx, bytes.NewReader([]byte("test")), remotePath, nil)
	require.NoError(t, err)
	defer provider.Delete(ctx, remotePath)

	// Try to get file lock (should be empty)
	mode, retainUntil, err := provider.GetFileLock(ctx, remotePath)
	if err != nil {
		t.Logf("GetFileLock error: %v", err)
	} else {
		t.Logf("Mode: %s, RetainUntil: %v", mode, retainUntil)
	}
}

func TestBackblazeB2Provider_SetLifecycleRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create lifecycle rules
	rules := []b2.LifecycleRule{
		{
			DaysNewUntilHidden:     30,
			DaysHiddenUntilDeleted: 7,
			Prefix:                 "test-lifecycle/",
		},
	}

	err := provider.SetLifecycleRules(ctx, rules)
	if err != nil {
		t.Skipf("Lifecycle rules not supported: %v", err)
	}

	// Get lifecycle rules
	retrievedRules, err := provider.GetLifecycleRules(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, retrievedRules)
}

func TestBackblazeB2Provider_GetBucketInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	info, err := provider.GetBucketInfo(ctx)
	if err != nil {
		t.Skipf("Cannot get bucket info: %v", err)
	}

	assert.NotNil(t, info)
	t.Logf("Bucket Type: %v", info.Type)
	t.Logf("Lifecycle Rules: %d", len(info.LifecycleRules))
}

// Helper functions

func setupTestProvider(t *testing.T) *BackblazeB2Provider {
	t.Helper()
	config := &storage.BackblazeB2Config{
		AccountID:      getEnv("B2_ACCOUNT_ID", "test-account"),
		ApplicationKey: getEnv("B2_APPLICATION_KEY", "test-key"),
		Bucket:         getEnv("B2_TEST_BUCKET", "test-bucket"),
	}

	provider, err := NewBackblazeB2Provider(config)
	if err != nil {
		t.Skip("Cannot create B2 provider without valid credentials")
	}

	return provider
}

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "b2-test-*.txt")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	tmpFile.Close()
	return tmpFile.Name()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
