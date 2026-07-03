package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bp;bpolicy
// +kubebuilder:printcolumn:name="Database",type=string,JSONPath=`.spec.databaseType`
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Last Backup",type=date,JSONPath=`.status.lastBackupTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BackupPolicy defines a backup policy for database backups.
type BackupPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupPolicySpec   `json:"spec,omitempty"`
	Status BackupPolicyStatus `json:"status,omitempty"`
}

// BackupPolicySpec defines the desired state of BackupPolicy.
type BackupPolicySpec struct {
	// Type of database (postgres, mysql, mongodb, sqlite)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=postgres;mysql;mongodb;sqlite
	DatabaseType string `json:"databaseType"`

	// Reference to database credentials
	DatabaseRef *DatabaseReference `json:"databaseRef,omitempty"`

	// Type of backup (full, incremental, differential)
	// +kubebuilder:validation:Enum=full;incremental;differential
	// +kubebuilder:default=full
	BackupType string `json:"backupType,omitempty"`

	// Cron schedule for automated backups
	// +kubebuilder:validation:Pattern=`^(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\d+(ns|us|µs|ms|s|m|h))+)|((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5,7})$`
	Schedule string `json:"schedule,omitempty"`

	// Retention policy for backups
	// +kubebuilder:validation:Required
	Retention RetentionPolicy `json:"retention"`

	// Storage configuration for backups
	Storage *StorageConfig `json:"storage,omitempty"`

	// Compression settings
	Compression *CompressionConfig `json:"compression,omitempty"`

	// Encryption settings
	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// Backup verification settings
	Verification *VerificationConfig `json:"verification,omitempty"`

	// Notification settings
	Notifications *NotificationConfig `json:"notifications,omitempty"`

	// Number of parallel backup jobs
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	Parallelism int `json:"parallelism,omitempty"`

	// Timeout for backup operations in seconds
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=86400
	// +kubebuilder:default=3600
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Suspend scheduled backups
	// +kubebuilder:default=false
	Suspended bool `json:"suspended,omitempty"`

	// Tags to attach to backups
	Tags map[string]string `json:"tags,omitempty"`

	// Name of the secret containing database credentials, injected into the
	// backup job via envFrom.
	DatabaseSecret string `json:"databaseSecret,omitempty"`

	// Container image used to run backup jobs.
	// +kubebuilder:default="db-backup:latest"
	Image string `json:"image,omitempty"`

	// DeletionPolicy controls cleanup behavior when the policy is deleted
	// (Retain or Delete).
	// +kubebuilder:validation:Enum=Retain;Delete
	// +kubebuilder:default=Retain
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// Additional labels applied to backup job pods.
	Labels map[string]string `json:"labels,omitempty"`

	// Resource requirements for backup job containers.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector constrains backup job pods to matching nodes.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations applied to backup job pods.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity rules applied to backup job pods.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// DatabaseReference references database credentials.
type DatabaseReference struct {
	// Name of secret containing database credentials
	SecretName string `json:"secretName"`

	// Namespace of the secret (defaults to policy namespace)
	Namespace string `json:"namespace,omitempty"`
}

// RetentionPolicy defines backup retention.
type RetentionPolicy struct {
	// Number of daily backups to retain
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=365
	Daily int `json:"daily"`

	// Number of weekly backups to retain
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=52
	Weekly int `json:"weekly,omitempty"`

	// Number of monthly backups to retain
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=120
	Monthly int `json:"monthly,omitempty"`

	// Number of yearly backups to retain
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Yearly int `json:"yearly,omitempty"`
}

// StorageConfig defines storage configuration.
type StorageConfig struct {
	// Storage provider (s3, azure, gcp, local)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;azure;gcp;local
	Provider string `json:"provider"`

	// Storage bucket name
	Bucket string `json:"bucket,omitempty"`

	// Storage region
	Region string `json:"region,omitempty"`

	// Base path for backups
	Path string `json:"path,omitempty"`

	// Secret containing storage credentials
	SecretName string `json:"secretName,omitempty"`

	// CredentialsSecret containing storage credentials, injected into the
	// backup job via envFrom.
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

// CompressionConfig defines compression settings.
type CompressionConfig struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Compression algorithm (gzip, zstd, lz4)
	// +kubebuilder:validation:Enum=gzip;zstd;lz4
	// +kubebuilder:default=zstd
	Algorithm string `json:"algorithm,omitempty"`

	// Compression level
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=22
	// +kubebuilder:default=3
	Level int `json:"level,omitempty"`
}

// EncryptionConfig defines encryption settings.
type EncryptionConfig struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Encryption algorithm (aes-256-gcm, chacha20-poly1305)
	// +kubebuilder:validation:Enum=aes-256-gcm;chacha20-poly1305
	// +kubebuilder:default=aes-256-gcm
	Algorithm string `json:"algorithm,omitempty"`

	// Secret containing encryption key
	KeySecretName string `json:"keySecretName,omitempty"`
}

