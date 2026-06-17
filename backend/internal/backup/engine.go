// Package backup provides the backup orchestration engine
package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// Engine orchestrates backup operations
type Engine struct {
	config *Config
}

// Config holds backup engine configuration
type Config struct {
	TempDirectory      string
	ParallelOperations int
	DefaultCompression string
	EnableEncryption   bool
	EncryptionKey      string
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
		config: config,
	}
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
	if err := os.MkdirAll(e.config.TempDirectory, 0700); err != nil {
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

	if opts.ProgressCallback != nil {
		opts.ProgressCallback(Progress{
			Stage:      "finalizing",
			Percentage: 90,
			Message:    "Finalizing backup...",
		})
	}

	// Save metadata
	if err := e.saveMetadata(metadata); err != nil {
		metadata.Status = database.BackupStatusFailed
		return metadata, err
	}

	// Complete
	metadata.EndTime = time.Now()
	metadata.Duration = metadata.EndTime.Sub(metadata.StartTime)
	metadata.Status = database.BackupStatusSuccess

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

	// Check if backup file exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		return pkgErrors.ErrValidationFailed(fmt.Sprintf("backup file not found: %s", metadata.BackupPath))
	}

	// Verify checksum
	checksum, err := e.calculateChecksum(metadata.BackupPath)
	if err != nil {
		return err
	}

	if checksum != metadata.Checksum {
		return pkgErrors.ErrValidationFailed("checksum mismatch")
	}

	return nil
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
	if err := os.MkdirAll(metadataDir, 0700); err != nil {
		return err
	}

	metadataPath := filepath.Join(metadataDir, metadata.ID+".json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0600)
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
