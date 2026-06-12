# AWS S3 Module Variables

variable "bucket_name" {
  description = "Name of the S3 bucket"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "force_destroy" {
  description = "Allow bucket to be destroyed even if it contains objects"
  type        = bool
  default     = false
}

variable "versioning_enabled" {
  description = "Enable versioning for the bucket"
  type        = bool
  default     = true
}

variable "kms_key_arn" {
  description = "ARN of KMS key for encryption (null for AES256)"
  type        = string
  default     = null
}

variable "enable_lifecycle_rules" {
  description = "Enable lifecycle rules"
  type        = bool
  default     = true
}

variable "transition_to_glacier_days" {
  description = "Days until transition to Glacier"
  type        = number
  default     = 90
}

variable "transition_to_deep_archive_days" {
  description = "Days until transition to Deep Archive"
  type        = number
  default     = 180
}

variable "backup_retention_days" {
  description = "Days to retain backups (0 = never delete)"
  type        = number
  default     = 0
}

variable "noncurrent_version_expiration_days" {
  description = "Days to retain noncurrent versions"
  type        = number
  default     = 30
}

variable "enable_logging" {
  description = "Enable S3 access logging"
  type        = bool
  default     = true
}

variable "logging_bucket" {
  description = "Bucket name for access logs (null to create new)"
  type        = string
  default     = null
}

variable "enable_replication" {
  description = "Enable cross-region replication"
  type        = bool
  default     = false
}

variable "replication_bucket_arn" {
  description = "ARN of destination bucket for replication"
  type        = string
  default     = null
}

variable "replication_storage_class" {
  description = "Storage class for replicated objects"
  type        = string
  default     = "STANDARD"
}

variable "replication_kms_key_arn" {
  description = "KMS key ARN for replication encryption"
  type        = string
  default     = null
}

variable "enable_inventory" {
  description = "Enable S3 inventory"
  type        = bool
  default     = false
}

variable "enable_intelligent_tiering" {
  description = "Enable intelligent tiering"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
