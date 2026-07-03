// Package elasticsearch provides Elasticsearch/OpenSearch database driver implementation
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// ElasticsearchDriver implements the database.Driver interface for Elasticsearch/OpenSearch.
type ElasticsearchDriver struct {
	client        *elasticsearch.Client
	config        *database.ConnectionConfig
	repositoryMgr *RepositoryManager
	isOpenSearch  bool
}

func init() {
	database.RegisterDriver("elasticsearch", func() database.Driver {
		return NewElasticsearchDriver()
	})
	database.RegisterDriver("opensearch", func() database.Driver {
		driver := NewElasticsearchDriver()
		driver.isOpenSearch = true
		return driver
	})
}

// NewElasticsearchDriver creates a new Elasticsearch driver instance.
func NewElasticsearchDriver() *ElasticsearchDriver {
	return &ElasticsearchDriver{}
}

// Connect establishes a connection to the Elasticsearch cluster.
func (d *ElasticsearchDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%d", config.Host, config.Port),
		},
		Username: config.Username,
		Password: config.Password,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Test connection
	res, err := client.Info()
	if err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return pkgErrors.ErrDatabaseConnection(fmt.Errorf("elasticsearch error: %s", res.Status()))
	}

	d.client = client
	d.config = config
	d.repositoryMgr = NewRepositoryManager(client)
	return nil
}

// Disconnect closes the database connection.
func (d *ElasticsearchDriver) Disconnect() error {
	// Elasticsearch client doesn't need explicit disconnect
	return nil
}

// Ping tests the database connection.
func (d *ElasticsearchDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}

	res, err := d.client.Ping()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping failed: %s", res.Status())
	}

	return nil
}

// Backup creates a backup of the Elasticsearch cluster.
func (d *ElasticsearchDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Create or verify repository
	repoName := "db_backup_repo"
	if err := d.repositoryMgr.CreateRepository(ctx, repoName, opts.OutputDir); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Create snapshot
	snapshotName := fmt.Sprintf("snapshot_%s", result.ID)
	if err := d.createSnapshot(ctx, repoName, snapshotName, opts); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Wait for snapshot to complete
	if err := d.waitForSnapshot(ctx, repoName, snapshotName); err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	// Get snapshot info
	info, err := d.getSnapshotInfo(ctx, repoName, snapshotName)
	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	result.BackupPath = fmt.Sprintf("%s/%s", repoName, snapshotName)
	result.BackupSize = info.Size
	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	result.Metadata["snapshot_name"] = snapshotName
	result.Metadata["repository"] = repoName
	result.Metadata["indices"] = info.Indices

	return result, nil
}

// createSnapshot creates a new snapshot.
func (d *ElasticsearchDriver) createSnapshot(ctx context.Context, repo, snapshot string, opts *database.BackupOptions) error {
	body := map[string]interface{}{
		"indices":              "*",
		"ignore_unavailable":   true,
		"include_global_state": true,
	}

	// Filter by indices if specified
	if opts.IncludeSchemas != nil && len(opts.IncludeSchemas) > 0 {
		body["indices"] = strings.Join(opts.IncludeSchemas, ",")
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot body: %w", err)
	}

	res, err := d.client.Snapshot.Create(
		repo,
		snapshot,
		d.client.Snapshot.Create.WithBody(bytes.NewReader(bodyJSON)),
		d.client.Snapshot.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("snapshot creation failed: %s, body: %s", res.Status(), string(body))
	}

	return nil
}

// waitForSnapshot waits for snapshot to complete.
func (d *ElasticsearchDriver) waitForSnapshot(ctx context.Context, repo, snapshot string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("snapshot timeout")
		case <-ticker.C:
			res, err := d.client.Snapshot.Status(
				d.client.Snapshot.Status.WithRepository(repo),
				d.client.Snapshot.Status.WithSnapshot(snapshot),
			)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			var status map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
				return err
			}

			snapshots, ok := status["snapshots"].([]interface{})
			if !ok || len(snapshots) == 0 {
				return nil // Snapshot completed
			}

			snap := snapshots[0].(map[string]interface{})
			state := snap["state"].(string)

			if state == "SUCCESS" {
				return nil
			} else if state == "FAILED" || state == "PARTIAL" {
				return fmt.Errorf("snapshot failed with state: %s", state)
			}
		}
	}
}

