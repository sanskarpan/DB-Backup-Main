package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupConverter converts backups between universal format and provider-specific formats
type BackupConverter struct {
	chunkSize int64
}

// NewBackupConverter creates a new backup converter
func NewBackupConverter(chunkSize int64) *BackupConverter {
	if chunkSize <= 0 {
		chunkSize = 10 * 1024 * 1024 // 10MB default
	}

	return &BackupConverter{
		chunkSize: chunkSize,
	}
}

// ToUniversalFormat converts a regular backup file to universal format
func (c *BackupConverter) ToUniversalFormat(ctx context.Context, inputFile, outputDir, databaseType, databaseName string) error {
	// Open input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Create universal backup writer
	writer := NewUniversalBackupWriter(outputDir, databaseType, databaseName, c.chunkSize)

	// Write data
	if err := writer.WriteData(file); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Detect compression from file extension
	if ext := filepath.Ext(inputFile); ext != "" {
		switch ext {
		case ".gz", ".gzip":
			writer.SetCompression("gzip")
		case ".zst", ".zstd":
			writer.SetCompression("zstd")
		case ".lz4":
			writer.SetCompression("lz4")
		}
	}

	// Finalize backup
	if err := writer.Finalize(); err != nil {
		return fmt.Errorf("failed to finalize backup: %w", err)
	}

	return nil
}

// FromUniversalFormat converts a universal format backup to a single file
func (c *BackupConverter) FromUniversalFormat(ctx context.Context, inputDir, outputFile string) error {
	// Create universal backup reader
	reader, err := NewUniversalBackupReader(inputDir)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	// Verify integrity
	if err := reader.VerifyIntegrity(); err != nil {
		return fmt.Errorf("backup integrity check failed: %w", err)
	}

	// Read all data
	data, err := reader.ReadData()
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// MigrateBetweenProviders migrates a backup from one provider to another
func (c *BackupConverter) MigrateBetweenProviders(ctx context.Context, sourceProvider, targetProvider, backupID string) error {
	// This would integrate with the actual storage providers
	// For now, it's a placeholder showing the structure

	// 1. Download from source provider to temp directory (universal format)
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("backup-migration-%s", backupID))
	defer os.RemoveAll(tempDir)

	// 2. Upload to target provider from temp directory
	// Implementation would depend on storage provider interfaces

	return nil
}

// ValidateUniversalBackup validates a universal format backup
func (c *BackupConverter) ValidateUniversalBackup(inputDir string) (*ValidationResult, error) {
	reader, err := NewUniversalBackupReader(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Check manifest
	manifest := reader.GetManifest()
	if manifest == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "missing manifest")
		return result, nil
	}

	// Check metadata
	metadata := reader.GetMetadata()
	if metadata == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "missing metadata")
		return result, nil
	}

	// Verify integrity
	if err := reader.VerifyIntegrity(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("integrity check failed: %v", err))
	}

	// Check chunks exist
	for _, chunkRef := range manifest.Chunks {
		chunkPath := filepath.Join(inputDir, chunkRef.Path)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing chunk: %s", chunkRef.ChunkID))
		}
	}

	result.ChunkCount = len(manifest.Chunks)
	result.TotalSize = manifest.TotalSize
	result.DatabaseType = manifest.DatabaseType
	result.BackupID = manifest.BackupID

	return result, nil
}

// GetBackupInfo retrieves backup information without reading data
func (c *BackupConverter) GetBackupInfo(inputDir string) (*BackupInfo, error) {
	reader, err := NewUniversalBackupReader(inputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	manifest := reader.GetManifest()
	metadata := reader.GetMetadata()

	return &BackupInfo{
		BackupID:     manifest.BackupID,
		DatabaseType: manifest.DatabaseType,
		DatabaseName: manifest.DatabaseName,
		BackupType:   metadata.BackupType,
		CreatedAt:    manifest.CreatedAt,
		TotalSize:    manifest.TotalSize,
		ChunkCount:   len(manifest.Chunks),
		Compression:  manifest.Compression,
		Encrypted:    manifest.Encryption != nil,
		Format:       manifest.Format,
		Version:      manifest.Version,
		Tags:         metadata.Tags,
	}, nil
}

// ExportManifest exports just the manifest for cataloging
func (c *BackupConverter) ExportManifest(inputDir, outputFile string) error {
	reader, err := NewUniversalBackupReader(inputDir)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	manifest := reader.GetManifest()

	// Write manifest to output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	return nil
}

// StreamBackupData streams backup data without loading everything into memory
func (c *BackupConverter) StreamBackupData(inputDir string, writer io.Writer) error {
	reader, err := NewUniversalBackupReader(inputDir)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	manifest := reader.GetManifest()

	for _, chunkRef := range manifest.Chunks {
		chunkPath := filepath.Join(inputDir, chunkRef.Path)
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk %s: %w", chunkRef.ChunkID, err)
		}

		if _, err := writer.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk data: %w", err)
		}
	}

	return nil
}

// ValidationResult represents the result of backup validation
type ValidationResult struct {
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors,omitempty"`
	ChunkCount   int      `json:"chunk_count"`
	TotalSize    int64    `json:"total_size"`
	DatabaseType string   `json:"database_type"`
	BackupID     string   `json:"backup_id"`
}

// BackupInfo contains summary information about a backup
type BackupInfo struct {
	BackupID     string            `json:"backup_id"`
	DatabaseType string            `json:"database_type"`
	DatabaseName string            `json:"database_name"`
	BackupType   string            `json:"backup_type"`
	CreatedAt    time.Time         `json:"created_at"`
	TotalSize    int64             `json:"total_size"`
	ChunkCount   int               `json:"chunk_count"`
	Compression  string            `json:"compression,omitempty"`
	Encrypted    bool              `json:"encrypted"`
	Format       string            `json:"format"`
	Version      string            `json:"version"`
	Tags         map[string]string `json:"tags,omitempty"`
}
