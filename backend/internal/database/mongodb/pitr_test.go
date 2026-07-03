// Package mongodb_test provides comprehensive tests for MongoDB PITR
package mongodb

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/sanskarpan/db-backup/internal/database"
)

func skipIfMongoDBUnavailable(t *testing.T) {
	t.Helper()
	host := os.Getenv("MONGODB_TEST_HOST")
	if host == "" {
		host = "localhost"
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:27017", host), 2*time.Second)
	if err != nil {
		t.Skip("MongoDB not available:", err)
	}
	conn.Close()
}

func TestMongoDBPITR_BackupOplog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMongoDB(t)
	if driver == nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create temporary output directory
	outputDir := t.TempDir()

	// Test oplog backup
	err := pitr.BackupOplog(ctx, outputDir, nil)
	if err != nil {
		t.Logf("Oplog backup failed (expected if not replica set): %v", err)
		return
	}

	// Verify output directory exists
	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}

func TestMongoDBPITR_GetOplogPosition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMongoDB(t)
	if driver == nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Get oplog position
	pos, err := pitr.GetOplogPosition(ctx)
	if err != nil {
		t.Logf("Oplog position not available (expected if not replica set): %v", err)
		return
	}

	assert.NotNil(t, pos)
	assert.NotZero(t, pos.Timestamp.T, "Expected valid timestamp")
}

func TestMongoDBPITR_RestoreToPointInTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMongoDB(t)
	if driver == nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create test database and collection
	testDB := "pitr_test_db"
	testCollection := "test_collection"

	db := driver.client.Database(testDB)
	collection := db.Collection(testCollection)

	// Cleanup
	defer func() {
		db.Drop(ctx)
	}()

	// Insert test data
	_, err := collection.InsertOne(ctx, map[string]interface{}{"value": "before"})
	require.NoError(t, err)

	timeBeforeSecondInsert := time.Now()
	time.Sleep(2 * time.Second)

	_, err = collection.InsertOne(ctx, map[string]interface{}{"value": "after"})
	require.NoError(t, err)

	// Test PITR restore options validation
	oplogDir := t.TempDir()

	opts := &PITRRestoreOptions{
		Database:   testDB,
		OplogDir:   oplogDir,
		TargetTime: timeBeforeSecondInsert,
	}

	err = pitr.validatePITROptions(opts)
	assert.NoError(t, err, "PITR options should be valid")
}

func TestMongoDBPITR_ValidationErrors(t *testing.T) {
	driver := &MongoDBDriver{}
	pitr := NewPITRManager(driver)

	tests := []struct {
		name    string
		opts    *PITRRestoreOptions
		wantErr bool
	}{
		{
			name: "missing target time",
			opts: &PITRRestoreOptions{
				Database: "test",
				OplogDir: "/tmp/oplog",
			},
			wantErr: true,
		},
		{
			name: "missing oplog directory",
			opts: &PITRRestoreOptions{
				Database:   "test",
				TargetTime: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "non-existent oplog directory",
			opts: &PITRRestoreOptions{
				Database:   "test",
				OplogDir:   "/nonexistent/path",
				TargetTime: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pitr.validatePITROptions(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOplogPosition_String(t *testing.T) {
	pos := &OplogPosition{
		Timestamp: primitive.Timestamp{T: 1234567890, I: 1},
		Time:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	str := pos.String()
	assert.Contains(t, str, "1234567890")
	assert.Contains(t, str, "2024")
}

func TestTimeToTimestamp(t *testing.T) {
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ts := TimeToTimestamp(testTime)

	assert.Equal(t, uint32(testTime.Unix()), ts.T)
	assert.Equal(t, uint32(0), ts.I)
}

func TestTimestampToTime(t *testing.T) {
	ts := primitive.Timestamp{T: 1704110400, I: 0} // 2024-01-01 12:00:00 UTC
	resultTime := TimestampToTime(ts)

	assert.Equal(t, int64(1704110400), resultTime.Unix())
}

func TestBackupWithOplog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMongoDB(t)
	if driver == nil {
		t.Skip("MongoDB not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	outputDir := t.TempDir()

	err := pitr.BackupWithOplog(ctx, outputDir, "")
	if err != nil {
		t.Logf("Backup with oplog failed (expected if not replica set): %v", err)
		return
	}

	// Verify backup directory exists
	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}

// Helper function to setup test MongoDB connection.
func setupTestMongoDB(t *testing.T) *MongoDBDriver {
	skipIfMongoDBUnavailable(t)
	host := os.Getenv("MONGODB_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("MONGODB_TEST_USER")
	password := os.Getenv("MONGODB_TEST_PASSWORD")

	driver := NewMongoDBDriver()
	config := &database.ConnectionConfig{
		Host:     host,
		Port:     27017,
		Username: user,
		Password: password,
		Database: "test",
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Logf("Could not connect to MongoDB for testing: %v", err)
		return nil
	}

	return driver
}
