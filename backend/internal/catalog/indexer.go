package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// BackupDocument represents a backup document in the catalog.
type BackupDocument struct {
	// ID is the unique backup identifier
	ID string `json:"id"`

	// DatabaseName is the name of the database
	DatabaseName string `json:"database_name"`

	// DatabaseType is the type of database (mysql, postgres, mongodb, etc.)
	DatabaseType string `json:"database_type"`

	// BackupType is the type of backup (full, incremental, differential)
	BackupType string `json:"backup_type"`

	// Status is the backup status (success, failed, in_progress)
	Status string `json:"status"`

	// SizeBytes is the backup size in bytes
	SizeBytes int64 `json:"size_bytes"`

	// CompressedSize is the compressed size in bytes
	CompressedSize int64 `json:"compressed_size,omitempty"`

	// Duration is the backup duration in seconds
	Duration float64 `json:"duration"`

	// StorageProvider is the storage provider (s3, gcs, azure, local)
	StorageProvider string `json:"storage_provider"`

	// StoragePath is the path in the storage backend
	StoragePath string `json:"storage_path"`

	// CreatedAt is the backup creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is the backup expiration timestamp
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Tags are user-defined tags for categorization
	Tags map[string]string `json:"tags,omitempty"`

	// Metadata contains additional backup metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ChecksumType is the type of checksum (sha256, md5)
	ChecksumType string `json:"checksum_type,omitempty"`

	// Checksum is the backup checksum
	Checksum string `json:"checksum,omitempty"`

	// ErrorMessage contains error details if backup failed
	ErrorMessage string `json:"error_message,omitempty"`

	// Tables contains the list of tables backed up
	Tables []string `json:"tables,omitempty"`

	// Schemas contains the list of schemas backed up
	Schemas []string `json:"schemas,omitempty"`

	// ParentBackupID is the ID of the parent backup (for incremental backups)
	ParentBackupID string `json:"parent_backup_id,omitempty"`

	// ChildBackupIDs are the IDs of child backups (for full backups)
	ChildBackupIDs []string `json:"child_backup_ids,omitempty"`

	// RestoreCount tracks how many times this backup was restored
	RestoreCount int `json:"restore_count"`

	// LastRestoreAt is the timestamp of the last restore
	LastRestoreAt *time.Time `json:"last_restore_at,omitempty"`

	// AccessCount tracks how many times this backup was accessed
	AccessCount int `json:"access_count"`

	// LastAccessAt is the timestamp of the last access
	LastAccessAt *time.Time `json:"last_access_at,omitempty"`
}

// CatalogIndexer indexes backup metadata into Elasticsearch.
//
//nolint:revive // keeps public name stable; used as catalog.CatalogIndexer by other packages
type CatalogIndexer struct {
	client    *ElasticsearchClient
	indexName string
}

// NewCatalogIndexer creates a new catalog indexer.
func NewCatalogIndexer(client *ElasticsearchClient) (*CatalogIndexer, error) {
	if client == nil {
		return nil, fmt.Errorf("elasticsearch client cannot be nil")
	}

	indexer := &CatalogIndexer{
		client:    client,
		indexName: "backups",
	}

	return indexer, nil
}

// Initialize creates the backups index with proper mapping.
func (ci *CatalogIndexer) Initialize(ctx context.Context) error {
	// Define mapping for backup documents
	mapping := map[string]interface{}{
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type": "keyword",
			},
			"database_name": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"database_type": map[string]interface{}{
				"type": "keyword",
			},
			"backup_type": map[string]interface{}{
				"type": "keyword",
			},
			"status": map[string]interface{}{
				"type": "keyword",
			},
			"size_bytes": map[string]interface{}{
				"type": "long",
			},
			"compressed_size": map[string]interface{}{
				"type": "long",
			},
			"duration": map[string]interface{}{
				"type": "double",
			},
			"storage_provider": map[string]interface{}{
				"type": "keyword",
			},
			"storage_path": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"created_at": map[string]interface{}{
				"type": "date",
			},
			"expires_at": map[string]interface{}{
				"type": "date",
			},
			"tags": map[string]interface{}{
				"type":    "object",
				"enabled": true,
			},
			"metadata": map[string]interface{}{
				"type":    "object",
				"enabled": true,
			},
			"checksum_type": map[string]interface{}{
				"type": "keyword",
			},
			"checksum": map[string]interface{}{
				"type": "keyword",
			},
			"error_message": map[string]interface{}{
				"type": "text",
			},
			"tables": map[string]interface{}{
				"type": "keyword",
			},
			"schemas": map[string]interface{}{
				"type": "keyword",
			},
			"parent_backup_id": map[string]interface{}{
				"type": "keyword",
			},
			"child_backup_ids": map[string]interface{}{
				"type": "keyword",
			},
			"restore_count": map[string]interface{}{
				"type": "integer",
			},
			"last_restore_at": map[string]interface{}{
				"type": "date",
			},
			"access_count": map[string]interface{}{
				"type": "integer",
			},
			"last_access_at": map[string]interface{}{
				"type": "date",
			},
		},
	}

	// Check if index already exists
	exists, err := ci.client.IndexExists(ctx, ci.indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return nil // Index already exists
	}

	// Create the index
	if err := ci.client.CreateIndex(ctx, ci.indexName, mapping); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// IndexBackup indexes a single backup document.
