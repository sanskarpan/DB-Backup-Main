// Package redis provides Redis database driver implementation
package redis

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sanskarpan/db-backup/internal/database"
	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
	"github.com/sanskarpan/db-backup/pkg/utils"
)

// RedisDriver implements the database.Driver interface for Redis
type RedisDriver struct {
	client      *redis.Client
	config      *database.ConnectionConfig
	pitrManager *PITRManager
	clusterMode bool
}

func init() {
	database.RegisterDriver("redis", func() database.Driver {
		return NewRedisDriver()
	})
}

// NewRedisDriver creates a new Redis driver instance
func NewRedisDriver() *RedisDriver {
	return &RedisDriver{}
}

// Connect establishes a connection to the Redis database
func (d *RedisDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
	// Convert database string to int for Redis
	dbNum := 0
	if config.Database != "" {
		var err error
		dbNum, err = strconv.Atoi(config.Database)
		if err != nil {
			return pkgErrors.ErrDatabaseConnection(fmt.Errorf("invalid database number: %w", err))
		}
	}

	// Build Redis client options
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       dbNum,
	}

	if config.SSLMode != "" && config.SSLMode != "disable" {
		// Add TLS configuration if needed
		// opts.TLSConfig = &tls.Config{...}
	}

	// Create client
	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return pkgErrors.ErrDatabaseConnection(err)
	}

	// Check if cluster mode
	info, err := client.Info(ctx, "replication").Result()
	if err == nil {
		d.clusterMode = strings.Contains(info, "cluster_enabled:1")
	}

	d.client = client
	d.config = config
	d.pitrManager = NewPITRManager(d)
	return nil
}

// Disconnect closes the database connection
func (d *RedisDriver) Disconnect() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// Ping tests the database connection
func (d *RedisDriver) Ping(ctx context.Context) error {
	if d.client == nil {
		return pkgErrors.New(pkgErrors.ErrorTypeDatabase, "not connected to database")
	}
	return d.client.Ping(ctx).Err()
}

// Backup creates a backup of the Redis database
func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
	result := &database.BackupResult{
		ID:        utils.GenerateBackupID(),
		StartTime: time.Now(),
		Metadata:  opts.Metadata,
		Status:    database.BackupStatusInProgress,
	}

	// Choose backup method based on configuration
	backupType := opts.BackupType
	if backupType == "" {
		backupType = "rdb" // Default to RDB snapshot
	}

	var err error
	switch backupType {
	case "rdb":
		err = d.backupRDB(ctx, opts, result)
	case "aof":
		err = d.backupAOF(ctx, opts, result)
	case "both":
		// Backup both RDB and AOF
		if err = d.backupRDB(ctx, opts, result); err != nil {
			break
		}
		err = d.backupAOF(ctx, opts, result)
	default:
		err = fmt.Errorf("unsupported backup type: %s", backupType)
	}

	if err != nil {
		result.Status = database.BackupStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseBackup(err)
	}

	result.Status = database.BackupStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// backupRDB creates an RDB snapshot backup
