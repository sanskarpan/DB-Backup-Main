// Package backup provides the backup orchestration engine
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/notification"
	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// Notifier is the minimal notification sink used by the engine to report backup
// outcomes. *notification.Router satisfies it. A nil Notifier disables
// notifications.
type Notifier interface {
	Send(ctx context.Context, notif *notification.Notification) error
}

// Engine orchestrates backup operations.
type Engine struct {
	config   *Config
	provider storage.Provider
	notifier Notifier
}

// Config holds backup engine configuration.
type Config struct {
	TempDirectory      string
	ParallelOperations int
	DefaultCompression string
	EnableEncryption   bool
	EncryptionKey      string

	// StorageProvider is the optional remote storage backend. When set, backups
	// are uploaded here after being dumped locally. When nil, backups remain in
	// the local temp directory.
	StorageProvider storage.Provider

	// Notifier is the optional notification sink. When set, a completion
	// notification (success or failure) is dispatched after each backup. When
	// nil, no notifications are sent.
	Notifier Notifier
}

// CreateOptions holds options for creating a backup.
type CreateOptions struct {
	// Database connection
	DatabaseType database.DatabaseType
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string

	// Backup options
	Databases     []string
	AllDatabases  bool
	Tables        []string
	ExcludeTables []string

	// Storage and compression
	Compression      database.CompressionType
	CompressionLevel int
	Encrypt          bool
	EncryptionKey    string

	// Metadata
	Name string
	Tags map[string]string

	// Immutability (WORM / object-lock) options. When Immutable is true the
	// uploaded artifact is locked so it cannot be deleted before its retention
	// expires. This requires a storage provider that implements
	// storage.ImmutableProvider; otherwise the backup fails (fail-closed).
	Immutable bool
	// RetentionUntil is the point in time until which an immutable backup is
	// protected from deletion. Required (in the future) when Immutable is true
	// unless LegalHold is set.
	RetentionUntil time.Time
	// LockMode is the object-lock retention mode (storage.LockModeGovernance or
	// storage.LockModeCompliance). Defaults to GOVERNANCE when empty.
	LockMode string
	// LegalHold, when true, applies an indefinite legal hold to an immutable
	// backup in addition to (or instead of) a retention period.
	LegalHold bool

	// Callbacks
	ProgressCallback func(progress Progress)
}

// Progress represents backup progress.
type Progress struct {
	Stage       string
	Percentage  float64
	Message     string
	BytesTotal  int64
	BytesCopied int64
}

// NewEngine creates a new backup engine.
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:   config,
		provider: config.StorageProvider,
		notifier: config.Notifier,
	}
}

// SetStorageProvider injects (or replaces) the storage provider used for
// uploading backups. Passing nil disables remote uploads.
func (e *Engine) SetStorageProvider(provider storage.Provider) {
	e.provider = provider
}

// SetNotifier injects (or replaces) the notification sink used to report backup
// outcomes. Passing nil disables notifications.
func (e *Engine) SetNotifier(notifier Notifier) {
	e.notifier = notifier
}

// remoteBackupPath returns the deterministic remote path for a backup artifact.
func remoteBackupPath(backupID, fileName string) string {
	return fmt.Sprintf("backups/%s/%s", backupID, fileName)
}

// reportProgress invokes the optional progress callback with a simple stage
// update, doing nothing when no callback is configured.
func reportProgress(opts *CreateOptions, stage string, pct float64, msg string) {
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{Stage: stage, Percentage: pct, Message: msg})
	}
}

// CreateBackup creates a new backup and dispatches a completion notification
// (success or failure) through the configured notifier, if any. This is the
// single integration point for backup notifications: callers such as the
// scheduler delegate here, so they must not send their own notifications. A
// notification failure never affects the backup result.
func (e *Engine) CreateBackup(ctx context.Context, opts *CreateOptions) (*models.BackupMetadata, error) {
	metadata, err := e.createBackup(ctx, opts)
	e.notifyResult(ctx, metadata, err)
	return metadata, err
}

