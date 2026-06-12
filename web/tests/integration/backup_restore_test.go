// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/repository"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BackupRestoreIntegrationSuite struct {
	suite.Suite
	tempDir    string
	backupDir  string
	metadataDir string
	repo       repository.Repository
}

func (s *BackupRestoreIntegrationSuite) SetupSuite() {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir

	s.backupDir = filepath.Join(tempDir, "backups")
	s.metadataDir = filepath.Join(tempDir, "metadata")

	os.MkdirAll(s.backupDir, 0755)
	os.MkdirAll(s.metadataDir, 0755)

	// Create repository
	repo, err := repository.NewFileRepository(s.metadataDir)
	require.NoError(s.T(), err)
	s.repo = repo
}

func (s *BackupRestoreIntegrationSuite) TearDownSuite() {
	os.RemoveAll(s.tempDir)
}

func (s *BackupRestoreIntegrationSuite) TestFullBackupRestoreCycle() {
	ctx := context.Background()

	// Skip if database not available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		s.T().Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	// Configure backup engine
	engineCfg := &backup.Config{
		TempDirectory:      s.backupDir,
		ParallelOperations: 1,
		DefaultCompression: "gzip",
	}
	backupEngine := backup.NewEngine(engineCfg)

	// Create backup options
	opts := &backup.CreateOptions{
		DatabaseType:     database.DatabaseTypePostgreSQL,
		Host:             "localhost",
		Port:             5432,
		Username:         "postgres",
		Password:         "postgres",
		Database:         "testdb",
		Compression:      database.CompressionGzip,
		CompressionLevel: 6,
		Encrypt:          false,
		Tags: map[string]string{
			"env":  "test",
			"type": "integration",
		},
	}

	// Create backup
	metadata, err := backupEngine.CreateBackup(ctx, opts)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), metadata.ID)
	assert.NotEmpty(s.T(), metadata.BackupPath)
	assert.Greater(s.T(), metadata.Size, int64(0))
	assert.Equal(s.T(), models.BackupStatusCompleted, metadata.Status)

	// Save metadata
	err = s.repo.Save(ctx, metadata)
	require.NoError(s.T(), err)

	// Verify backup file exists
	_, err = os.Stat(metadata.BackupPath)
	assert.NoError(s.T(), err)

	// Configure restore engine
	restoreCfg := &restore.Config{
		TempDirectory: s.tempDir,
		ValidateFirst: true,
	}
	restoreEngine := restore.NewEngine(restoreCfg)

	// Create restore options
	restoreOpts := &restore.RestoreOptions{
		BackupID:       metadata.ID,
		TargetHost:     "localhost",
		TargetPort:     5432,
		TargetUsername: "postgres",
		TargetPassword: "postgres",
		TargetDatabase: "testdb_restored",
		SkipValidation: false,
		Force:          true,
	}

	// Perform restore
	result, err := restoreEngine.RestoreBackup(ctx, metadata, restoreOpts)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), metadata.ID, result.BackupID)
	assert.Equal(s.T(), database.RestoreStatusSuccess, result.Status)
	assert.Greater(s.T(), len(result.RestoredTables), 0)
}

func (s *BackupRestoreIntegrationSuite) TestBackupWithCompression() {
	ctx := context.Background()

	if os.Getenv("TEST_DATABASE_URL") == "" {
		s.T().Skip("Skipping: TEST_DATABASE_URL not set")
	}

	engineCfg := &backup.Config{
		TempDirectory: s.backupDir,
	}
	engine := backup.NewEngine(engineCfg)

	compressionTypes := []database.CompressionType{
		database.CompressionGzip,
		database.CompressionZstd,
		database.CompressionLZ4,
		database.CompressionNone,
	}

	for _, compression := range compressionTypes {
		s.T().Run(string(compression), func(t *testing.T) {
			opts := &backup.CreateOptions{
				DatabaseType: database.DatabaseTypePostgreSQL,
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				Password:     "postgres",
				Database:     "testdb",
				Compression:  compression,
			}

			metadata, err := engine.CreateBackup(ctx, opts)
			require.NoError(t, err)
			assert.Equal(t, compression, metadata.Compression)

			if compression != database.CompressionNone {
				assert.Less(t, metadata.CompressedSize, metadata.Size)
			}
		})
	}
}

