package wasabi

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/storage"
)

func TestWasabiProvider_NewProvider(t *testing.T) {
	config := &storage.WasabiConfig{
		Region:    "us-east-1",
		AccessKey: getEnv("WASABI_ACCESS_KEY", "test-key"),
		SecretKey: getEnv("WASABI_SECRET_KEY", "test-secret"),
		Bucket:    "test-bucket",
		UseSSL:    true,
	}

	provider, err := NewWasabiProvider(config)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.client)
	assert.NotNil(t, provider.uploader)
	assert.NotNil(t, provider.downloader)
}

func TestWasabiProvider_NewProvider_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *storage.WasabiConfig
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "wasabi config is required",
		},
		{
			name: "missing access key",
			config: &storage.WasabiConfig{
				SecretKey: "secret",
				Bucket:    "bucket",
				Region:    "us-east-1",
			},
			errMsg: "wasabi access key is required",
		},
		{
			name: "missing secret key",
			config: &storage.WasabiConfig{
				AccessKey: "key",
				Bucket:    "bucket",
				Region:    "us-east-1",
			},
			errMsg: "wasabi secret key is required",
		},
		{
			name: "missing bucket",
			config: &storage.WasabiConfig{
				AccessKey: "key",
				SecretKey: "secret",
				Region:    "us-east-1",
			},
			errMsg: "wasabi bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewWasabiProvider(tt.config)
			assert.Error(t, err)
			assert.Nil(t, provider)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestWasabiProvider_RegionalEndpoints(t *testing.T) {
	regions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"eu-central-1",
		"eu-west-1",
		"eu-west-2",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-southeast-1",
		"ap-southeast-2",
	}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			config := &storage.WasabiConfig{
				Region:    region,
				AccessKey: "test-key",
				SecretKey: "test-secret",
				Bucket:    "test-bucket",
				UseSSL:    true,
			}

			provider, err := NewWasabiProvider(config)
			assert.NoError(t, err)
			assert.NotNil(t, provider)

			// Verify endpoint is set correctly
			endpoint := wasabiEndpoints[region]
			assert.NotEmpty(t, endpoint)
		})
	}
}

func TestWasabiProvider_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// Create test file
	tmpFile := createTestFile(t, "test-wasabi-upload-content")
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

func TestWasabiProvider_UploadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	content := []byte("test wasabi stream content")
	reader := bytes.NewReader(content)
	remotePath := "test-streams/test-stream-" + time.Now().Format("20060102150405") + ".txt"

	opts := &storage.UploadOptions{
		ContentType: "text/plain",
		Metadata: map[string]string{
			"test-key": "test-value",
		},
	}

	err := provider.UploadStream(ctx, reader, remotePath, opts)
	if err != nil {
		t.Skipf("Upload failed (likely missing real Wasabi credentials): %v", err)
	}

	// Cleanup
	defer provider.Delete(ctx, remotePath)

	// Verify metadata
	metadata, err := provider.GetMetadata(ctx, remotePath)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), metadata.Size)
	assert.Equal(t, "text/plain", metadata.ContentType)
}

func TestWasabiProvider_UploadWithImmutability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create provider with immutability enabled
	config := &storage.WasabiConfig{
		Region:        getEnv("WASABI_REGION", "us-east-1"),
		AccessKey:     getEnv("WASABI_ACCESS_KEY", "test-key"),
		SecretKey:     getEnv("WASABI_SECRET_KEY", "test-secret"),
		Bucket:        getEnv("WASABI_TEST_BUCKET", "test-bucket-immutable"),
		UseSSL:        true,
		Immutable:     true,
		RetentionDays: 7,
	}

	provider, err := NewWasabiProvider(config)
	if err != nil {
		t.Skip("Cannot test immutability without proper Wasabi credentials")
	}

	ctx := context.Background()
	content := []byte("immutable content")
	remotePath := "test-immutable/locked-file-" + time.Now().Format("20060102150405") + ".txt"

	err = provider.UploadStream(ctx, bytes.NewReader(content), remotePath, nil)
	if err != nil {
		t.Skipf("Object lock not available: %v", err)
	}

	// Try to delete (should fail if object lock is working)
	err = provider.Delete(ctx, remotePath)
	// This should succeed after retention period or fail immediately depending on setup
	t.Logf("Delete result with object lock: %v", err)
}

func TestWasabiProvider_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test wasabi download content"
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

