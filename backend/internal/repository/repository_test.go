package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
)

func TestFileRepository_Save(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	metadata := &models.BackupMetadata{
		ID:           "backup-2024-01-15-10-00-01-aabbcc01",
		Name:         "test-backup",
		DatabaseType: "mysql",
		Database:     "testdb",
		Size:         1024,
		StartTime:    time.Now(),
		Status:       database.BackupStatusCompleted,
	}

	// Test Save
	err = repo.Save(context.Background(), metadata)
	assert.NoError(t, err)

	// Verify file exists
	metadataPath := filepath.Join(tempDir, "backup-2024-01-15-10-00-01-aabbcc01.json")
	_, err = os.Stat(metadataPath)
	assert.NoError(t, err)
}

func TestFileRepository_Get(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	originalMetadata := &models.BackupMetadata{
		ID:           "backup-2024-01-15-10-00-02-aabbcc02",
		Name:         "test-backup-2",
		DatabaseType: database.DatabaseTypePostgreSQL,
		Database:     "testdb",
		Size:         2048,
		StartTime:    time.Now(),
		Status:       database.BackupStatusCompleted,
		Tags:         map[string]string{"env": "test"},
	}

	// Save first
	err = repo.Save(context.Background(), originalMetadata)
	require.NoError(t, err)

	// Test Get
	retrievedMetadata, err := repo.Get(context.Background(), "backup-2024-01-15-10-00-02-aabbcc02")
	assert.NoError(t, err)
	assert.Equal(t, originalMetadata.ID, retrievedMetadata.ID)
	assert.Equal(t, originalMetadata.Name, retrievedMetadata.Name)
	assert.Equal(t, originalMetadata.DatabaseType, retrievedMetadata.DatabaseType)
	assert.Equal(t, originalMetadata.Tags["env"], retrievedMetadata.Tags["env"])
}

func TestFileRepository_Get_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	// Test Get non-existent backup
	_, err = repo.Get(context.Background(), "non-existent")
	assert.Error(t, err)
}

func TestFileRepository_List(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Create multiple backups
	backups := []*models.BackupMetadata{
		{
			ID:           "backup-2024-01-15-10-01-01-aabbcc11",
			Name:         "mysql-backup",
			DatabaseType: "mysql",
			Database:     "mydb",
			Size:         1000,
			StartTime:    time.Now().Add(-2 * time.Hour),
			Status:       database.BackupStatusCompleted,
			Tags:         map[string]string{"env": "prod"},
		},
		{
			ID:           "backup-2024-01-15-10-01-02-aabbcc12",
			Name:         "postgres-backup",
			DatabaseType: database.DatabaseTypePostgreSQL,
			Database:     "pgdb",
			Size:         2000,
			StartTime:    time.Now().Add(-1 * time.Hour),
			Status:       database.BackupStatusCompleted,
			Tags:         map[string]string{"env": "dev"},
		},
		{
			ID:           "backup-2024-01-15-10-01-03-aabbcc13",
			Name:         "mongo-backup",
			DatabaseType: database.DatabaseTypeMongoDB,
			Database:     "mongodb",
			Size:         3000,
			StartTime:    time.Now(),
			Status:       database.BackupStatusCompleted,
			Tags:         map[string]string{"env": "prod"},
		},
	}

	for _, b := range backups {
		err = repo.Save(ctx, b)
		require.NoError(t, err)
	}

	// Test List all
	result, err := repo.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Test List with database filter
	result, err = repo.List(ctx, &ListFilter{Database: "mydb"})
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "mydb", result[0].Database)

	// Test List with type filter
	result, err = repo.List(ctx, &ListFilter{DatabaseType: "postgres"})
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, database.DatabaseTypePostgreSQL, result[0].DatabaseType)

	// Test List with tag filter
	result, err = repo.List(ctx, &ListFilter{Tags: map[string]string{"env": "prod"}})
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Test List with limit
	result, err = repo.List(ctx, &ListFilter{Limit: 2})
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Test List sorted by size
	result, err = repo.List(ctx, &ListFilter{SortBy: "size", SortOrder: "asc"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1000), result[0].Size)
	assert.Equal(t, int64(3000), result[len(result)-1].Size)
}

func TestFileRepository_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	metadata := &models.BackupMetadata{
		ID:           "backup-2024-01-15-10-02-01-ddee0001",
		Name:         "delete-test",
		DatabaseType: "mysql",
		Database:     "testdb",
		Size:         1024,
		StartTime:    time.Now(),
		Status:       database.BackupStatusCompleted,
	}

	// Save first
	ctx := context.Background()
	err = repo.Save(ctx, metadata)
	require.NoError(t, err)

	// Test Delete
	err = repo.Delete(ctx, "backup-2024-01-15-10-02-01-ddee0001")
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = repo.Get(ctx, "backup-2024-01-15-10-02-01-ddee0001")
	assert.Error(t, err)
}

func TestFileRepository_Update(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	repo, err := NewFileRepository(tempDir)
	require.NoError(t, err)

	metadata := &models.BackupMetadata{
		ID:           "backup-2024-01-15-10-03-01-ffee0001",
		Name:         "update-test",
		DatabaseType: "mysql",
		Database:     "testdb",
		Size:         1024,
		StartTime:    time.Now(),
		Status:       database.BackupStatusInProgress,
	}

	ctx := context.Background()
	err = repo.Save(ctx, metadata)
	require.NoError(t, err)

	// Update status
	metadata.Status = database.BackupStatusCompleted
	metadata.EndTime = time.Now()

	err = repo.Update(ctx, metadata)
	assert.NoError(t, err)

	// Verify update
	updated, err := repo.Get(ctx, "backup-2024-01-15-10-03-01-ffee0001")
	require.NoError(t, err)
	assert.Equal(t, database.BackupStatusCompleted, updated.Status)
}
