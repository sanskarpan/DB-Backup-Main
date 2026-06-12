terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

# Storage Account
resource "azurerm_storage_account" "main" {
  name                     = var.storage_account_name
  resource_group_name      = var.resource_group_name
  location                 = var.location
  account_tier             = var.account_tier
  account_replication_type = var.replication_type
  account_kind             = var.account_kind
  access_tier              = var.access_tier

  enable_https_traffic_only       = true
  min_tls_version                 = "TLS1_2"
  allow_nested_items_to_be_public = false
  shared_access_key_enabled       = true

  blob_properties {
    versioning_enabled       = var.enable_versioning
    change_feed_enabled      = var.enable_change_feed
    last_access_time_enabled = true

    dynamic "delete_retention_policy" {
      for_each = var.soft_delete_retention_days > 0 ? [1] : []
      content {
        days = var.soft_delete_retention_days
      }
    }

    dynamic "container_delete_retention_policy" {
      for_each = var.container_soft_delete_retention_days > 0 ? [1] : []
      content {
        days = var.container_soft_delete_retention_days
      }
    }
  }

  network_rules {
    default_action             = var.default_network_rule
    bypass                     = ["AzureServices"]
    ip_rules                   = var.allowed_ip_ranges
    virtual_network_subnet_ids = var.allowed_subnet_ids
  }

  identity {
    type = "SystemAssigned"
  }

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}

# Blob Container for backups
resource "azurerm_storage_container" "backups" {
  name                  = var.backup_container_name
  storage_account_name  = azurerm_storage_account.main.name
  container_access_type = "private"
}

# Blob Container for logs
resource "azurerm_storage_container" "logs" {
  name                  = var.logs_container_name
  storage_account_name  = azurerm_storage_account.main.name
  container_access_type = "private"
}

# Lifecycle Management Policy
resource "azurerm_storage_management_policy" "main" {
  count              = var.enable_lifecycle_management ? 1 : 0
  storage_account_id = azurerm_storage_account.main.id

  rule {
    name    = "backup-lifecycle"
    enabled = true

    filters {
      prefix_match = ["backups/"]
      blob_types   = ["blockBlob"]
    }

    actions {
      base_blob {
        tier_to_cool_after_days_since_modification_greater_than    = var.tier_to_cool_after_days
        tier_to_archive_after_days_since_modification_greater_than = var.tier_to_archive_after_days
        delete_after_days_since_modification_greater_than          = var.delete_after_days
      }

      snapshot {
        delete_after_days_since_creation_greater_than = var.delete_snapshots_after_days
      }
    }
  }

  rule {
    name    = "logs-lifecycle"
    enabled = true

    filters {
      prefix_match = ["logs/"]
      blob_types   = ["blockBlob"]
    }

    actions {
      base_blob {
        delete_after_days_since_modification_greater_than = var.delete_logs_after_days
      }
    }
  }
}

# Private Endpoint for Blob Storage
resource "azurerm_private_endpoint" "blob" {
  count               = var.enable_private_endpoint ? 1 : 0
  name                = "${var.storage_account_name}-blob-pe"
  location            = var.location
  resource_group_name = var.resource_group_name
  subnet_id           = var.private_endpoint_subnet_id

  private_service_connection {
    name                           = "${var.storage_account_name}-blob-psc"
    private_connection_resource_id = azurerm_storage_account.main.id
    is_manual_connection           = false
    subresource_names              = ["blob"]
  }

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}

# Private DNS Zone for Blob Storage
resource "azurerm_private_dns_zone" "blob" {
  count               = var.enable_private_endpoint ? 1 : 0
  name                = "privatelink.blob.core.windows.net"
  resource_group_name = var.resource_group_name

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}

# Link Private DNS Zone to VNet
resource "azurerm_private_dns_zone_virtual_network_link" "blob" {
  count                 = var.enable_private_endpoint ? 1 : 0
  name                  = "${var.storage_account_name}-blob-dns-link"
  resource_group_name   = var.resource_group_name
  private_dns_zone_name = azurerm_private_dns_zone.blob[0].name
  virtual_network_id    = var.vnet_id

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}

# Private DNS Zone Group
resource "azurerm_private_dns_zone_group" "blob" {
  count               = var.enable_private_endpoint ? 1 : 0
  name                = "${var.storage_account_name}-blob-dns-group"
  private_endpoint_id = azurerm_private_endpoint.blob[0].id

  private_dns_zone_ids = [
    azurerm_private_dns_zone.blob[0].id
  ]
}
