# Shell Completions for DB Backup Manager

Intelligent tab-completion for the `db-backup` command across multiple shells.

## Features

- ✅ **Bash** completion with caching
- ✅ **Zsh** completion with descriptions and oh-my-zsh support
- ✅ **Fish** completion with rich descriptions
- ✅ **PowerShell** completion for Windows
- ✅ **Dynamic completion** - database names, backup IDs, schedules, etc.
- ✅ **Context-aware suggestions** - suggests relevant flags based on command
- ✅ **Caching** - fast completions with 5-minute cache TTL
- ✅ **Smart filtering** - filters out already-used flags

## Quick Start

### Automatic Installation

Run the installer to automatically detect your shell and install:

```bash
cd completions
./install.sh
```

### Manual Installation

#### Bash

```bash
# System-wide (requires sudo)
sudo cp bash/db-backup /etc/bash_completion.d/

# User-local
mkdir -p ~/.bash_completion.d
cp bash/db-backup ~/.bash_completion.d/
echo "[[ -f ~/.bash_completion.d/db-backup ]] && source ~/.bash_completion.d/db-backup" >> ~/.bashrc
```

#### Zsh

```bash
# For oh-my-zsh users
cp zsh/_db-backup ~/.oh-my-zsh/completions/

# Standard zsh
sudo cp zsh/_db-backup /usr/local/share/zsh/site-functions/

# User-local
mkdir -p ~/.zfunc
cp zsh/_db-backup ~/.zfunc/
echo 'fpath=(~/.zfunc $fpath)' >> ~/.zshrc
echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
```

#### Fish

```bash
# User-local
mkdir -p ~/.config/fish/completions
cp fish/db-backup.fish ~/.config/fish/completions/
```

#### PowerShell (Windows)

```powershell
# Load for current session
. .\powershell\db-backup.ps1

# Add to PowerShell profile
Add-Content $PROFILE ". $PWD\powershell\db-backup.ps1"
```

## Usage Examples

### Basic Command Completion

```bash
db-backup <TAB>
# Shows: backup restore database schedule retention monitoring compliance ...

db-backup backup <TAB>
# Shows: create list delete download restore verify info cleanup stats
```

### Dynamic Database Completion

```bash
db-backup backup create --database <TAB>
# Shows: production-db staging-db development-db test-db

db-backup restore full --backup <TAB>
# Shows: backup-2024-01-13-001 backup-2024-01-13-002 ...
```

### Context-Aware Flag Completion

```bash
db-backup backup create --<TAB>
# Shows only relevant flags: --database --type --provider --compression --encryption ...

# After typing --database flag
db-backup backup create --database prod --<TAB>
# Shows remaining flags (excludes --database)
```

### Provider and Type Completion

```bash
db-backup backup create --provider <TAB>
# Shows: local s3 gcs azure minio wasabi backblaze digitalocean

db-backup database add --type <TAB>
# Shows: postgres mysql mongodb redis sqlite cassandra dynamodb ...
```

### Compliance Framework Completion

```bash
db-backup compliance scan --framework <TAB>
# Shows: gdpr hipaa sox pci-dss iso27001 ccpa
```

## Features by Shell

### Bash

- ✅ Command and subcommand completion
- ✅ Flag completion with context-awareness
- ✅ Dynamic completion for databases, backups, schedules
- ✅ File path completion for --output and --config
- ✅ Caching with 5-minute TTL
- ✅ Alias support (dbb, dbbackup)

### Zsh

- ✅ All Bash features
- ✅ **Rich descriptions** for each completion option
- ✅ **Grouped completions** (commands, flags, values)
- ✅ **oh-my-zsh compatibility**
- ✅ **Argument completion** with validation
- ✅ Better formatting and visual presentation

### Fish

- ✅ All Bash features
- ✅ **Inline descriptions** shown while typing
- ✅ **Smart filtering** as you type
- ✅ **Condition-based completion** (only show relevant options)
- ✅ **Colorized output** with syntax highlighting
- ✅ Superior user experience

### PowerShell

- ✅ All Bash features
- ✅ **Parameter completion** with Tab and Ctrl+Space
- ✅ **Type-aware completion** for parameters
- ✅ **Custom completion results** with descriptions
- ✅ Windows-native paths and conventions

## Dynamic Completion

The completion scripts query the `db-backup` CLI to get real-time data:

### Cached Data

The following data is cached for 5 minutes to improve performance:

- Database names
- Backup IDs
- Schedule names
- Retention policies
- Monitoring alerts
- Tags

### Cache Location

- **Bash/Zsh/Fish**: `~/.cache/db-backup/completion/`
- **PowerShell**: `%LOCALAPPDATA%\db-backup\completion\`

### Clear Cache

```bash
# Unix-like systems
rm -rf ~/.cache/db-backup/completion/

# Windows PowerShell
Remove-Item -Recurse "$env:LOCALAPPDATA\db-backup\completion"
```

## Troubleshooting

### Completions Not Working

**Bash:**
```bash
# Verify installation
ls -la /etc/bash_completion.d/db-backup
# or
ls -la ~/.bash_completion.d/db-backup

# Reload
source ~/.bash_completion.d/db-backup
# or restart shell
```

**Zsh:**
```bash
# Verify installation
ls -la ~/.oh-my-zsh/completions/_db-backup
# or
ls -la /usr/local/share/zsh/site-functions/_db-backup