// notifyResult builds and sends a backup completion notification. It is
// nil-safe (no notifier or no metadata is a no-op) and never returns an error:
// a notification failure is logged and swallowed so it cannot fail the backup.
func (e *Engine) notifyResult(ctx context.Context, metadata *models.BackupMetadata, backupErr error) {
	if e.notifier == nil || metadata == nil {
		return
	}

	notif := buildBackupNotification(metadata, backupErr)
	if err := e.notifier.Send(ctx, notif); err != nil {
		log.Printf("backup: failed to send %s notification for backup %s: %v",
			notif.Level, metadata.ID, err)
	}
}

// buildBackupNotification summarizes a backup outcome as a *notification.
// Notification. The "backup" and event ("success"/"failure") tags let each
// channel honor its NotifyOn configuration.
func buildBackupNotification(metadata *models.BackupMetadata, backupErr error) *notification.Notification {
	level := notification.LevelSuccess
	event := "success"
	title := fmt.Sprintf("Backup succeeded: %s", metadata.Database)
	message := fmt.Sprintf("Backup %s of database %q completed successfully.", metadata.ID, metadata.Database)

	if backupErr != nil {
		level = notification.LevelError
		event = "failure"
		title = fmt.Sprintf("Backup failed: %s", metadata.Database)
		message = fmt.Sprintf("Backup %s of database %q failed: %v", metadata.ID, metadata.Database, backupErr)
	}

	meta := map[string]interface{}{
		"backup_id": metadata.ID,
		"database":  metadata.Database,
		"status":    string(metadata.Status),
		"size":      metadata.Size,
		"duration":  metadata.Duration.String(),
	}
	if backupErr != nil {
		meta["error"] = backupErr.Error()
	}

	return &notification.Notification{
		Title:    title,
		Message:  message,
		Level:    level,
		Metadata: meta,
		Tags:     []string{"backup", event},
	}
}

