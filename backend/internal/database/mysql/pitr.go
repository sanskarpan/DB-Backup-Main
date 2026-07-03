// Package mysql provides Point-in-Time Recovery for MySQL
package mysql

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// PITRManager handles Point-in-Time Recovery for MySQL.
type PITRManager struct {
	driver *MySQLDriver
}

// NewPITRManager creates a new PITR manager.
func NewPITRManager(driver *MySQLDriver) *PITRManager {
	return &PITRManager{
		driver: driver,
	}
}

// BackupBinaryLogs backs up MySQL binary logs for PITR.
func (p *PITRManager) BackupBinaryLogs(ctx context.Context, outputDir string, since *time.Time) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return pkgErrors.ErrDatabaseBackup(err).WithMetadata("output_dir", outputDir)
	}

	// Get list of binary logs
	binlogs, err := p.getBinaryLogs(ctx)
	if err != nil {
		return err
	}

	// Filter logs if 'since' is provided
	if since != nil {
		binlogs, err = p.filterBinaryLogsSince(ctx, binlogs, *since)
		if err != nil {
			return err
		}
	}

	// Flush logs to ensure current log is complete
	if err := p.flushBinaryLogs(ctx); err != nil {
		return err
	}

	// Copy each binary log file
	for _, logName := range binlogs {
		if err := p.copyBinaryLog(ctx, logName, outputDir); err != nil {
			return fmt.Errorf("failed to backup binary log %s: %w", logName, err)
		}
	}

	// Save binary log index
	if err := p.saveBinaryLogIndex(binlogs, filepath.Join(outputDir, "binlog.index")); err != nil {
		return err
	}

	return nil
}

// RestoreToPointInTime restores database to a specific point in time.
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

	// Step 2: Apply binary logs up to the point in time
	if err := p.applyBinaryLogs(ctx, opts); err != nil {
		return fmt.Errorf("failed to apply binary logs: %w", err)
	}

	// Step 3: Verify the restore
	if !opts.SkipVerification {
		if err := p.verifyPITRRestore(ctx, opts); err != nil {
			return fmt.Errorf("PITR verification failed: %w", err)
		}
	}

	return nil
}

// GetBinaryLogPosition gets the current binary log position.
func (p *PITRManager) GetBinaryLogPosition(ctx context.Context) (*BinaryLogPosition, error) {
	query := "SHOW MASTER STATUS"
	rows, err := p.driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeDatabase, "no binary log position found - binary logging may not be enabled")
	}

	var pos BinaryLogPosition
	var ignore interface{}
	err = rows.Scan(&pos.File, &pos.Position, &ignore, &ignore, &ignore)
	if err != nil {
		return nil, err
	}

	return &pos, nil
}

// Helper methods

func (p *PITRManager) getBinaryLogs(ctx context.Context) ([]string, error) {
	query := "SHOW BINARY LOGS"
	rows, err := p.driver.db.QueryContext(ctx, query)
	if err != nil {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeDatabase, fmt.Sprintf("failed to list binary logs: %v - ensure binary logging is enabled", err))
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var logName string
		var fileSize int64
		if err := rows.Scan(&logName, &fileSize); err != nil {
			return nil, err
		}
		logs = append(logs, logName)
	}

	return logs, nil
}

func (p *PITRManager) filterBinaryLogsSince(ctx context.Context, logs []string, since time.Time) ([]string, error) {
	var filtered []string

	for _, logName := range logs {
		// Get first event time in this log
		firstEventTime, err := p.getFirstEventTime(ctx, logName)
		if err != nil {
			// If we can't read the log, include it to be safe
			filtered = append(filtered, logName)
			continue
		}

		// Include logs that have events after 'since'
		if firstEventTime.After(since) || firstEventTime.Equal(since) {
			filtered = append(filtered, logName)
		}
	}

	return filtered, nil
}

func (p *PITRManager) getFirstEventTime(ctx context.Context, logName string) (time.Time, error) {
	// Use mysqlbinlog to read the first timestamp
	cmd := exec.CommandContext(
		ctx, "mysqlbinlog",
		"--start-position=4",   // Skip magic number
		"--stop-position=1000", // Read only first event
		"--short-form",
		logName,
	)

	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	// Parse timestamp from output
	// Format: #YYMMDD HH:MM:SS
	re := regexp.MustCompile(`#(\d{6})\s+(\d{1,2}):(\d{2}):(\d{2})`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 5 {
		return time.Time{}, fmt.Errorf("could not parse timestamp from binary log")
	}

	// Parse the timestamp
	dateStr := matches[1]
	hour, _ := strconv.Atoi(matches[2])
	min, _ := strconv.Atoi(matches[3])
	sec, _ := strconv.Atoi(matches[4])

	year, _ := strconv.Atoi("20" + dateStr[0:2])
	month, _ := strconv.Atoi(dateStr[2:4])
	day, _ := strconv.Atoi(dateStr[4:6])

	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC), nil
}

