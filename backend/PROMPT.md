# Database Backup Utility - Complete Implementation Prompt

## Executive Summary

Build a production-ready, enterprise-grade database backup and restoration utility in Go with both CLI and web frontend interfaces. The system must support multiple database types (MySQL, PostgreSQL, MongoDB, SQLite), cloud storage integration (AWS S3, GCS, Azure), compression, encryption, scheduling, and comprehensive monitoring.

---

## Project Specifications

### Technical Stack

#### Backend
- **Language**: Go 1.21+
- **CLI Framework**: Cobra + Viper
- **Web Framework**: Gin or Echo
- **Database**: PostgreSQL (for metadata storage)
- **Caching**: Redis
- **Message Queue**: RabbitMQ or NATS (for async operations)
- **Logging**: Zerolog or Zap
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry + Jaeger

#### Frontend
- **Framework**: React 18+ with TypeScript
- **Build Tool**: Vite
- **UI Library**: Tailwind CSS + shadcn/ui
- **State Management**: Zustand or Redux Toolkit
- **API Client**: Axios
- **Charts**: Recharts or Chart.js
- **Real-time**: WebSocket

#### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Kubernetes
- **IaC**: Terraform
- **CI/CD**: GitHub Actions
- **Monitoring**: Prometheus + Grafana
- **Log Aggregation**: ELK Stack or Loki

---

## Detailed Implementation Requirements

### PHASE 1: Project Foundation

#### 1.1 Directory Structure
```
db-backup/
├── cmd/
│   ├── cli/                    # CLI entry point
│   │   └── main.go
│   ├── server/                 # Web server entry point
│   │   └── main.go
│   └── worker/                 # Background worker entry point
│       └── main.go
├── internal/
│   ├── backup/                 # Backup operations
│   │   ├── engine.go
│   │   ├── scheduler.go
│   │   ├── validator.go
│   │   └── metadata.go
│   ├── restore/                # Restore operations
│   │   ├── engine.go
│   │   ├── validator.go
│   │   └── pitr.go
│   ├── database/               # Database drivers
│   │   ├── interface.go
│   │   ├── mysql/
│   │   │   ├── driver.go
│   │   │   ├── backup.go
│   │   │   └── restore.go
│   │   ├── postgres/
│   │   │   ├── driver.go
│   │   │   ├── backup.go
│   │   │   └── restore.go
│   │   ├── mongodb/
│   │   │   ├── driver.go
│   │   │   ├── backup.go
│   │   │   └── restore.go
│   │   └── sqlite/
│   │       ├── driver.go
│   │       ├── backup.go
│   │       └── restore.go
│   ├── storage/                # Cloud storage
│   │   ├── interface.go
│   │   ├── s3/
│   │   │   └── client.go
│   │   ├── gcs/
│   │   │   └── client.go
│   │   ├── azure/
│   │   │   └── client.go
│   │   └── local/
│   │       └── client.go
│   ├── compression/            # Compression handlers
│   │   ├── interface.go
│   │   ├── gzip.go
│   │   ├── zstd.go
│   │   └── lz4.go
│   ├── encryption/             # Encryption handlers
│   │   ├── interface.go
│   │   ├── aes.go
│   │   └── keymanager.go
│   ├── notification/           # Notification system
│   │   ├── interface.go
│   │   ├── slack.go
│   │   ├── email.go
│   │   └── webhook.go
│   ├── api/                    # REST API
│   │   ├── handlers/
│   │   │   ├── backup.go
│   │   │   ├── database.go
│   │   │   ├── schedule.go
│   │   │   ├── user.go
│   │   │   └── metrics.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── logging.go
│   │   │   ├── cors.go
│   │   │   └── ratelimit.go
│   │   └── routes.go
│   ├── auth/                   # Authentication
│   │   ├── jwt.go
│   │   ├── oauth.go
│   │   └── rbac.go
│   ├── config/                 # Configuration
│   │   ├── config.go
│   │   └── validator.go
│   ├── logger/                 # Logging
│   │   ├── logger.go
│   │   └── hooks.go
│   ├── metrics/                # Metrics
│   │   ├── prometheus.go
│   │   └── custom.go
│   ├── models/                 # Data models
│   │   ├── backup.go
│   │   ├── database.go
│   │   ├── schedule.go
│   │   └── user.go
│   └── repository/             # Data access layer
│       ├── backup.go
│       ├── database.go
│       ├── schedule.go
│       └── user.go
├── pkg/                        # Public packages
│   ├── utils/
│   │   ├── crypto.go
│   │   ├── hash.go
│   │   └── validator.go
│   └── errors/
│       └── errors.go
├── web/                        # Frontend application
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── services/
│   │   ├── store/
│   │   ├── types/
│   │   └── utils/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── migrations/                 # Database migrations
│   ├── 001_init.up.sql
│   └── 001_init.down.sql
├── deployments/                # Deployment configs
│   ├── docker/
│   │   ├── Dockerfile.cli
│   │   ├── Dockerfile.server
│   │   └── docker-compose.yml
│   ├── kubernetes/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   └── terraform/
│       └── main.tf
├── scripts/                    # Utility scripts
│   ├── build.sh
│   ├── test.sh
│   └── deploy.sh
├── tests/                      # Tests
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
├── docs/                       # Documentation
│   ├── api/
│   ├── guides/
│   └── architecture/
├── .github/
│   └── workflows/
│       └── ci-cd.yml
├── Makefile
├── go.mod
├── go.sum
├── README.md
└── CHECKLIST.md
```

#### 1.2 Configuration File Structure
```yaml
# config.yaml
server:
  host: 0.0.0.0
  port: 8080
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  
database:
  metadata:
    type: postgres
    host: localhost
    port: 5432
    name: backup_metadata
    user: backup_user
    password: ""
    max_connections: 25
    
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0

logging:
  level: info              # debug, info, warn, error
  format: json             # json, text
  output: stdout           # stdout, file
  file:
    path: /var/log/backup.log
    max_size: 100          # MB
    max_backups: 5
    max_age: 30            # days
    compress: true

backup:
  default_compression: zstd
  compression_level: 3
  encryption:
    enabled: false
    algorithm: aes-256-gcm
    key_file: ""
  retention:
    daily: 7
    weekly: 4
    monthly: 12
  temp_directory: /tmp/backups
  parallel_operations: 4

storage:
  default_provider: s3
  providers:
    s3:
      enabled: true
      region: us-east-1
      bucket: my-backups
      access_key: ""
      secret_key: ""
      endpoint: ""         # For S3-compatible services
      use_path_style: false
    gcs:
      enabled: false
      project: ""
      bucket: ""
      credentials_file: ""
    azure:
      enabled: false
      account_name: ""
      account_key: ""
      container: ""
    local:
      enabled: true
      path: /backup/storage

notifications:
  slack:
    enabled: false
    webhook_url: ""
    channel: "#backups"
    notify_on:
      - success
      - failure
      - warning
  email:
    enabled: false
    smtp_host: ""
    smtp_port: 587
    username: ""
    password: ""
    from: ""
    to: []
  webhook:
    enabled: false
    url: ""
    method: POST
    headers: {}

metrics:
  enabled: true
  prometheus:
    port: 9090
    path: /metrics
  
tracing:
  enabled: false
  jaeger:
    endpoint: http://localhost:14268/api/traces
    service_name: db-backup

security:
  jwt:
    secret: ""
    expiration: 24h
  api_keys:
    enabled: true
  rate_limiting:
    enabled: true
    requests_per_minute: 100
```

