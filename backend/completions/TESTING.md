# Shell Completion Testing Guide

This document provides a comprehensive testing guide for the db-backup shell completions.

## Prerequisites

Before testing, ensure you have:

1. **Shell Access**: Access to the shell you want to test (bash, zsh, fish, or PowerShell)
2. **db-backup CLI**: The CLI should be built and in your PATH
3. **Completion Scripts**: All completion scripts are in the `completions/` directory

## Quick Test Script

Run this automated test to verify basic functionality:

```bash
cd completions
./test-completions.sh
```

This will:
- Verify syntax of all shell scripts
- Test the installation script
- Check for common issues
- Provide a summary report

## Manual Testing by Shell

### Bash Completion Testing

#### 1. Source the Completion Script

```bash
source bash/db-backup
```

#### 2. Test Basic Command Completion

```bash
# Type this and press TAB:
db-backup <TAB>

# Expected output:
# backup restore database schedule retention monitoring compliance ...
```

#### 3. Test Subcommand Completion

```bash
# Type this and press TAB:
db-backup backup <TAB>

# Expected output:
# create list delete download restore verify info cleanup stats
```

#### 4. Test Flag Completion

```bash
# Type this and press TAB:
db-backup backup create --<TAB>

# Expected output:
# --database --type --provider --compression --encryption ...
```

#### 5. Test Dynamic Database Completion

```bash
# First ensure you have databases configured
db-backup database add --name test-db --type postgres

# Then test:
db-backup backup create --database <TAB>

# Expected: Should show database names from CLI
```

#### 6. Test Provider Completion

```bash
db-backup backup create --provider <TAB>

# Expected output:
# local s3 gcs azure minio wasabi backblaze digitalocean
```

#### 7. Test Caching

```bash
# First completion (queries API):
time db-backup backup create --database <TAB>

# Second completion within 5 minutes (uses cache):
time db-backup backup create --database <TAB>

# Second completion should be significantly faster
```

#### 8. Verify Completion Function

```bash
# Check function is loaded:
type _db_backup_completion

# Should output: _db_backup_completion is a function
```

### Zsh Completion Testing

#### 1. Load the Completion Script

```zsh
# Add to fpath and load:
fpath=(./zsh $fpath)
autoload -Uz compinit && compinit
```

#### 2. Test Basic Completion

```zsh
db-backup <TAB>

# Expected: Commands with descriptions
# backup    -- Manage backups
# restore   -- Restore databases
# database  -- Manage database configurations
```

#### 3. Test Rich Descriptions

```zsh
db-backup backup create --provider <TAB>

# Expected: Providers with descriptions
# local           -- Local filesystem storage
# s3              -- Amazon S3
# gcs             -- Google Cloud Storage
```

#### 4. Test Grouped Completions

```zsh
db-backup backup create --<TAB>

# Expected: Flags grouped by category with descriptions
```

#### 5. Test oh-my-zsh Compatibility

If using oh-my-zsh:

```zsh
# Copy to completions directory:
cp zsh/_db-backup ~/.oh-my-zsh/completions/

# Reload:
exec zsh

# Test:
db-backup <TAB>
```

### Fish Completion Testing

#### 1. Load the Completion Script

```fish
source fish/db-backup.fish
```

#### 2. Test Inline Descriptions

```fish
db-backup <TAB>

# Expected: Commands with inline descriptions as you type
# backup (Manage backups)
# restore (Restore databases)
```

#### 3. Test Smart Filtering

```fish
# Type partial command:
db-backup ba<TAB>

# Expected: Only shows 'backup' (filters in real-time)
```

#### 4. Test Condition-Based Completion

```fish
# Type:
db-backup backup create --database <TAB>

# Expected: Only shows databases (not other values)
```

#### 5. Test Completion Functions

```fish
# Verify functions are loaded:
functions -q __db_backup_databases
echo $status

# Expected: 0 (function exists)
```

### PowerShell Completion Testing (Windows)

#### 1. Load the Completion Script

```powershell
. .\powershell\db-backup.ps1
```

#### 2. Test Parameter Completion

```powershell
# Type and press TAB:
db-backup backup create -<TAB>

# Expected: Parameter suggestions with descriptions
```

#### 3. Test Value Completion

```powershell
# Type and press TAB or Ctrl+Space:
db-backup backup create --provider <TAB>

# Expected: Provider names
# local, s3, gcs, azure, ...
```

#### 4. Test ArgumentCompleter Registration

```powershell
# Verify completer is registered:
Get-ArgumentCompleter -Native -CommandName db-backup

# Expected: Should show registered completer
```

