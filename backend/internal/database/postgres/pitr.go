// Package postgres provides Point-in-Time Recovery for PostgreSQL
package postgres

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// PITRManager handles Point-in-Time Recovery for PostgreSQL.
type PITRManager struct {
	driver *PostgreSQLDriver
}

// NewPITRManager creates a new PITR manager.
func NewPITRManager(driver *PostgreSQLDriver) *PITRManager {
	return &PITRManager{
		driver: driver,
	}
}

// BackupWALFiles backs up PostgreSQL WAL (Write-Ahead Log) files for PITR.
func (p *PITRManager) BackupWALFiles(ctx context.Context, outputDir string, since *time.Time) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return pkgErrors.ErrDatabaseBackup(err).WithMetadata("output_dir", outputDir)
	}

	// Get WAL directory
	walDir, err := p.getWALDirectory(ctx)
	if err != nil {
		return err
	}

	// Force a WAL switch to ensure current WAL is complete
	if err := p.switchWAL(ctx); err != nil {
		return err
	}

	// Get current WAL location
	walLocation, err := p.getCurrentWALLocation(ctx)
	if err != nil {
		return err
	}

	// Copy WAL files
	if err := p.copyWALFiles(walDir, outputDir, since); err != nil {
		return err
	}

	// Save WAL metadata
	if err := p.saveWALMetadata(walLocation, filepath.Join(outputDir, "wal.metadata")); err != nil {
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

	// Step 1: Stop PostgreSQL if running (for restore)
	// Note: This would typically be done externally or require elevated privileges

	// Step 2: Restore base backup
	if err := p.restoreBaseBackup(ctx, opts); err != nil {
		return fmt.Errorf("failed to restore base backup: %w", err)
	}

	// Step 3: Configure recovery.conf or recovery.signal for WAL replay
	if err := p.configureRecovery(opts); err != nil {
		return fmt.Errorf("failed to configure recovery: %w", err)
	}

	// Step 4: Copy WAL files to pg_wal directory
	if err := p.prepareWALFiles(opts); err != nil {
		return fmt.Errorf("failed to prepare WAL files: %w", err)
	}

	// Step 5: Verify recovery
	if !opts.SkipVerification {
		if err := p.verifyPITRRestore(ctx, opts); err != nil {
			return fmt.Errorf("PITR verification failed: %w", err)
		}
	}

	return nil
}

// GetWALLocation gets the current WAL location.
func (p *PITRManager) GetWALLocation(ctx context.Context) (*WALLocation, error) {
	return p.getCurrentWALLocation(ctx)
}

// Helper methods

func (p *PITRManager) getWALDirectory(ctx context.Context) (string, error) {
	var dataDir string
	query := "SHOW data_directory"
	err := p.driver.db.QueryRowContext(ctx, query).Scan(&dataDir)
	if err != nil {
		return "", err
	}

	walDir := filepath.Join(dataDir, "pg_wal")
	// For older PostgreSQL versions (< 10), it's pg_xlog
	if _, err := os.Stat(walDir); os.IsNotExist(err) {
		walDir = filepath.Join(dataDir, "pg_xlog")
	}

	return walDir, nil
}

func (p *PITRManager) switchWAL(ctx context.Context) error {
	_, err := p.driver.db.ExecContext(ctx, "SELECT pg_switch_wal()")
	if err != nil {
		// Try older function name for PostgreSQL < 10
		_, err = p.driver.db.ExecContext(ctx, "SELECT pg_switch_xlog()")
	}
	return err
}

func (p *PITRManager) getCurrentWALLocation(ctx context.Context) (*WALLocation, error) {
	var lsn string
	query := "SELECT pg_current_wal_lsn()"
	err := p.driver.db.QueryRowContext(ctx, query).Scan(&lsn)
	if err != nil {
		// Try older function name for PostgreSQL < 10
		query = "SELECT pg_current_xlog_location()"
		err = p.driver.db.QueryRowContext(ctx, query).Scan(&lsn)
		if err != nil {
			return nil, err
		}
	}

	return &WALLocation{
		LSN:       lsn,
		Timestamp: time.Now(),
	}, nil
}

func (p *PITRManager) copyWALFiles(walDir, outputDir string, since *time.Time) error {
	// List all WAL files
	files, err := os.ReadDir(walDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// WAL files have specific naming pattern (24 hex characters)
		if !isWALFile(file.Name()) {
			continue
		}

		// Filter by modification time if 'since' is provided
		if since != nil {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(*since) {
				continue
			}
		}

		// Copy WAL file
		sourcePath := filepath.Join(walDir, file.Name())
		destPath := filepath.Join(outputDir, file.Name())

		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy WAL file %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (p *PITRManager) saveWALMetadata(location *WALLocation, metadataPath string) error {
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "LSN: %s\nTimestamp: %s\n",
		location.LSN,
		location.Timestamp.Format(time.RFC3339))

	return err
}

func (p *PITRManager) validatePITROptions(opts *PITRRestoreOptions) error {
	if opts.TargetTime.IsZero() {
		return pkgErrors.ErrValidationFailed("target time is required for PITR")
	}

	if opts.WALDirectory == "" {
		return pkgErrors.ErrValidationFailed("WAL directory is required")
	}

	if _, err := os.Stat(opts.WALDirectory); os.IsNotExist(err) {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("WAL directory not found: %s", opts.WALDirectory))
	}

	if opts.DataDirectory == "" {
		return pkgErrors.ErrValidationFailed("data directory is required for PITR restore")
	}

	return nil
}