func (ci *CatalogIndexer) IndexBackup(ctx context.Context, backup *BackupDocument) error {
	if backup == nil {
		return fmt.Errorf("backup document cannot be nil")
	}

	if backup.ID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}

	return ci.client.IndexDocument(ctx, ci.indexName, backup.ID, backup)
}

// IndexBackups indexes multiple backup documents in bulk.
func (ci *CatalogIndexer) IndexBackups(ctx context.Context, backups []*BackupDocument) error {
	if len(backups) == 0 {
		return nil
	}

	bulkDocs := make([]BulkDocument, len(backups))
	for i, backup := range backups {
		if backup.ID == "" {
			return fmt.Errorf("backup at index %d has empty ID", i)
		}
		bulkDocs[i] = BulkDocument{
			ID:       backup.ID,
			Document: backup,
		}
	}

	return ci.client.BulkIndex(ctx, ci.indexName, bulkDocs)
}

// GetBackup retrieves a backup document by ID.
func (ci *CatalogIndexer) GetBackup(ctx context.Context, backupID string) (*BackupDocument, error) {
	if backupID == "" {
		return nil, fmt.Errorf("backup ID cannot be empty")
	}

	doc, err := ci.client.GetDocument(ctx, ci.indexName, backupID)
	if err != nil {
		return nil, err
	}

	// Convert map to BackupDocument
	backup := &BackupDocument{}
	if err := convertMapToStruct(doc, backup); err != nil {
		return nil, fmt.Errorf("failed to convert document: %w", err)
	}

	return backup, nil
}

// DeleteBackup deletes a backup document by ID.
func (ci *CatalogIndexer) DeleteBackup(ctx context.Context, backupID string) error {
	if backupID == "" {
		return fmt.Errorf("backup ID cannot be empty")
	}

	return ci.client.DeleteDocument(ctx, ci.indexName, backupID)
}

// UpdateBackupStatus updates the status of a backup.
func (ci *CatalogIndexer) UpdateBackupStatus(ctx context.Context, backupID, status string) error {
	backup, err := ci.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	backup.Status = status
	return ci.IndexBackup(ctx, backup)
}

// IncrementRestoreCount increments the restore count for a backup.
func (ci *CatalogIndexer) IncrementRestoreCount(ctx context.Context, backupID string) error {
	backup, err := ci.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	backup.RestoreCount++
	now := time.Now()
	backup.LastRestoreAt = &now

	return ci.IndexBackup(ctx, backup)
}

// IncrementAccessCount increments the access count for a backup.
func (ci *CatalogIndexer) IncrementAccessCount(ctx context.Context, backupID string) error {
	backup, err := ci.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	backup.AccessCount++
	now := time.Now()
	backup.LastAccessAt = &now

	return ci.IndexBackup(ctx, backup)
}

// AddChildBackup adds a child backup ID to a parent backup.
func (ci *CatalogIndexer) AddChildBackup(ctx context.Context, parentID, childID string) error {
	backup, err := ci.GetBackup(ctx, parentID)
	if err != nil {
		return err
	}

	// Check if child already exists
	for _, id := range backup.ChildBackupIDs {
		if id == childID {
			return nil // Already exists
		}
	}

	backup.ChildBackupIDs = append(backup.ChildBackupIDs, childID)
	return ci.IndexBackup(ctx, backup)
}

// CountBackups returns the total number of backups in the catalog.
func (ci *CatalogIndexer) CountBackups(ctx context.Context) (int64, error) {
	query := map[string]interface{}{
		"match_all": map[string]interface{}{},
	}

	return ci.client.Count(ctx, ci.indexName, query)
}

// CountBackupsByStatus returns the number of backups with a given status.
func (ci *CatalogIndexer) CountBackupsByStatus(ctx context.Context, status string) (int64, error) {
	query := map[string]interface{}{
		"term": map[string]interface{}{
			"status": status,
		},
	}

	return ci.client.Count(ctx, ci.indexName, query)
}

// CountBackupsByDatabase returns the number of backups for a given database.
func (ci *CatalogIndexer) CountBackupsByDatabase(ctx context.Context, databaseName string) (int64, error) {
	query := map[string]interface{}{
		"match": map[string]interface{}{
			"database_name.keyword": databaseName,
		},
	}

	return ci.client.Count(ctx, ci.indexName, query)
}

// GetExpiredBackups returns backups that have expired.
func (ci *CatalogIndexer) GetExpiredBackups(ctx context.Context, limit int) ([]*BackupDocument, error) {
	if limit <= 0 {
		limit = 100
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"expires_at": map[string]interface{}{
					"lt": time.Now().Format(time.RFC3339),
				},
			},
		},
		"size": limit,
		"sort": []map[string]interface{}{
			{
				"expires_at": map[string]interface{}{
					"order": "asc",
				},
			},
		},
	}

	response, err := ci.client.Search(ctx, ci.indexName, query)
	if err != nil {
		return nil, err
	}

	backups := make([]*BackupDocument, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		backup := &BackupDocument{}
		if err := convertMapToStruct(hit.Source, backup); err != nil {
			return nil, fmt.Errorf("failed to convert document: %w", err)
		}
		backups[i] = backup
	}

	return backups, nil
}

// convertMapToStruct converts a map to a struct using JSON marshaling.
func convertMapToStruct(m map[string]interface{}, v interface{}) error {
	// This is a simple conversion using JSON encoding/decoding
	// In production, you might want to use a more efficient method
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	return nil
}