## Testing Installation

### Automatic Installation Test

```bash
# Run installer in dry-run mode:
cd completions
./install.sh --help

# Install for your shell:
./install.sh

# Expected: Script detects shell and installs to correct location
```

### Manual Installation Verification

#### Bash

```bash
# Check installation:
ls -la ~/.bash_completion.d/db-backup
# or
ls -la /etc/bash_completion.d/db-backup

# Check if sourced in bashrc:
grep db-backup ~/.bashrc
```

#### Zsh

```bash
# Check installation:
ls -la ~/.oh-my-zsh/completions/_db-backup
# or
ls -la /usr/local/share/zsh/site-functions/_db-backup

# Check fpath:
echo $fpath
```

#### Fish

```bash
# Check installation:
ls -la ~/.config/fish/completions/db-backup.fish

# Test function loading:
functions -q __db_backup_databases
```

## Testing Cache Functionality

### 1. Cache Creation Test

```bash
# Clear cache:
rm -rf ~/.cache/db-backup/completion/

# Trigger completion (creates cache):
db-backup backup create --database <TAB>

# Verify cache created:
ls -la ~/.cache/db-backup/completion/
```

### 2. Cache Expiry Test

```bash
# Create cache entry:
db-backup backup create --database <TAB>

# Check timestamp:
stat ~/.cache/db-backup/completion/databases

# Wait 6 minutes (or modify TTL to 10 seconds for testing)

# Trigger completion again:
db-backup backup create --database <TAB>

# Verify cache was regenerated (new timestamp)
```

### 3. Cache Performance Test

```bash
# First run (no cache):
time db-backup backup create --database <TAB>

# Second run (with cache):
time db-backup backup create --database <TAB>

# Expected: Second run should be 5-10x faster
```

## Testing Dynamic Completion

### Database Completion

```bash
# Add test databases:
db-backup database add --name prod-db --type postgres
db-backup database add --name staging-db --type mysql

# Test completion shows both:
db-backup backup create --database <TAB>

# Expected: prod-db staging-db
```

### Backup ID Completion

```bash
# Create test backups:
db-backup backup create --database prod-db

# Test completion shows backup IDs:
db-backup restore full --backup <TAB>

# Expected: Lists of backup IDs with timestamps
```

### Schedule Completion

```bash
# Create test schedule:
db-backup schedule create --name daily-backup --cron "0 2 * * *"

# Test completion:
db-backup schedule update --name <TAB>

# Expected: daily-backup
```

## Common Issues and Troubleshooting

### Issue 1: Completions Not Working

**Bash:**
```bash
# Verify function is loaded:
type _db_backup_completion

# If not loaded, source again:
source ~/.bash_completion.d/db-backup
```

**Zsh:**
```zsh
# Verify fpath includes completion directory:
echo $fpath

# Reload completions:
exec zsh
```

**Fish:**
```fish
# Verify function exists:
functions -q __db_backup_databases

# Reload:
exec fish
```

### Issue 2: Slow Completions

```bash
# Check API connectivity:
db-backup database list --format=name

# Check cache directory permissions:
ls -la ~/.cache/db-backup/completion/

# Test cache TTL:
# Modify script to set TTL to 0 for debugging
```

### Issue 3: Empty Completions

```bash
# Verify CLI is in PATH:
which db-backup

# Test CLI directly:
db-backup database list --format=name

# Check for errors:
db-backup backup create --database <TAB> 2>&1 | grep -i error
```

### Issue 4: Cache Stale Data

```bash
# Clear cache:
rm -rf ~/.cache/db-backup/completion/

# Regenerate:
db-backup backup create --database <TAB>
```

## Automated Test Suite

Create a test script `test-completions.sh`:

