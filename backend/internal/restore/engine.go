// Package restore provides the restore orchestration engine
package restore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// ErrMalwareDetected indicates the pre-restore malware scan found a threat at or
// above the configured severity threshold and the restore was aborted to avoid
// restoring infected data.
var ErrMalwareDetected = errors.New("pre-restore malware scan detected a threat above the configured severity threshold")

// ArtifactScanner scans an on-disk backup artifact for malware/ransomware
// indicators before it is restored. It is satisfied by *ransomware.Detector.
type ArtifactScanner interface {
	// ScanFile scans the file at path and returns a threat report describing
	// what, if anything, was found.
	ScanFile(ctx context.Context, path string) (*ransomware.ThreatReport, error)
}

// Engine orchestrates restore operations.
type Engine struct {
	config   *Config
	provider storage.Provider
	scanner  ArtifactScanner
}

// Config holds restore engine configuration.
type Config struct {
	TempDirectory string
	ValidateFirst bool

	// StorageProvider is the optional remote storage backend. When set, remote
	// backups are downloaded into TempDirectory before being restored.
	StorageProvider storage.Provider

	// ArtifactScanner is the optional malware/ransomware scanner run against the
	// local backup artifact before it is restored. When nil, scanning is skipped.
	ArtifactScanner ArtifactScanner

	// MalwareScanThreshold is the minimum threat severity that aborts a restore.
	// When empty it defaults to ransomware.ThreatLevelHigh.
	MalwareScanThreshold ransomware.ThreatLevel

	// MalwareScanWarnOnly, when true, downgrades scan failures and detections to
	// warnings surfaced in the result instead of aborting the restore. It
	// defaults to false so scans fail closed.
	MalwareScanWarnOnly bool
}

// RestoreOptions holds options for restoring a backup.
//
//nolint:revive // keeps public name stable; renaming would break other packages
type RestoreOptions struct {
	// Backup to restore
	BackupID string

	// Target database
	TargetHost     string
	TargetPort     int
	TargetUsername string
	TargetPassword string
	TargetDatabase string

	// Restore options
	PointInTime   *time.Time
	Tables        []string
	ExcludeTables []string
	DropExisting  bool

	// Decryption
	Decrypt       bool
	DecryptionKey string

	// Flags
	SkipValidation bool
	Force          bool

	// SkipMalwareScan disables the pre-restore malware/ransomware scan for this
	// restore even when the engine has a scanner configured.
	SkipMalwareScan bool

	// Callbacks
	ProgressCallback func(progress Progress)
}

// Progress represents restore progress.
type Progress struct {
	Stage       string
	Percentage  float64
	Message     string
	BytesTotal  int64
	BytesCopied int64
}

// RestoreResult contains the result of a restore operation.
//
//nolint:revive // keeps public name stable; renaming would break other packages
type RestoreResult struct {
	BackupID       string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	RestoredTables []string
	RowsRestored   int64
	Status         database.RestoreStatus
	Error          error

	// ThreatReport holds the pre-restore malware scan result when a scan ran.
	// It is populated for both clean scans and detections so callers can audit
	// what was found, and is the source of the abort details on ErrMalwareDetected.
	ThreatReport *ransomware.ThreatReport
}

// NewEngine creates a new restore engine.
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:   config,
		provider: config.StorageProvider,
		scanner:  config.ArtifactScanner,
	}
}

// SetStorageProvider injects (or replaces) the storage provider used for
// downloading remote backups. Passing nil disables remote downloads.
func (e *Engine) SetStorageProvider(provider storage.Provider) {
	e.provider = provider
}

// SetArtifactScanner injects (or replaces) the malware/ransomware scanner run
// against a backup artifact before it is restored. Passing nil disables
// pre-restore scanning.
func (e *Engine) SetArtifactScanner(scanner ArtifactScanner) {
	e.scanner = scanner
}