// VerificationConfig defines verification settings.
type VerificationConfig struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Verification method (checksum, restore-test)
	// +kubebuilder:validation:Enum=checksum;restore-test
	// +kubebuilder:default=checksum
	Method string `json:"method,omitempty"`
}

// NotificationConfig defines notification settings.
type NotificationConfig struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Webhook URL for notifications
	WebhookURL string `json:"webhookURL,omitempty"`

	// +kubebuilder:default=false
	OnSuccess bool `json:"onSuccess,omitempty"`

	// +kubebuilder:default=true
	OnFailure bool `json:"onFailure,omitempty"`
}

// BackupPolicyStatus defines the observed state of BackupPolicy.
type BackupPolicyStatus struct {
	// Current phase of the policy
	// +kubebuilder:validation:Enum=Active;Suspended;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions of the policy
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Time of last successful backup
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`

	// Scheduled time for next backup
	NextBackupTime *metav1.Time `json:"nextBackupTime,omitempty"`

	// Total number of backups created
	TotalBackups int `json:"totalBackups,omitempty"`

	// Number of successful backups
	SuccessfulBackups int `json:"successfulBackups,omitempty"`

	// Number of failed backups
	FailedBackups int `json:"failedBackups,omitempty"`

	// Size of last backup in bytes
	LastBackupSize int64 `json:"lastBackupSize,omitempty"`

	// Most recent generation observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true

// BackupPolicyList contains a list of BackupPolicy.
type BackupPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupPolicy `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rj;restore
// +kubebuilder:printcolumn:name="Backup ID",type=string,JSONPath=`.spec.backupID`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Progress",type=integer,JSONPath=`.status.progress`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RestoreJob defines a database restore operation.
type RestoreJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RestoreJobSpec   `json:"spec,omitempty"`
	Status RestoreJobStatus `json:"status,omitempty"`
}

