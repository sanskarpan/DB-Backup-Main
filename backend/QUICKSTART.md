# Quick Start Guide

## For Developers

### Prerequisites

1. **Install Go 1.21+**
   ```bash
   # macOS
   brew install go

   # Linux
   wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
   ```

2. **Install MySQL Tools** (for MySQL driver)
   ```bash
   # macOS
   brew install mysql-client

   # Ubuntu/Debian
   sudo apt-get install mysql-client

   # RHEL/CentOS
   sudo yum install mysql
   ```

3. **Install Make** (usually pre-installed on Unix systems)

### Getting Started

1. **Clone and Setup**
   ```bash
   cd /Users/sanskar/dev/db-backup

   # Download Go dependencies
   go mod download

   # Or use make
   make deps
   ```

2. **Build the CLI**
   ```bash
   # Build CLI binary
   make build-cli

   # Binary will be at: bin/db-backup
   ```

3. **Create Configuration**
   ```bash
   # Copy example config
   cp config.yaml.example config.yaml

   # Edit as needed
   vim config.yaml
   ```

4. **Run the CLI**
   ```bash
   # Show help
   ./bin/db-backup --help

   # Show version
   ./bin/db-backup version

   # List available commands
   ./bin/db-backup
   ```

### Development Workflow

1. **Make Changes**
   ```bash
   # Edit code in your preferred editor
   vim internal/database/mysql/driver.go
   ```

2. **Format and Check**
   ```bash
   # Format code
   make fmt

   # Run linter (if installed)
   make lint

   # Run vet
   make vet
   ```

3. **Test**
   ```bash
   # Run all tests
   make test

   # Run tests with coverage
   make test-coverage

   # Open coverage report
   open coverage.html
   ```

4. **Build**
   ```bash
   # Build all binaries
   make build

   # Or just CLI
   make build-cli
   ```

### Project Structure Navigation

```
db-backup/
├── cmd/cli/              # Start here for CLI commands
│   ├── main.go          # Entry point
│   └── commands/        # All CLI commands
│       ├── root.go      # Root command & global flags
│       ├── backup.go    # Backup command
│       ├── restore.go   # Restore command
│       ├── list.go      # List command
│       └── version.go   # Version command
│
├── internal/             # Private application code
│   ├── config/          # Configuration management
│   │   └── config.go    # Config structs & loading
│   ├── logger/          # Logging system
│   │   └── logger.go    # Logger wrapper
│   └── database/        # Database drivers
│       ├── interface.go # Driver interface (IMPORTANT!)
│       └── mysql/       # MySQL implementation
│           └── driver.go
│
└── pkg/                 # Public reusable packages
    └── errors/          # Custom error types
        ├── errors.go
        └── errors_test.go
```

### Common Tasks

#### Add a New Database Driver

1. Create package: `internal/database/postgres/`
2. Implement `database.Driver` interface
3. Add to `DriverFactory` in `interface.go`
4. Create tests

#### Add a New CLI Command

1. Create file in `cmd/cli/commands/`
2. Define command with Cobra
3. Add to `init()` in your file: `rootCmd.AddCommand(yourCmd)`
4. Implement `RunE` function

#### Add a New Configuration Option

1. Add field to struct in `internal/config/config.go`
2. Add default in `setDefaults()`
3. Add validation in `validate()` if needed
4. Update `config.yaml.example`

### Testing MySQL Backup (Example)

```bash
# Make sure MySQL is running
# Option 1: Local MySQL
mysql -u root -e "CREATE DATABASE testdb;"
mysql -u root testdb < test_data.sql

# Option 2: Docker MySQL
docker run -d --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=test \
  -e MYSQL_DATABASE=testdb \
  -p 3306:3306 \
  mysql:8.0

# Run backup (when engine is implemented)
./bin/db-backup backup \
  --type mysql \
  --host localhost \
  --user root \
  --password test \
  --database testdb \
  --compression gzip

# Check backups directory
ls -lh ./backups/
```

### Debugging Tips

1. **Enable Debug Logging**
   ```bash
   ./bin/db-backup backup --verbose [other flags]
   # or
   export DBBACKUP_LOGGING_LEVEL=debug
   ```

2. **Use Dry Run**
   ```bash
   ./bin/db-backup backup --dry-run [other flags]
   ```

3. **Check Configuration**
   ```bash
   # Add a config dump command or check logs
   ./bin/db-backup --verbose version
   ```

### Next Development Tasks

Based on PROGRESS.md, the immediate next steps are:

1. **Implement Backup Engine** (`internal/backup/engine.go`)
   - Orchestrate backup workflow
   - Manage metadata
   - Handle progress tracking

2. **Implement Local Storage** (`internal/storage/local/client.go`)
   - File system operations
   - Upload/download abstraction

3. **Wire Everything Together**
   - Update `backup.go` command to use engine
   - Update `restore.go` command to use engine
   - Update `list.go` to read from storage

4. **Add Tests**
   - Unit tests for each component
   - Integration tests with real MySQL

### Useful Make Commands

```bash
make help              # Show all commands
make build             # Build everything
make build-cli         # Build just the CLI
make test              # Run tests
make test-coverage     # Generate coverage report
make clean             # Remove build artifacts
make fmt               # Format code
make vet               # Run go vet
make lint              # Run golangci-lint
make deps              # Download dependencies
make run-cli           # Run CLI without building
```

### Environment Setup for Development

```bash
# Set Go environment
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install air for hot reload (optional)
go install github.com/cosmtrek/air@latest
```

### Troubleshooting

**"go: command not found"**
- Install Go or add to PATH

**"mysqldump: command not found"**
- Install MySQL client tools

**"permission denied" when running binary**
- Run: `chmod +x bin/db-backup`

**Tests failing**
- Run: `go mod tidy`
- Check if all dependencies are downloaded

**Import errors**
- Make sure you're in the project directory
- Run: `go mod download`

### Resources

- [Go Documentation](https://golang.org/doc/)
- [Cobra CLI](https://github.com/spf13/cobra)
- [Viper Config](https://github.com/spf13/viper)
- [Zerolog](https://github.com/rs/zerolog)
- [Project CHECKLIST.md](CHECKLIST.md) - Full implementation plan
- [Project PROGRESS.md](PROGRESS.md) - Current status

### Getting Help

- Check [README.md](README.md) for overview
- Check [PROGRESS.md](PROGRESS.md) for current status
- Check [CHECKLIST.md](CHECKLIST.md) for implementation plan
- Open an issue on GitHub

---

**Happy Coding! 🚀**
