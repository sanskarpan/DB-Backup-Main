// Package mongodb provides Point-in-Time Recovery for MongoDB
package mongodb

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// PITRManager handles Point-in-Time Recovery for MongoDB
type PITRManager struct {
	driver *MongoDBDriver
}

// NewPITRManager creates a new PITR manager
func NewPITRManager(driver *MongoDBDriver) *PITRManager {
	return &PITRManager{
		driver: driver,
	}
}

// BackupOplog backs up MongoDB oplog for PITR
func (p *PITRManager) BackupOplog(ctx context.Context, outputDir string, since *time.Time) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return pkgErrors.ErrDatabaseBackup(err).WithMetadata("output_dir", outputDir)
	}

	// Get oplog position
	oplogPos, err := p.getCurrentOplogPosition(ctx)
	if err != nil {
		return err
	}

	// Build mongodump command for oplog
	args := p.buildOplogDumpArgs(outputDir, since)

	cmd := exec.CommandContext(ctx, "mongodump", args...)

	if err := cmd.Run(); err != nil {
		return pkgErrors.ErrDatabaseBackup(err).WithMetadata("command", "mongodump")
	}

	// Save oplog metadata
	if err := p.saveOplogMetadata(oplogPos, filepath.Join(outputDir, "oplog.metadata")); err != nil {
		return err
	}

	return nil
}

// RestoreToPointInTime restores database to a specific point in time
func (p *PITRManager) RestoreToPointInTime(ctx context.Context, opts *PITRRestoreOptions) error {
	// Validate options
	if err := p.validatePITROptions(opts); err != nil {
		return err
	}

	// Step 1: Restore base backup
	if opts.BaseBackupPath != "" {
		if err := p.restoreBaseBackup(ctx, opts); err != nil {
			return fmt.Errorf("failed to restore base backup: %w", err)
		}
	}

	// Step 2: Apply oplog up to the point in time
	if err := p.applyOplog(ctx, opts); err != nil {
		return fmt.Errorf("failed to apply oplog: %w", err)
	}

	// Step 3: Verify the restore
	if !opts.SkipVerification {
		if err := p.verifyPITRRestore(ctx, opts); err != nil {
			return fmt.Errorf("PITR verification failed: %w", err)
		}
	}

	return nil
}

// GetOplogPosition gets the current oplog position
func (p *PITRManager) GetOplogPosition(ctx context.Context) (*OplogPosition, error) {
	return p.getCurrentOplogPosition(ctx)
}

// Helper methods

func (p *PITRManager) getCurrentOplogPosition(ctx context.Context) (*OplogPosition, error) {
	// Get the local database (where oplog resides)
	localDB := p.driver.client.Database("local")

	// Get the latest oplog entry
	oplogCollection := localDB.Collection("oplog.rs")

	opts := options.FindOne().SetSort(bson.D{{Key: "$natural", Value: -1}})
	var result bson.M

	err := oplogCollection.FindOne(ctx, bson.D{}, opts).Decode(&result)
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeDatabase, fmt.Sprintf("failed to get oplog position: %v", err))
	}

	// Extract timestamp from oplog entry
	ts, ok := result["ts"].(primitive.Timestamp)
	if !ok {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeDatabase, "invalid oplog timestamp format")
	}

	return &OplogPosition{
		Timestamp: ts,
		Time:      time.Now(),
	}, nil
}

func (p *PITRManager) buildOplogDumpArgs(outputDir string, since *time.Time) []string {
	args := []string{
		"--host", p.driver.config.Host,
		"--port", fmt.Sprintf("%d", p.driver.config.Port),
		"--db", "local",
		"--collection", "oplog.rs",
		"--out", outputDir,
		"--gzip",
	}

	if p.driver.config.Username != "" {
		args = append(args, "--username", p.driver.config.Username)
	}

	if p.driver.config.Password != "" {
		args = append(args, "--password", p.driver.config.Password)
	}

	// Filter oplog entries by timestamp if provided
	if since != nil {
		// Convert time to MongoDB timestamp (seconds + increment)
		ts := primitive.Timestamp{
			T: uint32(since.Unix()),
			I: 0,
		}
		query := fmt.Sprintf(`{"ts":{"$gte":{"$timestamp":{"t":%d,"i":%d}}}}`, ts.T, ts.I)
		args = append(args, "--query", query)
	}

	return args
}

func (p *PITRManager) saveOplogMetadata(pos *OplogPosition, metadataPath string) error {
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "Timestamp: T=%d I=%d\nTime: %s\n",
		pos.Timestamp.T,
		pos.Timestamp.I,
		pos.Time.Format(time.RFC3339))

	return err
}

func (p *PITRManager) validatePITROptions(opts *PITRRestoreOptions) error {
	if opts.TargetTime.IsZero() {
		return pkgErrors.ErrValidationFailed("target time is required for PITR")
	}

	if opts.OplogDir == "" {
		return pkgErrors.ErrValidationFailed("oplog directory is required")
	}

	if _, err := os.Stat(opts.OplogDir); os.IsNotExist(err) {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("oplog directory not found: %s", opts.OplogDir))
	}

	return nil
}

func (p *PITRManager) restoreBaseBackup(ctx context.Context, opts *PITRRestoreOptions) error {
	// Build mongorestore command
	args := []string{
		"--host", p.driver.config.Host,
		"--port", fmt.Sprintf("%d", p.driver.config.Port),
		"--gzip",
		"--drop", // Drop existing collections before restoring
	}

	if p.driver.config.Username != "" {
		args = append(args, "--username", p.driver.config.Username)
	}

	if p.driver.config.Password != "" {
		args = append(args, "--password", p.driver.config.Password)
	}

	if opts.Database != "" {
		args = append(args, "--db", opts.Database)
	}

	args = append(args, opts.BaseBackupPath)

	cmd := exec.CommandContext(ctx, "mongorestore", args...)

	if err := cmd.Run(); err != nil {
		return pkgErrors.ErrDatabaseRestore(err).WithMetadata("command", "mongorestore")
	}

	return nil
}