# Reload
exec zsh

# Debug
echo $fpath
# Should include completion directory
```

**Fish:**
```bash
# Verify installation
ls -la ~/.config/fish/completions/db-backup.fish

# Reload
exec fish

# Test function
functions -q __db_backup_databases && echo "Loaded"
```

**PowerShell:**
```powershell
# Verify loading
Get-ArgumentCompleter -Native -CommandName db-backup

# Reload
. .\powershell\db-backup.ps1
```

### Slow Completions

Completions cache data for 5 minutes. If still slow:

1. **Check network connectivity** to API server
2. **Verify db-backup CLI is in PATH**: `which db-backup`
3. **Test CLI directly**: `db-backup database list --format=name`
4. **Check cache directory** permissions

### Cache Issues

If completions show stale data:

```bash
# Clear cache (Unix)
rm -rf ~/.cache/db-backup/completion/

# Or wait 5 minutes for auto-expiry
```

### Permission Errors

If installation fails:

```bash
# Use user-local installation instead
./install.sh

# Or run with sudo for system-wide
sudo ./install.sh
```

## Advanced Configuration

### Custom Cache TTL

Edit the completion script and change:

**Bash:**
```bash
_DB_BACKUP_CACHE_TTL=300  # Change to desired seconds
```

**Zsh:**
```zsh
typeset -g _DB_BACKUP_CACHE_TTL=300  # Change to desired seconds
```

**Fish:**
```fish
set -g _DB_BACKUP_CACHE_TTL 300  # Change to desired seconds
```

**PowerShell:**
```powershell
$script:DB_BACKUP_CACHE_TTL = 300  # Change to desired seconds
```

### Disable Caching

Set TTL to 0 to always query live data:

```bash
_DB_BACKUP_CACHE_TTL=0
```

Note: This will slow down completions.

### Add Custom Aliases

**Bash:**
```bash
complete -F _db_backup_completion my-alias
```

**Zsh:**
```zsh
compdef _db-backup my-alias
```

**Fish:**
```fish
complete -c my-alias -w db-backup
```

**PowerShell:**
```powershell
Register-ArgumentCompleter -Native -CommandName my-alias -ScriptBlock $scriptblock
```

## Development

### Testing Completions

**Bash:**
```bash
# Source the completion script
source bash/db-backup

# Test completion
COMP_WORDS=(db-backup backup create --) COMP_CWORD=3
_db_backup_completion
echo "${COMPREPLY[@]}"
```

**Zsh:**
```zsh
# Load completion
autoload -Uz compinit && compinit
source zsh/_db-backup

# Test interactively
db-backup <TAB>
```

**Fish:**
```fish
# Load completion
source fish/db-backup.fish

# Test interactively
db-backup <TAB>
```

### Adding New Completions

1. **Update the completion script** for your shell
2. **Add to dynamic completion functions** if data is from API
3. **Test the completion** works correctly
4. **Update this README** with examples

### Generating from Cobra

The CLI uses Cobra which can generate completions:

```bash
# Generate Bash completion
db-backup completion bash > bash/db-backup-generated

# Generate Zsh completion
db-backup completion zsh > zsh/_db-backup-generated

# Generate Fish completion
db-backup completion fish > fish/db-backup-generated.fish

# Generate PowerShell completion
db-backup completion powershell > powershell/db-backup-generated.ps1
```

## Shell Support Matrix

| Feature | Bash | Zsh | Fish | PowerShell |
|---------|------|-----|------|------------|
| Basic commands | ✅ | ✅ | ✅ | ✅ |
| Subcommands | ✅ | ✅ | ✅ | ✅ |
| Flags | ✅ | ✅ | ✅ | ✅ |
| Dynamic data | ✅ | ✅ | ✅ | ✅ |
| Descriptions | ❌ | ✅ | ✅ | ✅ |
| Caching | ✅ | ✅ | ✅ | ✅ |
| File completion | ✅ | ✅ | ✅ | ✅ |
| Context-aware | ✅ | ✅ | ✅ | ✅ |

## Performance

Typical completion times:

- **First completion**: 50-200ms (queries API)
- **Cached completion**: 5-20ms (reads cache)
- **Static completion**: 1-5ms (no API call)

Cache expires after 5 minutes.

## Security

- Completions run in user context
- Cache files are user-owned with 644 permissions
- No passwords or secrets are cached
- Only non-sensitive data (names, IDs) is cached

## Uninstallation

```bash
# Automatic (detects shell)
./install.sh uninstall

# Manual - Bash
rm /etc/bash_completion.d/db-backup
# or
rm ~/.bash_completion.d/db-backup

# Manual - Zsh
rm ~/.oh-my-zsh/completions/_db-backup
# or
rm /usr/local/share/zsh/site-functions/_db-backup

# Manual - Fish
rm ~/.config/fish/completions/db-backup.fish

# Manual - PowerShell
# Remove from PowerShell profile
```

## Contributing

Improvements welcome! Please:

1. Test on your shell and OS
2. Ensure backward compatibility
3. Update this README
4. Add examples for new features

## License

Same as DB Backup Manager project.

## Support

- **Documentation**: This README
- **Issues**: GitHub Issues
- **Shell help**: `man bash`, `man zsh`, `man fish`, `Get-Help`