// createBackup performs the actual backup work.
func (e *Engine) createBackup(ctx context.Context, opts *CreateOptions) (*models.BackupMetadata, error) {
	// Generate backup ID
	backupID := utils.GenerateBackupID()

	// Create backup metadata
	metadata := &models.BackupMetadata{
		ID:           backupID,
		Name:         opts.Name,
		DatabaseType: opts.DatabaseType,
		Database:     opts.Database,
		Databases:    opts.Databases,
		StartTime:    time.Now(),
		Status:       database.BackupStatusInProgress,
		Tags:         opts.Tags,
		Compression:  opts.Compression,
		Encrypted:    opts.Encrypt,
	}

	if metadata.Name == "" {
		metadata.Name = fmt.Sprintf("%s-%s-%s",
			opts.DatabaseType,
			opts.Database,
			time.Now().Format("20060102-150405"))
	}

	// Update progress
	reportProgress(opts, "initializing", 0, "Initializing backup...")

	// Create database driver
	driver, err := database.CreateDriver(opts.DatabaseType)
	if err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, pkgErrors.ErrDatabaseBackup(err)
	}

	// Connect to database
	connConfig := &database.ConnectionConfig{
		Type:              opts.DatabaseType,
		Host:              opts.Host,
		Port:              opts.Port,
		Username:          opts.Username,
		Password:          opts.Password,
		Database:          opts.Database,
		ConnectionTimeout: 30 * time.Second,
		MaxConnections:    10,
	}

	reportProgress(opts, "connecting", 10, "Connecting to database...")

	if err = driver.Connect(ctx, connConfig); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, pkgErrors.ErrDatabaseConnection(err)
	}
	defer func() {
		if derr := driver.Disconnect(); derr != nil {
			log.Printf("failed to disconnect database driver: %v", derr)
		}
	}()

	// Get database version
	version, verr := driver.GetVersion(ctx)
	if verr != nil {
		log.Printf("failed to get database version: %v", verr)
	}
	metadata.DatabaseVersion = version

	// Ensure temp directory exists
	if err = os.MkdirAll(e.config.TempDirectory, 0o700); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}

	// Create temporary backup file
	backupFileName := fmt.Sprintf("%s.%s", backupID, e.getFileExtension(opts.Compression))
	backupPath := filepath.Join(e.config.TempDirectory, backupFileName)

	reportProgress(opts, "backing_up", 30, "Creating backup...")

	// Create backup options for driver
	backupOpts := &database.BackupOptions{
		Database:         opts.Database,
		Databases:        opts.Databases,
		AllDatabases:     opts.AllDatabases,
		Tables:           opts.Tables,
		ExcludeTables:    opts.ExcludeTables,
		OutputPath:       backupPath,
		Compression:      opts.Compression,
		ConsistentBackup: true,
		Metadata:         make(map[string]interface{}),
	}

	// Perform backup
	result, err := driver.Backup(ctx, backupOpts)
	if err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, pkgErrors.ErrDatabaseBackup(err)
	}

	// Update metadata from result
	metadata.Size = result.Size
	metadata.CompressedSize = result.CompressedSize
	metadata.Tables = result.Tables
	metadata.BackupPath = backupPath

	reportProgress(opts, "validating", 70, "Validating backup...")

	// Calculate checksum
	checksum, err := e.calculateChecksum(backupPath)
	if err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}
	metadata.Checksum = checksum

	// Upload the backup artifact to the configured storage provider (or record
	// the explicit local-temp location when none is configured). Never report
	// success for a backup that was not durably stored.
	if err = e.storeArtifact(ctx, metadata, backupID, backupFileName, backupPath, checksum, opts); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}

	reportProgress(opts, "finalizing", 90, "Finalizing backup...")

	// Complete
	metadata.EndTime = time.Now()
	metadata.Duration = metadata.EndTime.Sub(metadata.StartTime)
	metadata.Status = database.BackupStatusSuccess

	// Save metadata locally (reflects the final, successful state).
	if err := e.saveMetadata(metadata); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}

	// Upload the metadata JSON alongside the backup so a remote store is
	// self-describing. A failure here fails the backup as well.
	if e.provider != nil {
		metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			metadata.Status = database.BackupStatusFailed
			return metadata, err
		}
		metaRemotePath := remoteBackupPath(backupID, "metadata.json")
		if err := e.provider.UploadStream(ctx, bytes.NewReader(metadataJSON), metaRemotePath, &storage.UploadOptions{ContentType: "application/json"}); err != nil {
			metadata.Status = database.BackupStatusFailed
			return metadata, pkgErrors.ErrStorageUpload(err)
		}
	}

	reportProgress(opts, "completed", 100, "Backup completed successfully")

	return metadata, nil
}

// storeArtifact uploads the backup artifact to the configured storage provider
// and records its location on the metadata. When no provider is configured the
// backup stays in the local temp directory and the local path is recorded.
func (e *Engine) storeArtifact(ctx context.Context, metadata *models.BackupMetadata, backupID, backupFileName, backupPath, checksum string, opts *CreateOptions) error {
	if e.provider == nil {
		// Fail closed: never silently keep a non-immutable local copy when
		// immutability was required.
		if opts.Immutable {
			return pkgErrors.New(pkgErrors.ErrorTypeStorage,
				"immutable backup requested but no storage provider is configured")
		}
		metadata.StorageLocation = backupPath
		return nil
	}

	reportProgress(opts, "uploading", 80, "Uploading backup to storage...")

	remotePath := remoteBackupPath(backupID, backupFileName)
	uploadOpts := &storage.UploadOptions{
		ContentType: "application/octet-stream",
		Checksum:    checksum,
		Metadata: map[string]string{
			"backup_id": backupID,
			"database":  opts.Database,
		},
	}
	if opts.ProgressCallback != nil {
		uploadOpts.ProgressCallback = func(uploaded, total int64) {
			opts.ProgressCallback(Progress{
				Stage:       "uploading",
				Percentage:  80,
				Message:     "Uploading backup to storage...",
				BytesTotal:  total,
				BytesCopied: uploaded,
			})
		}
	}

	if err := e.provider.Upload(ctx, backupPath, remotePath, uploadOpts); err != nil {
		return pkgErrors.ErrStorageUpload(err)
	}

	// Apply immutability (object-lock) protection to the just-uploaded object.
	// This fails closed: if immutability was requested but cannot be applied,
	// the whole backup fails rather than being stored unprotected.
	if err := e.applyImmutability(ctx, metadata, remotePath, opts); err != nil {
		return err
	}

	// Record provider type + remote path. BackupPath keeps pointing at the
	// local temp copy so callers can still read it before cleanup.
	metadata.StorageLocation = fmt.Sprintf("%s://%s", e.provider.GetType(), remotePath)
	return nil
}