func (p *PITRManager) restoreBaseBackup(ctx context.Context, opts *PITRRestoreOptions) error {
	if opts.BaseBackupPath == "" {
		return nil // No base backup to restore
	}

	// Extract base backup to data directory
	cmd := exec.CommandContext(
		ctx, "pg_restore",
		"-h", p.driver.config.Host,
		"-p", fmt.Sprintf("%d", p.driver.config.Port),
		"-U", p.driver.config.Username,
		"-d", opts.Database,
		"-v",
		"--clean",
		"--if-exists",
		opts.BaseBackupPath,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.driver.config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (p *PITRManager) configureRecovery(opts *PITRRestoreOptions) error {
	// PostgreSQL 12+ uses recovery.signal and postgresql.auto.conf
	// Older versions use recovery.conf

	version, err := p.getPostgreSQLVersion()
	if err != nil {
		version = 12 // Assume newer version if we can't determine
	}

	if version >= 12 {
		return p.configureRecoveryModern(opts)
	}
	return p.configureRecoveryLegacy(opts)
}

func (p *PITRManager) configureRecoveryModern(opts *PITRRestoreOptions) error {
	// Create recovery.signal file
	signalPath := filepath.Join(opts.DataDirectory, "recovery.signal")
	if err := os.WriteFile(signalPath, []byte(""), 0o600); err != nil {
		return err
	}

	// Append to postgresql.auto.conf
	autoConfPath := filepath.Join(opts.DataDirectory, "postgresql.auto.conf")
	file, err := os.OpenFile(autoConfPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	recoveryConfig := fmt.Sprintf(
		`
# PITR Recovery Configuration
restore_command = 'cp %s/%%f %%p'
recovery_target_time = '%s'
recovery_target_action = 'promote'
`,
		opts.WALDirectory,
		opts.TargetTime.Format("2006-01-02 15:04:05"),
	)

	if _, err := file.WriteString(recoveryConfig); err != nil {
		return err
	}

	return nil
}

func (p *PITRManager) configureRecoveryLegacy(opts *PITRRestoreOptions) error {
	// Create recovery.conf file
	recoveryConfPath := filepath.Join(opts.DataDirectory, "recovery.conf")
	file, err := os.Create(recoveryConfPath)
	if err != nil {
		return err
	}
	defer file.Close()

	recoveryConfig := fmt.Sprintf(
		`restore_command = 'cp %s/%%f %%p'
recovery_target_time = '%s'
recovery_target_action = 'promote'
`,
		opts.WALDirectory,
		opts.TargetTime.Format("2006-01-02 15:04:05"),
	)

	if _, err := file.WriteString(recoveryConfig); err != nil {
		return err
	}

	return nil
}

func (p *PITRManager) prepareWALFiles(opts *PITRRestoreOptions) error {
	// WAL files are already in WALDirectory, just ensure they're accessible
	// The restore_command in recovery configuration will handle copying them

	return nil
}

func (p *PITRManager) verifyPITRRestore(ctx context.Context, opts *PITRRestoreOptions) error {
	// Check database is accessible
	if err := p.driver.Ping(ctx); err != nil {
		return pkgErrors.ErrValidationFailed("database not accessible after PITR restore")
	}

	// Verify we're not still in recovery mode
	var inRecovery bool
	err := p.driver.db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	if err != nil {
		return err
	}

	if inRecovery {
		return pkgErrors.ErrValidationFailed("database still in recovery mode")
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

	return nil
}

func (p *PITRManager) getPostgreSQLVersion() (int, error) {
	version, err := p.driver.GetVersion(context.Background())
	if err != nil {
		return 0, err
	}

	// Parse version string (e.g., "PostgreSQL 14.2...")
	var major int
	_, err = fmt.Sscanf(version, "PostgreSQL %d", &major)
	return major, err
}

// PITRRestoreOptions holds options for PITR restore.
type PITRRestoreOptions struct {
	Database         string
	BaseBackupPath   string
	WALDirectory     string
	DataDirectory    string
	TargetTime       time.Time
	SkipVerification bool
}

// WALLocation represents a location in PostgreSQL WAL.
type WALLocation struct {
	LSN       string
	Timestamp time.Time
}

// String returns string representation of WAL location.
func (w *WALLocation) String() string {
	return fmt.Sprintf("LSN: %s, Time: %s", w.LSN, w.Timestamp.Format(time.RFC3339))
}

// Helper functions

func isWALFile(filename string) bool {
	// WAL files are 24 hex characters
	if len(filename) != 24 {
		return false
	}

	for _, c := range filename {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}

	return true
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// CreateBaseBackup creates a base backup using pg_basebackup.
func (p *PITRManager) CreateBaseBackup(ctx context.Context, outputDir string, format string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	args := []string{
		"-h", p.driver.config.Host,
		"-p", fmt.Sprintf("%d", p.driver.config.Port),
		"-U", p.driver.config.Username,
		"-D", outputDir,
		"-Ft",         // tar format
		"-z",          // gzip compression
		"-P",          // progress
		"-v",          // verbose
		"-X", "fetch", // include WAL files
	}

	if format != "" {
		args = append(args, "-F", format)
	}

	cmd := exec.CommandContext(ctx, "pg_basebackup", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.driver.config.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_basebackup failed: %w, output: %s", err, string(output))
	}

	return nil
}

// ArchiveWAL archives a WAL file (typically called by PostgreSQL's archive_command).
func (p *PITRManager) ArchiveWAL(walFile, archiveDir string) error {
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return err
	}

	destPath := filepath.Join(archiveDir, filepath.Base(walFile))
	return copyFile(walFile, destPath)
}