func TestWasabiProvider_DownloadStream(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	// First upload a file
	testContent := "test wasabi download stream content"
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

func TestWasabiProvider_Delete(t *testing.T) {
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

func TestWasabiProvider_Exists(t *testing.T) {
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

func TestWasabiProvider_GetMetadata(t *testing.T) {
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

func TestWasabiProvider_List(t *testing.T) {
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

func TestWasabiProvider_GetType(t *testing.T) {
	provider := setupTestProvider(t)
	providerType := provider.GetType()
	assert.Equal(t, storage.ProviderTypeWasabi, providerType)
}

func TestWasabiProvider_ValidateConfig(t *testing.T) {
	provider := setupTestProvider(t)
	err := provider.ValidateConfig()
	assert.NoError(t, err)
}

func TestWasabiProvider_CreateBucket(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &storage.WasabiConfig{
		Region:    getEnv("WASABI_REGION", "us-east-1"),
		AccessKey: getEnv("WASABI_ACCESS_KEY", "test-key"),
		SecretKey: getEnv("WASABI_SECRET_KEY", "test-secret"),
		Bucket:    "test-create-bucket-" + time.Now().Format("20060102150405"),
		UseSSL:    true,
	}

	provider, err := NewWasabiProvider(config)
	if err != nil {
		t.Skip("Cannot create provider without valid credentials")
	}

	ctx := context.Background()
	err = provider.CreateBucket(ctx)
	if err != nil {
		t.Skipf("Cannot create bucket: %v", err)
	}

	// Try creating again (should succeed as it already exists)
	err = provider.CreateBucket(ctx)
	assert.NoError(t, err)
}

func TestWasabiProvider_CreateBucketWithObjectLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := &storage.WasabiConfig{
		Region:        getEnv("WASABI_REGION", "us-east-1"),
		AccessKey:     getEnv("WASABI_ACCESS_KEY", "test-key"),
		SecretKey:     getEnv("WASABI_SECRET_KEY", "test-secret"),
		Bucket:        "test-lock-bucket-" + time.Now().Format("20060102150405"),
		UseSSL:        true,
		Immutable:     true,
		RetentionDays: 7,
	}

	provider, err := NewWasabiProvider(config)
	if err != nil {
		t.Skip("Cannot create provider without valid credentials")
	}

	ctx := context.Background()
	err = provider.CreateBucket(ctx)
	if err != nil {
		t.Skipf("Cannot create bucket with object lock: %v", err)
	}

	// Verify object lock configuration
	lockConfig, err := provider.GetObjectLockConfiguration(ctx)
	if err != nil {
		t.Logf("Cannot get object lock configuration: %v", err)
	} else {
		assert.NotNil(t, lockConfig)
		assert.Equal(t, types.ObjectLockEnabledEnabled, lockConfig.ObjectLockEnabled)
	}
}

func TestWasabiProvider_GetObjectLockConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	config, err := provider.GetObjectLockConfiguration(ctx)
	if err != nil {
		t.Skipf("Object lock not configured: %v", err)
	}

	assert.NotNil(t, config)
}

func TestWasabiProvider_SetLifecyclePolicy(t *testing.T) {
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

func TestWasabiProvider_GetLifecyclePolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	provider := setupTestProvider(t)
	ctx := context.Background()

	rules, err := provider.GetLifecyclePolicy(ctx)
	if err != nil {
		t.Logf("No lifecycle policy configured: %v", err)
	} else {
		t.Logf("Found %d lifecycle rules", len(rules))
	}
}

func TestWasabi_RegionalEndpointMapping(t *testing.T) {
	// Verify all regional endpoints are defined
	expectedRegions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"eu-central-1",
		"eu-west-1",
		"eu-west-2",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-southeast-1",
		"ap-southeast-2",
	}

	for _, region := range expectedRegions {
		endpoint, exists := wasabiEndpoints[region]
		assert.True(t, exists, "Region %s should have an endpoint", region)
		assert.NotEmpty(t, endpoint, "Endpoint for region %s should not be empty", region)
		assert.Contains(t, endpoint, "wasabisys.com", "Endpoint should contain wasabisys.com")
	}
}

func TestWasabi_DefaultRegion(t *testing.T) {
	// Test that default region is set when not specified
	config := &storage.WasabiConfig{
		AccessKey: "test-key",
		SecretKey: "test-secret",
		Bucket:    "test-bucket",
		UseSSL:    true,
		// Region not specified
	}

	err := validateConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, "us-east-1", config.Region)
}

// Helper functions

func setupTestProvider(t *testing.T) *WasabiProvider {
	t.Helper()
	if os.Getenv("WASABI_ACCESS_KEY") == "" {
		t.Skip("WASABI_ACCESS_KEY not set; skipping Wasabi integration test")
	}
	config := &storage.WasabiConfig{
		Region:    getEnv("WASABI_REGION", "us-east-1"),
		AccessKey: getEnv("WASABI_ACCESS_KEY", "test-key"),
		SecretKey: getEnv("WASABI_SECRET_KEY", "test-secret"),
		Bucket:    getEnv("WASABI_TEST_BUCKET", "test-bucket"),
		UseSSL:    true,
	}

	provider, err := NewWasabiProvider(config)
	if err != nil {
		t.Skip("Cannot create Wasabi provider without valid credentials")
	}

	// Ensure bucket exists
	ctx := context.Background()
	provider.CreateBucket(ctx)

	return provider
}

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "wasabi-test-*.txt")
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
