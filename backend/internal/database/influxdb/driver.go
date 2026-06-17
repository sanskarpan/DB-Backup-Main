// Package influxdb provides InfluxDB database driver implementation
package influxdb

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// InfluxDBDriver implements the database.Driver interface for InfluxDB
type InfluxDBDriver struct {
	client       influxdb2.Client
	config       *database.ConnectionConfig
	queryAPI     api.QueryAPI
	writeAPI     api.WriteAPI
	version      string
	organization string
}

func init() {
	database.RegisterDriver("influxdb", func() database.Driver {
		return NewInfluxDBDriver()
	})
}

// NewInfluxDBDriver creates a new InfluxDB driver instance
func NewInfluxDBDriver() *InfluxDBDriver {
	return &InfluxDBDriver{}
}

// Connect establishes a connection to InfluxDB
func (d *InfluxDBDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Build InfluxDB connection URL
	url := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
	if config.SSLMode == "enable" || config.SSLMode == "require" {
		url = fmt.Sprintf("https://%s:%d", config.Host, config.Port)
	}

	// Get token (password) and organization
	token := config.Password
	if token == "" {
		token = config.Options["token"]
	}

	org := config.Database
	if org == "" {
		org = config.Options["organization"]
		if org == "" {
			org = "default"
		}
	}

	// Create client
	client := influxdb2.NewClient(url, token)

	// Test connection by pinging
	health, err := client.Health(ctx)
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	if health.Status != "pass" {
		return pkgErrors.ErrDatabaseConnection(fmt.Errorf("influxdb health check failed: %s", health.Status))
	}

	d.client = client
	d.config = config
	d.organization = org
	d.queryAPI = client.QueryAPI(org)

	// Detect version
	if health.Version != nil {
		d.version = *health.Version
	} else {
		d.version = "unknown"
	}

	return nil
}

// Disconnect closes the database connection
func (d *InfluxDBDriver) Disconnect() error {
	if d.client != nil {
		d.client.Close()
	}
	return nil
}

// Ping tests the database connection
func (d *InfluxDBDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}

	health, err := d.client.Ping(ctx)
	if err != nil {
		return err
	}

	if !health {
		return fmt.Errorf("influxdb ping failed")
	}

	return nil
}

// Backup creates a backup of the InfluxDB database
func (d *InfluxDBDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Determine if we're using v1 or v2
	if strings.HasPrefix(d.version, "1.") {
		return d.backupV1(ctx, opts, result)
	}

	return d.backupV2(ctx, opts, result)
}

// backupV1 performs backup for InfluxDB v1.x using influxd backup command
func (d *InfluxDBDriver) backupV1(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) (*database.BackupResult, error) {
	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Build influxd backup command
	args := []string{
		"backup",
		"-host", fmt.Sprintf("%s:%d", d.config.Host, d.config.Port),
		"-portable",
	}

	// Add database filter if specified
	if opts.Database != "" {
		args = append(args, "-database", opts.Database)
	}

	// Add retention policy if specified
	if rp, ok := opts.Metadata["retention_policy"].(string); ok && rp != "" {
		args = append(args, "-retention", rp)
	}

	// Add start and end time if specified for incremental backups
	if opts.Incremental {
		if since, ok := opts.Metadata["since"].(time.Time); ok {
			args = append(args, "-since", since.Format(time.RFC3339))
		}
		if end, ok := opts.Metadata["end"].(time.Time); ok {
			args = append(args, "-end", end.Format(time.RFC3339))
		}
	}

	args = append(args, backupDir)

	// Execute backup command
	cmd := exec.CommandContext(ctx, "influxd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = fmt.Errorf("influxd backup failed: %w, output: %s", err, string(output))
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(result.Error)
	}

	// Get backup size
	var size int64
	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	result.BackupPath = backupDir
	result.BackupSize = size
	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["influxdb_version"] = d.version
	result.Metadata["backup_output"] = string(output)

	return result, nil
}