---

### PHASE 2: Core CLI Implementation

#### 2.1 CLI Commands Structure

```go
// Root command
db-backup [command] [flags]

Commands:
  backup      Create a database backup
  restore     Restore from a backup
  list        List available backups
  delete      Delete a backup
  schedule    Manage backup schedules
  config      Manage configuration
  server      Start the web server
  version     Show version information

Global Flags:
  -c, --config string      Config file path (default: ./config.yaml)
  -v, --verbose           Verbose output
      --log-level string  Log level (debug|info|warn|error)
      --log-file string   Log file path
```

#### 2.2 Backup Command Implementation

```go
// Command: db-backup backup [flags]
Flags:
  -t, --type string          Database type (mysql|postgres|mongodb|sqlite)
  -h, --host string          Database host
  -P, --port int             Database port
  -u, --user string          Database user
  -p, --password string      Database password
  -d, --database string      Database name
      --databases strings    Multiple databases (comma-separated)
      --all-databases        Backup all databases
      --tables strings       Specific tables (comma-separated)
      --exclude-tables strings  Exclude tables
      
      --compression string   Compression type (gzip|zstd|lz4|none)
      --compress-level int   Compression level (1-9)
      
      --encrypt             Enable encryption
      --encryption-key string  Encryption key or key file
      
      --storage string      Storage provider (s3|gcs|azure|local)
      --storage-path string Custom storage path
      
      --name string         Backup name (auto-generated if not provided)
      --tags strings        Tags for backup (key=value)
      
      --notify              Send notifications
      --dry-run             Simulate backup without execution

Examples:
  # Basic MySQL backup
  db-backup backup --type mysql --host localhost --user root \
    --password secret --database mydb

  # PostgreSQL backup with compression and upload to S3
  db-backup backup --type postgres --host localhost \
    --database mydb --compression zstd --storage s3

  # MongoDB backup with encryption
  db-backup backup --type mongodb --host localhost \
    --database mydb --encrypt --encryption-key /path/to/key

  # Backup all MySQL databases
  db-backup backup --type mysql --host localhost \
    --all-databases --compression gzip

  # SQLite backup to local storage
  db-backup backup --type sqlite --database /path/to/db.sqlite \
    --storage local --storage-path /backups
```

#### 2.3 Restore Command Implementation

```go
// Command: db-backup restore [backup-id] [flags]
Flags:
  --target-type string      Target database type
  --target-host string      Target host
  --target-port int         Target port
  --target-user string      Target user
  --target-password string  Target password
  --target-database string  Target database name
  
  --point-in-time string    Restore to specific timestamp (RFC3339)
  --tables strings          Restore specific tables only
  --exclude-tables strings  Exclude tables from restore
  
  --skip-validation        Skip pre-restore validation
  --force                  Force restore without confirmation
  
  --decrypt                Decrypt backup
  --decryption-key string  Decryption key or key file
  
  --download-only          Download backup without restoring
  --download-path string   Download destination path

Examples:
  # Restore from backup ID
  db-backup restore backup-123-456 --target-host localhost

  # Point-in-time restore for PostgreSQL
  db-backup restore backup-123-456 \
    --point-in-time "2025-01-01T12:00:00Z"

  # Restore specific tables
  db-backup restore backup-123-456 \
    --tables users,orders --target-database newdb

  # Restore encrypted backup
  db-backup restore backup-123-456 \
    --decrypt --decryption-key /path/to/key
```

#### 2.4 List Command Implementation

```go
// Command: db-backup list [flags]
Flags:
  --database string      Filter by database name
  --type string         Filter by database type
  --storage string      Filter by storage provider
  --from string         Start date (RFC3339)
  --to string           End date (RFC3339)
  --tags strings        Filter by tags
  --format string       Output format (table|json|yaml)
  --limit int           Limit results (default: 50)
  --sort string         Sort by (date|size|name)
  --order string        Sort order (asc|desc)

Examples:
  # List all backups
  db-backup list

  # List MySQL backups from last week
  db-backup list --type mysql \
    --from "2024-12-20T00:00:00Z"

  # List backups with specific tags
  db-backup list --tags "env=production,app=api"

  # List in JSON format
  db-backup list --format json
```

#### 2.5 Schedule Command Implementation

```go
// Command: db-backup schedule [subcommand] [flags]
Subcommands:
  create      Create a new schedule
  list        List all schedules
  update      Update a schedule
  delete      Delete a schedule
  run         Run a schedule manually
  enable      Enable a schedule
  disable     Disable a schedule

# Create schedule
db-backup schedule create [flags]
Flags:
  --name string           Schedule name
  --cron string          Cron expression
  --database-id string   Database configuration ID
  --type string          Database type
  --compression string   Compression type
  --storage string       Storage provider
  --retention-days int   Retention period
  --encrypt             Enable encryption
  --notify              Enable notifications
  --tags strings        Tags for scheduled backups

Examples:
  # Daily backup at 2 AM
  db-backup schedule create --name "daily-prod-backup" \
    --cron "0 2 * * *" --database-id prod-mysql

  # Weekly backup every Sunday
  db-backup schedule create --name "weekly-backup" \
    --cron "0 3 * * 0" --database-id prod-pg \
    --compression zstd --encrypt

  # List all schedules
  db-backup schedule list

  # Run schedule manually
  db-backup schedule run daily-prod-backup
```

---

### PHASE 3: Database Driver Implementation

#### 3.1 Database Interface

