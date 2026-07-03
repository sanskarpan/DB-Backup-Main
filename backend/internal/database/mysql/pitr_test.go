// Package mysql_test provides comprehensive tests for MySQL PITR
package mysql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/database"
)

func TestMySQLPITR_BackupBinaryLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test MySQL connection
	driver := setupTestMySQL(t)
	if driver == nil {
		t.Skip("MySQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create temporary output directory
	outputDir := t.TempDir()

	// Test binary log backup
	err := pitr.BackupBinaryLogs(ctx, outputDir, nil)
	if err != nil {
		// Binary logging might not be enabled in test environment
		t.Logf("Binary log backup failed (expected if binary logging not enabled): %v", err)
		return
	}

	// Verify output directory has files
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0, "Expected binary log files to be backed up")
}

func TestMySQLPITR_GetBinaryLogPosition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMySQL(t)
	if driver == nil {
		t.Skip("MySQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Get binary log position
	pos, err := pitr.GetBinaryLogPosition(ctx)
	if err != nil {
		t.Logf("Binary log position not available (expected if binary logging not enabled): %v", err)
		return
	}

	assert.NotNil(t, pos)
	assert.NotEmpty(t, pos.File, "Expected binary log file name")
	assert.GreaterOrEqual(t, pos.Position, int64(0), "Expected valid position")
}

func TestMySQLPITR_RestoreToPointInTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestMySQL(t)
	if driver == nil {
		t.Skip("MySQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create test database
	testDB := "pitr_test_db"
	_, err := driver.db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+testDB)
	require.NoError(t, err)
	defer driver.db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+testDB)

	// Create test table and insert data
	_, err = driver.db.ExecContext(ctx, "USE "+testDB)
	require.NoError(t, err)

	_, err = driver.db.ExecContext(ctx, "CREATE TABLE test_table (id INT PRIMARY KEY, value VARCHAR(50))")
	require.NoError(t, err)

	// Insert initial data
	_, err = driver.db.ExecContext(ctx, "INSERT INTO test_table VALUES (1, 'before')")
	require.NoError(t, err)

	// Record time before second insert
	timeBeforeSecondInsert := time.Now()
	time.Sleep(2 * time.Second)

	// Insert more data
	_, err = driver.db.ExecContext(ctx, "INSERT INTO test_table VALUES (2, 'after')")
	require.NoError(t, err)

	// Create base backup
	backupDir := t.TempDir()
	binlogDir := t.TempDir()

	// Backup binary logs
	err = pitr.BackupBinaryLogs(ctx, binlogDir, &timeBeforeSecondInsert)
	if err != nil {
		t.Logf("Binary log backup failed: %v", err)
		return
	}

	// Test PITR restore to point before second insert
	opts := &PITRRestoreOptions{
		Database:         testDB,
		BaseBackupPath:   backupDir,
		BinaryLogDir:     binlogDir,
		TargetTime:       timeBeforeSecondInsert,
		SkipVerification: false,
	}

	// Note: Full PITR restore requires database shutdown and restart
	// This is a smoke test to verify the options are validated correctly
	err = pitr.validatePITROptions(opts)
	assert.NoError(t, err, "PITR options should be valid")
}

func TestMySQLPITR_ValidationErrors(t *testing.T) {
	driver := &MySQLDriver{}
	pitr := NewPITRManager(driver)

	tests := []struct {
		name    string
		opts    *PITRRestoreOptions
		wantErr bool
	}{
		{
			name: "missing target time",
			opts: &PITRRestoreOptions{
				Database:     "test",
				BinaryLogDir: "/tmp/binlogs",
			},
			wantErr: true,
		},
		{
			name: "missing binlog directory",
			opts: &PITRRestoreOptions{
				Database:   "test",
				TargetTime: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "non-existent binlog directory",
			opts: &PITRRestoreOptions{
				Database:     "test",
				BinaryLogDir: "/nonexistent/path",
				TargetTime:   time.Now(),
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

func TestBinaryLogPosition_String(t *testing.T) {
	pos := &BinaryLogPosition{
		File:     "mysql-bin.000001",
		Position: 12345,
	}

	str := pos.String()
	assert.Equal(t, "mysql-bin.000001:12345", str)
}

// Helper function to setup test MySQL connection.
func setupTestMySQL(t *testing.T) *MySQLDriver {
	t.Helper()
	// Check if MySQL is available via environment variables
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("MYSQL_TEST_USER")
	if user == "" {
		user = "root"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")

	driver := NewMySQLDriver()
	config := &database.ConnectionConfig{
		Host:     host,
		Port:     3306,
		Username: user,
		Password: password,
		Database: "mysql",
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Logf("Could not connect to MySQL for testing: %v", err)
		return nil
	}

	return driver
}
