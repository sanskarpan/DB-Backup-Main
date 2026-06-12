# Fish completion script for db-backup
# Supports intelligent context-aware suggestions

# ============================================================================
# Cache Configuration
# ============================================================================

set -g _DB_BACKUP_CACHE_DIR (string replace -r '^$' "$HOME/.cache" -- $XDG_CACHE_HOME)/db-backup/completion
set -g _DB_BACKUP_CACHE_TTL 300

# ============================================================================
# Helper Functions
# ============================================================================

function __db_backup_cache_file
    echo "$_DB_BACKUP_CACHE_DIR/$argv[1].cache"
end

function __db_backup_cache_valid
    set cache_file $argv[1]

    test -f "$cache_file"; or return 1

    set cache_time (stat -f %m "$cache_file" 2>/dev/null; or stat -c %Y "$cache_file" 2>/dev/null)
    set current_time (date +%s)
    set age (math $current_time - $cache_time)

    test $age -le $_DB_BACKUP_CACHE_TTL; and return 0
    return 1
end

function __db_backup_cache_read
    set cache_file $argv[1]

    if __db_backup_cache_valid "$cache_file"
        cat "$cache_file"
        return 0
    end

    return 1
end

function __db_backup_cache_write
    set key $argv[1]
    set data $argv[2]

    mkdir -p "$_DB_BACKUP_CACHE_DIR"
    set cache_file (__db_backup_cache_file "$key")
    echo "$data" > "$cache_file"
end

function __db_backup_exec
    set cmd $argv[1]
    set cache_key $argv[2]

    set cache_file (__db_backup_cache_file "$cache_key")

    # Try cache first
    set cached (__db_backup_cache_read "$cache_file")
    if test $status -eq 0
        echo "$cached"
        return 0
    end

    # Execute command
    set result (eval $cmd 2>/dev/null)
    if test $status -eq 0
        __db_backup_cache_write "$cache_key" "$result"
        echo "$result"
        return 0
    end

    return 1
end

# ============================================================================
# Dynamic Completion Functions
# ============================================================================

function __db_backup_databases
    __db_backup_exec "db-backup database list --format=name" "databases" | string split \n | string match -v ''
end

function __db_backup_backups
    set database $argv[1]
    if test -n "$database"
        __db_backup_exec "db-backup backup list --database='$database' --format=id" "backups_$database" | string split \n
    else
        __db_backup_exec "db-backup backup list --format=id" "backups_all" | string split \n
    end
end

function __db_backup_schedules
    __db_backup_exec "db-backup schedule list --format=name" "schedules" | string split \n | string match -v ''
end

function __db_backup_providers
    printf '%s\n' local s3 gcs azure minio wasabi backblaze digitalocean
end

function __db_backup_database_types
    printf '%s\n' postgres mysql mongodb redis sqlite cassandra dynamodb elasticsearch influxdb timescaledb
end

function __db_backup_compression_formats
    printf '%s\n' gzip zstd lz4 none
end

function __db_backup_encryption_algorithms
    printf '%s\n' aes256 aes128 chacha20
end

function __db_backup_retention_policies
    __db_backup_exec "db-backup retention list --format=name" "retention_policies" | string split \n | string match -v ''
end

function __db_backup_compliance_frameworks
    printf '%s\n' gdpr hipaa sox pci-dss iso27001 ccpa
end

function __db_backup_alerts
    __db_backup_exec "db-backup monitoring alerts --format=id" "alerts" | string split \n | string match -v ''
end

function __db_backup_tags
    __db_backup_exec "db-backup tag list --format=name" "tags" | string split \n | string match -v ''
end

# ============================================================================
# Condition Functions
# ============================================================================

function __db_backup_using_command
    set cmd (commandline -opc)
    test (count $cmd) -ge 2; and test "$cmd[2]" = "$argv[1]"
end

function __db_backup_using_subcommand
    set cmd (commandline -opc)
    test (count $cmd) -ge 3; and test "$cmd[2]" = "$argv[1]"; and test "$cmd[3]" = "$argv[2]"
end

function __db_backup_needs_database
    set cmd (commandline -opc)
    not string match -q -- '*--database*' $cmd
end

function __db_backup_needs_backup
    set cmd (commandline -opc)
    not string match -q -- '*--backup*' $cmd
end

# ============================================================================
# Main Commands
# ============================================================================

