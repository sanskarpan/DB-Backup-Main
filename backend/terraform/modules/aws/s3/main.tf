# AWS S3 Module for Backup Storage
# Creates S3 buckets with versioning, encryption, lifecycle policies, and replication

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Primary Backup Bucket
resource "aws_s3_bucket" "backup" {
  bucket        = var.bucket_name
  force_destroy = var.force_destroy

  tags = merge(
    var.tags,
    {
      Name        = var.bucket_name
      Purpose     = "Database Backups"
      Environment = var.environment
    }
  )
}

# Bucket Versioning
resource "aws_s3_bucket_versioning" "backup" {
  bucket = aws_s3_bucket.backup.id

  versioning_configuration {
    status = var.versioning_enabled ? "Enabled" : "Suspended"
  }
}

# Server-Side Encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "backup" {
  bucket = aws_s3_bucket.backup.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = var.kms_key_arn != null ? "aws:kms" : "AES256"
      kms_master_key_id = var.kms_key_arn
    }
    bucket_key_enabled = var.kms_key_arn != null ? true : false
  }
}

# Block Public Access
resource "aws_s3_bucket_public_access_block" "backup" {
  bucket = aws_s3_bucket.backup.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Bucket Lifecycle Configuration
resource "aws_s3_bucket_lifecycle_configuration" "backup" {
  count  = var.enable_lifecycle_rules ? 1 : 0
  bucket = aws_s3_bucket.backup.id

  # Transition to Glacier after retention days
  rule {
    id     = "transition-to-glacier"
    status = "Enabled"

    filter {
      prefix = ""
    }

    transition {
      days          = var.transition_to_glacier_days
      storage_class = "GLACIER"
    }

    transition {
      days          = var.transition_to_deep_archive_days
      storage_class = "DEEP_ARCHIVE"
    }
  }

  # Delete old versions
  rule {
    id     = "expire-old-versions"
    status = var.versioning_enabled ? "Enabled" : "Disabled"

    filter {
      prefix = ""
    }

    noncurrent_version_expiration {
      noncurrent_days = var.noncurrent_version_expiration_days
    }
  }

  # Abort incomplete multipart uploads
  rule {
    id     = "abort-incomplete-multipart-uploads"
    status = "Enabled"

    filter {
      prefix = ""
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  # Delete old backups
  dynamic "rule" {
    for_each = var.backup_retention_days > 0 ? [1] : []
    content {
      id     = "delete-old-backups"
      status = "Enabled"

      filter {
        prefix = ""
      }

      expiration {
        days = var.backup_retention_days
      }
    }
  }
}

# Bucket Logging
resource "aws_s3_bucket_logging" "backup" {
  count  = var.enable_logging ? 1 : 0
  bucket = aws_s3_bucket.backup.id

  target_bucket = var.logging_bucket != null ? var.logging_bucket : aws_s3_bucket.logs[0].id
  target_prefix = "s3-access-logs/${var.bucket_name}/"
}

# Logging Bucket
resource "aws_s3_bucket" "logs" {
  count  = var.enable_logging && var.logging_bucket == null ? 1 : 0
  bucket = "${var.bucket_name}-logs"

  tags = merge(
    var.tags,
    {
      Name    = "${var.bucket_name}-logs"
      Purpose = "S3 Access Logs"
    }
  )
}

resource "aws_s3_bucket_public_access_block" "logs" {
  count  = var.enable_logging && var.logging_bucket == null ? 1 : 0
  bucket = aws_s3_bucket.logs[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Bucket Policy for logging
resource "aws_s3_bucket_policy" "logs" {
  count  = var.enable_logging && var.logging_bucket == null ? 1 : 0
  bucket = aws_s3_bucket.logs[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3ServerAccessLogsPolicy"
        Effect = "Allow"
        Principal = {
          Service = "logging.s3.amazonaws.com"
        }
        Action   = "s3:PutObject"
        Resource = "${aws_s3_bucket.logs[0].arn}/*"
        Condition = {
          StringEquals = {
            "aws:SourceAccount" = data.aws_caller_identity.current.account_id
          }
        }
      }
    ]
  })
}

# Replication Configuration (for disaster recovery)
resource "aws_s3_bucket_replication_configuration" "backup" {
  count = var.enable_replication ? 1 : 0

  # Must have versioning enabled
  depends_on = [aws_s3_bucket_versioning.backup]

  role   = aws_iam_role.replication[0].arn
  bucket = aws_s3_bucket.backup.id

  rule {
    id     = "replicate-all"
    status = "Enabled"

    filter {
      prefix = ""
    }

    destination {
      bucket        = var.replication_bucket_arn
      storage_class = var.replication_storage_class

      encryption_configuration {
        replica_kms_key_id = var.replication_kms_key_arn
      }
    }

    delete_marker_replication {
      status = "Enabled"
    }
  }
}

# Replication IAM Role
resource "aws_iam_role" "replication" {
  count = var.enable_replication ? 1 : 0
  name  = "${var.bucket_name}-replication-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "s3.amazonaws.com"
      }
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "replication" {
  count = var.enable_replication ? 1 : 0
  name  = "${var.bucket_name}-replication-policy"
  role  = aws_iam_role.replication[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "s3:GetReplicationConfiguration",
          "s3:ListBucket"
        ]
        Effect = "Allow"
        Resource = [
          aws_s3_bucket.backup.arn
        ]
      },
      {
        Action = [
          "s3:GetObjectVersionForReplication",
          "s3:GetObjectVersionAcl",
          "s3:GetObjectVersionTagging"
        ]
        Effect = "Allow"
        Resource = [
          "${aws_s3_bucket.backup.arn}/*"
        ]
      },
      {
        Action = [
          "s3:ReplicateObject",
          "s3:ReplicateDelete",
          "s3:ReplicateTags"
        ]
        Effect = "Allow"
        Resource = [
          "${var.replication_bucket_arn}/*"
        ]
      }
    ]
  })
}

# Bucket Metrics
resource "aws_s3_bucket_metric" "backup" {
  bucket = aws_s3_bucket.backup.id
  name   = "EntireBucket"
}

# Bucket Inventory
resource "aws_s3_bucket_inventory" "backup" {
  count  = var.enable_inventory ? 1 : 0
  bucket = aws_s3_bucket.backup.id
  name   = "weekly-inventory"

  included_object_versions = "All"

  schedule {
    frequency = "Weekly"
  }

  destination {
    bucket {
      format     = "Parquet"
      bucket_arn = aws_s3_bucket.backup.arn
      prefix     = "inventory"

      encryption {
        sse_s3 {}
      }
    }
  }

  optional_fields = [
    "Size",
    "LastModifiedDate",
    "StorageClass",
    "ETag",
    "IsMultipartUploaded",
    "ReplicationStatus",
    "EncryptionStatus"
  ]
}

# Intelligent Tiering Configuration
resource "aws_s3_bucket_intelligent_tiering_configuration" "backup" {
  count  = var.enable_intelligent_tiering ? 1 : 0
  bucket = aws_s3_bucket.backup.id
  name   = "entire-bucket"

  tiering {
    access_tier = "ARCHIVE_ACCESS"
    days        = 90
  }

  tiering {
    access_tier = "DEEP_ARCHIVE_ACCESS"
    days        = 180
  }
}

# Data Sources
data "aws_caller_identity" "current" {}
