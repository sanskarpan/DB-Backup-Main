package backup.authorization

# Default deny
default allow = false

# Allow backup operations for admin users
allow {
    input.metadata.user_role == "admin"
    input.operation in ["create", "read", "update", "delete"]
}

# Allow backup creation for regular users on their own databases
allow {
    input.metadata.user_role == "user"
    input.operation == "create"
    input.metadata.database_owner == input.user_id
}

# Allow backup read for users who have access to the database
allow {
    input.metadata.user_role == "user"
    input.operation == "read"
    user_has_database_access
}

# Allow backup restore for users with restore permission
allow {
    input.operation == "restore"
    input.metadata.user_permissions[_] == "backup:restore"
}

# Check if user has database access
user_has_database_access {
    input.metadata.database_access_list[_] == input.user_id
}

# Deny if database is marked as restricted
deny {
    input.metadata.database_restricted == true
    input.metadata.user_role != "admin"
}

# Deny if backup size exceeds quota
deny {
    input.metadata.user_quota_gb < input.metadata.backup_size_gb
    input.operation == "create"
}

# Reason for denial
reason = msg {
    not allow
    msg := "Access denied: Insufficient permissions for backup operation"
}

reason = msg {
    deny
    input.metadata.database_restricted == true
    msg := "Access denied: Database is restricted"
}

reason = msg {
    deny
    input.metadata.user_quota_gb < input.metadata.backup_size_gb
    msg := sprintf("Access denied: Backup size %.2f GB exceeds quota of %.2f GB", [
        input.metadata.backup_size_gb,
        input.metadata.user_quota_gb
    ])
}