// RestoreBackup restores a database from backup.
func (e *Engine) RestoreBackup(ctx context.Context, backupMetadata *models.BackupMetadata, opts *RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{
		BackupID:  backupMetadata.ID,
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Update progress
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "initializing",
			Percentage: 0,
			Message:    "Initializing restore...",
		})
	}

	// Ensure a local copy of the backup artifact exists, downloading it from
	// remote storage when necessary.
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "downloading",
			Percentage: 5,
			Message:    "Preparing backup artifact...",
		})
	}

	localPath, err := e.ensureLocalBackup(ctx, backupMetadata)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	// Validate backup if required
	if !opts.SkipValidation && e.config.ValidateFirst {
		if opts.ProgressCallback != nil {
			opts.ProgressCallback(Progress{
				Stage:      "validating",
				Percentage: 10,
				Message:    "Validating backup...",
			})
		}

		if err = e.validateBackup(ctx, backupMetadata, localPath); err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = err
			return result, err
		}
	}

	// Scan the local artifact for malware/ransomware before touching the target.
	// A detection at/above the configured severity (or a scan failure, unless
	// warn-only) aborts the restore so infected data is never restored.
	if e.scanEnabled(opts) {
		if opts.ProgressCallback != nil {
			opts.ProgressCallback(Progress{
				Stage:      "scanning",
				Percentage: 15,
				Message:    "Scanning backup for malware...",
			})
		}

		report, scanErr := e.scanArtifact(ctx, localPath, opts)
		if report != nil {
			result.ThreatReport = report
		}
		if scanErr != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = scanErr
			return result, scanErr
		}
	}

	// Create database driver for target
	driver, err := database.CreateDriver(backupMetadata.DatabaseType)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Connect to target database
	connConfig := &database.ConnectionConfig{
		Type:              backupMetadata.DatabaseType,
		Host:              opts.TargetHost,
		Port:              opts.TargetPort,
		Username:          opts.TargetUsername,
		Password:          opts.TargetPassword,
		Database:          opts.TargetDatabase,
		ConnectionTimeout: 30 * time.Second,
		MaxConnections:    10,
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "connecting",
			Percentage: 20,
			Message:    "Connecting to target database...",
		})
	}

	if err = driver.Connect(ctx, connConfig); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseConnection(err)
	}
	defer func() {
		// Best-effort disconnect; the restore result already reflects the outcome.
		if derr := driver.Disconnect(); derr != nil {
			_ = derr
		}
	}()

	// Prepare restore options
	restoreOpts := &database.RestoreOptions{
		Database:       opts.TargetDatabase,
		SourceBackup:   localPath,
		Tables:         opts.Tables,
		ExcludeTables:  opts.ExcludeTables,
		PointInTime:    opts.PointInTime,
		SkipValidation: opts.SkipValidation,
		DropExisting:   opts.DropExisting,
		Parallel:       4,
	}

	// Validate restore if required
	if !opts.SkipValidation {
		if err = driver.ValidateRestore(ctx, restoreOpts); err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = err
			return result, err
		}
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "restoring",
			Percentage: 40,
			Message:    "Restoring database...",
		})
	}

	// Perform restore
	restoreResult, err := driver.Restore(ctx, restoreOpts)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	// Update result from driver result
	result.RestoredTables = restoreResult.RestoredTables
	result.RowsRestored = restoreResult.RowsRestored

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "verifying",
			Percentage: 90,
			Message:    "Verifying restore...",
		})
	}

	// Verify restore (basic check)
	if err := e.verifyRestore(ctx, driver, opts); err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		return result, err
	}

	// Complete
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Status = database.RestoreStatusSuccess

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "completed",
			Percentage: 100,
			Message:    "Restore completed successfully",
		})
	}

	return result, nil
}

// RestorePointInTime performs point-in-time recovery.
func (e *Engine) RestorePointInTime(ctx context.Context, backupMetadata *models.BackupMetadata, targetTime time.Time, opts *RestoreOptions) (*RestoreResult, error) {
	// Set point-in-time
	opts.PointInTime = &targetTime

	// Check if database supports PITR
	driver, err := database.CreateDriver(backupMetadata.DatabaseType)
	if err != nil {
		return nil, err
	}

	if !driver.SupportsPITR() {
		return nil, pkgErrors.ErrValidationFailed(fmt.Sprintf("database type %s does not support point-in-time recovery", backupMetadata.DatabaseType))
	}

	// Perform restore with PITR
	return e.RestoreBackup(ctx, backupMetadata, opts)
}

// RestoreTables performs a selective restore of only the named tables from the
// backup described by backupMetadata, delegating to RestoreBackup with the
// table list applied to the restore options. The returned result reports which
// tables were actually restored via RestoreResult.RestoredTables.
//
// Table-granular restore ultimately depends on the target database driver
// honoring RestoreOptions.Tables; drivers that ignore it will restore the full
// backup regardless of the requested subset.
func (e *Engine) RestoreTables(ctx context.Context, backupMetadata *models.BackupMetadata, tables []string, opts *RestoreOptions) (*RestoreResult, error) {
	if len(tables) == 0 {
		return nil, pkgErrors.ErrValidationFailed("selective restore requires at least one table")
	}
	if opts == nil {
		opts = &RestoreOptions{}
	}
	opts.Tables = tables
	return e.RestoreBackup(ctx, backupMetadata, opts)
}

