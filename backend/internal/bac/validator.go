package bac

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ValidationResult represents the result of a validation.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Validator validates backup configurations.
type Validator struct {
	strictMode bool
}

// NewValidator creates a new configuration validator.
func NewValidator(strictMode bool) *Validator {
	return &Validator{
		strictMode: strictMode,
	}
}

// Validate validates a backup configuration.
func (v *Validator) Validate(config *BackupConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]string, 0),
	}

	// Validate API version
	v.validateAPIVersion(config, result)

	// Validate metadata
	v.validateMetadata(&config.Metadata, result)

	// Validate spec
	v.validateSpec(&config.Spec, result)

	// Set overall validity
	result.Valid = len(result.Errors) == 0

	return result
}

// validateAPIVersion validates the API version.
func (v *Validator) validateAPIVersion(config *BackupConfig, result *ValidationResult) {
	if config.APIVersion == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "apiVersion",
			Message: "API version is required",
			Code:    "REQUIRED_FIELD",
		})
		return
	}

	// Support v1, v1alpha1, v1beta1, etc.
	versionPattern := regexp.MustCompile(`^v\d+(alpha\d+|beta\d+)?$`)
	if !versionPattern.MatchString(config.APIVersion) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "apiVersion",
			Message: fmt.Sprintf("invalid API version format: %s", config.APIVersion),
			Code:    "INVALID_FORMAT",
		})
	}
}

// validateMetadata validates configuration metadata.
func (v *Validator) validateMetadata(metadata *ConfigMetadata, result *ValidationResult) {
	if metadata.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "metadata.name",
			Message: "name is required",
			Code:    "REQUIRED_FIELD",
		})
		return
	}

	// Name must be valid DNS subdomain
	namePattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !namePattern.MatchString(metadata.Name) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "metadata.name",
			Message: "name must consist of lowercase alphanumeric characters or '-'",
			Code:    "INVALID_FORMAT",
		})
	}

	// Validate version format
	if metadata.Version != "" {
		versionPattern := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
		if !versionPattern.MatchString(metadata.Version) {
			result.Warnings = append(result.Warnings, "metadata.version should follow semantic versioning (e.g., 1.0.0)")
		}
	}
}

// validateSpec validates the backup specification.
func (v *Validator) validateSpec(spec *BackupSpec, result *ValidationResult) {
	// Validate databases
	if len(spec.Databases) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "spec.databases",
			Message: "at least one database must be configured",
			Code:    "REQUIRED_FIELD",
		})
	} else {
		v.validateDatabases(spec.Databases, result)
	}

	// Validate schedules
	if len(spec.Schedules) == 0 {
		result.Warnings = append(result.Warnings, "no backup schedules configured")
	} else {
		v.validateSchedules(spec.Schedules, spec.Databases, result)
	}

	// Validate policies
	v.validatePolicies(spec.Policies, result)

	// Validate retention
	v.validateRetention(&spec.Retention, result)

	// Validate storage
	v.validateStorage(&spec.Storage, result)

	// Validate compliance
	if spec.Compliance.Standards != nil {
		v.validateCompliance(&spec.Compliance, result)
	}

	// Validate GitOps
	if spec.GitOps != nil && spec.GitOps.Enabled {
		v.validateGitOps(spec.GitOps, result)
	}
}

// validateDatabases validates database configurations.
func (v *Validator) validateDatabases(databases []DatabaseConfig, result *ValidationResult) {
	dbIDs := make(map[string]bool)

	for i := range databases {
		db := &databases[i]
		prefix := fmt.Sprintf("spec.databases[%d]", i)

		// Check for duplicate IDs
		if dbIDs[db.ID] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".id",
				Message: fmt.Sprintf("duplicate database ID: %s", db.ID),
				Code:    "DUPLICATE_ID",
			})
		}
		dbIDs[db.ID] = true

		// Validate required fields
		if db.ID == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".id",
				Message: "database ID is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		if db.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".name",
				Message: "database name is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		if db.Type == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".type",
				Message: "database type is required",
				Code:    "REQUIRED_FIELD",
			})
		} else {
			// Validate supported database types
			validTypes := []string{"postgres", "mysql", "mongodb", "mariadb", "oracle", "sqlserver", "cassandra", "redis"}
			if !contains(validTypes, db.Type) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s.type: %s may not be fully supported", prefix, db.Type))
			}
		}

		// Validate connection
		v.validateConnection(&db.Connection, prefix+".connection", result)
	}
}