// applyImmutability applies object-lock retention and/or legal hold to the
// uploaded object at remotePath when opts.Immutable is set, and records the
// applied protection on the metadata. It fails closed: if the configured
// provider does not implement storage.ImmutableProvider, or the retention
// request is invalid, the backup fails so an unprotected artifact is never
// reported as an immutable success. It is a no-op when immutability is not
// requested.
func (e *Engine) applyImmutability(ctx context.Context, metadata *models.BackupMetadata, remotePath string, opts *CreateOptions) error {
	if !opts.Immutable {
		return nil
	}

	imm, ok := e.provider.(storage.ImmutableProvider)
	if !ok {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("immutable backup requested but storage provider %q does not support object lock",
				e.provider.GetType()))
	}

	mode := opts.LockMode
	if mode == "" {
		mode = storage.LockModeGovernance
	}
	if !storage.ValidLockMode(mode) {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			fmt.Sprintf("invalid object-lock mode: %q (want %s or %s)",
				mode, storage.LockModeGovernance, storage.LockModeCompliance))
	}

	hasRetention := !opts.RetentionUntil.IsZero()
	if !hasRetention && !opts.LegalHold {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			"immutable backup requested but neither a retention period nor a legal hold was specified")
	}

	if hasRetention {
		if !opts.RetentionUntil.After(time.Now()) {
			return pkgErrors.New(pkgErrors.ErrorTypeValidation,
				"immutable backup retention must be in the future")
		}
		if err := imm.SetRetention(ctx, remotePath, opts.RetentionUntil, mode); err != nil {
			return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to apply object-lock retention")
		}
		until := opts.RetentionUntil
		metadata.ImmutableUntil = &until
		metadata.LockMode = mode
	}

	if opts.LegalHold {
		if err := imm.SetLegalHold(ctx, remotePath, true); err != nil {
			return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to apply legal hold")
		}
		metadata.LegalHold = true
	}

	metadata.Immutable = true
	return nil
}

// GetBackupImmutability reports the current object-lock protection for a
// backup. When the configured provider implements storage.ImmutableProvider it
// queries the provider for authoritative status; otherwise it falls back to the
// protection recorded on the metadata. A missing retention is not an error: it
// yields a zero until time.
func (e *Engine) GetBackupImmutability(ctx context.Context, metadata *models.BackupMetadata) (until time.Time, mode string, legalHold bool, err error) {
	imm, ok := e.provider.(storage.ImmutableProvider)
	remotePath, remote := parseRemotePath(metadata.StorageLocation)
	if e.provider == nil || !ok || !remote {
		// Fall back to recorded metadata.
		if metadata.ImmutableUntil != nil {
			until = *metadata.ImmutableUntil
		}
		return until, metadata.LockMode, metadata.LegalHold, nil
	}

	until, mode, err = imm.GetRetention(ctx, remotePath)
	if err != nil {
		if !errors.Is(err, storage.ErrNoRetention) {
			return time.Time{}, "", false, err
		}
		until, mode = time.Time{}, ""
	}

	legalHold, err = imm.GetLegalHold(ctx, remotePath)
	if err != nil {
		return time.Time{}, "", false, err
	}
	return until, mode, legalHold, nil
}

// ApplyLegalHold turns a legal hold on or off for a stored backup and persists
// the change on the metadata. It requires a provider that implements
// storage.ImmutableProvider and a remotely stored artifact; otherwise it
// returns an error. Object-lock allows extending protection, so enabling a
// legal hold is always permitted.
func (e *Engine) ApplyLegalHold(ctx context.Context, metadata *models.BackupMetadata, on bool) error {
	imm, ok := e.provider.(storage.ImmutableProvider)
	if e.provider == nil || !ok {
		return pkgErrors.New(pkgErrors.ErrorTypeStorage,
			"legal hold requires a storage provider that supports object lock")
	}

	remotePath, remote := parseRemotePath(metadata.StorageLocation)
	if !remote {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation,
			fmt.Sprintf("backup is not stored remotely: %s", metadata.StorageLocation))
	}

	if err := imm.SetLegalHold(ctx, remotePath, on); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to update legal hold")
	}

	metadata.LegalHold = on
	if on {
		metadata.Immutable = true
	}
	return e.saveMetadata(metadata)
}

