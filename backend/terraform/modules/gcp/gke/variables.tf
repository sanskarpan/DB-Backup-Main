variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
}

variable "location" {
  description = "GCP region or zone for the cluster"
  type        = string
  default     = "us-central1"
}

variable "node_locations" {
  description = "List of zones for node pools"
  type        = list(string)
  default     = ["us-central1-a", "us-central1-b", "us-central1-c"]
}

variable "network_name" {
  description = "Name of the VPC network"
  type        = string
}

variable "subnet_name" {
  description = "Name of the subnet"
  type        = string
}

variable "master_authorized_networks" {
  description = "List of master authorized networks"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = []
}

variable "enable_private_endpoint" {
  description = "Enable private endpoint for the cluster"
  type        = bool
  default     = false
}

variable "master_ipv4_cidr_block" {
  description = "IPv4 CIDR block for the master"
  type        = string
  default     = "172.16.0.0/28"
}

variable "release_channel" {
  description = "Release channel (RAPID, REGULAR, STABLE)"
  type        = string
  default     = "REGULAR"

  validation {
    condition     = contains(["RAPID", "REGULAR", "STABLE"], var.release_channel)
    error_message = "Release channel must be RAPID, REGULAR, or STABLE."
  }
}

variable "maintenance_start_time" {
  description = "Start time for maintenance window (HH:MM format)"
  type        = string
  default     = "03:00"
}

variable "enable_binary_authorization" {
  description = "Enable binary authorization"
  type        = bool
  default     = false
}

variable "system_node_count" {
  description = "Number of nodes in system pool"
  type        = number
  default     = 3
}

variable "system_min_node_count" {
  description = "Minimum nodes in system pool"
  type        = number
  default     = 2
}

variable "system_max_node_count" {
  description = "Maximum nodes in system pool"
  type        = number
  default     = 10
}

variable "system_machine_type" {
  description = "Machine type for system nodes"
  type        = string
  default     = "n2-standard-4"
}

variable "create_backup_node_pool" {
  description = "Create dedicated backup node pool"
  type        = bool
  default     = true
}

variable "backup_node_count" {
  description = "Number of nodes in backup pool"
  type        = number
  default     = 2
}

variable "backup_min_node_count" {
  description = "Minimum nodes in backup pool"
  type        = number
  default     = 1
}

variable "backup_max_node_count" {
  description = "Maximum nodes in backup pool"
  type        = number
  default     = 5
}

variable "backup_machine_type" {
  description = "Machine type for backup nodes"
  type        = string
  default     = "n2-standard-8"
}

variable "disk_size_gb" {
  description = "Disk size for nodes in GB"
  type        = number
  default     = 100
}

variable "labels" {
  description = "Labels to apply to resources"
  type        = map(string)
  default     = {}
}