// validateConnection validates connection configuration.
func (v *Validator) validateConnection(conn *ConnectionConfig, prefix string, result *ValidationResult) {
	if conn.Host == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".host",
			Message: "host is required",
			Code:    "REQUIRED_FIELD",
		})
	}

	if conn.Port <= 0 || conn.Port > 65535 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".port",
			Message: "port must be between 1 and 65535",
			Code:    "INVALID_VALUE",
		})
	}

	if conn.Username == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".username",
			Message: "username is required",
			Code:    "REQUIRED_FIELD",
		})
	}

	if conn.PasswordRef == "" {
		result.Warnings = append(result.Warnings, prefix+".passwordRef: password reference is recommended for security")
	}
}

// validateSchedules validates schedule configurations.
func (v *Validator) validateSchedules(schedules []ScheduleConfig, databases []DatabaseConfig, result *ValidationResult) {
	dbIDs := make(map[string]bool)
	for i := range databases {
		dbIDs[databases[i].ID] = true
	}

	for i, schedule := range schedules {
		prefix := fmt.Sprintf("spec.schedules[%d]", i)

		if schedule.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".name",
				Message: "schedule name is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		// Validate type
		validTypes := []string{"full", "incremental", "differential", "synthetic"}
		if !contains(validTypes, schedule.Type) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".type",
				Message: fmt.Sprintf("invalid schedule type: %s (valid: %s)", schedule.Type, strings.Join(validTypes, ", ")),
				Code:    "INVALID_VALUE",
			})
		}

		// Validate cron expression
		if schedule.CronExpression == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".cron",
				Message: "cron expression is required",
				Code:    "REQUIRED_FIELD",
			})
		} else if !v.isValidCronExpression(schedule.CronExpression) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".cron",
				Message: fmt.Sprintf("invalid cron expression: %s", schedule.CronExpression),
				Code:    "INVALID_FORMAT",
			})
		}

		// Validate database references
		for _, dbID := range schedule.Databases {
			if dbID != "*" && !dbIDs[dbID] {
				result.Errors = append(result.Errors, ValidationError{
					Field:   prefix + ".databases",
					Message: fmt.Sprintf("database ID not found: %s", dbID),
					Code:    "INVALID_REFERENCE",
				})
			}
		}

		// Validate window
		if schedule.Window != nil {
			v.validateWindow(schedule.Window, prefix+".window", result)
		}
	}
}

// validateWindow validates a time window.
func (v *Validator) validateWindow(window *Window, prefix string, result *ValidationResult) {
	timePattern := regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

	if !timePattern.MatchString(window.Start) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".start",
			Message: "start time must be in HH:MM format",
			Code:    "INVALID_FORMAT",
		})
	}

	if !timePattern.MatchString(window.End) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".end",
			Message: "end time must be in HH:MM format",
			Code:    "INVALID_FORMAT",
		})
	}

	// Validate timezone
	if window.Timezone != "" {
		_, err := time.LoadLocation(window.Timezone)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".timezone",
				Message: fmt.Sprintf("invalid timezone: %s", window.Timezone),
				Code:    "INVALID_VALUE",
			})
		}
	}
}