```go
package database

import (
    "context"
    "io"
)

// Driver interface that all database drivers must implement
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

type ConnectionConfig struct {
    Type              DatabaseType
    Host              string
    Port              int
    Username          string
    Password          string
    Database          string
    SSLMode           string
    ConnectionString  string
    Options           map[string]string
    ConnectionTimeout time.Duration
    MaxConnections    int
}

type BackupOptions struct {
    Database         string
    Databases        []string
    AllDatabases     bool
    Tables           []string
    ExcludeTables    []string
    Incremental      bool
    ConsistentBackup bool
    OutputPath       string
    Compression      CompressionType
    Parallel         int
    ChunkSize        int64
    Metadata         map[string]string
}

type RestoreOptions struct {
    Database        string
    SourceBackup    string
    Tables          []string
    ExcludeTables   []string
    PointInTime     *time.Time
    SkipValidation  bool
    Parallel        int
    DropExisting    bool
    Metadata        map[string]string
}

type BackupResult struct {
    ID               string
    StartTime        time.Time
    EndTime          time.Time
    Duration         time.Duration
    Size             int64
    CompressedSize   int64
    DatabaseVersion  string
    Tables           []TableInfo
    Checksum         string
    Metadata         map[string]string
    Status           BackupStatus
    Error            error
}

type RestoreResult struct {
    StartTime       time.Time
    EndTime         time.Time
    Duration        time.Duration
    RestoredTables  []string
    RowsRestored    int64
    Status          RestoreStatus
    Error           error
}
```

#### 3.2 MySQL Driver Implementation

```go
package mysql

import (
    "context"
    "database/sql"
    "fmt"
    "os/exec"
    "io"
    
    _ "github.com/go-sql-driver/mysql"
)

type MySQLDriver struct {
    db     *sql.DB
    config *database.ConnectionConfig
}

func NewMySQLDriver() *MySQLDriver {
    return &MySQLDriver{}
}

func (d *MySQLDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
        config.Username,
        config.Password,
        config.Host,
        config.Port,
        config.Database,
    )
    
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return fmt.Errorf("failed to open connection: %w", err)
    }
    
    db.SetMaxOpenConns(config.MaxConnections)
    db.SetMaxIdleConns(config.MaxConnections / 2)
    db.SetConnMaxLifetime(time.Hour)
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }
    
    d.db = db
    d.config = config
    return nil
}

func (d *MySQLDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    result := &database.BackupResult{
        ID:        generateBackupID(),
        StartTime: time.Now(),
        Metadata:  opts.Metadata,
    }
    
    // Build mysqldump command
    args := d.buildMySQLDumpArgs(opts)
    
    cmd := exec.CommandContext(ctx, "mysqldump", args...)
    cmd.Env = append(cmd.Env, fmt.Sprintf("MYSQL_PWD=%s", d.config.Password))
    
    // Create output file or stream
    output, err := os.Create(opts.OutputPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create output file: %w", err)
    }
    defer output.Close()
    
    // Handle compression if enabled
    var writer io.Writer = output
    if opts.Compression != database.CompressionNone {
        compressor, err := compression.NewCompressor(opts.Compression)
        if err != nil {
            return nil, err
        }
        writer = compressor.Wrap(writer)
        defer compressor.Close()
    }
    
    cmd.Stdout = writer
    cmd.Stderr = os.Stderr
    
    // Execute backup
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("mysqldump failed: %w", err)
    }
    
    // Get file info
    fileInfo, err := output.Stat()
    if err != nil {
        return nil, err
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Size = fileInfo.Size()
    result.Status = database.BackupStatusSuccess
    
    // Calculate checksum
    result.Checksum, err = calculateChecksum(opts.OutputPath)
    if err != nil {
        return nil, err
    }
    
    // Get table information
    result.Tables, err = d.getTableInfo(ctx, opts.Database)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

func (d *MySQLDriver) buildMySQLDumpArgs(opts *database.BackupOptions) []string {
    args := []string{
        fmt.Sprintf("--host=%s", d.config.Host),
        fmt.Sprintf("--port=%d", d.config.Port),
        fmt.Sprintf("--user=%s", d.config.Username),
        "--single-transaction",
        "--routines",
        "--triggers",
        "--events",
    }
    
    if opts.AllDatabases {
        args = append(args, "--all-databases")
    } else if len(opts.Databases) > 0 {
        args = append(args, "--databases")
        args = append(args, opts.Databases...)
    } else {
        args = append(args, opts.Database)
    }
    
    if len(opts.Tables) > 0 {
        args = append(args, opts.Tables...)
    }
    
    if len(opts.ExcludeTables) > 0 {
        for _, table := range opts.ExcludeTables {
            args = append(args, fmt.Sprintf("--ignore-table=%s.%s", opts.Database, table))
        }
    }
    
    return args
}

func (d *MySQLDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
    result := &database.RestoreResult{
        StartTime: time.Now(),
    }
    
    // Validate backup file
    if !opts.SkipValidation {
        if err := d.ValidateRestore(ctx, opts); err != nil {
            return nil, err
        }
    }
    
    // Build mysql command
    args := []string{
        fmt.Sprintf("--host=%s", d.config.Host),
        fmt.Sprintf("--port=%d", d.config.Port),
        fmt.Sprintf("--user=%s", d.config.Username),
    }
    
    if opts.Database != "" {
        args = append(args, opts.Database)
    }
    
    cmd := exec.CommandContext(ctx, "mysql", args...)
    cmd.Env = append(cmd.Env, fmt.Sprintf("MYSQL_PWD=%s", d.config.Password))
    
    // Open backup file
    backupFile, err := os.Open(opts.SourceBackup)
    if err != nil {
        return nil, fmt.Errorf("failed to open backup file: %w", err)
    }
    defer backupFile.Close()
    
    // Handle decompression
    var reader io.Reader = backupFile
    if isCompressed(opts.SourceBackup) {
        decompressor, err := compression.NewDecompressor(detectCompression(opts.SourceBackup))
        if err != nil {
            return nil, err
        }
        reader = decompressor.Wrap(reader)
    }
    
    cmd.Stdin = reader
    cmd.Stderr = os.Stderr
    
    // Execute restore
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("mysql restore failed: %w", err)
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Status = database.RestoreStatusSuccess
    
    return result, nil
}

// Implement remaining interface methods...
// GetDatabases, GetTables, GetVersion, etc.
```

#### 3.3 PostgreSQL Driver Implementation

