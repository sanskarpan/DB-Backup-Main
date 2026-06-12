# AWS S3 Module Outputs

output "bucket_id" {
  description = "ID of the S3 bucket"
  value       = aws_s3_bucket.backup.id
}

output "bucket_arn" {
  description = "ARN of the S3 bucket"
  value       = aws_s3_bucket.backup.arn
}

output "bucket_domain_name" {
  description = "Domain name of the S3 bucket"
  value       = aws_s3_bucket.backup.bucket_domain_name
}

output "bucket_regional_domain_name" {
  description = "Regional domain name of the S3 bucket"
  value       = aws_s3_bucket.backup.bucket_regional_domain_name
}

output "replication_role_arn" {
  description = "ARN of the replication IAM role"
  value       = var.enable_replication ? aws_iam_role.replication[0].arn : null
}

output "logging_bucket_id" {
  description = "ID of the logging bucket"
  value       = var.enable_logging && var.logging_bucket == null ? aws_s3_bucket.logs[0].id : var.logging_bucket
}