// RestoreJobSpec defines the desired state of RestoreJob.
type RestoreJobSpec struct {
	// ID of backup to restore
	// +kubebuilder:validation:Required
	BackupID string `json:"backupID"`

	// Location of backup file
	BackupLocation *BackupLocation `json:"backupLocation,omitempty"`

	// Target database configuration
	// +kubebuilder:validation:Required
	TargetDatabase TargetDatabase `json:"targetDatabase"`

	// Restore options
	RestoreOptions *RestoreOptions `json:"restoreOptions,omitempty"`

	// Post-restore verification
	Verification *RestoreVerification `json:"verification,omitempty"`

	// Notification settings
	Notifications *NotificationConfig `json:"notifications,omitempty"`

	// Timeout for restore operation in seconds
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=86400
	// +kubebuilder:default=7200
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Retry policy for failed restores
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	// PointInTime restores the database to a specific timestamp (PITR).
	PointInTime *metav1.Time `json:"pointInTime,omitempty"`

	// Container image used to run the restore job.
	// +kubebuilder:default="db-backup:latest"
	Image string `json:"image,omitempty"`

	// Additional labels applied to the restore job pod.
	Labels map[string]string `json:"labels,omitempty"`

	// Resource requirements for the restore job container.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector constrains the restore job pod to matching nodes.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations applied to the restore job pod.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity rules applied to the restore job pod.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// VolumeMounts attached to the restore job container.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// TTLSecondsAfterFinished controls automatic cleanup of the finished job.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// BackupLocation defines backup file location.
type BackupLocation struct {
	// Storage provider (s3, azure, gcp, local)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=s3;azure;gcp;local
	Provider string `json:"provider"`

	// Storage bucket name
	Bucket string `json:"bucket,omitempty"`

	// Storage region
	Region string `json:"region,omitempty"`

	// Path to backup file
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// Secret containing storage credentials
	SecretName string `json:"secretName,omitempty"`

	// CredentialsSecret containing storage credentials, injected into the
	// restore job via envFrom.
	CredentialsSecret string `json:"credentialsSecret,omitempty"`
}

// TargetDatabase defines target database configuration.
type TargetDatabase struct {
	// Database type (postgres, mysql, mongodb, sqlite)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=postgres;mysql;mongodb;sqlite
	Type string `json:"type"`

	// Secret containing target database credentials
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// Namespace of the secret
	Namespace string `json:"namespace,omitempty"`

	// Name of target database (overrides backup)
	DatabaseName string `json:"databaseName,omitempty"`
}

// RestoreOptions defines restore options.
type RestoreOptions struct {
	// Point-in-time for PITR restore
	PointInTime *metav1.Time `json:"pointInTime,omitempty"`

	// Tables to skip during restore
	SkipTables []string `json:"skipTables,omitempty"`

	// Only restore these tables
	IncludeTables []string `json:"includeTables,omitempty"`

	// Schemas to skip during restore
	SkipSchemas []string `json:"skipSchemas,omitempty"`

	// Only restore these schemas
	IncludeSchemas []string `json:"includeSchemas,omitempty"`

	// Drop existing database before restore
	// +kubebuilder:default=false
	DropExisting bool `json:"dropExisting,omitempty"`

	// Skip setting object ownership
	// +kubebuilder:default=false
	NoOwner bool `json:"noOwner,omitempty"`

	// Skip restoring access privileges
	// +kubebuilder:default=false
	NoACL bool `json:"noACL,omitempty"`

	// Restore only data, skip schema
	// +kubebuilder:default=false
	DataOnly bool `json:"dataOnly,omitempty"`

	// Restore only schema, skip data
	// +kubebuilder:default=false
	SchemaOnly bool `json:"schemaOnly,omitempty"`
}

// RestoreVerification defines post-restore verification.
type RestoreVerification struct {
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:default=true
	ValidateChecksum bool `json:"validateChecksum,omitempty"`

	// +kubebuilder:default=false
	ValidateRowCount bool `json:"validateRowCount,omitempty"`

	// Custom SQL query for validation
	CustomQuery string `json:"customQuery,omitempty"`
}

// RetryPolicy defines retry policy.
type RetryPolicy struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	MaxAttempts int `json:"maxAttempts,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=3600
	// +kubebuilder:default=60
	BackoffSeconds int `json:"backoffSeconds,omitempty"`
}

// RestoreJobStatus defines the observed state of RestoreJob.
type RestoreJobStatus struct {
	// Current phase of restore
	// +kubebuilder:validation:Enum=Pending;Downloading;Decompressing;Decrypting;Restoring;Verifying;Completed;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions of the restore job
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Time when restore started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Time when restore completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Duration of restore in seconds
	Duration float64 `json:"duration,omitempty"`

	// Size of backup being restored
	BackupSize int64 `json:"backupSize,omitempty"`

	// Size of restored data
	RestoredSize int64 `json:"restoredSize,omitempty"`

	// Restore progress percentage
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Progress int `json:"progress,omitempty"`

	// Number of restore attempts
	Attempts int `json:"attempts,omitempty"`

	// Last error message
	LastError string `json:"lastError,omitempty"`

	// Result of verification
	VerificationResult *VerificationResult `json:"verificationResult,omitempty"`

	// Most recent generation observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// VerificationResult defines verification result.
type VerificationResult struct {
	Passed        bool   `json:"passed"`
	ChecksumValid bool   `json:"checksumValid"`
	RowCountMatch bool   `json:"rowCountMatch"`
	Message       string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

// RestoreJobList contains a list of RestoreJob.
type RestoreJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestoreJob `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bs;bsched
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Policy",type=string,JSONPath=`.spec.backupPolicyRef.name`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Last Run",type=date,JSONPath=`.status.lastScheduleTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BackupSchedule defines a scheduled backup job.
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupScheduleSpec   `json:"spec,omitempty"`
	Status BackupScheduleStatus `json:"status,omitempty"`
}

// BackupScheduleSpec defines the desired state of BackupSchedule.
type BackupScheduleSpec struct {
	// Cron schedule expression
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(@(annually|yearly|monthly|weekly|daily|hourly|reboot))|(@every (\d+(ns|us|µs|ms|s|m|h))+)|((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5,7})$`
	Schedule string `json:"schedule"`

	// Reference to BackupPolicy
	// +kubebuilder:validation:Required
	BackupPolicyRef BackupPolicyReference `json:"backupPolicyRef"`

	// Timezone for schedule (defaults to UTC)
	// +kubebuilder:default=UTC
	Timezone string `json:"timezone,omitempty"`

	// Suspend scheduled backups
	// +kubebuilder:default=false
	Suspended bool `json:"suspended,omitempty"`

	// How to handle concurrent backups
	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	// +kubebuilder:default=Forbid
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`

	// Number of successful jobs to retain
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=3
	SuccessfulJobsHistoryLimit *int `json:"successfulJobsHistoryLimit,omitempty"`

	// Number of failed jobs to retain
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	FailedJobsHistoryLimit *int `json:"failedJobsHistoryLimit,omitempty"`

	// Deadline for starting job if missed schedule
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=86400
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Template for backup job
	BackupTemplate *BackupTemplate `json:"backupTemplate,omitempty"`

	// Allowed time window for backups
	MaintenanceWindow *MaintenanceWindow `json:"maintenanceWindow,omitempty"`

	// Notification settings
	Notifications *ScheduleNotificationConfig `json:"notifications,omitempty"`

	// CleanupOnDelete removes generated backup jobs when the schedule is deleted.
	// +kubebuilder:default=false
	CleanupOnDelete bool `json:"cleanupOnDelete,omitempty"`
}

// BackupPolicyReference references a BackupPolicy.
type BackupPolicyReference struct {
	// Name of BackupPolicy
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of BackupPolicy
	Namespace string `json:"namespace,omitempty"`
}

// BackupTemplate defines backup job template.
type BackupTemplate struct {
	// Metadata for backup job
	Metadata *BackupTemplateMetadata `json:"metadata,omitempty"`

	// Spec overrides from BackupPolicy
	Spec *BackupTemplateSpec `json:"spec,omitempty"`
}

// BackupTemplateMetadata defines metadata for backup job.
type BackupTemplateMetadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// BackupTemplateSpec defines spec overrides.
type BackupTemplateSpec struct {
	// +kubebuilder:validation:Enum=full;incremental;differential
	BackupType string `json:"backupType,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=86400
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// MaintenanceWindow defines allowed time window.
type MaintenanceWindow struct {
	// Start time (HH:MM format)
	// +kubebuilder:validation:Pattern=`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`
	StartTime string `json:"startTime,omitempty"`

	// End time (HH:MM format)
	// +kubebuilder:validation:Pattern=`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`
	EndTime string `json:"endTime,omitempty"`

	// Days of week (0=Sunday, 6=Saturday)
	// +kubebuilder:validation:items:Minimum=0
	// +kubebuilder:validation:items:Maximum=6
	Days []int `json:"days,omitempty"`
}

// ScheduleNotificationConfig defines schedule notification settings.
type ScheduleNotificationConfig struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	WebhookURL string `json:"webhookURL,omitempty"`

	// +kubebuilder:default=false
	OnSuccess bool `json:"onSuccess,omitempty"`

	// +kubebuilder:default=true
	OnFailure bool `json:"onFailure,omitempty"`

	// +kubebuilder:default=true
	OnMissedSchedule bool `json:"onMissedSchedule,omitempty"`
}

// BackupScheduleStatus defines the observed state.
type BackupScheduleStatus struct {
	// Current phase
	// +kubebuilder:validation:Enum=Active;Suspended;Failed
	Phase string `json:"phase,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last time schedule was triggered
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// Next scheduled time
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// Last successful backup time
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`

	// Currently running backup jobs
	ActiveJobs []ActiveJob `json:"activeJobs,omitempty"`

	// Total number of backup runs
	TotalRuns int `json:"totalRuns,omitempty"`

	// Number of successful runs
	SuccessfulRuns int `json:"successfulRuns,omitempty"`

	// Number of failed runs
	FailedRuns int `json:"failedRuns,omitempty"`

	// Number of missed schedules
	MissedRuns int `json:"missedRuns,omitempty"`

	// Most recent generation observed
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ActiveJob defines an active backup job.
type ActiveJob struct {
	Name      string       `json:"name"`
	Namespace string       `json:"namespace"`
	UID       string       `json:"uid"`
	StartTime *metav1.Time `json:"startTime,omitempty"`
}

// +kubebuilder:object:root=true

// BackupScheduleList contains a list of BackupSchedule.
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupSchedule `json:"items"`
}

func init() {
	// Register types with scheme
}