// scanEnabled reports whether a pre-restore malware scan should run for opts.
func (e *Engine) scanEnabled(opts *RestoreOptions) bool {
	return e.scanner != nil && !opts.SkipMalwareScan
}

// malwareThreshold returns the configured abort severity, defaulting to HIGH.
func (e *Engine) malwareThreshold() ransomware.ThreatLevel {
	if e.config.MalwareScanThreshold != "" {
		return e.config.MalwareScanThreshold
	}
	return ransomware.ThreatLevelHigh
}

// scanArtifact runs the configured scanner over the local backup artifact. It
// returns the threat report (when one is available) and a non-nil error when
// the restore must be aborted: either the scan itself failed (fail-closed) or a
// threat at/above the configured severity threshold was detected. When the
// engine is configured warn-only, failures and detections are surfaced via the
// report/logs but never produce an abort error.
func (e *Engine) scanArtifact(ctx context.Context, localPath string, _ *RestoreOptions) (*ransomware.ThreatReport, error) {
	report, err := e.scanner.ScanFile(ctx, localPath)
	if err != nil {
		if e.config.MalwareScanWarnOnly {
			return nil, nil //nolint:nilnil // warn-only: no report and no abort error by design
		}
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeValidation, "pre-restore malware scan failed")
	}
	if report == nil {
		return nil, nil //nolint:nilnil // no report available and no threat to abort on
	}

	if severityRank(report.ThreatLevel) >= severityRank(e.malwareThreshold()) && !e.config.MalwareScanWarnOnly {
		return report, fmt.Errorf("%w: %s %s threat in backup artifact %q: %s",
			ErrMalwareDetected, report.ThreatLevel, report.ThreatType, localPath, report.Description)
	}
	return report, nil
}

// severityRank maps a threat level to an orderable rank so severities can be
// compared against the configured abort threshold.
func severityRank(level ransomware.ThreatLevel) int {
	switch level {
	case ransomware.ThreatLevelCritical:
		return 4
	case ransomware.ThreatLevelHigh:
		return 3
	case ransomware.ThreatLevelMedium:
		return 2
	case ransomware.ThreatLevelLow:
		return 1
	case ransomware.ThreatLevelNone:
		return 0
	default:
		return 0
	}
}

// validateBackup validates the backup before restore. localPath, when
// non-empty, points at a locally available copy whose checksum is verified
// against the recorded metadata. When no local copy is available it falls back
// to confirming the artifact exists in remote storage.
func (e *Engine) validateBackup(ctx context.Context, metadata *models.BackupMetadata, localPath string) error {
	// Prefer verifying a local copy when we have one.
	if localPath != "" {
		return e.validateLocalBackup(metadata, localPath)
	}
	// No local copy: confirm the artifact exists in remote storage.
	return e.validateRemoteBackup(ctx, metadata)
}

// validateLocalBackup verifies an on-disk backup's size and checksum against
// the recorded metadata.
func (e *Engine) validateLocalBackup(metadata *models.BackupMetadata, localPath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup file not found: %s", localPath))
		}
		return err
	}

	// Size sanity check against recorded compressed/raw size.
	if metadata.CompressedSize > 0 && info.Size() != metadata.CompressedSize {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup size mismatch: expected %d bytes, got %d", metadata.CompressedSize, info.Size()))
	}

	if metadata.Checksum == "" {
		return nil
	}
	checksum, err := calculateChecksum(localPath)
	if err != nil {
		return err
	}
	if checksum != metadata.Checksum {
		return pkgErrors.ErrValidationFailed("checksum mismatch")
	}
	return nil
}

// validateRemoteBackup confirms the artifact exists in remote storage.
func (e *Engine) validateRemoteBackup(ctx context.Context, metadata *models.BackupMetadata) error {
	if e.provider == nil {
		return pkgErrors.ErrValidationFailed("backup artifact is not available locally or in storage")
	}
	remotePath, ok := parseRemotePath(metadata.StorageLocation)
	if !ok {
		return pkgErrors.ErrValidationFailed("backup artifact is not available locally or in storage")
	}
	exists, err := e.provider.Exists(ctx, remotePath)
	if err != nil {
		return err
	}
	if !exists {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup artifact not found in storage: %s", remotePath))
	}
	return nil
}