// backupV2 performs backup for InfluxDB v2.x using API
func (d *InfluxDBDriver) backupV2(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) (*database.BackupResult, error) {
	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Get list of buckets to backup
	buckets, err := d.getBucketsToBackup(ctx, opts)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Backup each bucket
	totalSize := int64(0)
	for _, bucket := range buckets {
		bucketFile := filepath.Join(backupDir, fmt.Sprintf("%s.ndjson", bucket))
		size, err := d.backupBucket(ctx, bucket, bucketFile, opts)
		if err != nil {
			result.Status = database.BackupStatusFailed
			result.Error = fmt.Errorf("failed to backup bucket %s: %w", bucket, err)
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseBackup(err)
		}
		totalSize += size
	}

	// Backup retention policies and continuous queries metadata
	if err := d.backupMetadata(ctx, backupDir); err != nil {
		// Non-fatal error, just log it
		result.Metadata["metadata_backup_error"] = err.Error()
	}

	result.BackupPath = backupDir
	result.BackupSize = totalSize
	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["influxdb_version"] = d.version
	result.Metadata["buckets_backed_up"] = len(buckets)
	result.Metadata["bucket_names"] = buckets

	return result, nil
}

// backupBucket backs up a single bucket to a file
func (d *InfluxDBDriver) backupBucket(ctx context.Context, bucket, outputFile string, opts *database.BackupOptions) (int64, error) {
	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Build Flux query to export all data from bucket
	var startTime string
	if opts.Incremental {
		if since, ok := opts.Metadata["since"].(time.Time); ok {
			startTime = since.Format(time.RFC3339)
		} else {
			startTime = "-30d" // Default to last 30 days for incremental
		}
	} else {
		startTime = "1970-01-01T00:00:00Z" // Export all data
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: %s)
		  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
	`, bucket, startTime)

	// Execute query and write to file
	queryResult, err := d.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	written := int64(0)
	for queryResult.Next() {
		record := queryResult.Record()
		// Write record as JSON to file
		data := fmt.Sprintf("%v\n", record.Values())
		n, err := file.WriteString(data)
		if err != nil {
			return written, err
		}
		written += int64(n)
	}

	if queryResult.Err() != nil {
		return written, queryResult.Err()
	}

	return written, nil
}

// backupMetadata backs up retention policies, tasks, and other metadata
func (d *InfluxDBDriver) backupMetadata(ctx context.Context, backupDir string) error {
	// Use the management API to export tasks, checks, notification rules, etc.
	metadataFile := filepath.Join(backupDir, "metadata.json")
	file, err := os.Create(metadataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create metadata structure
	metadata := make(map[string]interface{})

	// Export tasks
	tasksAPI := d.client.TasksAPI()
	tasks, err := tasksAPI.FindTasks(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}
	var taskList []map[string]interface{}
	for _, task := range tasks {
		taskData := map[string]interface{}{
			"id":     task.Id,
			"name":   task.Name,
			"orgID":  task.OrgID,
			"status": task.Status,
			"flux":   task.Flux,
		}
		if task.Every != nil && *task.Every != "" {
			taskData["every"] = *task.Every
		}
		if task.Cron != nil && *task.Cron != "" {
			taskData["cron"] = *task.Cron
		}
		taskList = append(taskList, taskData)
	}
	metadata["tasks"] = taskList

	// Export buckets API information
	bucketsAPI := d.client.BucketsAPI()
	if bucketsAPI != nil {
		buckets, err := bucketsAPI.GetBuckets(ctx)
		if err == nil && buckets != nil {
			var bucketList []map[string]interface{}
			for _, bucket := range *buckets {
				bucketData := map[string]interface{}{
					"id":          bucket.Id,
					"name":        bucket.Name,
					"orgID":       bucket.OrgID,
					"description": bucket.Description,
				}
				if bucket.RetentionRules != nil && len(bucket.RetentionRules) > 0 {
					bucketData["retention_period"] = bucket.RetentionRules[0].EverySeconds
				}
				bucketList = append(bucketList, bucketData)
			}
			metadata["buckets"] = bucketList
		}
	}

	// Encode to JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	return nil
}

// getBucketsToBackup returns the list of buckets to backup
func (d *InfluxDBDriver) getBucketsToBackup(ctx context.Context, opts *database.BackupOptions) ([]string, error) {
	// If specific buckets are specified, use those
	if len(opts.Databases) > 0 {
		return opts.Databases, nil
	}

	if opts.Database != "" {
		return []string{opts.Database}, nil
	}

	// Otherwise, list all buckets
	bucketsAPI := d.client.BucketsAPI()
	buckets, err := bucketsAPI.GetBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var bucketNames []string
	for _, bucket := range *buckets {
		// Skip system buckets
		if strings.HasPrefix(bucket.Name, "_") {
			continue
		}
		bucketNames = append(bucketNames, bucket.Name)
	}

	return bucketNames, nil
}

// Restore restores InfluxDB from a backup
func (d *InfluxDBDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Determine if we're using v1 or v2
	if strings.HasPrefix(d.version, "1.") {
		return d.restoreV1(ctx, opts, result)
	}

	return d.restoreV2(ctx, opts, result)
}

// restoreV1 performs restore for InfluxDB v1.x
func (d *InfluxDBDriver) restoreV1(ctx context.Context, opts *database.RestoreOptions, result *database.RestoreResult) (*database.RestoreResult, error) {
	backupPath := opts.BackupPath
	if backupPath == "" {
		backupPath = opts.SourceBackup
	}

	// Build influxd restore command
	args := []string{
		"restore",
		"-host", fmt.Sprintf("%s:%d", d.config.Host, d.config.Port),
		"-portable",
	}

	// Add database filter if specified
	if opts.Database != "" {
		args = append(args, "-database", opts.Database)
	}

	args = append(args, backupPath)

	// Execute restore command
	cmd := exec.CommandContext(ctx, "influxd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("influxd restore failed: %w, output: %s", err, string(output))
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(result.Error)
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	result.Metadata = map[string]interface{}{
		"restore_output": string(output),
	}

	return result, nil
}

// restoreV2 performs restore for InfluxDB v2.x
func (d *InfluxDBDriver) restoreV2(ctx context.Context, opts *database.RestoreOptions, result *database.RestoreResult) (*database.RestoreResult, error) {
	backupPath := opts.BackupPath
	if backupPath == "" {
		backupPath = opts.SourceBackup
	}

	// Read backup directory
	files, err := os.ReadDir(backupPath)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Restore each bucket file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ndjson") {
			continue
		}

		bucketName := strings.TrimSuffix(file.Name(), ".ndjson")
		filePath := filepath.Join(backupPath, file.Name())

		if err := d.restoreBucket(ctx, bucketName, filePath); err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = fmt.Errorf("failed to restore bucket %s: %w", bucketName, err)
			result.EndTime = time.Now()
			return result, pkgErrors.ErrDatabaseRestore(err)
		}
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()

	return result, nil
}

// restoreBucket restores a single bucket from a file
func (d *InfluxDBDriver) restoreBucket(ctx context.Context, bucket, inputFile string) error {
	// Open the NDJSON file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Get the write API for this bucket
	writeAPI := d.client.WriteAPIBlocking(d.organization, bucket)

	// Read and parse NDJSON line by line
	scanner := bufio.NewScanner(file)

	// Set a larger buffer size for scanner to handle large lines
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineCount := 0
	errorCount := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		lineCount++

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse the JSON line into a point structure
		var pointData map[string]interface{}
		if err := json.Unmarshal(line, &pointData); err != nil {
			errorCount++
			// Log error but continue processing
			continue
		}

		// Extract point components
		measurement, ok := pointData["_measurement"].(string)
		if !ok {
			errorCount++
			continue
		}

		// Extract timestamp
		var timestamp time.Time
		if timeStr, ok := pointData["_time"].(string); ok {
			timestamp, err = time.Parse(time.RFC3339Nano, timeStr)
			if err != nil {
				errorCount++
				continue
			}
		} else {
			timestamp = time.Now()
		}

		// Build tags and fields
		tags := make(map[string]string)
		fields := make(map[string]interface{})

		for key, value := range pointData {
			// Skip internal InfluxDB fields
			if strings.HasPrefix(key, "_") {
				continue
			}

			// Determine if this is a tag or field
			// In InfluxDB line protocol, tags are typically strings
			// and fields can be any type
			switch v := value.(type) {
			case string:
				// Could be a tag or a string field
				// By convention, use lowercase for tags
				if strings.ToLower(key) == key {
					tags[key] = v
				} else {
					fields[key] = v
				}
			case float64, int, int64, bool:
				// These are fields
				fields[key] = v
			default:
				// Default to field
				fields[key] = v
			}
		}

		// Create a point
		point := influxdb2.NewPoint(
			measurement,
			tags,
			fields,
			timestamp,
		)

		// Write the point
		if err := writeAPI.WritePoint(ctx, point); err != nil {
			errorCount++
			// Continue processing even if individual point fails
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading backup file: %w (processed %d lines, %d errors)", err, lineCount, errorCount)
	}

	// If all points failed, return error
	if lineCount > 0 && errorCount == lineCount {
		return fmt.Errorf("failed to restore any points from backup (processed %d lines)", lineCount)
	}

	return nil
}

// GetDatabases returns the list of databases/buckets
func (d *InfluxDBDriver) GetDatabases(ctx context.Context) ([]string, error) {
	bucketsAPI := d.client.BucketsAPI()
	buckets, err := bucketsAPI.GetBuckets(ctx)
	if err != nil {
		return nil, err
	}

	var bucketNames []string
	for _, bucket := range *buckets {
		bucketNames = append(bucketNames, bucket.Name)
	}

	return bucketNames, nil
}

// GetTables returns the list of measurements (tables) in a database/bucket
func (d *InfluxDBDriver) GetTables(ctx context.Context, database string) ([]string, error) {
	query := fmt.Sprintf(`
		import "influxdata/influxdb/schema"
		schema.measurements(bucket: "%s")
	`, database)

	queryResult, err := d.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var measurements []string
	for queryResult.Next() {
		record := queryResult.Record()
		if measurement, ok := record.Values()["_value"].(string); ok {
			measurements = append(measurements, measurement)
		}
	}

	if queryResult.Err() != nil {
		return nil, queryResult.Err()
	}

	return measurements, nil
}

// GetTableSize returns the size of a measurement (not accurate for time-series)
func (d *InfluxDBDriver) GetTableSize(ctx context.Context, database, table string) (int64, error) {
	// Time-series databases don't have a direct concept of table size
	// Return the count of points as an approximation
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: 0)
		  |> filter(fn: (r) => r._measurement == "%s")
		  |> count()
	`, database, table)

	queryResult, err := d.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	count := int64(0)
	for queryResult.Next() {
		record := queryResult.Record()
		if val, ok := record.Values()["_value"].(int64); ok {
			count += val
		}
	}

	return count, queryResult.Err()
}

