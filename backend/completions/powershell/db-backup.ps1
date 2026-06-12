# PowerShell completion script for db-backup
# Supports intelligent context-aware suggestions

# ============================================================================
# Cache Configuration
# ============================================================================

$script:DB_BACKUP_CACHE_DIR = Join-Path $env:LOCALAPPDATA "db-backup\completion"
$script:DB_BACKUP_CACHE_TTL = 300  # 5 minutes

# ============================================================================
# Helper Functions
# ============================================================================

function Get-DbBackupCacheFile {
    param([string]$Key)
    return Join-Path $script:DB_BACKUP_CACHE_DIR "$Key.cache"
}

function Test-DbBackupCacheValid {
    param([string]$CacheFile)

    if (-not (Test-Path $CacheFile)) {
        return $false
    }

    $cacheTime = (Get-Item $CacheFile).LastWriteTime
    $currentTime = Get-Date
    $age = ($currentTime - $cacheTime).TotalSeconds

    return $age -le $script:DB_BACKUP_CACHE_TTL
}

function Read-DbBackupCache {
    param([string]$CacheFile)

    if (Test-DbBackupCacheValid $CacheFile) {
        return Get-Content $CacheFile -Raw
    }

    return $null
}

function Write-DbBackupCache {
    param(
        [string]$Key,
        [string]$Data
    )

    if (-not (Test-Path $script:DB_BACKUP_CACHE_DIR)) {
        New-Item -ItemType Directory -Path $script:DB_BACKUP_CACHE_DIR -Force | Out-Null
    }

    $cacheFile = Get-DbBackupCacheFile $Key
    $Data | Out-File -FilePath $cacheFile -Encoding UTF8
}

function Invoke-DbBackupExec {
    param(
        [string]$Command,
        [string]$CacheKey
    )

    $cacheFile = Get-DbBackupCacheFile $CacheKey

    # Try cache first
    $cached = Read-DbBackupCache $cacheFile
    if ($cached) {
        return $cached
    }

    # Execute command
    try {
        $result = Invoke-Expression $Command 2>$null
        if ($result) {
            Write-DbBackupCache $CacheKey $result
            return $result
        }
    }
    catch {
        return $null
    }

    return $null
}

# ============================================================================
# Dynamic Completion Functions
# ============================================================================