```go
package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "os/exec"
    
    _ "github.com/lib/pq"
)

type PostgreSQLDriver struct {
    db     *sql.DB
    config *database.ConnectionConfig
}

func NewPostgreSQLDriver() *PostgreSQLDriver {
    return &PostgreSQLDriver{}
}

func (d *PostgreSQLDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    result := &database.BackupResult{
        ID:        generateBackupID(),
        StartTime: time.Now(),
        Metadata:  opts.Metadata,
    }
    
    // Use pg_dump with custom format for better performance
    args := []string{
        "-h", d.config.Host,
        "-p", fmt.Sprintf("%d", d.config.Port),
        "-U", d.config.Username,
        "-F", "c",  // Custom format
        "-f", opts.OutputPath,
        "-v",       // Verbose
    }
    
    if opts.Parallel > 1 {
        args = append(args, "-j", fmt.Sprintf("%d", opts.Parallel))
    }
    
    if opts.ConsistentBackup {
        args = append(args, "--serializable-deferrable")
    }
    
    if len(opts.Tables) > 0 {
        for _, table := range opts.Tables {
            args = append(args, "-t", table)
        }
    }
    
    if len(opts.ExcludeTables) > 0 {
        for _, table := range opts.ExcludeTables {
            args = append(args, "-T", table)
        }
    }
    
    args = append(args, opts.Database)
    
    cmd := exec.CommandContext(ctx, "pg_dump", args...)
    cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", d.config.Password))
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
    }
    
    // Get backup file info
    fileInfo, err := os.Stat(opts.OutputPath)
    if err != nil {
        return nil, err
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Size = fileInfo.Size()
    result.Status = database.BackupStatusSuccess
    
    return result, nil
}

func (d *PostgreSQLDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
    result := &database.RestoreResult{
        StartTime: time.Now(),
    }
    
    // Use pg_restore for custom format backups
    args := []string{
        "-h", d.config.Host,
        "-p", fmt.Sprintf("%d", d.config.Port),
        "-U", d.config.Username,
        "-d", opts.Database,
        "-v",
        "--no-owner",
        "--no-acl",
    }
    
    if opts.Parallel > 1 {
        args = append(args, "-j", fmt.Sprintf("%d", opts.Parallel))
    }
    
    if opts.DropExisting {
        args = append(args, "--clean")
    }
    
    if len(opts.Tables) > 0 {
        for _, table := range opts.Tables {
            args = append(args, "-t", table)
        }
    }
    
    args = append(args, opts.SourceBackup)
    
    cmd := exec.CommandContext(ctx, "pg_restore", args...)
    cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", d.config.Password))
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("pg_restore failed: %w, output: %s", err, string(output))
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Status = database.RestoreStatusSuccess
    
    return result, nil
}

// Implement PITR support
func (d *PostgreSQLDriver) RestorePointInTime(ctx context.Context, opts *database.RestoreOptions) error {
    // Implementation for PostgreSQL PITR using WAL archives
    // This requires:
    // 1. Base backup
    // 2. WAL archives
    // 3. recovery.conf or recovery.signal configuration
    
    // Create recovery configuration
    recoveryConf := fmt.Sprintf(`
restore_command = 'cp /path/to/wal_archive/%%f %%p'
recovery_target_time = '%s'
recovery_target_action = 'promote'
`, opts.PointInTime.Format(time.RFC3339))
    
    // Write recovery configuration
    // Restart PostgreSQL to apply recovery
    // Monitor recovery process
    
    return nil
}
```

#### 3.4 MongoDB Driver Implementation

```go
package mongodb

import (
    "context"
    "fmt"
    "os/exec"
    
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBDriver struct {
    client *mongo.Client
    config *database.ConnectionConfig
}

func NewMongoDBDriver() *MongoDBDriver {
    return &MongoDBDriver{}
}

func (d *MongoDBDriver) Connect(ctx context.Context, config *database.ConnectionConfig) error {
    clientOpts := options.Client().
        ApplyURI(d.buildConnectionString(config)).
        SetMaxPoolSize(uint64(config.MaxConnections))
    
    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }
    
    if err := client.Ping(ctx, nil); err != nil {
        return fmt.Errorf("failed to ping: %w", err)
    }
    
    d.client = client
    d.config = config
    return nil
}

func (d *MongoDBDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    result := &database.BackupResult{
        ID:        generateBackupID(),
        StartTime: time.Now(),
        Metadata:  opts.Metadata,
    }
    
    // Build mongodump command
    args := []string{
        "--host", d.config.Host,
        "--port", fmt.Sprintf("%d", d.config.Port),
        "--username", d.config.Username,
        "--password", d.config.Password,
        "--out", opts.OutputPath,
        "--gzip",  // Always compress
    }
    
    if opts.Database != "" {
        args = append(args, "--db", opts.Database)
    }
    
    if len(opts.Tables) > 0 { // Collections in MongoDB
        for _, collection := range opts.Tables {
            args = append(args, "--collection", collection)
        }
    }
    
    // Add oplog for point-in-time consistency
    if opts.ConsistentBackup {
        args = append(args, "--oplog")
    }
    
    if opts.Parallel > 1 {
        args = append(args, "--numParallelCollections", fmt.Sprintf("%d", opts.Parallel))
    }
    
    cmd := exec.CommandContext(ctx, "mongodump", args...)
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("mongodump failed: %w, output: %s", err, string(output))
    }
    
    // Calculate total size
    totalSize, err := dirSize(opts.OutputPath)
    if err != nil {
        return nil, err
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Size = totalSize
    result.Status = database.BackupStatusSuccess
    
    return result, nil
}

func (d *MongoDBDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
    result := &database.RestoreResult{
        StartTime: time.Now(),
    }
    
    // Build mongorestore command
    args := []string{
        "--host", d.config.Host,
        "--port", fmt.Sprintf("%d", d.config.Port),
        "--username", d.config.Username,
        "--password", d.config.Password,
        "--gzip",
    }
    
    if opts.Database != "" {
        args = append(args, "--db", opts.Database)
    }
    
    if opts.DropExisting {
        args = append(args, "--drop")
    }
    
    if opts.Parallel > 1 {
        args = append(args, "--numParallelCollections", fmt.Sprintf("%d", opts.Parallel))
    }
    
    // Restore from oplog for PITR
    if opts.PointInTime != nil {
        args = append(args, 
            "--oplogReplay",
            "--oplogLimit", fmt.Sprintf("%d:%d", 
                opts.PointInTime.Unix(), 
                0,
            ),
        )
    }
    
    args = append(args, opts.SourceBackup)
    
    cmd := exec.CommandContext(ctx, "mongorestore", args...)
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("mongorestore failed: %w, output: %s", err, string(output))
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Status = database.RestoreStatusSuccess
    
    return result, nil
}
```

#### 3.5 SQLite Driver Implementation