// verifyRestore verifies the restore was successful.
func (e *Engine) verifyRestore(ctx context.Context, driver database.Driver, opts *RestoreOptions) error {
	// Basic connectivity check
	if err := driver.Ping(ctx); err != nil {
		return pkgErrors.ErrValidationFailed("restore verification failed: database not accessible")
	}

	// Check if database exists
	databases, err := driver.GetDatabases(ctx)
	if err != nil {
		return pkgErrors.ErrValidationFailed("restore verification failed: cannot list databases")
	}

	found := false
	for _, db := range databases {
		if db == opts.TargetDatabase {
			found = true
			break
		}
	}

	if !found && opts.TargetDatabase != "" {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("restore verification failed: database %s not found", opts.TargetDatabase))
	}

	return nil
}

// DownloadBackup downloads a backup artifact to destinationPath and verifies
// its checksum against the recorded metadata. When destinationPath is empty a
// file inside the configured TempDirectory is used. It works for both remote
// backups (fetched via the storage provider) and local backups (copied from
// their existing path).
func (e *Engine) DownloadBackup(ctx context.Context, backupMetadata *models.BackupMetadata, destinationPath string) error {
	if destinationPath == "" {
		destinationPath = filepath.Join(e.config.TempDirectory, backupArtifactName(backupMetadata))
	}

	remotePath, isRemote := parseRemotePath(backupMetadata.StorageLocation)

	switch {
	case isRemote:
		if e.provider == nil {
			return pkgErrors.ErrValidationFailed("backup is stored remotely but no storage provider is configured")
		}
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o700); err != nil {
			return err
		}
		if err := e.provider.Download(ctx, remotePath, destinationPath); err != nil {
			return pkgErrors.ErrStorageDownload(err)
		}
	default:
		// Local backup: copy from the existing path if it differs.
		src := backupMetadata.BackupPath
		if src == "" {
			src = backupMetadata.StorageLocation
		}
		if src == "" {
			return pkgErrors.ErrValidationFailed("backup has no local path")
		}
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup file not found: %s", src))
			}
			return err
		}
		if src != destinationPath {
			if err := copyFile(src, destinationPath); err != nil {
				return err
			}
		}
	}

	// Verify the downloaded artifact against the recorded checksum.
	if backupMetadata.Checksum != "" {
		checksum, err := calculateChecksum(destinationPath)
		if err != nil {
			return err
		}
		if checksum != backupMetadata.Checksum {
			return pkgErrors.ErrValidationFailed("checksum mismatch after download")
		}
	}

	return nil
}

// ensureLocalBackup guarantees a local, checksum-verified copy of the backup
// artifact and returns its path. Remote backups are downloaded into
// TempDirectory; local backups are used in place.
func (e *Engine) ensureLocalBackup(ctx context.Context, metadata *models.BackupMetadata) (string, error) {
	remotePath, isRemote := parseRemotePath(metadata.StorageLocation)

	if isRemote {
		dest := filepath.Join(e.config.TempDirectory, backupArtifactName(metadata))
		if e.provider == nil {
			return "", pkgErrors.ErrValidationFailed("backup is stored remotely but no storage provider is configured")
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return "", err
		}
		if err := e.provider.Download(ctx, remotePath, dest); err != nil {
			return "", pkgErrors.ErrStorageDownload(err)
		}
		return dest, nil
	}

	// Local backup: use BackupPath if present, otherwise the storage location.
	localPath := metadata.BackupPath
	if localPath == "" {
		localPath = metadata.StorageLocation
	}
	return localPath, nil
}

// backupArtifactName derives a stable local filename for a backup artifact.
func backupArtifactName(metadata *models.BackupMetadata) string {
	if remotePath, ok := parseRemotePath(metadata.StorageLocation); ok {
		return filepath.Base(remotePath)
	}
	if metadata.BackupPath != "" {
		return filepath.Base(metadata.BackupPath)
	}
	return metadata.ID
}

// parseRemotePath extracts the remote object path from a "<type>://<path>"
// storage location. Returns false when the location is not a remote reference.
func parseRemotePath(storageLocation string) (string, bool) {
	const sep = "://"
	idx := strings.Index(storageLocation, sep)
	if idx < 0 {
		return "", false
	}
	return storageLocation[idx+len(sep):], true
}

// calculateChecksum computes the SHA256 checksum of a file.
func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