func (p *PITRManager) flushBinaryLogs(ctx context.Context) error {
	_, err := p.driver.db.ExecContext(ctx, "FLUSH BINARY LOGS")
	return err
}

func (p *PITRManager) copyBinaryLog(ctx context.Context, logName, outputDir string) error {
	destPath := filepath.Join(outputDir, logName)

	// Use mysqlbinlog to read the log (safer than direct file copy)
	cmd := exec.CommandContext(
		ctx, "mysqlbinlog",
		"--read-from-remote-server",
		fmt.Sprintf("--host=%s", p.driver.config.Host),
		fmt.Sprintf("--port=%d", p.driver.config.Port),
		fmt.Sprintf("--user=%s", p.driver.config.Username),
		fmt.Sprintf("--password=%s", p.driver.config.Password),
		"--result-file="+destPath,
		logName,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqlbinlog failed: %w", err)
	}

	return nil
}

func (p *PITRManager) saveBinaryLogIndex(logs []string, indexPath string) error {
	file, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, log := range logs {
		_, err := writer.WriteString(log + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (p *PITRManager) validatePITROptions(opts *PITRRestoreOptions) error {
	if opts.TargetTime.IsZero() {
		return pkgErrors.ErrValidationFailed("target time is required for PITR")
	}

	if opts.BinaryLogDir == "" {
		return pkgErrors.ErrValidationFailed("binary log directory is required")
	}

	if _, err := os.Stat(opts.BinaryLogDir); os.IsNotExist(err) {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("binary log directory not found: %s", opts.BinaryLogDir))
	}

	return nil
}

func (p *PITRManager) restoreBaseBackup(ctx context.Context, opts *PITRRestoreOptions) error {
	// Create restore options for base backup
	restoreOpts := &database.RestoreOptions{
		Database:     opts.Database,
		SourceBackup: opts.BaseBackupPath,
		DropExisting: true,
		Parallel:     4,
	}

	// Restore base backup
	_, err := p.driver.Restore(ctx, restoreOpts)
	return err
}

func (p *PITRManager) applyBinaryLogs(ctx context.Context, opts *PITRRestoreOptions) error {
	// Get list of binary logs to apply
	indexPath := filepath.Join(opts.BinaryLogDir, "binlog.index")
	logs, err := p.readBinaryLogIndex(indexPath)
	if err != nil {
		return err
	}

	// Apply each binary log up to the target time
	for _, logName := range logs {
		logPath := filepath.Join(opts.BinaryLogDir, logName)

		// Check if log exists
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue
		}

		// Apply this binary log up to target time
		if err := p.applyBinaryLogToTime(ctx, logPath, opts); err != nil {
			return fmt.Errorf("failed to apply binary log %s: %w", logName, err)
		}
	}

	return nil
}

func (p *PITRManager) applyBinaryLogToTime(ctx context.Context, logPath string, opts *PITRRestoreOptions) error {
	// Build mysqlbinlog command
	cmd := exec.CommandContext(
		ctx, "mysqlbinlog",
		"--stop-datetime="+opts.TargetTime.Format("2006-01-02 15:04:05"),
		logPath,
	)

	// Get output
	binlogOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("mysqlbinlog command failed: %w", err)
	}

	// Apply to database using mysql client
	mysqlCmd := exec.CommandContext(
		ctx, "mysql",
		fmt.Sprintf("--host=%s", p.driver.config.Host),
		fmt.Sprintf("--port=%d", p.driver.config.Port),
		fmt.Sprintf("--user=%s", p.driver.config.Username),
		fmt.Sprintf("--password=%s", p.driver.config.Password),
		opts.Database,
	)

	mysqlCmd.Stdin = strings.NewReader(string(binlogOutput))

	// Capture stderr for errors
	stderr, err := mysqlCmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := mysqlCmd.Start(); err != nil {
		return err
	}

	// Read stderr
	stderrOutput, _ := io.ReadAll(stderr)

	if err := mysqlCmd.Wait(); err != nil {
		return fmt.Errorf("mysql command failed: %w, stderr: %s", err, string(stderrOutput))
	}

	return nil
}

func (p *PITRManager) readBinaryLogIndex(indexPath string) ([]string, error) {
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		log := strings.TrimSpace(scanner.Text())
		if log != "" {
			logs = append(logs, log)
		}
	}

	return logs, scanner.Err()
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

	// Get tables to verify data exists
	tables, err := p.driver.GetTables(ctx, opts.Database)
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		return pkgErrors.ErrValidationFailed("no tables found after PITR restore")
	}

	return nil
}

// PITRRestoreOptions holds options for PITR restore.
type PITRRestoreOptions struct {
	Database         string
	BaseBackupPath   string
	BinaryLogDir     string
	TargetTime       time.Time
	SkipVerification bool
}

// BinaryLogPosition represents a position in MySQL binary log.
type BinaryLogPosition struct {
	File     string
	Position int64
}

// String returns string representation of binary log position.
func (b *BinaryLogPosition) String() string {
	return fmt.Sprintf("%s:%d", b.File, b.Position)
}