func (d *RedisDriver) backupRDB(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) error {
	// Trigger BGSAVE to create RDB snapshot
	if err := d.client.BgSave(ctx).Err(); err != nil {
		return fmt.Errorf("failed to trigger BGSAVE: %w", err)
	}

	// Wait for BGSAVE to complete
	if err := d.waitForBGSave(ctx); err != nil {
		return fmt.Errorf("BGSAVE failed: %w", err)
	}

	// Get RDB file location
	configDir, err := d.client.ConfigGet(ctx, "dir").Result()
	if err != nil {
		return fmt.Errorf("failed to get RDB directory: %w", err)
	}

	configDBFilename, err := d.client.ConfigGet(ctx, "dbfilename").Result()
	if err != nil {
		return fmt.Errorf("failed to get RDB filename: %w", err)
	}

	var rdbDir, rdbFilename string
	if len(configDir) >= 2 {
		rdbDir = configDir[1].(string)
	}
	if len(configDBFilename) >= 2 {
		rdbFilename = configDBFilename[1].(string)
	}

	rdbPath := filepath.Join(rdbDir, rdbFilename)

	// Copy RDB file to backup location
	backupPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.rdb", result.ID))
	if err := copyFile(rdbPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy RDB file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	result.BackupPath = backupPath
	result.BackupSize = fileInfo.Size()
	result.Metadata["backup_type"] = "rdb"
	result.Metadata["rdb_version"] = d.getRDBVersion(ctx)

	return nil
}

// backupAOF backs up the Append-Only File
func (d *RedisDriver) backupAOF(ctx context.Context, opts *database.BackupOptions, result *database.BackupResult) error {
	// Get AOF file location
	configDir, err := d.client.ConfigGet(ctx, "dir").Result()
	if err != nil {
		return fmt.Errorf("failed to get AOF directory: %w", err)
	}

	configAOFFilename, err := d.client.ConfigGet(ctx, "appendfilename").Result()
	if err != nil {
		return fmt.Errorf("failed to get AOF filename: %w", err)
	}

	var aofDir, aofFilename string
	if len(configDir) >= 2 {
		aofDir = configDir[1].(string)
	}
	if len(configAOFFilename) >= 2 {
		aofFilename = configAOFFilename[1].(string)
	}

	aofPath := filepath.Join(aofDir, aofFilename)

	// Trigger AOF rewrite for consistent backup
	if err := d.client.BgRewriteAOF(ctx).Err(); err != nil {
		return fmt.Errorf("failed to trigger BGREWRITEAOF: %w", err)
	}

	// Wait for rewrite to complete
	if err := d.waitForAOFRewrite(ctx); err != nil {
		return fmt.Errorf("AOF rewrite failed: %w", err)
	}

	// Copy AOF file to backup location
	backupPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.aof", result.ID))
	if err := copyFile(aofPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy AOF file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	result.BackupPath = backupPath
	result.BackupSize = fileInfo.Size()
	result.Metadata["backup_type"] = "aof"

	return nil
}

// Restore restores the Redis database from a backup
func (d *RedisDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
	result := &database.RestoreResult{
		ID:        utils.GenerateRestoreID(),
		StartTime: time.Now(),
		Status:    database.RestoreStatusInProgress,
	}

	// Determine backup type from file extension
	backupType := "rdb"
	if strings.HasSuffix(opts.BackupPath, ".aof") {
		backupType = "aof"
	}

	var err error
	switch backupType {
	case "rdb":
		err = d.restoreRDB(ctx, opts, result)
	case "aof":
		err = d.restoreAOF(ctx, opts, result)
	default:
		err = fmt.Errorf("unsupported backup type")
	}

	if err != nil {
		result.Status = database.RestoreStatusFailed
		result.Error = err
		result.EndTime = time.Now()
		return result, pkgErrors.ErrDatabaseRestore(err)
	}

	result.Status = database.RestoreStatusCompleted
	result.EndTime = time.Now()
	return result, nil
}

// restoreRDB restores from an RDB snapshot
func (d *RedisDriver) restoreRDB(ctx context.Context, opts *database.RestoreOptions, result *database.RestoreResult) error {
	// Get Redis data directory
	configDir, err := d.client.ConfigGet(ctx, "dir").Result()
	if err != nil {
		return fmt.Errorf("failed to get RDB directory: %w", err)
	}

	configDBFilename, err := d.client.ConfigGet(ctx, "dbfilename").Result()
	if err != nil {
		return fmt.Errorf("failed to get RDB filename: %w", err)
	}

	var rdbDir, rdbFilename string
	if len(configDir) >= 2 {
		rdbDir = configDir[1].(string)
	}
	if len(configDBFilename) >= 2 {
		rdbFilename = configDBFilename[1].(string)
	}

	targetPath := filepath.Join(rdbDir, rdbFilename)

	// Flush existing data if requested
	if opts.DropExisting {
		if err := d.client.FlushAll(ctx).Err(); err != nil {
			return fmt.Errorf("failed to flush database: %w", err)
		}
	}

	// Shutdown Redis to perform restore
	// Note: In production, this should be done carefully
	if err := d.client.Shutdown(ctx).Err(); err != nil {
		// Ignore error as shutdown may close the connection
	}

	// Copy backup file to Redis data directory
	if err := copyFile(opts.BackupPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy RDB file: %w", err)
	}

	// Redis will automatically load the RDB file on restart
	// In production, you would restart Redis server here

	result.Metadata["backup_type"] = "rdb"
	return nil
}

// restoreAOF restores from an AOF file
func (d *RedisDriver) restoreAOF(ctx context.Context, opts *database.RestoreOptions, result *database.RestoreResult) error {
	// Get AOF directory
	configDir, err := d.client.ConfigGet(ctx, "dir").Result()
	if err != nil {
		return fmt.Errorf("failed to get AOF directory: %w", err)
	}

	configAOFFilename, err := d.client.ConfigGet(ctx, "appendfilename").Result()
	if err != nil {
		return fmt.Errorf("failed to get AOF filename: %w", err)
	}

	var aofDir, aofFilename string
	if len(configDir) >= 2 {
		aofDir = configDir[1].(string)
	}
	if len(configAOFFilename) >= 2 {
		aofFilename = configAOFFilename[1].(string)
	}

	targetPath := filepath.Join(aofDir, aofFilename)

	// Flush existing data if requested
	if opts.DropExisting {
		if err := d.client.FlushAll(ctx).Err(); err != nil {
			return fmt.Errorf("failed to flush database: %w", err)
		}
	}

	// Copy backup AOF file
	if err := copyFile(opts.BackupPath, targetPath); err != nil {
		return fmt.Errorf("failed to copy AOF file: %w", err)
	}

	// Restart Redis to load AOF
	// In production, restart Redis server here

	result.Metadata["backup_type"] = "aof"
	return nil
}

// GetDatabaseSize returns the total size of the database
func (d *RedisDriver) GetDatabaseSize(ctx context.Context) (int64, error) {
	// Get memory usage
	info, err := d.client.Info(ctx, "memory").Result()
	if err != nil {
		return 0, err
	}

	// Parse used_memory from info
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				var size int64
				fmt.Sscanf(parts[1], "%d", &size)
				return size, nil
			}
		}
	}

	return 0, fmt.Errorf("could not determine database size")
}