```bash
#!/bin/bash

echo "Testing DB Backup Completions..."
echo "================================="

# Test 1: Syntax Check
echo "1. Checking Bash syntax..."
bash -n bash/db-backup && echo "✓ Bash syntax OK" || echo "✗ Bash syntax ERROR"

echo "2. Checking Zsh syntax..."
zsh -n zsh/_db-backup && echo "✓ Zsh syntax OK" || echo "✗ Zsh syntax ERROR"

echo "3. Checking install script..."
bash -n install.sh && echo "✓ Install script OK" || echo "✗ Install script ERROR"

# Test 2: Shell Detection
echo "4. Testing shell detection..."
SHELL_TYPE=$(bash install.sh --help | grep -q "Usage:" && echo "✓ Help works" || echo "✗ Help broken")
echo "$SHELL_TYPE"

# Test 3: File Existence
echo "5. Checking completion files..."
[[ -f bash/db-backup ]] && echo "✓ Bash completion exists" || echo "✗ Bash missing"
[[ -f zsh/_db-backup ]] && echo "✓ Zsh completion exists" || echo "✗ Zsh missing"
[[ -f fish/db-backup.fish ]] && echo "✓ Fish completion exists" || echo "✗ Fish missing"
[[ -f powershell/db-backup.ps1 ]] && echo "✓ PowerShell completion exists" || echo "✗ PowerShell missing"

# Test 4: Source Test (Bash)
echo "6. Testing Bash completion loading..."
if source bash/db-backup 2>/dev/null; then
    type _db_backup_completion >/dev/null 2>&1 && echo "✓ Bash completion loads" || echo "✗ Bash function not defined"
else
    echo "✗ Bash completion failed to load"
fi

echo ""
echo "================================="
echo "Basic tests complete!"
echo ""
echo "For full testing, manually test in each shell."
echo "See TESTING.md for detailed instructions."
```

## Performance Benchmarks

### Expected Performance

- **First completion (no cache)**: 50-200ms
- **Cached completion**: 5-20ms
- **Static completion**: 1-5ms
- **Cache TTL**: 5 minutes (300 seconds)

### Measuring Performance

```bash
# Bash timing:
time (db-backup backup create --database <TAB>)

# Zsh timing:
time db-backup backup create --database <TAB>

# Fish timing:
time db-backup backup create --database <TAB>
```

## Security Testing

### 1. Permission Check

```bash
# Cache files should be user-owned:
ls -la ~/.cache/db-backup/completion/
# Expected: User ownership, 644 permissions
```

### 2. No Sensitive Data in Cache

```bash
# Inspect cache files:
cat ~/.cache/db-backup/completion/*

# Expected: Only names and IDs, no passwords or secrets
```

### 3. Script Injection Test

```bash
# Ensure no command injection in database names:
db-backup database add --name "; rm -rf /" --type postgres
db-backup backup create --database <TAB>

# Expected: Should escape or handle safely
```

## Completion Coverage Checklist

- [x] Basic command completion
- [x] Subcommand completion
- [x] Flag completion
- [x] Dynamic database completion
- [x] Dynamic backup ID completion
- [x] Dynamic schedule completion
- [x] Provider completion
- [x] Database type completion
- [x] Compression format completion
- [x] Encryption algorithm completion
- [x] Compliance framework completion
- [x] File path completion (--output, --config)
- [x] Context-aware flag filtering
- [x] Caching functionality
- [x] Cache expiry
- [x] Alias support (dbb, dbbackup)
- [x] Multi-shell support (bash, zsh, fish, powershell)
- [x] Installation automation
- [x] Error handling

## Test Results Template

```
# DB Backup Completion Test Results
Date: _______________
Tester: _______________
OS: _______________
Shell: _______________

## Test Results

| Test | Status | Notes |
|------|--------|-------|
| Bash syntax | ☐ Pass ☐ Fail | |
| Zsh syntax | ☐ Pass ☐ Fail | |
| Fish syntax | ☐ Pass ☐ Fail | |
| PowerShell syntax | ☐ Pass ☐ Fail | |
| Installation | ☐ Pass ☐ Fail | |
| Basic completion | ☐ Pass ☐ Fail | |
| Subcommand completion | ☐ Pass ☐ Fail | |
| Flag completion | ☐ Pass ☐ Fail | |
| Dynamic database | ☐ Pass ☐ Fail | |
| Dynamic backup | ☐ Pass ☐ Fail | |
| Provider completion | ☐ Pass ☐ Fail | |
| Type completion | ☐ Pass ☐ Fail | |
| Caching works | ☐ Pass ☐ Fail | |
| Cache expires | ☐ Pass ☐ Fail | |
| Performance OK | ☐ Pass ☐ Fail | |
| No security issues | ☐ Pass ☐ Fail | |

## Overall Status

☐ All tests passed
☐ Some tests failed (see notes)
☐ Major issues found

## Recommendations

_____________________
```

## Conclusion

After completing all tests above, the shell completions should:

1. ✅ Load correctly in all supported shells
2. ✅ Provide accurate command/flag suggestions
3. ✅ Query dynamic data from the CLI
4. ✅ Cache results for performance
5. ✅ Handle errors gracefully
6. ✅ Install automatically
7. ✅ Provide rich descriptions (zsh, fish, powershell)
8. ✅ Work with shell aliases
9. ✅ Expire cache after TTL
10. ✅ Maintain security (no sensitive data exposure)

For any issues found during testing, refer to the troubleshooting section or file a bug report.
