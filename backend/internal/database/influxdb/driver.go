// Package influxdb provides InfluxDB database driver implementation
package influxdb

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// InfluxDBDriver implements the database.Driver interface for InfluxDB.
//
//nolint:revive // InfluxDBDriver is a public name used by other packages; keep stable.
type InfluxDBDriver struct {
	client       influxdb2.Client
	config       *database.ConnectionConfig
	queryAPI     api.QueryAPI
	version      string
	organization string
}

func init() {
	database.RegisterDriver("influxdb", func() database.Driver {
		return NewInfluxDBDriver()
	})
}

// NewInfluxDBDriver creates a new InfluxDB driver instance.
func NewInfluxDBDriver() *InfluxDBDriver {
	return &InfluxDBDriver{}
}

// Connect establishes a connection to InfluxDB.
func (d *InfluxDBDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Build InfluxDB connection URL
	hostPort := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	url := "http://" + hostPort
	if config.SSLMode == "enable" || config.SSLMode == "require" {
		url = "https://" + hostPort
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

// Disconnect closes the database connection.
func (d *InfluxDBDriver) Disconnect() error {
	if d.client != nil {
		d.client.Close()
	}
	return nil
}

// Ping tests the database connection.
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

// Backup creates a backup of the InfluxDB database.
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

// backupV1 performs backup for InfluxDB v1.x using influxd backup command.
func (d *InfluxDBDriver) backupV1(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) (*database.BackupResult, error) {
	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
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
	if walkErr := filepath.Walk(backupDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	}); walkErr != nil {
		size = 0
	}

	result.BackupPath = backupDir
	result.BackupSize = size
	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["influxdb_version"] = d.version
	result.Metadata["backup_output"] = string(output)

	return result, nil
}

// backupV2 performs backup for InfluxDB v2.x using API.
func (d *InfluxDBDriver) backupV2(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) (*database.BackupResult, error) {
	backupDir := filepath.Join(opts.OutputDir, result.ID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
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

// backupRecord is the on-disk serialization format for a single InfluxDB v2
// data point (one field value). It is written as one JSON object per line
// (NDJSON) so that a backup can be read back losslessly by restoreBucket.
// Measurement, timestamp, the field name/value and all tags are preserved.
type backupRecord struct {
	Time        time.Time         `json:"time"`
	Value       interface{}       `json:"value"`
	Tags        map[string]string `json:"tags,omitempty"`
	Measurement string            `json:"measurement"`
	Field       string            `json:"field"`
}

// reservedFluxColumns are the columns present in a (non-pivoted) Flux result
// that are not user-defined tags. Everything else in a record's Values() map
// is treated as a tag.
var reservedFluxColumns = map[string]struct{}{
	"result":       {},
	"table":        {},
	"_start":       {},
	"_stop":        {},
	"_time":        {},
	"_value":       {},
	"_field":       {},
	"_measurement": {},
}

// newBackupRecord builds a backupRecord from the components of a Flux record.
// The values map is the full record.Values(); any key that is not a reserved
// Flux column is stored as a tag (stringified to match line-protocol semantics).
func newBackupRecord(measurement, field string, value interface{}, ts time.Time, values map[string]interface{}) *backupRecord {
	rec := &backupRecord{
		Measurement: measurement,
		Time:        ts,
		Field:       field,
		Value:       value,
	}

	for key, v := range values {
		if _, reserved := reservedFluxColumns[key]; reserved {
			continue
		}
		if v == nil {
			continue
		}
		if rec.Tags == nil {
			rec.Tags = make(map[string]string)
		}
		if s, ok := v.(string); ok {
			rec.Tags[key] = s
		} else {
			rec.Tags[key] = fmt.Sprintf("%v", v)
		}
	}

	return rec
}

// toPoint converts a decoded backupRecord back into an InfluxDB write point.
func (r *backupRecord) toPoint() *write.Point {
	fields := map[string]interface{}{
		r.Field: r.Value,
	}
	return influxdb2.NewPoint(r.Measurement, r.Tags, fields, r.Time)
}

// backupBucket backs up a single bucket to a file.
func (d *InfluxDBDriver) backupBucket(ctx context.Context, bucket, outputFile string, opts *database.BackupOptions) (int64, error) {
	// Create output file
	file, err := os.Create(outputFile)
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

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

	// Note: the query is intentionally NOT pivoted. Keeping one row per field
	// value means _field/_value/_measurement are explicit and every remaining
	// column is unambiguously a tag, which lets the backup round-trip cleanly.
	query := fmt.Sprintf(`
		from(bucket: "%s")
		  |> range(start: %s)
	`, bucket, startTime)

	// Execute query and write to file
	queryResult, err := d.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, err
	}

	written := int64(0)
	for queryResult.Next() {
		record := queryResult.Record()

		rec := newBackupRecord(
			record.Measurement(),
			record.Field(),
			record.Value(),
			record.Time(),
			record.Values(),
		)

		encoded, err := json.Marshal(rec)
		if err != nil {
			return written, err
		}
		encoded = append(encoded, '\n')

		n, err := file.Write(encoded)
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

// backupMetadata backs up retention policies, tasks, and other metadata.
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
	taskList := make([]map[string]interface{}, 0, len(tasks))
	for i := range tasks {
		task := &tasks[i]
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
				if len(bucket.RetentionRules) > 0 {
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

// getBucketsToBackup returns the list of buckets to backup.
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

	bucketNames := make([]string, 0, len(*buckets))
	for _, bucket := range *buckets {
		// Skip system buckets
		if strings.HasPrefix(bucket.Name, "_") {
			continue
		}
		bucketNames = append(bucketNames, bucket.Name)
	}

	return bucketNames, nil
}

// Restore restores InfluxDB from a backup.
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

// restoreV1 performs restore for InfluxDB v1.x.
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

// restoreV2 performs restore for InfluxDB v2.x.
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

// restoreBucket restores a single bucket from a file.
func (d *InfluxDBDriver) restoreBucket(ctx context.Context, bucket, inputFile string) error {
	// Open the NDJSON file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

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

		// Skip empty lines
		if len(line) == 0 {
			continue
		}
		lineCount++

		// Decode the line using the same structured format written by
		// backupBucket, preserving measurement, tags, field and timestamp.
		var rec backupRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			errorCount++
			// Log error but continue processing
			continue
		}

		if rec.Measurement == "" || rec.Field == "" {
			errorCount++
			continue
		}

		// Write the point
		if err := writeAPI.WritePoint(ctx, rec.toPoint()); err != nil {
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

// GetDatabases returns the list of databases/buckets.
func (d *InfluxDBDriver) GetDatabases(ctx context.Context) ([]string, error) {
	bucketsAPI := d.client.BucketsAPI()
	buckets, err := bucketsAPI.GetBuckets(ctx)
	if err != nil {
		return nil, err
	}

	bucketNames := make([]string, 0, len(*buckets))
	for _, bucket := range *buckets {
		bucketNames = append(bucketNames, bucket.Name)
	}

	return bucketNames, nil
}

// GetTables returns the list of measurements (tables) in a database/bucket.
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

// GetTableSize returns the size of a measurement (not accurate for time-series).
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

// GetDatabaseSize returns the total on-disk size (in bytes) of the InfluxDB
// storage engine.
//
// InfluxDB v2 does not expose per-bucket byte size through its client API, but
// it publishes the on-disk shard size on the Prometheus /metrics endpoint via
// the storage_shard_disk_size gauge. We fetch and sum those gauges to obtain a
// real value. If the metric is not exposed (or metrics are unreachable) we
// return an honest error rather than a fabricated size.
func (d *InfluxDBDriver) GetDatabaseSize(ctx context.Context) (size int64, err error) {
	if d.client == nil {
		return 0, pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}

	metricsURL := strings.TrimSuffix(d.client.ServerURL(), "/") + "/metrics"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, http.NoBody)
	if err != nil {
		return 0, pkgErrors.Wrap(err, pkgErrors.ErrorTypeDatabase, "failed to build metrics request")
	}

	token := d.config.Password
	if token == "" {
		token = d.config.Options["token"]
	}
	if token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, pkgErrors.Wrap(err, pkgErrors.ErrorTypeDatabase, "failed to fetch InfluxDB metrics")
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, pkgErrors.New(pkgErrors.ErrorTypeDatabase,
			fmt.Sprintf("failed to fetch InfluxDB metrics: status %d", resp.StatusCode))
	}

	total, found, perr := sumShardDiskSize(resp.Body)
	if perr != nil {
		return 0, pkgErrors.Wrap(perr, pkgErrors.ErrorTypeDatabase, "failed to parse InfluxDB metrics")
	}
	if !found {
		return 0, pkgErrors.New(pkgErrors.ErrorTypeDatabase,
			"database size unavailable: InfluxDB did not expose the storage_shard_disk_size metric")
	}

	return total, nil
}

// sumShardDiskSize parses a Prometheus text-format metrics stream and sums the
// values of the storage_shard_disk_size gauge (bytes on disk). It reports
// whether any such metric line was found so callers can distinguish "size is
// zero" from "metric not exposed".
func sumShardDiskSize(r io.Reader) (total int64, found bool, err error) {
	const metricName = "storage_shard_disk_size"

	scanner := bufio.NewScanner(r)
	const maxCapacity = 1024 * 1024 // 1MB
	scanner.Buffer(make([]byte, maxCapacity), maxCapacity)

	var sum float64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, metricName) {
			continue
		}

		// Guard against metrics that merely share the same prefix
		// (e.g. storage_shard_disk_size_something). The character after the
		// metric name must start a label set or the value separator.
		rest := line[len(metricName):]
		if rest != "" && rest[0] != '{' && rest[0] != ' ' {
			continue
		}

		// Prometheus line: <name>[{labels}] <value> [timestamp]
		idx := strings.LastIndexByte(line, ' ')
		if idx < 0 {
			continue
		}
		v, perr := strconv.ParseFloat(line[idx+1:], 64)
		if perr != nil {
			continue
		}
		sum += v
		found = true
	}

	if serr := scanner.Err(); serr != nil {
		return 0, false, serr
	}

	return int64(sum), found, nil
}

// GetVersion returns the InfluxDB version.
func (d *InfluxDBDriver) GetVersion(ctx context.Context) (string, error) {
	return fmt.Sprintf("InfluxDB %s", d.version), nil
}

// StreamBackup streams backup data to a writer.
func (d *InfluxDBDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for InfluxDB")
}

// StreamRestore streams restore data from a reader.
func (d *InfluxDBDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for InfluxDB")
}

// GetBackupSize returns the estimated size of a backup.
func (d *InfluxDBDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	// Estimate based on current database size
	return d.GetDatabaseSize(ctx)
}

// ValidateRestore validates restore options.
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

// GetType returns the database type.
func (d *InfluxDBDriver) GetType() database.DatabaseType {
	return "influxdb"
}

// SupportsIncremental returns whether the driver supports incremental backups.
func (d *InfluxDBDriver) SupportsIncremental() bool {
	return true
}

// SupportsPITR returns whether the driver supports point-in-time recovery.
func (d *InfluxDBDriver) SupportsPITR() bool {
	// InfluxDB doesn't have native PITR, but we can do time-range based restores
	return false
}