func (s *BackupRestoreIntegrationSuite) TestBackupWithEncryption() {
	ctx := context.Background()

	if os.Getenv("TEST_DATABASE_URL") == "" {
		s.T().Skip("Skipping: TEST_DATABASE_URL not set")
	}

	engineCfg := &backup.Config{
		TempDirectory:    s.backupDir,
		EnableEncryption: true,
		EncryptionKey:    "test-encryption-key-32-bytes!!",
	}
	engine := backup.NewEngine(engineCfg)

	opts := &backup.CreateOptions{
		DatabaseType:  database.DatabaseTypePostgreSQL,
		Host:          "localhost",
		Port:          5432,
		Username:      "postgres",
		Password:      "postgres",
		Database:      "testdb",
		Encrypt:       true,
		EncryptionKey: "test-encryption-key-32-bytes!!",
	}

	metadata, err := engine.CreateBackup(ctx, opts)
	require.NoError(s.T(), err)
	assert.True(s.T(), metadata.Encrypted)

	// Verify encrypted backup can be restored
	restoreCfg := &restore.Config{
		TempDirectory: s.tempDir,
	}
	restoreEngine := restore.NewEngine(restoreCfg)

	restoreOpts := &restore.RestoreOptions{
		BackupID:       metadata.ID,
		TargetHost:     "localhost",
		TargetPort:     5432,
		TargetUsername: "postgres",
		TargetPassword: "postgres",
		TargetDatabase: "testdb_encrypted",
		Decrypt:        true,
		DecryptionKey:  "test-encryption-key-32-bytes!!",
		Force:          true,
	}

	result, err := restoreEngine.RestoreBackup(ctx, metadata, restoreOpts)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), database.RestoreStatusSuccess, result.Status)
}

func (s *BackupRestoreIntegrationSuite) TestRepositoryOperations() {
	ctx := context.Background()

	// Create multiple backups
	for i := 0; i < 5; i++ {
		metadata := &backup.BackupMetadata{
			ID:           fmt.Sprintf("test-backup-%d", i),
			Name:         fmt.Sprintf("backup-%d", i),
			DatabaseType: database.DatabaseTypePostgreSQL,
			Database:     "testdb",
			Size:         int64(1000 * (i + 1)),
			StartTime:    time.Now().Add(time.Duration(-i) * time.Hour),
			Status:       models.BackupStatusCompleted,
			Tags: map[string]string{
				"index": fmt.Sprintf("%d", i),
			},
		}

		err := s.repo.Save(ctx, metadata)
		require.NoError(s.T(), err)
	}

	// Test listing
	backups, err := s.repo.List(ctx, nil)
	require.NoError(s.T(), err)
	assert.Len(s.T(), backups, 5)

	// Test filtering by limit
	backups, err = s.repo.List(ctx, &repository.ListFilter{Limit: 3})
	require.NoError(s.T(), err)
	assert.Len(s.T(), backups, 3)

	// Test sorting
	backups, err = s.repo.List(ctx, &repository.ListFilter{
		SortBy:    "size",
		SortOrder: "asc",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1000), backups[0].Size)
	assert.Equal(s.T(), int64(5000), backups[4].Size)

	// Test deletion
	err = s.repo.Delete(ctx, "test-backup-0")
	require.NoError(s.T(), err)

	backups, err = s.repo.List(ctx, nil)
	require.NoError(s.T(), err)
	assert.Len(s.T(), backups, 4)
}

func TestBackupRestoreIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BackupRestoreIntegrationSuite))
}