function Get-DbBackupDatabases {
    $databases = Invoke-DbBackupExec "db-backup database list --format=name" "databases"
    if ($databases) {
        return $databases -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

function Get-DbBackupBackups {
    param([string]$Database)

    if ($Database) {
        $backups = Invoke-DbBackupExec "db-backup backup list --database='$Database' --format=id" "backups_$Database"
    }
    else {
        $backups = Invoke-DbBackupExec "db-backup backup list --format=id" "backups_all"
    }

    if ($backups) {
        return $backups -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

function Get-DbBackupSchedules {
    $schedules = Invoke-DbBackupExec "db-backup schedule list --format=name" "schedules"
    if ($schedules) {
        return $schedules -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

function Get-DbBackupProviders {
    return @(
        'local',
        's3',
        'gcs',
        'azure',
        'minio',
        'wasabi',
        'backblaze',
        'digitalocean'
    )
}

function Get-DbBackupDatabaseTypes {
    return @(
        'postgres',
        'mysql',
        'mongodb',
        'redis',
        'sqlite',
        'cassandra',
        'dynamodb',
        'elasticsearch',
        'influxdb',
        'timescaledb'
    )
}

function Get-DbBackupCompressionFormats {
    return @('gzip', 'zstd', 'lz4', 'none')
}

function Get-DbBackupEncryptionAlgorithms {
    return @('aes256', 'aes128', 'chacha20')
}

function Get-DbBackupRetentionPolicies {
    $policies = Invoke-DbBackupExec "db-backup retention list --format=name" "retention_policies"
    if ($policies) {
        return $policies -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

function Get-DbBackupComplianceFrameworks {
    return @('gdpr', 'hipaa', 'sox', 'pci-dss', 'iso27001', 'ccpa')
}

function Get-DbBackupAlerts {
    $alerts = Invoke-DbBackupExec "db-backup monitoring alerts --format=id" "alerts"
    if ($alerts) {
        return $alerts -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

function Get-DbBackupTags {
    $tags = Invoke-DbBackupExec "db-backup tag list --format=name" "tags"
    if ($tags) {
        return $tags -split "`n" | Where-Object { $_ -ne "" }
    }
    return @()
}

# ============================================================================
# Main Completion Function
# ============================================================================

$scriptblock = {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commands = @(
        'backup',
        'restore',
        'database',
        'schedule',
        'retention',
        'monitoring',
        'compliance',
        'policy',
        'encryption',
        'catalog',
        'tag',
        'export',
        'import',
        'health',
        'version',
        'help'
    )

    # Subcommands for each main command
    $subcommands = @{
        'backup'     = @('create', 'list', 'delete', 'download', 'restore', 'verify', 'info', 'cleanup', 'stats')
        'restore'    = @('full', 'incremental', 'point-in-time', 'download', 'validate', 'test')
        'database'   = @('add', 'list', 'remove', 'update', 'test', 'info', 'stats', 'migrate')
        'schedule'   = @('create', 'list', 'delete', 'update', 'enable', 'disable', 'run', 'info')
        'retention'  = @('create', 'list', 'delete', 'update', 'apply', 'cleanup', 'info')
        'monitoring' = @('status', 'alerts', 'metrics', 'dashboard', 'logs', 'enable', 'disable')
        'compliance' = @('scan', 'report', 'frameworks', 'violations', 'export', 'audit')
        'policy'     = @('create', 'list', 'delete', 'update', 'validate', 'test', 'export', 'import')
        'encryption' = @('enable', 'disable', 'rotate-keys', 'info', 'test', 'algorithms')
        'catalog'    = @('sync', 'rebuild', 'cleanup', 'stats', 'search', 'export', 'import')
        'tag'        = @('add', 'list', 'remove', 'search')
        'export'     = @('backups', 'databases', 'schedules', 'config', 'all')
        'import'     = @('backups', 'databases', 'schedules', 'config')
        'health'     = @('check', 'status', 'components', 'metrics')
    }

    $tokens = $commandAst.ToString() -split '\s+' | Where-Object { $_ -ne '' }

    # Find main command and subcommand
    $mainCmd = $null
    $subCmd = $null

    for ($i = 1; $i -lt $tokens.Count; $i++) {
        if ($commands -contains $tokens[$i]) {
            $mainCmd = $tokens[$i]
            if ($i + 1 -lt $tokens.Count -and $subcommands[$mainCmd] -contains $tokens[$i + 1]) {
                $subCmd = $tokens[$i + 1]
            }
            break
        }
    }

    # Complete main commands
    if (-not $mainCmd) {
        $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    # Complete subcommands
    if ($mainCmd -and -not $subCmd) {
        $subcommands[$mainCmd] | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    # Find the last parameter
    $lastToken = $tokens[-1]
    $prevToken = if ($tokens.Count -gt 1) { $tokens[-2] } else { $null }

    # Complete parameter values based on the previous parameter
    switch -Regex ($prevToken) {
        '^(--database|-d)$' {
            Get-DbBackupDatabases | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Database: $_")
            }
            return
        }
        '^(--backup|-b|--backup-id)$' {
            # Try to find database from previous args
            $database = $null
            for ($i = 0; $i -lt $tokens.Count; $i++) {
                if ($tokens[$i] -match '^(--database|-d)$' -and $i + 1 -lt $tokens.Count) {
                    $database = $tokens[$i + 1]
                    break
                }
            }
            Get-DbBackupBackups $database | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Backup: $_")
            }
            return
        }
        '^(--schedule|-s)$' {
            Get-DbBackupSchedules | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Schedule: $_")
            }
            return
        }
        '^(--provider|-p)$' {
            Get-DbBackupProviders | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Provider: $_")
            }
            return
        }
        '^(--type|-t)$' {
            Get-DbBackupDatabaseTypes | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Type: $_")
            }
            return
        }
        '^(--compression|-c)$' {
            Get-DbBackupCompressionFormats | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Compression: $_")
            }
            return
        }
        '^(--encryption|-e)$' {
            Get-DbBackupEncryptionAlgorithms | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Encryption: $_")
            }
            return
        }
        '^--retention$' {
            Get-DbBackupRetentionPolicies | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Retention: $_")
            }
            return
        }
        '^(--framework|-f)$' {
            Get-DbBackupComplianceFrameworks | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Framework: $_")
            }
            return
        }
        '^--alert$' {
            Get-DbBackupAlerts | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Alert: $_")
            }
            return
        }
        '^--tag$' {
            Get-DbBackupTags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Tag: $_")
            }
            return
        }
        '^--format$' {
            @('json', 'yaml', 'table', 'csv', 'id', 'name') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Format: $_")
            }
            return
        }
        '^--log-level$' {
            @('debug', 'info', 'warn', 'error') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Log level: $_")
            }
            return
        }
        '^--interval$' {
            @('hourly', 'daily', 'weekly', 'monthly') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Interval: $_")
            }
            return
        }
        '^--priority$' {
            @('low', 'normal', 'high', 'critical') | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Priority: $_")
            }
            return
        }
        '^(--output|-o|--config)$' {
            # File path completion
            Get-ChildItem -Path . -File | Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_.Name, $_.Name, 'ParameterValue', $_.FullName)
            }
            return
        }
    }

    # Context-aware flag completion
    $flags = @()

    switch ("$mainCmd`:$subCmd") {
        'backup:create' {
            $flags = @('--database', '--type', '--provider', '--compression', '--encryption', '--retention',
                '--tag', '--description', '--full', '--incremental', '--verify', '--notify')
        }
        'backup:list' {
            $flags = @('--database', '--provider', '--status', '--from', '--to', '--limit', '--format',
                '--tag', '--sort', '--order')
        }
        'backup:delete' {
            $flags = @('--backup', '--force', '--purge', '--keep-metadata')
        }
        'backup:restore' {
            $flags = @('--backup', '--database', '--target', '--point-in-time', '--verify', '--dry-run')
        }
        'restore:full' {
            $flags = @('--backup', '--database', '--target', '--overwrite', '--verify', '--parallel')
        }
        'restore:incremental' {
            $flags = @('--backup', '--database', '--target', '--base-backup', '--verify')
        }
        'restore:point-in-time' {
            $flags = @('--database', '--timestamp', '--target', '--verify', '--dry-run')
        }
        'database:add' {
            $flags = @('--name', '--type', '--host', '--port', '--username', '--password', '--database',
                '--ssl', '--connection-string', '--test')
        }
        'database:list' {
            $flags = @('--type', '--status', '--format', '--sort', '--limit')
        }
        'database:update' {
            $flags = @('--database', '--host', '--port', '--username', '--password', '--ssl', '--test')
        }
        'schedule:create' {
            $flags = @('--name', '--database', '--interval', '--time', '--retention', '--enabled',
                '--compression', '--encryption', '--notify', '--tag')
        }
        'schedule:list' {
            $flags = @('--status', '--format', '--sort')
        }
        'schedule:update' {
            $flags = @('--schedule', '--interval', '--time', '--enabled', '--retention')
        }
        'retention:create' {
            $flags = @('--name', '--keep-daily', '--keep-weekly', '--keep-monthly', '--keep-yearly',
                '--min-backups', '--max-backups')
        }
        'retention:apply' {
            $flags = @('--database', '--policy', '--dry-run', '--force')
        }
        'monitoring:status' {
            $flags = @('--format', '--refresh', '--watch')
        }
        'monitoring:alerts' {
            $flags = @('--status', '--severity', '--from', '--to', '--format')
        }
        'compliance:scan' {
            $flags = @('--framework', '--database', '--report', '--fix', '--format')
        }
        'compliance:report' {
            $flags = @('--framework', '--from', '--to', '--format', '--output')
        }
        'encryption:enable' {
            $flags = @('--database', '--algorithm', '--key-file', '--rotate', '--verify')
        }
        'policy:create' {
            $flags = @('--name', '--type', '--rules', '--enabled', '--validate')
        }
        default {
            $flags = @('--help', '--version', '--config', '--log-level', '--verbose', '--quiet', '--json', '--no-color')
        }
    }

    # Add common flags
    $flags += @('--help', '--verbose', '--quiet', '--json', '--no-color')

    # Filter out already used flags
    $usedFlags = $tokens | Where-Object { $_ -like '--*' }
    $availableFlags = $flags | Where-Object { $usedFlags -notcontains $_ }

    # Complete flags
    $availableFlags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
    }
}

# ============================================================================
# Register Completion
# ============================================================================

# Register completion for db-backup
Register-ArgumentCompleter -Native -CommandName db-backup -ScriptBlock $scriptblock

# Also register for common aliases
Register-ArgumentCompleter -Native -CommandName dbb -ScriptBlock $scriptblock
Register-ArgumentCompleter -Native -CommandName dbbackup -ScriptBlock $scriptblock

# Export completion function for manual import
Export-ModuleMember -Function *
