terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

# AKS Cluster
resource "azurerm_kubernetes_cluster" "main" {
  name                = var.cluster_name
  location            = var.location
  resource_group_name = var.resource_group_name
  dns_prefix          = var.dns_prefix
  kubernetes_version  = var.kubernetes_version

  automatic_channel_upgrade = var.automatic_channel_upgrade
  sku_tier                  = var.sku_tier

  default_node_pool {
    name                = "system"
    node_count          = var.system_node_count
    vm_size             = var.system_node_vm_size
    vnet_subnet_id      = var.vnet_subnet_id
    type                = "VirtualMachineScaleSets"
    enable_auto_scaling = var.enable_auto_scaling
    min_count           = var.enable_auto_scaling ? var.min_node_count : null
    max_count           = var.enable_auto_scaling ? var.max_node_count : null
    os_disk_size_gb     = var.os_disk_size_gb
    zones               = var.availability_zones

    upgrade_settings {
      max_surge = "33%"
    }

    tags = merge(
      var.tags,
      {
        "ManagedBy" = "Terraform"
        "Project"   = "db-backup"
        "NodePool"  = "system"
      }
    )
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin     = "azure"
    network_policy     = "azure"
    dns_service_ip     = var.dns_service_ip
    service_cidr       = var.service_cidr
    load_balancer_sku  = "standard"
    outbound_type      = "loadBalancer"
  }

  azure_active_directory_role_based_access_control {
    managed            = true
    azure_rbac_enabled = true
  }

  dynamic "oms_agent" {
    for_each = var.enable_log_analytics ? [1] : []
    content {
      log_analytics_workspace_id = var.log_analytics_workspace_id
    }
  }

  dynamic "key_vault_secrets_provider" {
    for_each = var.enable_key_vault_secrets_provider ? [1] : []
    content {
      secret_rotation_enabled = true
    }
  }

  dynamic "microsoft_defender" {
    for_each = var.enable_microsoft_defender ? [1] : []
    content {
      log_analytics_workspace_id = var.log_analytics_workspace_id
    }
  }

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}

# User Node Pool for Database Backup workloads
resource "azurerm_kubernetes_cluster_node_pool" "backup" {
  count                 = var.create_backup_node_pool ? 1 : 0
  name                  = "backup"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.main.id
  vm_size               = var.backup_node_vm_size
  node_count            = var.backup_node_count
  vnet_subnet_id        = var.vnet_subnet_id
  enable_auto_scaling   = var.enable_auto_scaling
  min_count             = var.enable_auto_scaling ? var.backup_min_node_count : null
  max_count             = var.enable_auto_scaling ? var.backup_max_node_count : null
  os_disk_size_gb       = var.os_disk_size_gb
  zones                 = var.availability_zones

  node_labels = {
    "workload" = "backup"
    "role"     = "backup-worker"
  }

  node_taints = var.backup_node_taints

  upgrade_settings {
    max_surge = "33%"
  }

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
      "NodePool"  = "backup"
    }
  )
}

# Role assignment for AKS to manage network
resource "azurerm_role_assignment" "aks_network" {
  scope                = var.vnet_id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_kubernetes_cluster.main.identity[0].principal_id
}

# Role assignment for AKS to pull images from ACR
resource "azurerm_role_assignment" "aks_acr" {
  count                = var.acr_id != "" ? 1 : 0
  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
}