// ValidateBackup validates a backup file.
func (e *Engine) ValidateBackup(ctx context.Context, backupID string) error {
	// Load metadata
	metadata, err := e.loadMetadata(backupID)
	if err != nil {
		return err
	}

	// If a local copy exists, verify its checksum directly.
	if _, statErr := os.Stat(metadata.BackupPath); statErr == nil {
		checksum, err := e.calculateChecksum(metadata.BackupPath)
		if err != nil {
			return err
		}
		if checksum != metadata.Checksum {
			return pkgErrors.ErrValidationFailed("checksum mismatch")
		}
		return nil
	}

	// No local copy: confirm the artifact exists in remote storage.
	if e.provider != nil {
		remotePath, ok := parseRemotePath(metadata.StorageLocation)
		if ok {
			exists, err := e.provider.Exists(ctx, remotePath)
			if err != nil {
				return err
			}
			if !exists {
				return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup artifact not found in storage: %s", remotePath))
			}
			return nil
		}
	}

	return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup file not found: %s", metadata.BackupPath))
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

// ListBackups lists all live (non-soft-deleted) backups. Backups sitting in the
// recycle bin (DeletedAt != nil) are excluded; use ListDeletedBackups for those.
func (e *Engine) ListBackups(ctx context.Context) ([]*models.BackupMetadata, error) {
	return e.listBackups(func(m *models.BackupMetadata) bool { return m.DeletedAt == nil })
}

// ListDeletedBackups lists only soft-deleted backups: the recycle bin.
func (e *Engine) ListDeletedBackups(ctx context.Context) ([]*models.BackupMetadata, error) {
	return e.listBackups(func(m *models.BackupMetadata) bool { return m.DeletedAt != nil })
}

// listBackups reads every metadata file and returns those matching the include
// predicate. Invalid metadata files are skipped.
func (e *Engine) listBackups(include func(*models.BackupMetadata) bool) ([]*models.BackupMetadata, error) {
	metadataDir := filepath.Join(e.config.TempDirectory, "metadata")

	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		return []*models.BackupMetadata{}, nil
	}

	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return nil, err
	}

	var backups []*models.BackupMetadata
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		backupID := entry.Name()[:len(entry.Name())-5] // Remove .json
		metadata, err := e.loadMetadata(backupID)
		if err != nil {
			continue // Skip invalid metadata files
		}

		if include(metadata) {
			backups = append(backups, metadata)
		}
	}

	return backups, nil
}

// GetBackup retrieves backup metadata.
func (e *Engine) GetBackup(ctx context.Context, backupID string) (*models.BackupMetadata, error) {
	return e.loadMetadata(backupID)
}

// DeleteBackup soft-deletes a backup: it stamps DeletedAt and persists the
// metadata, moving the backup into the recycle bin. The artifact and metadata
// file are retained so the deletion can be undone via RestoreDeletedBackup or
// made permanent via PurgeBackup. Deleting an already soft-deleted backup is a
// no-op that succeeds (idempotent).
func (e *Engine) DeleteBackup(ctx context.Context, backupID string) error {
	metadata, err := e.loadMetadata(backupID)
	if err != nil {
		return err
	}

	if metadata.DeletedAt != nil {
		return nil // Already in the recycle bin.
	}

	now := time.Now()
	metadata.DeletedAt = &now
	return e.saveMetadata(metadata)
}

