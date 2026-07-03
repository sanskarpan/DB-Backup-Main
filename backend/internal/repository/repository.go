// Package repository provides storage and retrieval of backup metadata
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/models"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/validation"
)

// Repository manages backup metadata storage.
type Repository interface {
	// Save saves backup metadata
	Save(ctx context.Context, metadata *models.BackupMetadata) error

	// Get retrieves backup metadata by ID
	Get(ctx context.Context, id string) (*models.BackupMetadata, error)

	// List lists all backups with optional filtering
	List(ctx context.Context, filter *ListFilter) ([]*models.BackupMetadata, error)

	// Delete removes backup metadata
	Delete(ctx context.Context, id string) error

	// Update updates backup metadata
	Update(ctx context.Context, metadata *models.BackupMetadata) error
}

// ListFilter defines filtering options for listing backups.
type ListFilter struct {
	Database     string
	DatabaseType string
	StorageType  string
	Tags         map[string]string
	From         *time.Time
	To           *time.Time
	Limit        int
	SortBy       string // date, size, name
	SortOrder    string // asc, desc
}

// FileRepository implements Repository using filesystem storage.
type FileRepository struct {
	baseDir string
}

// NewFileRepository creates a new file-based repository.
func NewFileRepository(baseDir string) (*FileRepository, error) {
	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create repository directory")
	}

	return &FileRepository{
		baseDir: baseDir,
	}, nil
}

// Save saves backup metadata to file.
func (r *FileRepository) Save(ctx context.Context, metadata *models.BackupMetadata) error {
	if metadata.ID == "" {
		return pkgErrors.New(pkgErrors.ErrorTypeValidation, "backup ID is required")
	}

	// Create metadata file path
	metadataPath, err := r.getMetadataPath(metadata.ID)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeValidation, "invalid backup ID")
	}

	// Ensure directory exists
	if err = os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to create metadata directory")
	}

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeInternal, "failed to marshal metadata")
	}

	// Write to file with secure permissions (owner read/write only)
	if err := os.WriteFile(metadataPath, data, 0o600); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to write metadata file")
	}

	return nil
}

// Get retrieves backup metadata by ID.
func (r *FileRepository) Get(ctx context.Context, id string) (*models.BackupMetadata, error) {
	metadataPath, err := r.getMetadataPath(id)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeValidation, "invalid backup ID")
	}

	// Check if file exists
	if _, err = os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("backup not found: %s", id))
	}

	// Read file
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to read metadata file")
	}

	// Unmarshal JSON
	var metadata models.BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeInternal, "failed to unmarshal metadata")
	}

	return &metadata, nil
}

// List lists all backups with optional filtering.
func (r *FileRepository) List(ctx context.Context, filter *ListFilter) ([]*models.BackupMetadata, error) {
	var backups []*models.BackupMetadata

	// Walk through metadata directory
	err := filepath.Walk(r.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Read metadata file
		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // skip files that can't be read
		}

		// Unmarshal metadata
		var metadata models.BackupMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil //nolint:nilerr // skip invalid metadata files
		}

		// Apply filters
		if matchesFilter(&metadata, filter) {
			backups = append(backups, &metadata)
		}
		return nil
	})
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to list backups")
	}

	// Sort results
	if filter != nil && filter.SortBy != "" {
		r.sortBackups(backups, filter.SortBy, filter.SortOrder)
	} else {
		// Default: sort by date descending
		r.sortBackups(backups, "date", "desc")
	}

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(backups) > filter.Limit {
		backups = backups[:filter.Limit]
	}

	return backups, nil
}

// Delete removes backup metadata.
func (r *FileRepository) Delete(ctx context.Context, id string) error {
	metadataPath, err := r.getMetadataPath(id)
	if err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeValidation, "invalid backup ID")
	}

	// Check if file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return pkgErrors.New(pkgErrors.ErrorTypeNotFound, fmt.Sprintf("backup not found: %s", id))
	}

	// Remove file
	if err := os.Remove(metadataPath); err != nil {
		return pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage, "failed to delete metadata file")
	}

	return nil
}

// Update updates backup metadata.
func (r *FileRepository) Update(ctx context.Context, metadata *models.BackupMetadata) error {
	// Verify backup exists
	if _, err := r.Get(ctx, metadata.ID); err != nil {
		return err
	}

	// Save updated metadata
	return r.Save(ctx, metadata)
}

// getMetadataPath returns the file path for backup metadata.
func (r *FileRepository) getMetadataPath(id string) (string, error) {
	// Validate backup ID to prevent path traversal
	if err := validation.ValidateBackupID(id); err != nil {
		return "", fmt.Errorf("invalid backup ID: %w", err)
	}

	path := filepath.Join(r.baseDir, fmt.Sprintf("%s.json", id))

	// Additional safety check: ensure the path is within baseDir
	sanitized, err := validation.SanitizePath(path, r.baseDir)
	if err != nil {
		return "", fmt.Errorf("path traversal detected: %w", err)
	}

	return sanitized, nil
}

// sortBackups sorts backups based on criteria.
func (r *FileRepository) sortBackups(backups []*models.BackupMetadata, sortBy, order string) {
	sort.Slice(backups, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "size":
			less = backups[i].Size < backups[j].Size
		case "name":
			less = backups[i].Name < backups[j].Name
		default: // date
			less = backups[i].StartTime.Before(backups[j].StartTime)
		}

		if order == "asc" {
			return less
		}
		return !less
	})
}

// matchesFilter reports whether the metadata satisfies the given filter.
func matchesFilter(metadata *models.BackupMetadata, filter *ListFilter) bool {
	if filter == nil {
		return true
	}
	if filter.Database != "" && metadata.Database != filter.Database {
		return false
	}
	if filter.DatabaseType != "" && string(metadata.DatabaseType) != filter.DatabaseType {
		return false
	}
	if filter.From != nil && metadata.StartTime.Before(*filter.From) {
		return false
	}
	if filter.To != nil && metadata.StartTime.After(*filter.To) {
		return false
	}
	if len(filter.Tags) > 0 && !matchTags(metadata.Tags, filter.Tags) {
		return false
	}
	return true
}

// matchTags checks if metadata tags match filter tags.
func matchTags(metadataTags, filterTags map[string]string) bool {
	for key, value := range filterTags {
		if metadataTags[key] != value {
			return false
		}
	}
	return true
}