# Main commands
complete -c db-backup -f -n "__fish_use_subcommand" -a backup -d "Backup operations"
complete -c db-backup -f -n "__fish_use_subcommand" -a restore -d "Restore operations"
complete -c db-backup -f -n "__fish_use_subcommand" -a database -d "Database management"
complete -c db-backup -f -n "__fish_use_subcommand" -a schedule -d "Schedule management"
complete -c db-backup -f -n "__fish_use_subcommand" -a retention -d "Retention policy management"
complete -c db-backup -f -n "__fish_use_subcommand" -a monitoring -d "Monitoring and alerts"
complete -c db-backup -f -n "__fish_use_subcommand" -a compliance -d "Compliance scanning and reporting"
complete -c db-backup -f -n "__fish_use_subcommand" -a policy -d "Policy management"
complete -c db-backup -f -n "__fish_use_subcommand" -a encryption -d "Encryption management"
complete -c db-backup -f -n "__fish_use_subcommand" -a catalog -d "Backup catalog operations"
complete -c db-backup -f -n "__fish_use_subcommand" -a tag -d "Tag management"
complete -c db-backup -f -n "__fish_use_subcommand" -a export -d "Export configurations"
complete -c db-backup -f -n "__fish_use_subcommand" -a import -d "Import configurations"
complete -c db-backup -f -n "__fish_use_subcommand" -a health -d "Health checks"
complete -c db-backup -f -n "__fish_use_subcommand" -a version -d "Show version"
complete -c db-backup -f -n "__fish_use_subcommand" -a help -d "Show help"

# ============================================================================
# Backup Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command backup" -a create -d "Create a new backup"
complete -c db-backup -f -n "__db_backup_using_command backup" -a list -d "List backups"
complete -c db-backup -f -n "__db_backup_using_command backup" -a delete -d "Delete a backup"
complete -c db-backup -f -n "__db_backup_using_command backup" -a download -d "Download a backup"
complete -c db-backup -f -n "__db_backup_using_command backup" -a restore -d "Restore from backup"
complete -c db-backup -f -n "__db_backup_using_command backup" -a verify -d "Verify backup integrity"
complete -c db-backup -f -n "__db_backup_using_command backup" -a info -d "Show backup information"
complete -c db-backup -f -n "__db_backup_using_command backup" -a cleanup -d "Cleanup old backups"
complete -c db-backup -f -n "__db_backup_using_command backup" -a stats -d "Show backup statistics"

# Backup flags
complete -c db-backup -n "__db_backup_using_command backup" -l database -s d -d "Database name" -a "(__db_backup_databases)"
complete -c db-backup -n "__db_backup_using_command backup" -l type -s t -d "Database type" -a "(__db_backup_database_types)"
complete -c db-backup -n "__db_backup_using_command backup" -l provider -s p -d "Storage provider" -a "(__db_backup_providers)"
complete -c db-backup -n "__db_backup_using_command backup" -l compression -s c -d "Compression format" -a "(__db_backup_compression_formats)"
complete -c db-backup -n "__db_backup_using_command backup" -l encryption -s e -d "Encryption algorithm" -a "(__db_backup_encryption_algorithms)"
complete -c db-backup -n "__db_backup_using_command backup" -l retention -d "Retention policy" -a "(__db_backup_retention_policies)"
complete -c db-backup -n "__db_backup_using_command backup" -l tag -d "Tag" -a "(__db_backup_tags)"
complete -c db-backup -n "__db_backup_using_command backup" -l description -d "Backup description"
complete -c db-backup -n "__db_backup_using_command backup" -l full -d "Full backup"
complete -c db-backup -n "__db_backup_using_command backup" -l incremental -d "Incremental backup"
complete -c db-backup -n "__db_backup_using_command backup" -l verify -d "Verify after backup"
complete -c db-backup -n "__db_backup_using_command backup" -l notify -d "Send notifications"

# ============================================================================
# Restore Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command restore" -a full -d "Full database restore"
complete -c db-backup -f -n "__db_backup_using_command restore" -a incremental -d "Incremental restore"
complete -c db-backup -f -n "__db_backup_using_command restore" -a point-in-time -d "Point-in-time recovery"
complete -c db-backup -f -n "__db_backup_using_command restore" -a download -d "Download backup file"
complete -c db-backup -f -n "__db_backup_using_command restore" -a validate -d "Validate backup before restore"
complete -c db-backup -f -n "__db_backup_using_command restore" -a test -d "Test restore without applying"