```go
package sqlite

import (
    "context"
    "database/sql"
    "fmt"
    "io"
    "os"
    
    _ "github.com/mattn/go-sqlite3"
)

type SQLiteDriver struct {
    db     *sql.DB
    config *database.ConnectionConfig
}

func NewSQLiteDriver() *SQLiteDriver {
    return &SQLiteDriver{}
}

func (d *SQLiteDriver) Backup(ctx context.Context, opts *database.BackupOptions) (*database.BackupResult, error) {
    result := &database.BackupResult{
        ID:        generateBackupID(),
        StartTime: time.Now(),
        Metadata:  opts.Metadata,
    }
    
    // SQLite online backup using VACUUM INTO (SQLite 3.27.0+)
    // or file copy for older versions
    
    // Method 1: VACUUM INTO (preferred)
    query := fmt.Sprintf("VACUUM INTO '%s'", opts.OutputPath)
    if _, err := d.db.ExecContext(ctx, query); err != nil {
        // Fallback to file copy
        if err := d.fileCopyBackup(opts.OutputPath); err != nil {
            return nil, err
        }
    }
    
    // Get file info
    fileInfo, err := os.Stat(opts.OutputPath)
    if err != nil {
        return nil, err
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Size = fileInfo.Size()
    result.Status = database.BackupStatusSuccess
    
    return result, nil
}

func (d *SQLiteDriver) fileCopyBackup(destPath string) error {
    srcFile, err := os.Open(d.config.Database)
    if err != nil {
        return err
    }
    defer srcFile.Close()
    
    destFile, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer destFile.Close()
    
    // Lock database for consistent backup
    d.db.Exec("BEGIN IMMEDIATE")
    defer d.db.Exec("ROLLBACK")
    
    _, err = io.Copy(destFile, srcFile)
    return err
}

func (d *SQLiteDriver) Restore(ctx context.Context, opts *database.RestoreOptions) (*database.RestoreResult, error) {
    result := &database.RestoreResult{
        StartTime: time.Now(),
    }
    
    // Close existing connection
    if err := d.Disconnect(); err != nil {
        return nil, err
    }
    
    // Copy backup file to target location
    if err := copyFile(opts.SourceBackup, opts.Database); err != nil {
        return nil, fmt.Errorf("failed to copy backup: %w", err)
    }
    
    // Reopen connection
    if err := d.Connect(ctx, d.config); err != nil {
        return nil, err
    }
    
    // Verify integrity
    var integrityCheck string
    if err := d.db.QueryRow("PRAGMA integrity_check").Scan(&integrityCheck); err != nil {
        return nil, err
    }
    
    if integrityCheck != "ok" {
        return nil, fmt.Errorf("integrity check failed: %s", integrityCheck)
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    result.Status = database.RestoreStatusSuccess
    
    return result, nil
}
```

---

### PHASE 4: Cloud Storage Implementation

#### 4.1 Storage Interface

```go
package storage

import (
    "context"
    "io"
)

type Provider interface {
    // Upload operations
    Upload(ctx context.Context, localPath, remotePath string, opts *UploadOptions) error
    UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *UploadOptions) error
    
    // Download operations
    Download(ctx context.Context, remotePath, localPath string) error
    DownloadStream(ctx context.Context, remotePath string) (io.ReadCloser, error)
    
    // File operations
    Delete(ctx context.Context, remotePath string) error
    Exists(ctx context.Context, remotePath string) (bool, error)
    GetMetadata(ctx context.Context, remotePath string) (*FileMetadata, error)
    List(ctx context.Context, prefix string) ([]*FileMetadata, error)
    
    // Management
    GetType() ProviderType
    ValidateConfig() error
}

type UploadOptions struct {
    ContentType     string
    Metadata        map[string]string
    StorageClass    string
    ServerSideEncryption bool
    ACL             string
    ChunkSize       int64
    Parallel        int
    Checksum        string
}

type FileMetadata struct {
    Path            string
    Size            int64
    LastModified    time.Time
    ContentType     string
    Checksum        string
    StorageClass    string
    Metadata        map[string]string
}
```

#### 4.2 AWS S3 Implementation

```go
package s3

import (
    "context"
    "fmt"
    "io"
    "os"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Provider struct {
    client   *s3.Client
    uploader *manager.Uploader
    bucket   string
    config   *S3Config
}

type S3Config struct {
    Region          string
    Bucket          string
    AccessKeyID     string
    SecretAccessKey string
    Endpoint        string
    UsePathStyle    bool
    SSE             string
}

func NewS3Provider(cfg *S3Config) (*S3Provider, error) {
    awsCfg, err := config.LoadDefaultConfig(context.Background(),
        config.WithRegion(cfg.Region),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(
                cfg.AccessKeyID,
                cfg.SecretAccessKey,
                "",
            ),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        if cfg.Endpoint != "" {
            o.BaseEndpoint = aws.String(cfg.Endpoint)
        }
        o.UsePathStyle = cfg.UsePathStyle
    })
    
    uploader := manager.NewUploader(client, func(u *manager.Uploader) {
        u.PartSize = 64 * 1024 * 1024 // 64MB parts
        u.Concurrency = 10
    })
    
    return &S3Provider{
        client:   client,
        uploader: uploader,
        bucket:   cfg.Bucket,
        config:   cfg,
    }, nil
}

func (p *S3Provider) Upload(ctx context.Context, localPath, remotePath string, opts *storage.UploadOptions) error {
    file, err := os.Open(localPath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()
    
    return p.UploadStream(ctx, file, remotePath, opts)
}

func (p *S3Provider) UploadStream(ctx context.Context, reader io.Reader, remotePath string, opts *storage.UploadOptions) error {
    input := &s3.PutObjectInput{
        Bucket: aws.String(p.bucket),
        Key:    aws.String(remotePath),
        Body:   reader,
    }
    
    if opts != nil {
        if opts.ContentType != "" {
            input.ContentType = aws.String(opts.ContentType)
        }
        
        if opts.ServerSideEncryption {
            input.ServerSideEncryption = types.ServerSideEncryptionAes256
        }
        
        if len(opts.Metadata) > 0 {
            input.Metadata = opts.Metadata
        }
        
        if opts.StorageClass != "" {
            input.StorageClass = types.StorageClass(opts.StorageClass)
        }
    }
    
    // Use uploader for multipart upload
    _, err := p.uploader.Upload(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to upload to S3: %w", err)
    }
    
    return nil
}

func (p *S3Provider) Download(ctx context.Context, remotePath, localPath string) error {
    file, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    downloader := manager.NewDownloader(p.client, func(d *manager.Downloader) {
        d.PartSize = 64 * 1024 * 1024
        d.Concurrency = 10
    })
    
    _, err = downloader.Download(ctx, file, &s3.GetObjectInput{
        Bucket: aws.String(p.bucket),
        Key:    aws.String(remotePath),
    })
    
    return err
}

func (p *S3Provider) List(ctx context.Context, prefix string) ([]*storage.FileMetadata, error) {
    var files []*storage.FileMetadata
    
    paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
        Bucket: aws.String(p.bucket),
        Prefix: aws.String(prefix),
    })
    
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        
        for _, obj := range page.Contents {
            files = append(files, &storage.FileMetadata{
                Path:         *obj.Key,
                Size:         *obj.Size,
                LastModified: *obj.LastModified,
                Checksum:     *obj.ETag,
            })
        }
    }
    
    return files, nil
}

func (p *S3Provider) Delete(ctx context.Context, remotePath string) error {
    _, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(p.bucket),
        Key:    aws.String(remotePath),
    })
    return err
}
```

