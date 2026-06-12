terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

resource "azurerm_resource_group" "main" {
  name     = var.name
  location = var.location

  tags = merge(
    var.tags,
    {
      "ManagedBy" = "Terraform"
      "Project"   = "db-backup"
    }
  )
}