# Restore flags
complete -c db-backup -n "__db_backup_using_command restore" -l backup -s b -d "Backup ID" -a "(__db_backup_backups)"
complete -c db-backup -n "__db_backup_using_command restore" -l database -s d -d "Database name" -a "(__db_backup_databases)"
complete -c db-backup -n "__db_backup_using_command restore" -l target -d "Target database"
complete -c db-backup -n "__db_backup_using_command restore" -l point-in-time -d "Timestamp for PITR"
complete -c db-backup -n "__db_backup_using_command restore" -l verify -d "Verify after restore"
complete -c db-backup -n "__db_backup_using_command restore" -l dry-run -d "Dry run mode"
complete -c db-backup -n "__db_backup_using_command restore" -l overwrite -d "Overwrite existing"
complete -c db-backup -n "__db_backup_using_command restore" -l parallel -d "Parallel restore threads"

# ============================================================================
# Database Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command database" -a add -d "Add a new database"
complete -c db-backup -f -n "__db_backup_using_command database" -a list -d "List databases"
complete -c db-backup -f -n "__db_backup_using_command database" -a remove -d "Remove a database"
complete -c db-backup -f -n "__db_backup_using_command database" -a update -d "Update database configuration"
complete -c db-backup -f -n "__db_backup_using_command database" -a test -d "Test database connection"
complete -c db-backup -f -n "__db_backup_using_command database" -a info -d "Show database information"
complete -c db-backup -f -n "__db_backup_using_command database" -a stats -d "Show database statistics"
complete -c db-backup -f -n "__db_backup_using_command database" -a migrate -d "Migrate database"

# Database flags
complete -c db-backup -n "__db_backup_using_command database" -l name -d "Database name"
complete -c db-backup -n "__db_backup_using_command database" -l type -s t -d "Database type" -a "(__db_backup_database_types)"
complete -c db-backup -n "__db_backup_using_command database" -l host -d "Database host"
complete -c db-backup -n "__db_backup_using_command database" -l port -d "Database port"
complete -c db-backup -n "__db_backup_using_command database" -l username -s u -d "Username"
complete -c db-backup -n "__db_backup_using_command database" -l password -s p -d "Password"
complete -c db-backup -n "__db_backup_using_command database" -l database -s d -d "Database name"
complete -c db-backup -n "__db_backup_using_command database" -l ssl -d "Enable SSL"
complete -c db-backup -n "__db_backup_using_command database" -l connection-string -d "Connection string"
complete -c db-backup -n "__db_backup_using_command database" -l test -d "Test connection"

# ============================================================================
# Schedule Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command schedule" -a create -d "Create a schedule"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a list -d "List schedules"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a delete -d "Delete a schedule"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a update -d "Update a schedule"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a enable -d "Enable a schedule"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a disable -d "Disable a schedule"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a run -d "Run a schedule now"
complete -c db-backup -f -n "__db_backup_using_command schedule" -a info -d "Show schedule information"

# Schedule flags
complete -c db-backup -n "__db_backup_using_command schedule" -l name -d "Schedule name"
complete -c db-backup -n "__db_backup_using_command schedule" -l schedule -s s -d "Schedule to operate on" -a "(__db_backup_schedules)"
complete -c db-backup -n "__db_backup_using_command schedule" -l database -s d -d "Database" -a "(__db_backup_databases)"
complete -c db-backup -n "__db_backup_using_command schedule" -l interval -d "Backup interval" -a "hourly daily weekly monthly"
complete -c db-backup -n "__db_backup_using_command schedule" -l time -d "Backup time"
complete -c db-backup -n "__db_backup_using_command schedule" -l retention -d "Retention policy" -a "(__db_backup_retention_policies)"
complete -c db-backup -n "__db_backup_using_command schedule" -l enabled -d "Enable schedule"
complete -c db-backup -n "__db_backup_using_command schedule" -l compression -s c -d "Compression" -a "(__db_backup_compression_formats)"
complete -c db-backup -n "__db_backup_using_command schedule" -l encryption -s e -d "Encryption" -a "(__db_backup_encryption_algorithms)"
complete -c db-backup -n "__db_backup_using_command schedule" -l notify -d "Send notifications"
complete -c db-backup -n "__db_backup_using_command schedule" -l tag -d "Tag" -a "(__db_backup_tags)"

# ============================================================================
# Retention Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command retention" -a create -d "Create retention policy"
complete -c db-backup -f -n "__db_backup_using_command retention" -a list -d "List retention policies"
complete -c db-backup -f -n "__db_backup_using_command retention" -a delete -d "Delete retention policy"
complete -c db-backup -f -n "__db_backup_using_command retention" -a update -d "Update retention policy"
complete -c db-backup -f -n "__db_backup_using_command retention" -a apply -d "Apply retention policy"
complete -c db-backup -f -n "__db_backup_using_command retention" -a cleanup -d "Cleanup old backups"
complete -c db-backup -f -n "__db_backup_using_command retention" -a info -d "Show policy information"