---

### PHASE 5: Web API Implementation

#### 5.1 API Server Setup

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/sanskarpan/db-backup/internal/api"
    "github.com/sanskarpan/db-backup/internal/config"
    "github.com/sanskarpan/db-backup/internal/logger"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }
    
    // Initialize logger
    log := logger.New(cfg.Logging)
    
    // Initialize Gin router
    if cfg.Server.Mode == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    router := gin.New()
    router.Use(gin.Recovery())
    router.Use(api.LoggerMiddleware(log))
    router.Use(api.CORSMiddleware())
    router.Use(api.RateLimitMiddleware(cfg.Security.RateLimiting))
    
    // Initialize API
    apiServer, err := api.NewServer(cfg, log)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to initialize API server")
    }
    
    // Register routes
    apiServer.RegisterRoutes(router)
    
    // Create HTTP server
    srv := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    // Start server
    go func() {
        log.Info().
            Str("addr", srv.Addr).
            Msg("Starting API server")
        
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal().Err(err).Msg("Failed to start server")
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Info().Msg("Shutting down server...")
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal().Err(err).Msg("Server forced to shutdown")
    }
    
    log.Info().Msg("Server exited")
}
```

#### 5.2 API Routes

```go
package api

import (
    "github.com/gin-gonic/gin"
)

func (s *Server) RegisterRoutes(router *gin.Engine) {
    // Health check
    router.GET("/health", s.handleHealthCheck)
    router.GET("/ready", s.handleReadinessCheck)
    
    // API v1
    v1 := router.Group("/api/v1")
    {
        // Authentication
        auth := v1.Group("/auth")
        {
            auth.POST("/login", s.handleLogin)
            auth.POST("/logout", s.handleLogout)
            auth.POST("/refresh", s.handleRefreshToken)
        }
        
        // Backups (requires authentication)
        backups := v1.Group("/backups")
        backups.Use(s.authMiddleware())
        {
            backups.POST("", s.handleCreateBackup)
            backups.GET("", s.handleListBackups)
            backups.GET("/:id", s.handleGetBackup)
            backups.DELETE("/:id", s.handleDeleteBackup)
            backups.POST("/:id/restore", s.handleRestoreBackup)
            backups.GET("/:id/download", s.handleDownloadBackup)
            backups.GET("/:id/logs", s.handleGetBackupLogs)
        }
        
        // Databases
        databases := v1.Group("/databases")
        databases.Use(s.authMiddleware())
        {
            databases.POST("", s.handleCreateDatabase)
            databases.GET("", s.handleListDatabases)
            databases.GET("/:id", s.handleGetDatabase)
            databases.PUT("/:id", s.handleUpdateDatabase)
            databases.DELETE("/:id", s.handleDeleteDatabase)
            databases.POST("/:id/test", s.handleTestDatabaseConnection)
        }
        
        // Schedules
        schedules := v1.Group("/schedules")
        schedules.Use(s.authMiddleware())
        {
            schedules.POST("", s.handleCreateSchedule)
            schedules.GET("", s.handleListSchedules)
            schedules.GET("/:id", s.handleGetSchedule)
            schedules.PUT("/:id", s.handleUpdateSchedule)
            schedules.DELETE("/:id", s.handleDeleteSchedule)
            schedules.POST("/:id/run", s.handleRunSchedule)
            schedules.POST("/:id/enable", s.handleEnableSchedule)
            schedules.POST("/:id/disable", s.handleDisableSchedule)
        }
        
        // Users (admin only)
        users := v1.Group("/users")
        users.Use(s.authMiddleware(), s.adminMiddleware())
        {
            users.POST("", s.handleCreateUser)
            users.GET("", s.handleListUsers)
            users.GET("/:id", s.handleGetUser)
            users.PUT("/:id", s.handleUpdateUser)
            users.DELETE("/:id", s.handleDeleteUser)
        }
        
        // Metrics
        metrics := v1.Group("/metrics")
        metrics.Use(s.authMiddleware())
        {
            metrics.GET("", s.handleGetMetrics)
            metrics.GET("/prometheus", s.handlePrometheusMetrics)
        }
        
        // Logs
        logs := v1.Group("/logs")
        logs.Use(s.authMiddleware())
        {
            logs.GET("", s.handleGetLogs)
            logs.GET("/stream", s.handleStreamLogs)
        }
        
        // Settings
        settings := v1.Group("/settings")
        settings.Use(s.authMiddleware())
        {
            settings.GET("", s.handleGetSettings)
            settings.PUT("", s.handleUpdateSettings)
        }
    }
    
    // WebSocket
    router.GET("/ws", s.handleWebSocket)
}
```

#### 5.3 Backup Handlers

```go
package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
)

type CreateBackupRequest struct {
    DatabaseID   string            `json:"database_id" binding:"required"`
    Name         string            `json:"name"`
    Compression  string            `json:"compression"`
    Encryption   bool              `json:"encryption"`
    Storage      string            `json:"storage"`
    Notify       bool              `json:"notify"`
    Tags         map[string]string `json:"tags"`
}

func (s *Server) handleCreateBackup(c *gin.Context) {
    var req CreateBackupRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Get user from context
    userID := c.GetString("user_id")
    
    // Get database configuration
    dbConfig, err := s.repos.Database.GetByID(c.Request.Context(), req.DatabaseID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
        return
    }
    
    // Create backup job
    backup, err := s.backupEngine.CreateBackup(c.Request.Context(), &backup.CreateOptions{
        DatabaseConfig: dbConfig,
        Name:          req.Name,
        Compression:   req.Compression,
        Encryption:    req.Encryption,
        Storage:       req.Storage,
        Tags:          req.Tags,
        UserID:        userID,
    })
    if err != nil {
        s.log.Error().Err(err).Msg("Failed to create backup")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup"})
        return
    }
    
    // Send notification if requested
    if req.Notify {
        go s.notifier.NotifyBackupStarted(backup)
    }
    
    c.JSON(http.StatusCreated, backup)
}