// RestoreDeletedBackup undeletes a soft-deleted backup, clearing DeletedAt and
// moving it out of the recycle bin. It errors if the backup does not exist or is
// not currently soft-deleted.
func (e *Engine) RestoreDeletedBackup(ctx context.Context, backupID string) error {
	metadata, err := e.loadMetadata(backupID)
	if err != nil {
		return err
	}

	if metadata.DeletedAt == nil {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup is not in the recycle bin: %s", backupID))
	}

	metadata.DeletedAt = nil
	return e.saveMetadata(metadata)
}

// PurgeBackup permanently removes a backup's artifact and metadata. It is only
// allowed on backups already in the recycle bin (soft-deleted); purging a live
// backup is rejected so a permanent deletion always requires an explicit
// soft-delete first. After a successful purge the backup is unrecoverable.
func (e *Engine) PurgeBackup(ctx context.Context, backupID string) error {
	metadata, err := e.loadMetadata(backupID)
	if err != nil {
		return err
	}

	if metadata.DeletedAt == nil {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup must be soft-deleted before it can be purged: %s", backupID))
	}

	return e.purge(ctx, metadata)
}

// purge removes the artifact (from the remote provider when the storage location
// is remote, plus any local temp copy) and the metadata file. It is the shared
// hard-delete used by PurgeBackup and PurgeExpired.
func (e *Engine) purge(ctx context.Context, metadata *models.BackupMetadata) error {
	// Remove the remote artifact (and its self-describing metadata.json) when the
	// location is a "<type>://<path>" reference.
	if e.provider != nil {
		if remotePath, ok := parseRemotePath(metadata.StorageLocation); ok {
			if err := e.provider.Delete(ctx, remotePath); err != nil {
				return fmt.Errorf("delete remote artifact %s: %w", remotePath, err)
			}
			metaRemotePath := remoteBackupPath(metadata.ID, "metadata.json")
			if err := e.provider.Delete(ctx, metaRemotePath); err != nil {
				return fmt.Errorf("delete remote metadata %s: %w", metaRemotePath, err)
			}
		}
	}

	// Remove any local temp copy of the artifact.
	if metadata.BackupPath != "" {
		if err := os.Remove(metadata.BackupPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Remove the metadata file last so a failure above leaves a recoverable
	// recycle-bin entry rather than an orphaned artifact.
	metadataPath := filepath.Join(e.config.TempDirectory, "metadata", metadata.ID+".json")
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// PurgeExpired permanently purges every soft-deleted backup whose DeletedAt is
// older than the given retention window, returning the number purged. A backup
// that fails to purge aborts the sweep and the error is returned along with the
// count purged so far.
func (e *Engine) PurgeExpired(ctx context.Context, retention time.Duration) (int, error) {
	deleted, err := e.ListDeletedBackups(ctx)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-retention)
	purged := 0
	for _, metadata := range deleted {
		if metadata.DeletedAt == nil || metadata.DeletedAt.After(cutoff) {
			continue
		}
		if err := e.purge(ctx, metadata); err != nil {
			return purged, err
		}
		purged++
	}

	return purged, nil
}

// calculateChecksum calculates SHA256 checksum of a file.
func (e *Engine) calculateChecksum(filePath string) (string, error) {
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

// saveMetadata saves backup metadata to disk.
func (e *Engine) saveMetadata(metadata *models.BackupMetadata) error {
	metadataDir := filepath.Join(e.config.TempDirectory, "metadata")
	if err := os.MkdirAll(metadataDir, 0o700); err != nil {
		return err
	}

	metadataPath := filepath.Join(metadataDir, metadata.ID+".json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0o600)
}

// loadMetadata loads backup metadata from disk.
func (e *Engine) loadMetadata(backupID string) (*models.BackupMetadata, error) {
	metadataPath := filepath.Join(e.config.TempDirectory, "metadata", backupID+".json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, pkgErrors.ErrValidationFailed(fmt.Sprintf("backup not found: %s", backupID))
		}
		return nil, err
	}

	var metadata models.BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// getFileExtension returns the file extension based on compression type.
func (e *Engine) getFileExtension(compression database.CompressionType) string {
	switch compression {
	case database.CompressionGzip:
		return "sql.gz"
	case database.CompressionZstd:
		return "sql.zst"
	case database.CompressionLZ4:
		return "sql.lz4"
	default:
		return "sql"
	}
}