# Retention flags
complete -c db-backup -n "__db_backup_using_command retention" -l name -d "Policy name"
complete -c db-backup -n "__db_backup_using_command retention" -l database -s d -d "Database" -a "(__db_backup_databases)"
complete -c db-backup -n "__db_backup_using_command retention" -l policy -d "Policy" -a "(__db_backup_retention_policies)"
complete -c db-backup -n "__db_backup_using_command retention" -l keep-daily -d "Keep daily backups"
complete -c db-backup -n "__db_backup_using_command retention" -l keep-weekly -d "Keep weekly backups"
complete -c db-backup -n "__db_backup_using_command retention" -l keep-monthly -d "Keep monthly backups"
complete -c db-backup -n "__db_backup_using_command retention" -l keep-yearly -d "Keep yearly backups"
complete -c db-backup -n "__db_backup_using_command retention" -l min-backups -d "Minimum backups"
complete -c db-backup -n "__db_backup_using_command retention" -l max-backups -d "Maximum backups"
complete -c db-backup -n "__db_backup_using_command retention" -l dry-run -d "Dry run mode"
complete -c db-backup -n "__db_backup_using_command retention" -l force -d "Force operation"

# ============================================================================
# Monitoring Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command monitoring" -a status -d "Show monitoring status"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a alerts -d "Show alerts"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a metrics -d "Show metrics"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a dashboard -d "Open dashboard"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a logs -d "Show logs"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a enable -d "Enable monitoring"
complete -c db-backup -f -n "__db_backup_using_command monitoring" -a disable -d "Disable monitoring"

# Monitoring flags
complete -c db-backup -n "__db_backup_using_command monitoring" -l alert -d "Alert ID" -a "(__db_backup_alerts)"
complete -c db-backup -n "__db_backup_using_command monitoring" -l status -d "Filter by status" -a "active resolved acknowledged"
complete -c db-backup -n "__db_backup_using_command monitoring" -l severity -d "Filter by severity" -a "low medium high critical"
complete -c db-backup -n "__db_backup_using_command monitoring" -l from -d "Start time"
complete -c db-backup -n "__db_backup_using_command monitoring" -l to -d "End time"
complete -c db-backup -n "__db_backup_using_command monitoring" -l refresh -d "Auto-refresh seconds"
complete -c db-backup -n "__db_backup_using_command monitoring" -l watch -d "Watch mode"

# ============================================================================
# Compliance Subcommands
# ============================================================================

complete -c db-backup -f -n "__db_backup_using_command compliance" -a scan -d "Scan for compliance"
complete -c db-backup -f -n "__db_backup_using_command compliance" -a report -d "Generate compliance report"
complete -c db-backup -f -n "__db_backup_using_command compliance" -a frameworks -d "List frameworks"
complete -c db-backup -f -n "__db_backup_using_command compliance" -a violations -d "List violations"
complete -c db-backup -f -n "__db_backup_using_command compliance" -a export -d "Export compliance data"
complete -c db-backup -f -n "__db_backup_using_command compliance" -a audit -d "Audit trail"

# Compliance flags
complete -c db-backup -n "__db_backup_using_command compliance" -l framework -s f -d "Framework" -a "(__db_backup_compliance_frameworks)"
complete -c db-backup -n "__db_backup_using_command compliance" -l database -s d -d "Database" -a "(__db_backup_databases)"
complete -c db-backup -n "__db_backup_using_command compliance" -l report -d "Generate report"
complete -c db-backup -n "__db_backup_using_command compliance" -l fix -d "Fix violations"
complete -c db-backup -n "__db_backup_using_command compliance" -l from -d "Start time"
complete -c db-backup -n "__db_backup_using_command compliance" -l to -d "End time"

# ============================================================================
# Common Flags (for all commands)
# ============================================================================

complete -c db-backup -l format -d "Output format" -a "json yaml table csv id name"
complete -c db-backup -l output -s o -d "Output file" -r
complete -c db-backup -l config -d "Config file" -r
complete -c db-backup -l log-level -d "Log level" -a "debug info warn error"
complete -c db-backup -l help -s h -d "Show help"
complete -c db-backup -l version -s v -d "Show version"
complete -c db-backup -l verbose -d "Verbose output"
complete -c db-backup -l quiet -s q -d "Quiet mode"
complete -c db-backup -l json -d "JSON output"
complete -c db-backup -l no-color -d "Disable colors"

# ============================================================================
# Aliases
# ============================================================================

# Also complete for common aliases
complete -c dbb -w db-backup
complete -c dbbackup -w db-backup