func (s *Server) handleListBackups(c *gin.Context) {
    // Parse query parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    databaseID := c.Query("database_id")
    status := c.Query("status")
    
    // Get backups
    backups, total, err := s.repos.Backup.List(c.Request.Context(), &repository.ListOptions{
        Page:       page,
        PageSize:   pageSize,
        DatabaseID: databaseID,
        Status:     status,
    })
    if err != nil {
        s.log.Error().Err(err).Msg("Failed to list backups")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "backups": backups,
        "total":   total,
        "page":    page,
        "page_size": pageSize,
    })
}

func (s *Server) handleGetBackup(c *gin.Context) {
    id := c.Param("id")
    
    backup, err := s.repos.Backup.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
        return
    }
    
    c.JSON(http.StatusOK, backup)
}

func (s *Server) handleRestoreBackup(c *gin.Context) {
    var req struct {
        TargetDatabaseID string    `json:"target_database_id" binding:"required"`
        PointInTime      *string   `json:"point_in_time"`
        Tables           []string  `json:"tables"`
        ExcludeTables    []string  `json:"exclude_tables"`
        Force            bool      `json:"force"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    id := c.Param("id")
    
    // Get backup
    backup, err := s.repos.Backup.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
        return
    }
    
    // Get target database
    targetDB, err := s.repos.Database.GetByID(c.Request.Context(), req.TargetDatabaseID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Target database not found"})
        return
    }
    
    // Parse point-in-time if provided
    var pit *time.Time
    if req.PointInTime != nil {
        t, err := time.Parse(time.RFC3339, *req.PointInTime)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid point_in_time format"})
            return
        }
        pit = &t
    }
    
    // Start restore operation
    restore, err := s.restoreEngine.RestoreBackup(c.Request.Context(), &restore.Options{
        Backup:         backup,
        TargetDatabase: targetDB,
        PointInTime:    pit,
        Tables:         req.Tables,
        ExcludeTables:  req.ExcludeTables,
        Force:          req.Force,
    })
    if err != nil {
        s.log.Error().Err(err).Msg("Failed to restore backup")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore backup"})
        return
    }
    
    c.JSON(http.StatusAccepted, restore)
}
```

---

### PHASE 6: Frontend Implementation

#### 6.1 Frontend Project Structure

```
web/
├── src/
│   ├── components/
│   │   ├── common/
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Modal.tsx
│   │   │   ├── Table.tsx
│   │   │   ├── Card.tsx
│   │   │   └── Spinner.tsx
│   │   ├── layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Footer.tsx
│   │   │   └── Layout.tsx
│   │   ├── backup/
│   │   │   ├── BackupList.tsx
│   │   │   ├── BackupCard.tsx
│   │   │   ├── CreateBackupModal.tsx
│   │   │   ├── BackupDetails.tsx
│   │   │   └── RestoreModal.tsx
│   │   ├── database/
│   │   │   ├── DatabaseList.tsx
│   │   │   ├── DatabaseForm.tsx
│   │   │   ├── DatabaseCard.tsx
│   │   │   └── ConnectionTest.tsx
│   │   ├── schedule/
│   │   │   ├── ScheduleList.tsx
│   │   │   ├── ScheduleForm.tsx
│   │   │   ├── CronBuilder.tsx
│   │   │   └── ScheduleHistory.tsx
│   │   ├── dashboard/
│   │   │   ├── StatsCard.tsx
│   │   │   ├── ActivityChart.tsx
│   │   │   ├── RecentBackups.tsx
│   │   │   └── SystemHealth.tsx
│   │   └── monitoring/
│   │       ├── MetricsChart.tsx
│   │       ├── LogViewer.tsx
│   │       └── AlertList.tsx
│   ├── pages/
│   │   ├── Dashboard.tsx
│   │   ├── Backups.tsx
│   │   ├── Databases.tsx
│   │   ├── Schedules.tsx
│   │   ├── Settings.tsx
│   │   ├── Monitoring.tsx
│   │   ├── Users.tsx
│   │   └── Login.tsx
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useBackups.ts
│   │   ├── useDatabases.ts
│   │   ├── useSchedules.ts
│   │   ├── useWebSocket.ts
│   │   └── useNotification.ts
│   ├── services/
│   │   ├── api.ts
│   │   ├── auth.ts
│   │   ├── backups.ts
│   │   ├── databases.ts
│   │   ├── schedules.ts
│   │   └── websocket.ts
│   ├── store/
│   │   ├── authSlice.ts
│   │   ├── backupSlice.ts
│   │   ├── databaseSlice.ts
│   │   ├── scheduleSlice.ts
│   │   └── store.ts
│   ├── types/
│   │   ├── backup.ts
│   │   ├── database.ts
│   │   ├── schedule.ts
│   │   ├── user.ts
│   │   └── api.ts
│   ├── utils/
│   │   ├── format.ts
│   │   ├── validation.ts
│   │   ├── constants.ts
│   │   └── helpers.ts
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── public/
│   ├── index.html
│   └── favicon.ico
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

#### 6.2 Key Frontend Components

```typescript
// Dashboard.tsx
import React, { useEffect } from 'react';
import { useBackups } from '@/hooks/useBackups';
import { useWebSocket } from '@/hooks/useWebSocket';
import StatsCard from '@/components/dashboard/StatsCard';
import ActivityChart from '@/components/dashboard/ActivityChart';
import RecentBackups from '@/components/dashboard/RecentBackups';

export default function Dashboard() {
  const { backups, stats, loading } = useBackups();
  const { connected, lastMessage } = useWebSocket('/ws');

  return (
    <div className="p-6">
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      
      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        <StatsCard
          title="Total Backups"
          value={stats.totalBackups}
          change={stats.backupChange}
          icon="database"
        />
        <StatsCard
          title="Success Rate"
          value={`${stats.successRate}%`}
          change={stats.rateChange}
          icon="check"
        />
        <StatsCard
          title="Storage Used"
          value={formatBytes(stats.storageUsed)}
          change={stats.storageChange}
          icon="hard-drive"
        />
        <StatsCard
          title="Active Schedules"
          value={stats.activeSchedules}
          icon="clock"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <Card>
          <ActivityChart data={stats.activityData} />
        </Card>
        <Card>
          <SystemHealth />
        </Card>
      </div>

      {/* Recent Backups */}
      <Card>
        <RecentBackups backups={backups} loading={loading} />
      </Card>
    </div>
  );
}

// CreateBackupModal.tsx
import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { createBackup } from '@/services/backups';
import Modal from '@/components/common/Modal';
import Input from '@/components/common/Input';
import Select from '@/components/common/Select';

interface CreateBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CreateBackupModal({ isOpen, onClose, onSuccess }: CreateBackupModalProps) {
  const { register, handleSubmit, formState: { errors } } = useForm();
  const [loading, setLoading] = useState(false);

  const onSubmit = async (data: any) => {
    setLoading(true);
    try {
      await createBackup(data);
      onSuccess();
      onClose();
    } catch (error) {
      console.error('Failed to create backup:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Create Backup">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Select
          label="Database"
          {...register('database_id', { required: true })}
          error={errors.database_id?.message}
        >
          {/* Database options */}
        </Select>

        <Input
          label="Backup Name (optional)"
          {...register('name')}
          placeholder="Auto-generated if empty"
        />

        <Select
          label="Compression"
          {...register('compression')}
          defaultValue="zstd"
        >
          <option value="none">None</option>
          <option value="gzip">Gzip</option>
          <option value="zstd">Zstandard</option>
          <option value="lz4">LZ4</option>
        </Select>

        <div className="flex items-center">
          <input
            type="checkbox"
            {...register('encryption')}
            className="mr-2"
          />
          <label>Enable Encryption</label>
        </div>

        <div className="flex items-center">
          <input
            type="checkbox"
            {...register('notify')}
            className="mr-2"
          />
          <label>Send Notification</label>
        </div>

        <div className="flex justify-end space-x-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Create Backup
          </Button>
        </div>
      </form>
    </Modal>
  );
}
```

---

### PHASE 7: Testing Strategy

#### 7.1 Unit Tests

```go
// backup_test.go
package backup_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/sanskarpan/db-backup/internal/backup"
)

func TestBackupEngine_CreateBackup(t *testing.T) {
    tests := []struct {
        name    string
        opts    *backup.CreateOptions
        wantErr bool
    }{
        {
            name: "successful backup",
            opts: &backup.CreateOptions{
                DatabaseConfig: &config.Database{
                    Type: "mysql",
                    Host: "localhost",
                },
                Compression: "zstd",
            },
            wantErr: false,
        },
        {
            name: "invalid database type",
            opts: &backup.CreateOptions{
                DatabaseConfig: &config.Database{
                    Type: "invalid",
                },
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := backup.NewEngine(mockConfig)
            result, err := engine.CreateBackup(context.Background(), tt.opts)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.NotEmpty(t, result.ID)
            }
        })
    }
}
```

#### 7.2 Integration Tests

```go
// integration_test.go
package integration_test

import (
    "context"
    "testing"
    
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestMySQLBackupRestore(t *testing.T) {
    ctx := context.Background()
    
    // Start MySQL container
    mysqlC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "mysql:8.0",
            ExposedPorts: []string{"3306/tcp"},
            Env: map[string]string{
                "MYSQL_ROOT_PASSWORD": "test",
                "MYSQL_DATABASE":      "testdb",
            },
            WaitingFor: wait.ForLog("ready for connections"),
        },
        Started: true,
    })
    if err != nil {
        t.Fatal(err)
    }
    defer mysqlC.Terminate(ctx)
    
    // Get connection details
    host, _ := mysqlC.Host(ctx)
    port, _ := mysqlC.MappedPort(ctx, "3306")
    
    // Test backup
    driver := mysql.NewMySQLDriver()
    err = driver.Connect(ctx, &database.ConnectionConfig{
        Host:     host,
        Port:     port.Int(),
        Username: "root",
        Password: "test",
        Database: "testdb",
    })
    assert.NoError(t, err)
    
    // Create test data
    // Perform backup
    // Verify backup
    // Restore backup
    // Verify restored data
}
```

#### 7.3 E2E Tests

```typescript
// e2e/backup.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Backup Management', () => {
  test('should create a backup', async ({ page }) => {
    await page.goto('/');
    
    // Login
    await page.fill('[name="email"]', 'admin@example.com');
    await page.fill('[name="password"]', 'password');
    await page.click('button[type="submit"]');
    
    // Navigate to backups
    await page.click('text=Backups');
    
    // Create backup
    await page.click('text=Create Backup');
    await page.selectOption('[name="database_id"]', 'db-1');
    await page.click('button:has-text("Create")');
    
    // Verify success
    await expect(page.locator('.success-message')).toBeVisible();
    await expect(page.locator('.backup-list')).toContainText('backup-');
  });

  test('should restore from backup', async ({ page }) => {
    // Similar test for restore operation
  });
});
```

---

### PHASE 8: Deployment

#### 8.1 Dockerfile

```dockerfile
# Multi-stage build for CLI
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN make build-cli