// getSnapshotInfo retrieves snapshot information.
func (d *ElasticsearchDriver) getSnapshotInfo(ctx context.Context, repo, snapshot string) (*SnapshotInfo, error) {
	res, err := d.client.Snapshot.Get(repo, []string{snapshot})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response struct {
		Snapshots []struct {
			Snapshot string   `json:"snapshot"`
			Indices  []string `json:"indices"`
			State    string   `json:"state"`
			Stats    struct {
				Total struct {
					Size int64 `json:"size_in_bytes"`
				} `json:"total"`
			} `json:"stats"`
		} `json:"snapshots"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Snapshots) == 0 {
		return nil, fmt.Errorf("snapshot not found")
	}

	snap := response.Snapshots[0]
	return &SnapshotInfo{
		Name:    snap.Snapshot,
		Indices: snap.Indices,
		Size:    snap.Stats.Total.Size,
		State:   snap.State,
	}, nil
}

// Restore restores the Elasticsearch cluster from a backup.
func (d *ElasticsearchDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Parse backup path (format: repository/snapshot)
	parts := strings.Split(opts.BackupPath, "/")
	if len(parts) != 2 {
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("invalid backup path format")
		return result, result.Error
	}

	repo := parts[0]
	snapshot := parts[1]

	// Close indices if requested
	if opts.DropExisting {
		// Close all indices
		res, err := d.client.Indices.Close([]string{"*"})
		if err != nil {
			result.Status = database.RestoreStatusFailed
			result.Error = err
			return result, pkgErrors.ErrDatabaseRestore(err)
		}
		res.Body.Close()
	}

	// Restore snapshot
	body := map[string]interface{}{
		"indices":              "*",
		"include_global_state": true,
	}

	bodyJSON, _ := json.Marshal(body)

	res, err := d.client.Snapshot.Restore(
		repo,
		snapshot,
		d.client.Snapshot.Restore.WithBody(bytes.NewReader(bodyJSON)),
	)
	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		result.Status = database.RestoreStatusFailed
		result.Error = fmt.Errorf("restore failed: %s, body: %s", res.Status(), string(body))
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(result.Error)
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// GetDatabaseSize returns the total size of the Elasticsearch cluster.
func (d *ElasticsearchDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	res, err := d.client.Cat.Indices(
		d.client.Cat.Indices.WithFormat("json"),
		d.client.Cat.Indices.WithBytes("b"),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var indices []struct {
		StoreSize string `json:"store.size"`
	}

	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return 0, err
	}

	var totalSize int64
	for _, idx := range indices {
		var size int64
		fmt.Sscanf(idx.StoreSize, "%d", &size)
		totalSize += size
	}

	return totalSize, nil
}

// GetVersion returns the Elasticsearch/OpenSearch version.
func (d *ElasticsearchDriver) GetVersion(ctx context.Context) (string, error) {
	res, err := d.client.Info()
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var info struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}

	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return "", err
	}

	if d.isOpenSearch {
		return fmt.Sprintf("OpenSearch %s", info.Version.Number), nil
	}
	return fmt.Sprintf("Elasticsearch %s", info.Version.Number), nil
}

// SnapshotInfo contains snapshot information.
type SnapshotInfo struct {
	Name    string
	Indices []string
	Size    int64
	State   string
}

// GetBackupSize estimates the size of a backup.
func (d *ElasticsearchDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	return d.GetDatabaseSize(ctx)
}

// StreamBackup streams a backup to the provided writer.
func (d *ElasticsearchDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for Elasticsearch")
}

// StreamRestore restores from a reader.
func (d *ElasticsearchDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for Elasticsearch")
}

// ValidateRestore validates that a restore can be performed.
func (d *ElasticsearchDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" {
		return fmt.Errorf("backup path is required")
	}
	return nil
}

// GetDatabases returns list of indices (Elasticsearch doesn't have databases).
func (d *ElasticsearchDriver) GetDatabases(ctx context.Context) ([]string, error) {
	res, err := d.client.Cat.Indices(
		d.client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var indices []struct {
		Index string `json:"index"`
	}

	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, err
	}

	var indexNames []string
	for _, idx := range indices {
		// Skip system indices
		if !strings.HasPrefix(idx.Index, ".") {
			indexNames = append(indexNames, idx.Index)
		}
	}

	return indexNames, nil
}

// GetTables returns list of indices in an index pattern (not applicable for Elasticsearch).
func (d *ElasticsearchDriver) GetTables(ctx context.Context, indexPattern string) ([]string, error) {
	// For Elasticsearch, tables are essentially indices
	return d.GetDatabases(ctx)
}

// GetTableSize returns the size of an index.
func (d *ElasticsearchDriver) GetTableSize(ctx context.Context, database, index string) (int64, error) {
	res, err := d.client.Cat.Indices(
		d.client.Cat.Indices.WithIndex(index),
		d.client.Cat.Indices.WithFormat("json"),
		d.client.Cat.Indices.WithBytes("b"),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var indices []struct {
		StoreSize string `json:"store.size"`
	}

	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return 0, err
	}

	if len(indices) == 0 {
		return 0, nil
	}

	var size int64
	fmt.Sscanf(indices[0].StoreSize, "%d", &size)
	return size, nil
}

// GetType returns the database type.
func (d *ElasticsearchDriver) GetType() database.DatabaseType {
	if d.isOpenSearch {
		return "opensearch"
	}
	return "elasticsearch"
}

// SupportsIncremental returns whether incremental backups are supported.
func (d *ElasticsearchDriver) SupportsIncremental() bool {
	return false
}

// SupportsPITR returns whether point-in-time recovery is supported.
func (d *ElasticsearchDriver) SupportsPITR() bool {
	return false
}
