package security.access

# Default deny all access
default allow = false

# Blocked IP addresses
blocked_ips := ["192.0.2.0", "198.51.100.0", "203.0.113.0"]

# Allowed IP ranges for admin access
admin_ip_ranges := ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]

# Sensitive resources requiring additional checks
sensitive_resources := ["user_data", "encryption_keys", "credentials", "audit_logs"]

# Admin actions
admin_actions := ["delete", "modify_permissions", "access_audit_logs", "modify_policies"]

# Allow access for admin role with proper IP restrictions
allow {
    input.user_role == "admin"
    not ip_is_blocked
    admin_from_safe_network
    not is_sensitive_action_without_mfa
}

# Allow read access for operator role
allow {
    input.user_role == "operator"
    input.action == "read"
    not ip_is_blocked
    not is_sensitive_resource
}

# Allow read/write access for user role on non-sensitive resources
allow {
    input.user_role == "user"
    input.action in ["read", "write"]
    not is_sensitive_resource
    not ip_is_blocked
    has_valid_session
}

# Allow specific actions for service accounts
allow {
    input.user_role == "service_account"
    input.action in ["read", "write", "backup"]
    valid_service_account_token
    not is_sensitive_resource
}

# Check if IP is blocked
ip_is_blocked {
    input.ip_address in blocked_ips
}

# Check if admin is from safe network
admin_from_safe_network {
    input.user_role == "admin"
    ip_in_allowed_range
}

ip_in_allowed_range {
    # In a real implementation, this would check CIDR ranges
    # For now, we'll do a simple prefix check
    startswith(input.ip_address, "10.")
}

ip_in_allowed_range {
    startswith(input.ip_address, "192.168.")
}

ip_in_allowed_range {
    startswith(input.ip_address, "172.")
}

# Check if resource is sensitive
is_sensitive_resource {
    input.resource in sensitive_resources
}

# Check if action requires MFA
is_sensitive_action_without_mfa {
    input.action in admin_actions
    not input.mfa_verified
}

# Check if user has valid session
has_valid_session {
    input.session_valid == true
    input.session_expired == false
}

# Check if service account token is valid
valid_service_account_token {
    input.service_account_token != ""
    input.token_expired == false
}

# Rate limiting check
rate_limit_exceeded {
    input.request_count > max_requests_per_minute
}

max_requests_per_minute = 100

# Time-based access control
within_allowed_hours {
    input.hour >= 8
    input.hour <= 18
}

# Require business hours for sensitive operations
allow {
    input.action in admin_actions
    within_allowed_hours
    input.user_role == "admin"
    input.mfa_verified == true
    not ip_is_blocked
}

# Deny if rate limit is exceeded
deny {
    rate_limit_exceeded
}

# Deny if from blocked country for sensitive data
deny {
    is_sensitive_resource
    input.country_code in ["XX", "YY"] # Placeholder for blocked countries
}

# Reasons for denial
reason = msg {
    not allow
    ip_is_blocked
    msg := sprintf("Access denied: IP address %s is blocked", [input.ip_address])
}

reason = msg {
    not allow
    is_sensitive_resource
    input.user_role != "admin"
    msg := sprintf("Access denied: Resource '%s' requires admin privileges", [input.resource])
}

reason = msg {
    not allow
    is_sensitive_action_without_mfa
    msg := sprintf("Access denied: Action '%s' requires MFA verification", [input.action])
}

reason = msg {
    not allow
    not has_valid_session
    msg := "Access denied: Invalid or expired session"
}

reason = msg {
    not allow
    rate_limit_exceeded
    msg := sprintf("Access denied: Rate limit exceeded (%d requests/min)", [max_requests_per_minute])
}

reason = msg {
    not allow
    input.action in admin_actions
    not within_allowed_hours
    msg := sprintf("Access denied: Admin action '%s' only allowed during business hours (8-18)", [input.action])
}

reason = msg {
    not allow
    input.user_role == "admin"
    not admin_from_safe_network
    msg := "Access denied: Admin access must be from approved network ranges"
}

# Details about the decision
details = {
    "user_role": input.user_role,
    "action": input.action,
    "resource": input.resource,
    "ip_address": input.ip_address,
    "is_sensitive": is_sensitive_resource,
    "mfa_verified": input.mfa_verified,
    "rate_limit_ok": not rate_limit_exceeded,
}
