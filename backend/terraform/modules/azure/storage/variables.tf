variable "storage_account_name" {
  description = "Name of the storage account (must be globally unique)"
  type        = string

  validation {
    condition     = length(var.storage_account_name) >= 3 && length(var.storage_account_name) <= 24
    error_message = "Storage account name must be between 3 and 24 characters."
  }

  validation {
    condition     = can(regex("^[a-z0-9]+$", var.storage_account_name))
    error_message = "Storage account name can only contain lowercase letters and numbers."
  }
}

variable "resource_group_name" {
  description = "Name of the resource group"
  type        = string
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "account_tier" {
  description = "Storage account tier"
  type        = string
  default     = "Standard"

  validation {
    condition     = contains(["Standard", "Premium"], var.account_tier)
    error_message = "Account tier must be Standard or Premium."
  }
}

variable "replication_type" {
  description = "Storage account replication type"
  type        = string
  default     = "GRS"

  validation {
    condition     = contains(["LRS", "GRS", "RAGRS", "ZRS", "GZRS", "RAGZRS"], var.replication_type)
    error_message = "Invalid replication type."
  }
}

variable "account_kind" {
  description = "Storage account kind"
  type        = string
  default     = "StorageV2"
}

variable "access_tier" {
  description = "Access tier for blob storage"
  type        = string
  default     = "Hot"

  validation {
    condition     = contains(["Hot", "Cool"], var.access_tier)
    error_message = "Access tier must be Hot or Cool."
  }
}

variable "enable_versioning" {
  description = "Enable blob versioning"
  type        = bool
  default     = true
}

variable "enable_change_feed" {
  description = "Enable blob change feed"
  type        = bool
  default     = true
}

variable "soft_delete_retention_days" {
  description = "Number of days to retain soft-deleted blobs"
  type        = number
  default     = 30
}

variable "container_soft_delete_retention_days" {
  description = "Number of days to retain soft-deleted containers"
  type        = number
  default     = 30
}

variable "default_network_rule" {
  description = "Default network access rule"
  type        = string
  default     = "Deny"

  validation {
    condition     = contains(["Allow", "Deny"], var.default_network_rule)
    error_message = "Default network rule must be Allow or Deny."
  }
}

variable "allowed_ip_ranges" {
  description = "List of allowed IP ranges"
  type        = list(string)
  default     = []
}

variable "allowed_subnet_ids" {
  description = "List of allowed subnet IDs"
  type        = list(string)
  default     = []
}

variable "backup_container_name" {
  description = "Name of the backups container"
  type        = string
  default     = "backups"
}

variable "logs_container_name" {
  description = "Name of the logs container"
  type        = string
  default     = "logs"
}

variable "enable_lifecycle_management" {
  description = "Enable lifecycle management policies"
  type        = bool
  default     = true
}

variable "tier_to_cool_after_days" {
  description = "Move blobs to cool tier after N days"
  type        = number
  default     = 30
}

variable "tier_to_archive_after_days" {
  description = "Move blobs to archive tier after N days"
  type        = number
  default     = 90
}

variable "delete_after_days" {
  description = "Delete blobs after N days"
  type        = number
  default     = 365
}

variable "delete_snapshots_after_days" {
  description = "Delete snapshots after N days"
  type        = number
  default     = 90
}

variable "delete_logs_after_days" {
  description = "Delete logs after N days"
  type        = number
  default     = 90
}

variable "enable_private_endpoint" {
  description = "Enable private endpoint for storage account"
  type        = bool
  default     = true
}

variable "private_endpoint_subnet_id" {
  description = "Subnet ID for private endpoint"
  type        = string
  default     = ""
}

variable "vnet_id" {
  description = "Virtual network ID for private DNS zone link"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}