func (p *PITRManager) applyOplog(ctx context.Context, opts *PITRRestoreOptions) error {
	// Path to oplog backup
	oplogPath := filepath.Join(opts.OplogDir, "local", "oplog.rs.bson.gz")

	// Check if oplog file exists
	if _, err := os.Stat(oplogPath); os.IsNotExist(err) {
		// Try without .gz extension
		oplogPath = filepath.Join(opts.OplogDir, "local", "oplog.rs.bson")
		if _, err := os.Stat(oplogPath); os.IsNotExist(err) {
			return pkgErrors.ErrValidationFailed(fmt.Sprintf("oplog file not found in %s", opts.OplogDir))
		}
	}

	// Calculate target timestamp
	targetTimestamp := primitive.Timestamp{
		T: uint32(opts.TargetTime.Unix()),
		I: 0,
	}

	// Build mongorestore command with oplog replay
	args := []string{
		"--host", p.driver.config.Host,
		"--port", fmt.Sprintf("%d", p.driver.config.Port),
		"--oplogReplay",
		"--oplogLimit", fmt.Sprintf("%d:%d", targetTimestamp.T, targetTimestamp.I),
		"--gzip",
	}

	if p.driver.config.Username != "" {
		args = append(args, "--username", p.driver.config.Username)
	}

	if p.driver.config.Password != "" {
		args = append(args, "--password", p.driver.config.Password)
	}

	if opts.Database != "" {
		args = append(args, "--db", opts.Database)
	}

	args = append(args, opts.OplogDir)

	cmd := exec.CommandContext(ctx, "mongorestore", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mongorestore oplog replay failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (p *PITRManager) verifyPITRRestore(ctx context.Context, opts *PITRRestoreOptions) error {
	// Check database is accessible
	if err := p.driver.Ping(ctx); err != nil {
		return pkgErrors.ErrValidationFailed("database not accessible after PITR restore")
	}

	// Verify database exists
	databases, err := p.driver.GetDatabases(ctx)
	if err != nil {
		return err
	}

	if opts.Database != "" {
		found := false
		for _, db := range databases {
			if db == opts.Database {
				found = true
				break
			}
		}

		if !found {
			return pkgErrors.ErrValidationFailed(fmt.Sprintf("database %s not found after PITR restore", opts.Database))
		}
	}

	// Verify collections exist
	if opts.Database != "" {
		collections, err := p.driver.GetTables(ctx, opts.Database)
		if err != nil {
			return err
		}

		if len(collections) == 0 {
			return pkgErrors.ErrValidationFailed("no collections found after PITR restore")
		}
	}

	// Verify we didn't restore beyond target time
	if err := p.verifyTimestamp(ctx, opts); err != nil {
		return err
	}

	return nil
}

func (p *PITRManager) verifyTimestamp(ctx context.Context, opts *PITRRestoreOptions) error {
	// Get the latest oplog entry after restore
	currentPos, err := p.getCurrentOplogPosition(ctx)
	if err != nil {
		// Oplog verification is optional - may not be available after restore
		return nil
	}

	// Convert target time to timestamp
	targetTimestamp := primitive.Timestamp{
		T: uint32(opts.TargetTime.Unix()),
		I: 0,
	}

	// Verify we didn't go past the target time
	if currentPos.Timestamp.T > targetTimestamp.T {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf(
			"restore went beyond target time: current %d > target %d",
			currentPos.Timestamp.T,
			targetTimestamp.T,
		))
	}

	return nil
}

// BackupWithOplog creates a backup with oplog for PITR
func (p *PITRManager) BackupWithOplog(ctx context.Context, outputDir string, database string) error {
	// Create mongodump with --oplog flag
	args := []string{
		"--host", p.driver.config.Host,
		"--port", fmt.Sprintf("%d", p.driver.config.Port),
		"--out", outputDir,
		"--oplog", // Include oplog for point-in-time consistency
		"--gzip",
	}

	if p.driver.config.Username != "" {
		args = append(args, "--username", p.driver.config.Username)
	}

	if p.driver.config.Password != "" {
		args = append(args, "--password", p.driver.config.Password)
	}

	if database != "" {
		args = append(args, "--db", database)
	}

	cmd := exec.CommandContext(ctx, "mongodump", args...)

	if err := cmd.Run(); err != nil {
		return pkgErrors.ErrDatabaseBackup(err)
	}

	return nil
}

// PITRRestoreOptions holds options for PITR restore
type PITRRestoreOptions struct {
	Database         string
	BaseBackupPath   string
	OplogDir         string
	TargetTime       time.Time
	SkipVerification bool
}

// OplogPosition represents a position in MongoDB oplog
type OplogPosition struct {
	Timestamp primitive.Timestamp
	Time      time.Time
}

// String returns string representation of oplog position
func (o *OplogPosition) String() string {
	return fmt.Sprintf("Timestamp: T=%d I=%d, Time: %s",
		o.Timestamp.T,
		o.Timestamp.I,
		o.Time.Format(time.RFC3339))
}

// TimeToTimestamp converts a time.Time to MongoDB timestamp
func TimeToTimestamp(t time.Time) primitive.Timestamp {
	return primitive.Timestamp{
		T: uint32(t.Unix()),
		I: 0,
	}
}

// TimestampToTime converts a MongoDB timestamp to time.Time
func TimestampToTime(ts primitive.Timestamp) time.Time {
	return time.Unix(int64(ts.T), 0)
}
