// Package backup provides the backup orchestration engine
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/storage"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// Engine orchestrates backup operations
type Engine struct {
	config   *Config
	provider storage.Provider
}

// Config holds backup engine configuration
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
}

// CreateOptions holds options for creating a backup
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

	// Callbacks
	ProgressCallback func(progress Progress)
}

// Progress represents backup progress
type Progress struct {
	Stage       string
	Percentage  float64
	Message     string
	BytesTotal  int64
	BytesCopied int64
}

// NewEngine creates a new backup engine
func NewEngine(config *Config) *Engine {
	return &Engine{
		config:   config,
		provider: config.StorageProvider,
	}
}

// SetStorageProvider injects (or replaces) the storage provider used for
// uploading backups. Passing nil disables remote uploads.
func (e *Engine) SetStorageProvider(provider storage.Provider) {
	e.provider = provider
}

// remoteBackupPath returns the deterministic remote path for a backup artifact.
func remoteBackupPath(backupID, fileName string) string {
	return fmt.Sprintf("backups/%s/%s", backupID, fileName)
}

// CreateBackup creates a new backup
func (e *Engine) CreateBackup(ctx context.Context, opts *CreateOptions) (*models.BackupMetadata, error) {
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
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "initializing",
			Percentage: 0,
			Message:    "Initializing backup...",
		})
	}

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

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "connecting",
			Percentage: 10,
			Message:    "Connecting to database...",
		})
	}

	if err := driver.Connect(ctx, connConfig); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, pkgErrors.ErrDatabaseConnection(err)
	}
	defer driver.Disconnect()

	// Get database version
	version, _ := driver.GetVersion(ctx)
	metadata.DatabaseVersion = version

	// Ensure temp directory exists
	if err = os.MkdirAll(e.config.TempDirectory, 0o700); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}

	// Create temporary backup file
	backupFileName := fmt.Sprintf("%s.%s", backupID, e.getFileExtension(opts.Compression))
	backupPath := filepath.Join(e.config.TempDirectory, backupFileName)

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "backing_up",
			Percentage: 30,
			Message:    "Creating backup...",
		})
	}

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

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "validating",
			Percentage: 70,
			Message:    "Validating backup...",
		})
	}

	// Calculate checksum
	checksum, err := e.calculateChecksum(backupPath)
	if err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}
	metadata.Checksum = checksum

	// Upload the backup artifact to the configured storage provider. If no
	// provider is configured the backup stays in the local temp directory and
	// the storage location is recorded explicitly.
	if e.provider != nil {
		if opts.ProgressCallback != nil {
			opts.ProgressCallback(Progress{
				Stage:      "uploading",
				Percentage: 80,
				Message:    "Uploading backup to storage...",
			})
		}

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
			// Never report success for a backup that was not durably stored.
			metadata.Status = database.BackupStatusFailed
			return metadata, pkgErrors.ErrStorageUpload(err)
		}

		// Record provider type + remote path. BackupPath keeps pointing at the
		// local temp copy so callers can still read it before cleanup.
		metadata.StorageLocation = fmt.Sprintf("%s://%s", e.provider.GetType(), remotePath)
	} else {
		// Explicit local-temp behavior.
		metadata.StorageLocation = backupPath
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "finalizing",
			Percentage: 90,
			Message:    "Finalizing backup...",
		})
	}

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

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "completed",
			Percentage: 100,
			Message:    "Backup completed successfully",
		})
	}

	return metadata, nil
}

// ValidateBackup validates a backup file
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

// ListBackups lists all available backups
func (e *Engine) ListBackups(ctx context.Context) ([]*models.BackupMetadata, error) {
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

		backups = append(backups, metadata)
	}

	return backups, nil
}

// GetBackup retrieves backup metadata
func (e *Engine) GetBackup(ctx context.Context, backupID string) (*models.BackupMetadata, error) {
	return e.loadMetadata(backupID)
}

// DeleteBackup deletes a backup and its metadata
func (e *Engine) DeleteBackup(ctx context.Context, backupID string) error {
	// Load metadata
	metadata, err := e.loadMetadata(backupID)
	if err != nil {
		return err
	}

	// Delete backup file
	if err := os.Remove(metadata.BackupPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete metadata file
	metadataPath := filepath.Join(e.config.TempDirectory, "metadata", backupID+".json")
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
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

// saveMetadata saves backup metadata to disk
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

// loadMetadata loads backup metadata from disk
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

// getFileExtension returns the file extension based on compression type
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