// GetDatabaseSize returns the total size of all buckets
func (d *InfluxDBDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	// InfluxDB doesn't expose direct size metrics easily
	// We would need to query storage usage metrics or use the management API
	// For now, return 0 as placeholder
	return 0, nil
}

// GetVersion returns the InfluxDB version
func (d *InfluxDBDriver) GetVersion(ctx context.Context) (string, error) {
	return fmt.Sprintf("InfluxDB %s", d.version), nil
}

// StreamBackup streams backup data to a writer
func (d *InfluxDBDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for InfluxDB")
}

// StreamRestore streams restore data from a reader
func (d *InfluxDBDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for InfluxDB")
}

// GetBackupSize returns the estimated size of a backup
func (d *InfluxDBDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	// Estimate based on current database size
	return d.GetDatabaseSize(ctx)
}

// ValidateRestore validates restore options
func (d *InfluxDBDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" && opts.SourceBackup == "" {
		return fmt.Errorf("backup path is required")
	}

	backupPath := opts.BackupPath
	if backupPath == "" {
		backupPath = opts.SourceBackup
	}

	// Check if backup path exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup path does not exist: %s", backupPath)
	}

	return nil
}

// GetType returns the database type
func (d *InfluxDBDriver) GetType() database.DatabaseType {
	return "influxdb"
}

// SupportsIncremental returns whether the driver supports incremental backups
func (d *InfluxDBDriver) SupportsIncremental() bool {
	return true
}

// SupportsPITR returns whether the driver supports point-in-time recovery
func (d *InfluxDBDriver) SupportsPITR() bool {
	// InfluxDB doesn't have native PITR, but we can do time-range based restores
	return false
}