# Final stage
FROM alpine:latest

RUN apk add --no-cache \
    ca-certificates \
    mysql-client \
    postgresql-client \
    mongodb-tools \
    sqlite

COPY --from=builder /app/bin/db-backup /usr/local/bin/

ENTRYPOINT ["db-backup"]
```

#### 8.2 Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: db-backup-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: db-backup-server
  template:
    metadata:
      labels:
        app: db-backup-server
    spec:
      containers:
      - name: server
        image: your-registry/db-backup-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_PATH
          value: /config/config.yaml
        volumeMounts:
        - name: config
          mountPath: /config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: db-backup-config
---
apiVersion: v1
kind: Service
metadata:
  name: db-backup-server
spec:
  selector:
    app: db-backup-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## Success Metrics

1. **Code Quality**
   - Test coverage > 80%
   - No critical security vulnerabilities
   - Code passes all linters

2. **Performance**
   - Backup creation < 5 minutes for 1GB database
   - API response time < 200ms (p95)
   - Support 1000+ concurrent users

3. **Reliability**
   - Backup success rate > 99.5%
   - System uptime > 99.9%
   - Recovery time < 30 minutes

4. **User Experience**
   - Intuitive UI/UX
   - Clear documentation
   - Responsive support

---

## Next Steps

1. Review and approve CHECKLIST.md
2. Set up development environment
3. Begin Phase 1 implementation
4. Regular progress reviews
5. Iterative testing and improvement

---

**This is a comprehensive guide. Follow the CHECKLIST.md for detailed phase-by-phase implementation.**