// validatePolicies validates policy configurations.
func (v *Validator) validatePolicies(policies []PolicyConfig, result *ValidationResult) {
	for i := range policies {
		policy := &policies[i]
		prefix := fmt.Sprintf("spec.policies[%d]", i)

		if policy.Name == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".name",
				Message: "policy name is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		// Validate compression
		if policy.Compression != "" {
			validCompression := []string{"gzip", "zstd", "lz4", "bzip2", "xz"}
			if !contains(validCompression, policy.Compression) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s.compression: %s may not be supported", prefix, policy.Compression))
			}
		}

		// Validate encryption
		if policy.Encryption != nil && policy.Encryption.Enabled {
			v.validateEncryption(policy.Encryption, prefix+".encryption", result)
		}

		// Validate timeout
		if policy.Timeout != "" {
			_, err := time.ParseDuration(policy.Timeout)
			if err != nil {
				result.Errors = append(result.Errors, ValidationError{
					Field:   prefix + ".timeout",
					Message: fmt.Sprintf("invalid timeout duration: %s", policy.Timeout),
					Code:    "INVALID_FORMAT",
				})
			}
		}

		// Validate retry policy
		if policy.RetryPolicy != nil {
			v.validateRetryPolicy(policy.RetryPolicy, prefix+".retryPolicy", result)
		}

		// Validate notifications
		for j, notif := range policy.Notifications {
			v.validateNotification(&notif, fmt.Sprintf("%s.notifications[%d]", prefix, j), result)
		}

		// Validate synthetic full config
		if policy.SyntheticFull != nil && policy.SyntheticFull.Enabled {
			v.validateSyntheticFull(policy.SyntheticFull, prefix+".syntheticFull", result)
		}
	}
}

// validateEncryption validates encryption configuration.
func (v *Validator) validateEncryption(enc *EncryptionConfig, prefix string, result *ValidationResult) {
	if enc.Algorithm == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".algorithm",
			Message: "encryption algorithm is required when encryption is enabled",
			Code:    "REQUIRED_FIELD",
		})
	} else {
		validAlgorithms := []string{"aes-256-gcm", "aes-256-cbc", "chacha20-poly1305"}
		if !contains(validAlgorithms, enc.Algorithm) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s.algorithm: %s may not be supported", prefix, enc.Algorithm))
		}
	}

	if enc.KeyRef == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".keyRef",
			Message: "encryption key reference is required when encryption is enabled",
			Code:    "REQUIRED_FIELD",
		})
	}
}

// validateRetryPolicy validates retry policy.
func (v *Validator) validateRetryPolicy(retry *RetryPolicy, prefix string, result *ValidationResult) {
	if retry.MaxAttempts <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".maxAttempts",
			Message: "maxAttempts must be greater than 0",
			Code:    "INVALID_VALUE",
		})
	}

	if retry.Delay != "" {
		_, err := time.ParseDuration(retry.Delay)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".delay",
				Message: fmt.Sprintf("invalid delay duration: %s", retry.Delay),
				Code:    "INVALID_FORMAT",
			})
		}
	}

	validBackoffTypes := []string{"linear", "exponential", "constant"}
	if retry.BackoffType != "" && !contains(validBackoffTypes, retry.BackoffType) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".backoffType",
			Message: fmt.Sprintf("invalid backoff type: %s (valid: %s)", retry.BackoffType, strings.Join(validBackoffTypes, ", ")),
			Code:    "INVALID_VALUE",
		})
	}
}

// validateNotification validates notification configuration.
func (v *Validator) validateNotification(notif *NotificationConfig, prefix string, result *ValidationResult) {
	validTypes := []string{"email", "slack", "webhook", "pagerduty", "teams", "discord"}
	if !contains(validTypes, notif.Type) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s.type: %s may not be supported", prefix, notif.Type))
	}

	if notif.Endpoint == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".endpoint",
			Message: "notification endpoint is required",
			Code:    "REQUIRED_FIELD",
		})
	} else if notif.Type == "webhook" || notif.Type == "slack" {
		// Validate URL format for webhook types
		_, err := url.ParseRequestURI(notif.Endpoint)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".endpoint",
				Message: "invalid URL format",
				Code:    "INVALID_FORMAT",
			})
		}
	}

	validEvents := []string{"success", "failure", "warning", "started", "completed"}
	for _, event := range notif.Events {
		if !contains(validEvents, event) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s.events: %s is not a standard event type", prefix, event))
		}
	}
}

