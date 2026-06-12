# Production Environment Outputs

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_oidc_issuer_url" {
  description = "EKS OIDC issuer URL"
  value       = module.eks.cluster_oidc_issuer_url
}

output "s3_backup_bucket_primary" {
  description = "Primary S3 backup bucket name"
  value       = module.s3_backup_primary.bucket_id
}

output "s3_backup_bucket_dr" {
  description = "DR S3 backup bucket name"
  value       = var.enable_cross_region_replication ? module.s3_backup_dr[0].bucket_id : null
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = aws_db_instance.metadata.endpoint
  sensitive   = true
}

output "rds_database_name" {
  description = "RDS database name"
  value       = aws_db_instance.metadata.db_name
}

output "efs_id" {
  description = "EFS filesystem ID"
  value       = aws_efs_file_system.backup_storage.id
}

output "kms_key_arn" {
  description = "KMS key ARN"
  value       = aws_kms_key.main.arn
}

output "kms_key_dr_arn" {
  description = "DR KMS key ARN"
  value       = var.enable_cross_region_replication ? aws_kms_key.dr[0].arn : null
}

output "kubeconfig_command" {
  description = "Command to update kubeconfig"
  value       = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

output "app_url" {
  description = "Application URL (after ingress is configured)"
  value       = "https://db-backup.${var.environment}.example.com"
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.app.name
}