// GetVersion returns the Redis server version
func (d *RedisDriver) GetVersion(ctx context.Context) (string, error) {
	info, err := d.client.Info(ctx, "server").Result()
	if err != nil {
		return "", err
	}

	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "redis_version:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "unknown", nil
}

// GetBackupSize estimates the size of a backup
func (d *RedisDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
	return d.GetDatabaseSize(ctx)
}

// StreamBackup streams a backup to the provided writer
func (d *RedisDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
	return fmt.Errorf("streaming backup not implemented for Redis")
}

// StreamRestore restores from a reader
func (d *RedisDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
	return fmt.Errorf("streaming restore not implemented for Redis")
}

// GetDatabases returns list of databases (Redis has numbered databases 0-15)
func (d *RedisDriver) GetDatabases(ctx context.Context) ([]string, error) {
	// Redis typically has 16 databases (0-15)
	databases := make([]string, 16)
	for i := 0; i < 16; i++ {
		databases[i] = strconv.Itoa(i)
	}
	return databases, nil
}

// GetTables returns list of key patterns (Redis doesn't have tables)
func (d *RedisDriver) GetTables(ctx context.Context, database string) ([]string, error) {
	// Redis doesn't have tables, return empty list
	return []string{}, nil
}

// GetTableSize returns the size of a table (not applicable for Redis)
func (d *RedisDriver) GetTableSize(ctx context.Context, database, table string) (int64, error) {
	return 0, fmt.Errorf("Redis does not have tables")
}

// GetType returns the database type
func (d *RedisDriver) GetType() database.DatabaseType {
	return "redis"
}

// SupportsIncremental returns whether incremental backups are supported
func (d *RedisDriver) SupportsIncremental() bool {
	return true // Redis supports incremental via AOF
}

// SupportsPITR returns whether true, time-based point-in-time recovery is
// supported. It is not for Redis: there is no continuous, timestamped write log
// that can be replayed to an arbitrary instant. RDB captures discrete snapshots
// and standard AOF files carry no per-command timestamps usable for
// second-granularity replay. Advertising PITR here would be dishonest, so we
// return false and let the restore engine reject time-based recovery requests
// with a clear error. Best-effort, snapshot-based recovery is still available
// via the PITRManager (see pitr.go).
func (d *RedisDriver) SupportsPITR() bool {
	return false
}

// ValidateRestore validates that a restore can be performed
func (d *RedisDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
	if opts.BackupPath == "" {
		return fmt.Errorf("backup path is required")
	}
	if _, err := os.Stat(opts.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", opts.BackupPath)
	}
	return nil
}

// waitForBGSave waits for BGSAVE operation to complete
func (d *RedisDriver) waitForBGSave(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("BGSAVE timeout")
		case <-ticker.C:
			info, err := d.client.Info(ctx, "persistence").Result()
			if err != nil {
				return err
			}

			// Check rdb_bgsave_in_progress
			if strings.Contains(info, "rdb_bgsave_in_progress:0") {
				return nil
			}
		}
	}
}

// waitForAOFRewrite waits for AOF rewrite operation to complete
func (d *RedisDriver) waitForAOFRewrite(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("AOF rewrite timeout")
		case <-ticker.C:
			info, err := d.client.Info(ctx, "persistence").Result()
			if err != nil {
				return err
			}

			// Check aof_rewrite_in_progress
			if strings.Contains(info, "aof_rewrite_in_progress:0") {
				return nil
			}
		}
	}
}

// getRDBVersion gets the RDB format version
func (d *RedisDriver) getRDBVersion(ctx context.Context) string {
	info, err := d.client.Info(ctx, "persistence").Result()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "rdb_last_save_time:") {
			return strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	return "unknown"
}

// copyFile copies a file from src to dst
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

	_, err = io.Copy(destFile, sourceFile)
	return err
}
