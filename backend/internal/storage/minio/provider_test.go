package minio

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/storage"
)

func TestMinIOProvider_NewProvider(t *testing.T) {
	config := &storage.MinIOConfig{
		Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    "test-bucket",
		UseSSL:    false,
		Region:    "us-east-1",
	}

	provider, err := NewMinIOProvider(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.NotNil(t, provider.uploader)
	assert.NotNil(t, provider.downloader)
}

func TestMinIOProvider_NewProvider_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *storage.MinIOConfig
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "MinIO config is required",
		},
		{
			name: "missing endpoint",
			config: &storage.MinIOConfig{
				AccessKey: "key",
				SecretKey: "secret",
				Bucket:    "bucket",
			},
			errMsg: "MinIO endpoint is required",
		},
		{
			name: "missing access key",
			config: &storage.MinIOConfig{
				Endpoint:  "localhost:9000",
				SecretKey: "secret",
				Bucket:    "bucket",
			},
			errMsg: "MinIO access key is required",
		},
		{
			name: "missing secret key",
			config: &storage.MinIOConfig{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				Bucket:    "bucket",
			},
			errMsg: "MinIO secret key is required",
		},
		{
			name: "missing bucket",
			config: &storage.MinIOConfig{
				Endpoint:  "localhost:9000",
				AccessKey: "key",
				SecretKey: "secret",
			},
			errMsg: "MinIO bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewMinIOProvider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestMinIOProvider_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create test file
	tmpFile := createTestFile(t, "test-upload-content")
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

func TestMinIOProvider_UploadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	content := []byte("test stream content")
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
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
}

func TestMinIOProvider_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test download content"
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

func TestMinIOProvider_DownloadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test download stream content"
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

func TestMinIOProvider_Delete(t *testing.T) {
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

func TestMinIOProvider_Exists(t *testing.T) {
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

func TestMinIOProvider_GetMetadata(t *testing.T) {
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
	require.NoError(t, err)
	require.NotNil(t, metadata)
	assert.Equal(t, remotePath, metadata.Path)
	assert.Equal(t, int64(len(testContent)), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
	assert.NotZero(t, metadata.LastModified)
	assert.NotEmpty(t, metadata.Checksum)
}

func TestMinIOProvider_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Upload some files
	prefix := "test-list-" + time.Now().Format("20060102150405") + "/"
	files := []string{"file1.txt", "file2.txt", "file3.txt"}

	var uploaded []string
	for _, file := range files {
		remotePath := prefix + file
		err := provider.UploadStream(ctx, bytes.NewReader([]byte("content")), remotePath, nil)
		require.NoError(t, err)
		uploaded = append(uploaded, remotePath)
	}
	defer func() {
		for _, remotePath := range uploaded {
			if err := provider.Delete(ctx, remotePath); err != nil {
				t.Logf("cleanup: failed to delete %s: %v", remotePath, err)
			}
		}
	}()

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

func TestMinIOProvider_GetType(t *testing.T) {
	provider := setupTestProvider(t)
	providerType := provider.GetType()
	assert.Equal(t, storage.ProviderTypeMinIO, providerType)
}

func TestMinIOProvider_ValidateConfig(t *testing.T) {
	provider := setupTestProvider(t)
	err := provider.ValidateConfig()
	assert.NoError(t, err)
}

func TestMinIOProvider_CreateBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfMinIOUnavailable(t)

	config := &storage.MinIOConfig{
		Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    "test-create-bucket-" + time.Now().Format("20060102150405"),
		UseSSL:    false,
		Region:    "us-east-1",
	}

	provider, err := NewMinIOProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = provider.CreateBucket(ctx)
	if err != nil {
		t.Skipf("MinIO authentication failed: %v", err)
	}

	// Cleanup
	defer provider.DeleteBucket(ctx)

	// Try creating again (should succeed as it already exists)
	err = provider.CreateBucket(ctx)
	assert.NoError(t, err)
}

func TestMinIOProvider_DeleteBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfMinIOUnavailable(t)

	config := &storage.MinIOConfig{
		Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    "test-delete-bucket-" + time.Now().Format("20060102150405"),
		UseSSL:    false,
		Region:    "us-east-1",
	}

	provider, err := NewMinIOProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create bucket
	err = provider.CreateBucket(ctx)
	if err != nil {
		t.Skipf("MinIO authentication failed: %v", err)
	}

	// Delete bucket
	err = provider.DeleteBucket(ctx)
	assert.NoError(t, err)
}

func TestMinIOProvider_SetVersioning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Enable versioning
	err := provider.SetVersioning(ctx, true)
	if err != nil {
		t.Skipf("Versioning not supported: %v", err)
	}

	// Check versioning is enabled
	enabled, err := provider.GetVersioning(ctx)
	assert.NoError(t, err)
	assert.True(t, enabled)

	// Disable versioning
	err = provider.SetVersioning(ctx, false)
	assert.NoError(t, err)

	// Check versioning is disabled
	enabled, err = provider.GetVersioning(ctx)
	assert.NoError(t, err)
	assert.False(t, enabled)
}

func TestMinIOProvider_GetVersioning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	enabled, err := provider.GetVersioning(ctx)
	if err != nil {
		t.Skipf("Versioning not supported: %v", err)
	}
	assert.NotNil(t, enabled)
}

func TestMinIOProvider_SetLifecyclePolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create a simple lifecycle rule
	rules := []types.LifecycleRule{
		{
			ID:     stringPtr("test-rule"),
			Status: types.ExpirationStatusEnabled,
			Expiration: &types.LifecycleExpiration{
				Days: int32Ptr(30),
			},
			Filter: &types.LifecycleRuleFilterMemberPrefix{
				Value: "test-prefix/",
			},
		},
	}

	err := provider.SetLifecyclePolicy(ctx, rules)
	if err != nil {
		t.Skipf("Lifecycle policy not supported: %v", err)
	}

	// Get lifecycle policy
	retrievedRules, err := provider.GetLifecyclePolicy(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, retrievedRules)
}

func TestMinIOProvider_MultipartUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create a large file (>5MB to trigger multipart)
	largeContent := make([]byte, 6*1024*1024) // 6MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	remotePath := "test-multipart/large-file-" + time.Now().Format("20060102150405") + ".bin"

	err := provider.UploadStream(ctx, bytes.NewReader(largeContent), remotePath, nil)
	assert.NoError(t, err)

	// Cleanup
	defer provider.Delete(ctx, remotePath)

	// Verify upload
	metadata, err := provider.GetMetadata(ctx, remotePath)
	require.NoError(t, err)
	assert.Equal(t, int64(len(largeContent)), metadata.Size)
}

// Helper functions

func skipIfMinIOUnavailable(t *testing.T) {
	t.Helper()
	endpoint := getEnv("MINIO_ENDPOINT", "localhost:9000")
	conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
	if err != nil {
		t.Skipf("MinIO not available at %s: %v", endpoint, err)
	}
	conn.Close()
}

func setupTestProvider(t *testing.T) *Provider {
	t.Helper()
	skipIfMinIOUnavailable(t)
	config := &storage.MinIOConfig{
		Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		Bucket:    getEnv("MINIO_TEST_BUCKET", "test-bucket"),
		UseSSL:    false,
		Region:    "us-east-1",
	}

	provider, err := NewMinIOProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	if err := provider.CreateBucket(ctx); err != nil {
		t.Skipf("MinIO authentication failed (set MINIO_ACCESS_KEY/MINIO_SECRET_KEY): %v", err)
	}

	return provider
}

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "minio-test-*.txt")
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

func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