// validateSyntheticFull validates synthetic full backup configuration.
func (v *Validator) validateSyntheticFull(synth *SyntheticFullConfig, prefix string, result *ValidationResult) {
	if synth.MaxChainLength <= 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".maxChainLength",
			Message: "maxChainLength must be greater than 0",
			Code:    "INVALID_VALUE",
		})
	}

	if synth.MaxChainAge != "" {
		_, err := time.ParseDuration(synth.MaxChainAge)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".maxChainAge",
				Message: fmt.Sprintf("invalid duration: %s", synth.MaxChainAge),
				Code:    "INVALID_FORMAT",
			})
		}
	}

	if synth.Schedule != "" {
		if !v.isValidCronExpression(synth.Schedule) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".schedule",
				Message: fmt.Sprintf("invalid cron expression: %s", synth.Schedule),
				Code:    "INVALID_FORMAT",
			})
		}
	}
}

// validateRetention validates retention configuration.
func (v *Validator) validateRetention(retention *RetentionConfig, result *ValidationResult) {
	prefix := "spec.retention"

	if retention.Daily < 0 || retention.Weekly < 0 || retention.Monthly < 0 || retention.Yearly < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix,
			Message: "retention periods cannot be negative",
			Code:    "INVALID_VALUE",
		})
	}

	if retention.Daily == 0 && retention.Weekly == 0 && retention.Monthly == 0 && retention.Yearly == 0 {
		result.Warnings = append(result.Warnings, prefix+": no retention periods configured, backups may accumulate indefinitely")
	}

	if retention.MinimumBackups < 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".minimumBackups",
			Message: "minimumBackups cannot be negative",
			Code:    "INVALID_VALUE",
		})
	}
}

// validateStorage validates storage configuration.
func (v *Validator) validateStorage(storage *StorageConfig, result *ValidationResult) {
	prefix := "spec.storage"

	validTypes := []string{"s3", "gcs", "azure", "local", "nfs", "minio"}
	if !contains(validTypes, storage.Type) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s.type: %s may not be supported", prefix, storage.Type))
	}

	if storage.Location == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".location",
			Message: "storage location is required",
			Code:    "REQUIRED_FIELD",
		})
	}

	// Validate redundancy
	if storage.Redundancy != nil && storage.Redundancy.Enabled {
		if len(storage.Redundancy.Destinations) == 0 {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".redundancy.destinations",
				Message: "at least one redundancy destination is required when redundancy is enabled",
				Code:    "REQUIRED_FIELD",
			})
		}
	}
}

// validateCompliance validates compliance configuration.
func (v *Validator) validateCompliance(compliance *ComplianceConfig, result *ValidationResult) {
	prefix := "spec.compliance"

	validStandards := []string{"GDPR", "HIPAA", "SOC2", "ISO27001", "PCI-DSS", "CCPA"}
	for _, standard := range compliance.Standards {
		if !contains(validStandards, standard) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s.standards: %s compliance validation may not be fully implemented", prefix, standard))
		}
	}

	// Validate immutability
	if compliance.Immutability != nil && compliance.Immutability.Enabled {
		if compliance.Immutability.LockPeriod == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".immutability.lockPeriod",
				Message: "lock period is required when immutability is enabled",
				Code:    "REQUIRED_FIELD",
			})
		} else {
			_, err := time.ParseDuration(compliance.Immutability.LockPeriod)
			if err != nil {
				result.Errors = append(result.Errors, ValidationError{
					Field:   prefix + ".immutability.lockPeriod",
					Message: fmt.Sprintf("invalid duration: %s", compliance.Immutability.LockPeriod),
					Code:    "INVALID_FORMAT",
				})
			}
		}
	}

	// Validate audit
	if compliance.Audit != nil && compliance.Audit.Enabled {
		validLogLevels := []string{"info", "verbose", "debug"}
		if !contains(validLogLevels, compliance.Audit.LogLevel) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".audit.logLevel",
				Message: fmt.Sprintf("invalid log level: %s (valid: %s)", compliance.Audit.LogLevel, strings.Join(validLogLevels, ", ")),
				Code:    "INVALID_VALUE",
			})
		}
	}

	// Validate data masking
	for i, rule := range compliance.DataMasking {
		rulePrefix := fmt.Sprintf("%s.dataMasking[%d]", prefix, i)

		if rule.Table == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   rulePrefix + ".table",
				Message: "table name is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		if len(rule.Columns) == 0 {
			result.Errors = append(result.Errors, ValidationError{
				Field:   rulePrefix + ".columns",
				Message: "at least one column is required",
				Code:    "REQUIRED_FIELD",
			})
		}

		validMethods := []string{"hash", "redact", "random", "preserve-format", "encrypt"}
		if !contains(validMethods, rule.Method) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   rulePrefix + ".method",
				Message: fmt.Sprintf("invalid masking method: %s (valid: %s)", rule.Method, strings.Join(validMethods, ", ")),
				Code:    "INVALID_VALUE",
			})
		}
	}
}

