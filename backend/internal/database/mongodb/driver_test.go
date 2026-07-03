package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

func testDriver() *MongoDBDriver {
	return &MongoDBDriver{
		config: &database.ConnectionConfig{
			Host: "localhost",
			Port: 27017,
		},
	}
}

func TestStreamBackup_Unsupported(t *testing.T) {
	driver := testDriver()
	err := driver.StreamBackup(t.Context(), &database.BackupOptions{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestStreamRestore_Unsupported(t *testing.T) {
	driver := testDriver()
	err := driver.StreamRestore(t.Context(), &database.RestoreOptions{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestRestore_MissingBackupDirFails(t *testing.T) {
	driver := testDriver()
	result, err := driver.Restore(t.Context(), &database.RestoreOptions{
		SourceBackup: "/nonexistent/backup/path-should-not-exist",
	})
	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, database.RestoreStatusFailed, result.Status)
}

func TestBuildMongoRestoreArgs(t *testing.T) {
	driver := &MongoDBDriver{
		config: &database.ConnectionConfig{
			Host:     "db.example.com",
			Port:     27018,
			Username: "admin",
			Password: "secret",
		},
	}

	args, err := driver.buildMongoRestoreArgs(&database.RestoreOptions{
		Database:     "appdb",
		SourceBackup: "/backups/appdb",
		DropExisting: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"--host", "db.example.com",
		"--port", "27018",
		"--gzip",
		"--username", "admin",
		"--password", "secret",
		"--db", "appdb",
		"--drop",
		"/backups/appdb",
	}, args)
}

func TestBuildMongoRestoreArgs_InvalidDatabaseName(t *testing.T) {
	driver := testDriver()
	_, err := driver.buildMongoRestoreArgs(&database.RestoreOptions{
		Database:     "bad;name",
		SourceBackup: "/backups/x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid database name")
}

func TestBuildMongoDumpArgs_InvalidCollectionName(t *testing.T) {
	driver := testDriver()
	_, err := driver.buildMongoDumpArgs(&database.BackupOptions{
		Database:   "appdb",
		OutputPath: "/tmp/out",
		Tables:     []string{"good", "bad name"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid collection name")
}
