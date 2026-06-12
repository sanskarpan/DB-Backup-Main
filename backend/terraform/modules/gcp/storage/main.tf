terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

# Cloud Storage Bucket
resource "google_storage_bucket" "main" {
  name     = var.bucket_name
  location = var.location
  project  = var.project_id

  storage_class = var.storage_class
  force_destroy = var.force_destroy

  uniform_bucket_level_access = true

  versioning {
    enabled = var.enable_versioning
  }

  dynamic "lifecycle_rule" {
    for_each = var.lifecycle_rules
    content {
      action {
        type          = lifecycle_rule.value.action.type
        storage_class = lookup(lifecycle_rule.value.action, "storage_class", null)
      }

      condition {
        age                        = lookup(lifecycle_rule.value.condition, "age", null)
        created_before             = lookup(lifecycle_rule.value.condition, "created_before", null)
        with_state                 = lookup(lifecycle_rule.value.condition, "with_state", null)
        matches_storage_class      = lookup(lifecycle_rule.value.condition, "matches_storage_class", null)
        num_newer_versions         = lookup(lifecycle_rule.value.condition, "num_newer_versions", null)
        days_since_noncurrent_time = lookup(lifecycle_rule.value.condition, "days_since_noncurrent_time", null)
      }
    }
  }

  encryption {
    default_kms_key_name = var.kms_key_name
  }

  logging {
    log_bucket        = var.logging_bucket
    log_object_prefix = var.logging_prefix
  }

  labels = merge(
    var.labels,
    {
      "managed-by" = "terraform"
      "project"    = "db-backup"
    }
  )
}

# Backup Folder (prefix)
resource "google_storage_bucket_object" "backups_folder" {
  name    = "backups/"
  content = " "
  bucket  = google_storage_bucket.main.name
}

# Logs Folder
resource "google_storage_bucket_object" "logs_folder" {
  name    = "logs/"
  content = " "
  bucket  = google_storage_bucket.main.name
}

# IAM binding for service account
resource "google_storage_bucket_iam_member" "admin" {
  count  = var.admin_service_account != "" ? 1 : 0
  bucket = google_storage_bucket.main.name
  role   = "roles/storage.admin"
  member = "serviceAccount:${var.admin_service_account}"
}

resource "google_storage_bucket_iam_member" "viewer" {
  for_each = toset(var.viewer_members)
  bucket   = google_storage_bucket.main.name
  role     = "roles/storage.objectViewer"
  member   = each.value
}
