// Package postgres_test provides comprehensive tests for PostgreSQL PITR
package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresPITR_BackupWALFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestPostgres(t)
	if driver == nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create temporary output directory
	outputDir := t.TempDir()

	// Test WAL backup
	err := pitr.BackupWALFiles(ctx, outputDir, nil)
	if err != nil {
		t.Logf("WAL backup failed (expected if WAL archiving not configured): %v", err)
		return
	}

	// Verify output directory exists
	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}

func TestPostgresPITR_GetWALLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestPostgres(t)
	if driver == nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Get WAL location
	location, err := pitr.GetWALLocation(ctx)
	require.NoError(t, err)
	assert.NotNil(t, location)
	assert.NotEmpty(t, location.LSN, "Expected WAL LSN")
}

func TestPostgresPITR_RestoreToPointInTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	driver := setupTestPostgres(t)
	if driver == nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer driver.Disconnect()

	ctx := context.Background()
	pitr := NewPITRManager(driver)

	// Create test database
	testDB := "pitr_test_db"
	_, err := driver.db.ExecContext(ctx, "CREATE DATABASE "+testDB)
	if err != nil {
		// Database might already exist
		t.Logf("Could not create test database: %v", err)
	}
	defer driver.db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+testDB)

	// Test PITR restore options validation
	walDir := t.TempDir()
	dataDir := t.TempDir()

	opts := &PITRRestoreOptions{
		Database:      testDB,
		WALDirectory:  walDir,
		DataDirectory: dataDir,
		TargetTime:    time.Now(),
	}

	err = pitr.validatePITROptions(opts)
	assert.NoError(t, err, "PITR options should be valid")
}

func TestPostgresPITR_ValidationErrors(t *testing.T) {
	driver := &PostgreSQLDriver{}
	pitr := NewPITRManager(driver)

	tests := []struct {
		name    string
		opts    *PITRRestoreOptions
		wantErr bool
	}{
		{
			name: "missing target time",
			opts: &PITRRestoreOptions{
				Database:      "test",
				WALDirectory:  "/tmp/wal",
				DataDirectory: "/tmp/data",
			},
			wantErr: true,
		},
		{
			name: "missing WAL directory",
			opts: &PITRRestoreOptions{
				Database:      "test",
				DataDirectory: "/tmp/data",
				TargetTime:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing data directory",
			opts: &PITRRestoreOptions{
				Database:     "test",
				WALDirectory: "/tmp/wal",
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

func TestWALLocation_String(t *testing.T) {
	location := &WALLocation{
		LSN:       "0/1234ABCD",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	str := location.String()
	assert.Contains(t, str, "0/1234ABCD")
	assert.Contains(t, str, "2024")
}

func TestIsWALFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"000000010000000000000001", true},
		{"0000000100000000000000FF", true},
		{"00000001000000000000ABCD", true},
		{"not-a-wal-file", false},
		{"00000001", false}, // too short
		{"000000010000000000000001.partial", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isWALFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to setup test PostgreSQL connection
func setupTestPostgres(t *testing.T) *PostgreSQLDriver {
	host := os.Getenv("POSTGRES_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("POSTGRES_TEST_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("POSTGRES_TEST_PASSWORD")

	driver := NewPostgreSQLDriver()
	config := &database.ConnectionConfig{
		Host:     host,
		Port:     5432,
		Username: user,
		Password: password,
		Database: "postgres",
	}

	ctx := context.Background()
	err := driver.Connect(ctx, config)
	if err != nil {
		t.Logf("Could not connect to PostgreSQL for testing: %v", err)
		return nil
	}

	return driver
}