// validateGitOps validates GitOps configuration.
func (v *Validator) validateGitOps(gitops *GitOpsConfig, result *ValidationResult) {
	prefix := "spec.gitOps"

	if gitops.Repository == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".repository",
			Message: "repository URL is required when GitOps is enabled",
			Code:    "REQUIRED_FIELD",
		})
	} else if !strings.HasPrefix(gitops.Repository, "http://") &&
		!strings.HasPrefix(gitops.Repository, "https://") &&
		!strings.HasPrefix(gitops.Repository, "git@") {
		// Validate repository URL format
		result.Warnings = append(result.Warnings, prefix+".repository: URL format may not be recognized")
	}

	if gitops.Branch == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".branch",
			Message: "branch is required when GitOps is enabled",
			Code:    "REQUIRED_FIELD",
		})
	}

	if gitops.SyncInterval != "" {
		_, err := time.ParseDuration(gitops.SyncInterval)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".syncInterval",
				Message: fmt.Sprintf("invalid duration: %s", gitops.SyncInterval),
				Code:    "INVALID_FORMAT",
			})
		}
	}
}

// isValidCronExpression validates a cron expression.
func (v *Validator) isValidCronExpression(expr string) bool {
	// Simple validation: cron should have 5 or 6 fields
	fields := strings.Fields(expr)
	if len(fields) < 5 || len(fields) > 6 {
		return false
	}

	// Additional validation could be added here
	return true
}

// contains checks if a slice contains a value.
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// ValidateScheduleConflicts checks for conflicting schedules.
func (v *Validator) ValidateScheduleConflicts(schedules []ScheduleConfig) []string {
	conflicts := make([]string, 0)

	// Group schedules by database
	dbSchedules := make(map[string][]ScheduleConfig)
	for _, schedule := range schedules {
		for _, dbID := range schedule.Databases {
			dbSchedules[dbID] = append(dbSchedules[dbID], schedule)
		}
	}

	// Check for overlapping windows
	for dbID, scheds := range dbSchedules {
		for i := 0; i < len(scheds); i++ {
			for j := i + 1; j < len(scheds); j++ {
				if v.schedulesOverlap(&scheds[i], &scheds[j]) {
					conflicts = append(conflicts, fmt.Sprintf("Schedules '%s' and '%s' may overlap for database '%s'",
						scheds[i].Name, scheds[j].Name, dbID))
				}
			}
		}
	}

	return conflicts
}

// schedulesOverlap checks if two schedules might overlap.
func (v *Validator) schedulesOverlap(s1, s2 *ScheduleConfig) bool {
	// If both have windows, check for time overlap
	if s1.Window != nil && s2.Window != nil {
		return v.windowsOverlap(s1.Window, s2.Window)
	}

	// Without detailed cron parsing, we can't determine overlap
	return false
}

// windowsOverlap checks if two time windows overlap.
func (v *Validator) windowsOverlap(w1, w2 *Window) bool {
	// Parse times
	start1, err1 := parseTimeWindow(w1.Start)
	end1, err2 := parseTimeWindow(w1.End)
	start2, err3 := parseTimeWindow(w2.Start)
	end2, err4 := parseTimeWindow(w2.End)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return false
	}

	// Check for overlap: start1 < end2 && start2 < end1
	return start1 < end2 && start2 < end1
}

// parseTimeWindow parses a time string (HH:MM) to minutes since midnight.
func parseTimeWindow(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format")
	}

	var hours, minutes int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hours, &minutes)
	if err != nil {
		return 0, err
	}

	return hours*60 + minutes, nil
}
