# Plugin Development Guide

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Database Driver Plugin](#database-driver-plugin)
- [Storage Provider Plugin](#storage-provider-plugin)
- [Notification Provider Plugin](#notification-provider-plugin)
- [Plugin Structure](#plugin-structure)
- [Testing Your Plugin](#testing-your-plugin)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Publishing Your Plugin](#publishing-your-plugin)

## Overview

The Database Backup Utility is designed with a plugin architecture that allows you to extend its functionality by creating custom:

- **Database Drivers**: Support for additional database types
- **Storage Providers**: Integration with new storage backends
- **Notification Providers**: Custom notification channels

All plugins implement Go interfaces and are compiled into the application or loaded as separate packages.

### Plugin Types

| Plugin Type | Interface | Purpose | Examples |
|-------------|-----------|---------|----------|
| Database Driver | `database.Driver` | Backup/restore operations for specific databases | Redis, Cassandra, Neo4j |
| Storage Provider | `storage.Provider` | Upload/download to storage backends | FTP, WebDAV, MinIO |
| Notification Provider | `notification.Notifier` | Send notifications via custom channels | Discord, Teams, PagerDuty |

## Quick Start

### 1. Set Up Development Environment

```bash
# Clone the repository
git clone https://github.com/your-org/db-backup.git
cd db-backup

# Install dependencies
go mod download

# Create plugin directory
mkdir -p plugins/drivers/redis
cd plugins/drivers/redis
```

### 2. Create Plugin Scaffold

```go
package redis

import (
    "context"
    "io"
    "time"

    "github.com/sanskarpan/db-backup/internal/database"
)

// RedisDriver implements the database.Driver interface for Redis
type RedisDriver struct {
    config *database.ConnectionConfig
    client *redis.Client
}

// Ensure RedisDriver implements database.Driver
var _ database.Driver = (*RedisDriver)(nil)

// NewRedisDriver creates a new Redis driver instance
func NewRedisDriver() *RedisDriver {
    return &RedisDriver{}
}
```

### 3. Register Your Plugin

```go
// plugins/drivers/redis/register.go
package redis

import "github.com/sanskarpan/db-backup/internal/database"

func init() {
    database.RegisterDriver("redis", func() database.Driver {
        return NewRedisDriver()
    })
}
```

### 4. Import in Main Application

```go
// cmd/db-backup/main.go
import (
    _ "github.com/sanskarpan/db-backup/plugins/drivers/redis"
)
```

## Database Driver Plugin

### Interface Definition

```go
// internal/database/interface.go
type Driver interface {
    // Connection management
    Connect(ctx context.Context, config *ConnectionConfig) error
    Disconnect() error
    Ping(ctx context.Context) error

    // Backup operations
    Backup(ctx context.Context, opts *BackupOptions) (*BackupResult, error)
    StreamBackup(ctx context.Context, opts *BackupOptions, writer io.Writer) error
    GetBackupSize(ctx context.Context, opts *BackupOptions) (int64, error)

    // Restore operations
    Restore(ctx context.Context, opts *RestoreOptions) (*RestoreResult, error)
    StreamRestore(ctx context.Context, opts *RestoreOptions, reader io.Reader) error
    ValidateRestore(ctx context.Context, opts *RestoreOptions) error

    // Metadata
    GetDatabases(ctx context.Context) ([]string, error)
    GetTables(ctx context.Context, database string) ([]string, error)
    GetTableSize(ctx context.Context, database, table string) (int64, error)
    GetVersion(ctx context.Context) (string, error)

    // Utility
    GetType() DatabaseType
    SupportsIncremental() bool
    SupportsPITR() bool
}
```

### Complete Example: Redis Driver

```go
package redis

import (
    "context"
    "fmt"
    "io"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/sanskarpan/db-backup/internal/database"
)

type RedisDriver struct {
    config *database.ConnectionConfig
    client *redis.Client
}

func NewRedisDriver() *RedisDriver {
    return &RedisDriver{}
}

// Connect establishes connection to Redis
func (d *RedisDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
    d.config = config

    // Parse Redis connection string or use individual params
    opts := &redis.Options{
        Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
        Password: config.Password,
        DB:       0, // Use default DB
    }

    if config.Options != nil {
        if db, ok := config.Options["database"]; ok {
            fmt.Sscanf(db, "%d", &opts.DB)
        }
    }

    d.client = redis.NewClient(opts)

    // Test connection
    return d.Ping(ctx)
}

// Disconnect closes the Redis connection
func (d *RedisDriver) Disconnect() error {
    if d.client != nil {
        return d.client.Close()
    }
    return nil
}

// Ping verifies the connection is alive
func (d *RedisDriver) Ping(ctx context.Context) error {
    return d.client.Ping(ctx).Err()
}

// Backup performs a Redis backup
func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    startTime := time.Now()

    // Use BGSAVE for non-blocking backup
    if err := d.client.BgSave(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to start background save: %w", err)
    }

    // Wait for BGSAVE to complete
    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(1 * time.Second):
            info, err := d.client.Info(ctx, "persistence").Result()
            if err != nil {
                return nil, err
            }

            // Check if BGSAVE is complete
            if !isBGSaveInProgress(info) {
                break
            }
        }
    }

    // Get RDB file location
    rdbPath, err := d.getRDBPath(ctx)
    if err != nil {
        return nil, err
    }

    // Copy RDB file to output path
    if err := copyFile(rdbPath, opts.OutputPath); err != nil {
        return nil, err
    }

    // Get file size
    size, err := getFileSize(opts.OutputPath)
    if err != nil {
        return nil, err
    }

    endTime := time.Now()

    return &database.BackupResult{
        ID:              generateID(),
        StartTime:       startTime,
        EndTime:         endTime,
        Duration:        endTime.Sub(startTime),
        Size:            size,
        CompressedSize:  size, // RDB is already compressed
        DatabaseVersion: d.getVersion(ctx),
        Status:          database.BackupStatusSuccess,
    }, nil
}

// StreamBackup streams backup data to a writer
func (d *RedisDriver) StreamBackup(ctx context.Context, opts *database.BackupOptions, writer io.Writer) error {
    // Get all keys
    keys, err := d.client.Keys(ctx, "*").Result()
    if err != nil {
        return err
    }

    // Stream each key and value
    for _, key := range keys {
        keyType, err := d.client.Type(ctx, key).Result()
        if err != nil {
            return err
        }

        switch keyType {
        case "string":
            val, err := d.client.Get(ctx, key).Result()
            if err != nil {
                return err
            }
            fmt.Fprintf(writer, "SET %s %s\n", key, val)

        case "list":
            vals, err := d.client.LRange(ctx, key, 0, -1).Result()
            if err != nil {
                return err
            }
            for _, val := range vals {
                fmt.Fprintf(writer, "RPUSH %s %s\n", key, val)
            }

        case "hash":
            vals, err := d.client.HGetAll(ctx, key).Result()
            if err != nil {
                return err
            }
            for field, val := range vals {
                fmt.Fprintf(writer, "HSET %s %s %s\n", key, field, val)
            }

        // Handle other types: set, zset, etc.
        }
    }

    return nil
}

// GetBackupSize estimates backup size
func (d *RedisDriver) GetBackupSize(ctx context.Context, opts *database.BackupOptions) (int64, error) {
    info, err := d.client.Info(ctx, "memory").Result()
    if err != nil {
        return 0, err
    }

    // Parse used_memory from info
    var usedMemory int64
    fmt.Sscanf(info, "used_memory:%d", &usedMemory)
    return usedMemory, nil
}

// Restore restores from a backup
func (d *RedisDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
    startTime := time.Now()

    // Flush database if requested
    if opts.DropExisting {
        if err := d.client.FlushDB(ctx).Err(); err != nil {
            return nil, err
        }
    }

    // Read and execute commands from backup file
    commands, err := readBackupFile(opts.SourceBackup)
    if err != nil {
        return nil, err
    }

    var rowsRestored int64
    for _, cmd := range commands {
        if err := d.executeCommand(ctx, cmd); err != nil {
            return nil, err
        }
        rowsRestored++
    }

    endTime := time.Now()

    return &database.RestoreResult{
        StartTime:    startTime,
        EndTime:      endTime,
        Duration:     endTime.Sub(startTime),
        RowsRestored: rowsRestored,
        Status:       database.RestoreStatusSuccess,
    }, nil
}

// StreamRestore restores from a reader
func (d *RedisDriver) StreamRestore(ctx context.Context, opts *database.RestoreOptions, reader io.Reader) error {
    scanner := bufio.NewScanner(reader)
    for scanner.Scan() {
        cmd := scanner.Text()
        if err := d.executeCommand(ctx, cmd); err != nil {
            return err
        }
    }
    return scanner.Err()
}

// ValidateRestore validates restore options
func (d *RedisDriver) ValidateRestore(ctx context.Context, opts *database.RestoreOptions) error {
    // Check if backup file exists and is valid
    if _, err := os.Stat(opts.SourceBackup); err != nil {
        return fmt.Errorf("backup file not found: %w", err)
    }

    // Verify connection
    if err := d.Ping(ctx); err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    return nil
}

// GetDatabases returns list of databases (Redis has numbered DBs)
func (d *RedisDriver) GetDatabases(ctx context.Context) ([]string, error) {
    // Redis has numbered databases (0-15 by default)
    databases := make([]string, 16)
    for i := 0; i < 16; i++ {
        databases[i] = fmt.Sprintf("db%d", i)
    }
    return databases, nil
}

// GetTables returns list of key patterns (Redis doesn't have tables)
func (d *RedisDriver) GetTables(ctx context.Context, database string) ([]string, error) {
    keys, err := d.client.Keys(ctx, "*").Result()
    if err != nil {
        return nil, err
    }
    return keys, nil
}

// GetTableSize returns memory used by a key pattern
func (d *RedisDriver) GetTableSize(ctx context.Context, database, pattern string) (int64, error) {
    keys, err := d.client.Keys(ctx, pattern).Result()
    if err != nil {
        return 0, err
    }

    var totalSize int64
    for _, key := range keys {
        size, err := d.client.MemoryUsage(ctx, key).Result()
        if err != nil {
            continue
        }
        totalSize += size
    }

    return totalSize, nil
}

// GetVersion returns Redis version
func (d *RedisDriver) GetVersion(ctx context.Context) (string, error) {
    info, err := d.client.Info(ctx, "server").Result()
    if err != nil {
        return "", err
    }

    // Parse redis_version
    var version string
    fmt.Sscanf(info, "redis_version:%s", &version)
    return version, nil
}

// GetType returns driver type
func (d *RedisDriver) GetType() database.DatabaseType {
    return database.DatabaseType("redis")
}

// SupportsIncremental indicates if incremental backups are supported
func (d *RedisDriver) SupportsIncremental() bool {
    return false
}

// SupportsPITR indicates if point-in-time recovery is supported
func (d *RedisDriver) SupportsPITR() bool {
    return false
}

// Helper functions
func (d *RedisDriver) getRDBPath(ctx context.Context) (string, error) {
    config, err := d.client.ConfigGet(ctx, "dir").Result()
    if err != nil {
        return "", err
    }

    dbfilename, err := d.client.ConfigGet(ctx, "dbfilename").Result()
    if err != nil {
        return "", err
    }

    return filepath.Join(config["dir"].(string), dbfilename["dbfilename"].(string)), nil
}

func (d *RedisDriver) executeCommand(ctx context.Context, cmd string) error {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return nil
    }

    args := make([]interface{}, len(parts[1:]))
    for i, arg := range parts[1:] {
        args[i] = arg
    }

    return d.client.Do(ctx, parts[0], args...).Err()
}
```

### Registration

```go
// plugins/drivers/redis/register.go
package redis

import "github.com/sanskarpan/db-backup/internal/database"

func init() {
    database.RegisterDriver("redis", func() database.Driver {
        return NewRedisDriver()
    })
}
```

## Storage Provider Plugin

### Interface Definition

```go
// internal/storage/interface.go
type Provider interface {
    // Upload uploads a file to storage
    Upload(ctx context.Context, localPath, remotePath string) error

    // Download downloads a file from storage
    Download(ctx context.Context, remotePath, localPath string) error

    // Delete deletes a file from storage
    Delete(ctx context.Context, remotePath string) error

    // List lists files in a path
    List(ctx context.Context, prefix string) ([]FileInfo, error)

    // Exists checks if a file exists
    Exists(ctx context.Context, remotePath string) (bool, error)

    // GetURL generates a signed URL
    GetURL(ctx context.Context, remotePath string, expiry time.Duration) (string, error)

    // GetType returns provider type
    GetType() string
}
```

### Example: FTP Storage Provider

```go
package ftp

import (
    "context"
    "io"
    "os"
    "time"

    "github.com/jlaffaye/ftp"
    "github.com/sanskarpan/db-backup/internal/storage"
)

type FTPProvider struct {
    host     string
    port     int
    username string
    password string
    basePath string
    conn     *ftp.ServerConn
}

func NewFTPProvider(config *storage.Config) (*FTPProvider, error) {
    return &FTPProvider{
        host:     config.Host,
        port:     config.Port,
        username: config.Username,
        password: config.Password,
        basePath: config.BasePath,
    }, nil
}

func (p *FTPProvider) connect() error {
    if p.conn != nil {
        return nil
    }

    conn, err := ftp.Dial(
        fmt.Sprintf("%s:%d", p.host, p.port),
        ftp.DialWithTimeout(30*time.Second),
    )
    if err != nil {
        return err
    }

    if err := conn.Login(p.username, p.password); err != nil {
        return err
    }

    p.conn = conn
    return nil
}

func (p *FTPProvider) Upload(ctx context.Context, localPath, remotePath string) error {
    if err := p.connect(); err != nil {
        return err
    }

    file, err := os.Open(localPath)
    if err != nil {
        return err
    }
    defer file.Close()

    fullPath := filepath.Join(p.basePath, remotePath)

    return p.conn.Stor(fullPath, file)
}

func (p *FTPProvider) Download(ctx context.Context, remotePath, localPath string) error {
    if err := p.connect(); err != nil {
        return err
    }

    fullPath := filepath.Join(p.basePath, remotePath)

    resp, err := p.conn.Retr(fullPath)
    if err != nil {
        return err
    }
    defer resp.Close()

    file, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = io.Copy(file, resp)
    return err
}

func (p *FTPProvider) Delete(ctx context.Context, remotePath string) error {
    if err := p.connect(); err != nil {
        return err
    }

    fullPath := filepath.Join(p.basePath, remotePath)
    return p.conn.Delete(fullPath)
}

func (p *FTPProvider) List(ctx context.Context, prefix string) ([]storage.FileInfo, error) {
    if err := p.connect(); err != nil {
        return nil, err
    }

    fullPath := filepath.Join(p.basePath, prefix)

    entries, err := p.conn.List(fullPath)
    if err != nil {
        return nil, err
    }

    var files []storage.FileInfo
    for _, entry := range entries {
        files = append(files, storage.FileInfo{
            Name:         entry.Name,
            Size:         int64(entry.Size),
            LastModified: entry.Time,
            IsDir:        entry.Type == ftp.EntryTypeFolder,
        })
    }

    return files, nil
}

func (p *FTPProvider) Exists(ctx context.Context, remotePath string) (bool, error) {
    if err := p.connect(); err != nil {
        return false, err
    }

    fullPath := filepath.Join(p.basePath, remotePath)

    _, err := p.conn.FileSize(fullPath)
    if err != nil {
        if isFTPNotFoundError(err) {
            return false, nil
        }
        return false, err
    }

    return true, nil
}

func (p *FTPProvider) GetURL(ctx context.Context, remotePath string, expiry time.Duration) (string, error) {
    // FTP doesn't support signed URLs
    return "", fmt.Errorf("signed URLs not supported for FTP")
}

func (p *FTPProvider) GetType() string {
    return "ftp"
}
```

## Notification Provider Plugin

### Interface Definition

```go
// internal/notification/interface.go
type Notifier interface {
    Send(ctx context.Context, notification *Notification) error
    GetType() ProviderType
    ValidateConfig() error
}
```

### Example: Discord Notification Provider

```go
package discord

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/sanskarpan/db-backup/internal/notification"
)

type DiscordProvider struct {
    webhookURL string
    username   string
    avatarURL  string
}

func NewDiscordProvider(config *notification.Config) *DiscordProvider {
    return &DiscordProvider{
        webhookURL: config.WebhookURL,
        username:   config.Username,
        avatarURL:  config.AvatarURL,
    }
}

type discordMessage struct {
    Username  string          `json:"username,omitempty"`
    AvatarURL string          `json:"avatar_url,omitempty"`
    Content   string          `json:"content,omitempty"`
    Embeds    []discordEmbed  `json:"embeds,omitempty"`
}

type discordEmbed struct {
    Title       string               `json:"title"`
    Description string               `json:"description"`
    Color       int                  `json:"color"`
    Timestamp   string               `json:"timestamp"`
    Fields      []discordEmbedField  `json:"fields,omitempty"`
}

type discordEmbedField struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Inline bool   `json:"inline"`
}

func (p *DiscordProvider) Send(ctx context.Context, notif *notification.Notification) error {
    embed := discordEmbed{
        Title:       notif.Title,
        Description: notif.Message,
        Color:       p.getColor(notif.Level),
        Timestamp:   notif.Timestamp.Format(time.RFC3339),
    }

    // Add metadata as fields
    for key, value := range notif.Metadata {
        embed.Fields = append(embed.Fields, discordEmbedField{
            Name:   key,
            Value:  fmt.Sprintf("%v", value),
            Inline: true,
        })
    }

    message := discordMessage{
        Username:  p.username,
        AvatarURL: p.avatarURL,
        Embeds:    []discordEmbed{embed},
    }

    body, err := json.Marshal(message)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", p.webhookURL, bytes.NewReader(body))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("discord returned status %d", resp.StatusCode)
    }

    return nil
}

func (p *DiscordProvider) getColor(level notification.NotificationLevel) int {
    switch level {
    case notification.LevelSuccess:
        return 0x00FF00 // Green
    case notification.LevelInfo:
        return 0x0099FF // Blue
    case notification.LevelWarning:
        return 0xFFAA00 // Orange
    case notification.LevelError:
        return 0xFF0000 // Red
    default:
        return 0x808080 // Gray
    }
}

func (p *DiscordProvider) GetType() notification.ProviderType {
    return notification.ProviderType("discord")
}

func (p *DiscordProvider) ValidateConfig() error {
    if p.webhookURL == "" {
        return fmt.Errorf("webhook URL is required")
    }
    return nil
}
```

## Plugin Structure

### Recommended Directory Structure

```
plugins/
├── drivers/              # Database drivers
│   ├── redis/
│   │   ├── driver.go
│   │   ├── driver_test.go
│   │   ├── register.go
│   │   └── README.md
│   ├── cassandra/
│   └── neo4j/
├── storage/              # Storage providers
│   ├── ftp/
│   │   ├── provider.go
│   │   ├── provider_test.go
│   │   ├── register.go
│   │   └── README.md
│   └── webdav/
└── notifications/        # Notification providers
    ├── discord/
    │   ├── provider.go
    │   ├── provider_test.go
    │   ├── register.go
    │   └── README.md
    └── teams/
```

### Plugin Metadata (plugin.yaml)

```yaml
name: redis-driver
version: 1.0.0
description: Redis database driver for db-backup
author: Your Name
repository: https://github.com/your-org/db-backup-redis
license: MIT

plugin_type: database_driver
supported_versions:
  - ">=1.0.0"

dependencies:
  - github.com/go-redis/redis/v8: ^8.11.5

configuration:
  - name: host
    type: string
    required: true
    description: Redis host
  - name: port
    type: integer
    default: 6379
    description: Redis port
  - name: password
    type: string
    required: false
    description: Redis password
```

## Testing Your Plugin

### Unit Tests

```go
// plugins/drivers/redis/driver_test.go
package redis

import (
    "context"
    "testing"

    "github.com/alicebob/miniredis/v2"
    "github.com/sanskarpan/db-backup/internal/database"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRedisDriver_Connect(t *testing.T) {
    // Start mock Redis server
    mr, err := miniredis.Run()
    require.NoError(t, err)
    defer mr.Close()

    // Create driver
    driver := NewRedisDriver()

    // Test connection
    config := &database.ConnectionConfig{
        Host: mr.Host(),
        Port: mr.Port(),
    }

    err = driver.Connect(context.Background(), config)
    assert.NoError(t, err)

    // Test ping
    err = driver.Ping(context.Background())
    assert.NoError(t, err)

    // Cleanup
    err = driver.Disconnect()
    assert.NoError(t, err)
}

func TestRedisDriver_Backup(t *testing.T) {
    mr, err := miniredis.Run()
    require.NoError(t, err)
    defer mr.Close()

    // Set test data
    mr.Set("key1", "value1")
    mr.Set("key2", "value2")

    driver := NewRedisDriver()
    config := &database.ConnectionConfig{
        Host: mr.Host(),
        Port: mr.Port(),
    }

    err = driver.Connect(context.Background(), config)
    require.NoError(t, err)

    // Perform backup
    opts := &database.BackupOptions{
        OutputPath: t.TempDir() + "/backup.rdb",
    }

    result, err := driver.Backup(context.Background(), opts)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, database.BackupStatusSuccess, result.Status)
    assert.Greater(t, result.Size, int64(0))
}
```

### Integration Tests

```go
func TestRedisDriver_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Use real Redis instance
    driver := NewRedisDriver()
    config := &database.ConnectionConfig{
        Host: "localhost",
        Port: 6379,
    }

    err := driver.Connect(context.Background(), config)
    require.NoError(t, err)
    defer driver.Disconnect()

    // Test full backup/restore cycle
    // ...
}
```

### Running Tests

```bash
# Unit tests only
go test ./plugins/drivers/redis/...

# With integration tests
go test -tags=integration ./plugins/drivers/redis/...

# With coverage
go test -cover ./plugins/drivers/redis/...

# Verbose output
go test -v ./plugins/drivers/redis/...
```

## Best Practices

### 1. Error Handling

Always wrap errors with context:

```go
func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    if err := d.Ping(ctx); err != nil {
        return nil, fmt.Errorf("connection check failed: %w", err)
    }

    if err := d.performBackup(ctx, opts); err != nil {
        return nil, fmt.Errorf("backup operation failed: %w", err)
    }

    return result, nil
}
```

### 2. Context Handling

Respect context cancellation:

```go
func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Perform operation with context
    result, err := d.performBackup(ctx, opts)
    if err != nil {
        return nil, err
    }

    return result, nil
}
```

### 3. Resource Cleanup

Always clean up resources:

```go
func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    file, err := os.Create(opts.OutputPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Ensure cleanup even if panic occurs
    defer func() {
        if err != nil {
            os.Remove(opts.OutputPath)
        }
    }()

    // Perform backup...
}
```

### 4. Configuration Validation

Validate configuration early:

```go
func (d *RedisDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
    // Validate required fields
    if config.Host == "" {
        return fmt.Errorf("host is required")
    }

    if config.Port <= 0 || config.Port > 65535 {
        return fmt.Errorf("invalid port: %d", config.Port)
    }

    // Connect...
}
```

### 5. Logging

Use structured logging:

```go
import "github.com/sanskarpan/db-backup/internal/logger"

func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    log := logger.FromContext(ctx)

    log.Info("Starting Redis backup", map[string]interface{}{
        "host":     d.config.Host,
        "database": opts.Database,
    })

    // Perform backup...

    log.Info("Backup completed", map[string]interface{}{
        "duration": result.Duration,
        "size":     result.Size,
    })

    return result, nil
}
```

### 6. Metrics

Emit metrics for monitoring:

```go
import "github.com/sanskarpan/db-backup/internal/metrics"

func (d *RedisDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    start := time.Now()
    defer func() {
        metrics.RecordBackupDuration("redis", time.Since(start))
    }()

    result, err := d.performBackup(ctx, opts)
    if err != nil {
        metrics.IncrementBackupFailures("redis")
        return nil, err
    }

    metrics.IncrementBackupSuccesses("redis")
    metrics.RecordBackupSize("redis", result.Size)

    return result, nil
}
```

## Examples

### Complete Example: Cassandra Driver

See `plugins/drivers/cassandra/` for a complete example including:
- Full driver implementation
- Comprehensive tests
- Documentation
- Configuration examples

### Complete Example: WebDAV Storage

See `plugins/storage/webdav/` for a complete example including:
- Storage provider implementation
- Authentication handling
- File operations
- Tests

## Publishing Your Plugin

### 1. Documentation

Create comprehensive documentation:

```markdown
# Redis Driver for DB Backup

## Installation

go get github.com/your-org/db-backup-redis

## Usage

import _ "github.com/your-org/db-backup-redis"

## Configuration

## Examples

## License
```

### 2. Versioning

Use semantic versioning:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### 3. Go Module

Publish as a Go module:

```bash
go mod init github.com/your-org/db-backup-redis
go mod tidy
```

### 4. README

Include badges and examples:

```markdown
# DB Backup Redis Driver

[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/db-backup-redis)](https://goreportcard.com/report/github.com/your-org/db-backup-redis)
[![GoDoc](https://godoc.org/github.com/your-org/db-backup-redis?status.svg)](https://godoc.org/github.com/your-org/db-backup-redis)
```

## Support

For plugin development support:
- Documentation: https://docs.backup.example.com
- GitHub Discussions: https://github.com/your-org/db-backup/discussions
- Email: plugins@example.com

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing plugins to the main repository.
