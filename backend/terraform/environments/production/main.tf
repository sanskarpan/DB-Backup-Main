# Production Environment - Complete Infrastructure
# This configuration deploys a full production-ready database backup infrastructure

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }

  backend "s3" {
    bucket         = "db-backup-terraform-state"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}

# Provider Configuration
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "db-backup"
      Environment = "production"
      ManagedBy   = "Terraform"
      Owner       = var.owner
      CostCenter  = var.cost_center
    }
  }
}

# Data Sources
data "aws_availability_zones" "available" {
  state = "available"
}

# KMS Key for Encryption
resource "aws_kms_key" "main" {
  description             = "KMS key for db-backup production encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "db-backup-production"
  }
}

resource "aws_kms_alias" "main" {
  name          = "alias/db-backup-production"
  target_key_id = aws_kms_key.main.key_id
}

# VPC Module
module "vpc" {
  source = "../../modules/aws/vpc"

  vpc_name           = "${var.project_name}-${var.environment}"
  vpc_cidr           = var.vpc_cidr
  availability_zones = slice(data.aws_availability_zones.available.names, 0, 3)
  cluster_name       = "${var.project_name}-${var.environment}"
  aws_region         = var.aws_region

  enable_nat_gateway      = true
  single_nat_gateway      = false # High availability
  enable_database_subnets = true
  enable_s3_endpoint      = true
  enable_ecr_endpoint     = true
  enable_flow_logs        = true

  tags = {
    Name        = "${var.project_name}-${var.environment}-vpc"
    Environment = var.environment
  }
}

# EKS Cluster Module
module "eks" {
  source = "../../modules/aws/eks"

  cluster_name       = "${var.project_name}-${var.environment}"
  kubernetes_version = var.kubernetes_version
  vpc_id             = module.vpc.vpc_id
  subnet_ids         = module.vpc.private_subnet_ids
  kms_key_arn        = aws_kms_key.main.arn

  endpoint_private_access = true
  endpoint_public_access  = true
  public_access_cidrs     = var.allowed_cidr_blocks

  cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  node_groups = {
    general = {
      desired_size    = 3
      max_size        = 10
      min_size        = 3
      instance_types  = ["t3.xlarge"]
      capacity_type   = "ON_DEMAND"
      disk_size       = 100
      max_unavailable = 1
      labels = {
        role = "general"
      }
      taints = []
      tags   = {}
    }
    backup = {
      desired_size    = 2
      max_size        = 8
      min_size        = 2
      instance_types  = ["r6i.2xlarge"] # Memory optimized for database operations
      capacity_type   = "ON_DEMAND"
      disk_size       = 200
      max_unavailable = 1
      labels = {
        role = "backup"
      }
      taints = [
        {
          key    = "workload"
          value  = "backup"
          effect = "NoSchedule"
        }
      ]
      tags = {
        Workload = "backup"
      }
    }
  }

  enable_ebs_csi_driver = true
  enable_efs_csi_driver = true

  tags = {
    Name        = "${var.project_name}-${var.environment}-eks"
    Environment = var.environment
  }

  depends_on = [module.vpc]
}

# S3 Backup Bucket - Primary Region
module "s3_backup_primary" {
  source = "../../modules/aws/s3"

  bucket_name = "${var.project_name}-backups-${var.environment}-${var.aws_region}"
  environment = var.environment
  kms_key_arn = aws_kms_key.main.arn

  versioning_enabled              = true
  enable_lifecycle_rules          = true
  transition_to_glacier_days      = var.glacier_transition_days
  transition_to_deep_archive_days = var.deep_archive_transition_days
  backup_retention_days           = var.backup_retention_days

  enable_logging   = true
  enable_inventory = true

  # Enable replication for disaster recovery
  enable_replication        = var.enable_cross_region_replication
  replication_bucket_arn    = var.enable_cross_region_replication ? module.s3_backup_dr[0].bucket_arn : null
  replication_storage_class = "STANDARD_IA"
  replication_kms_key_arn   = var.enable_cross_region_replication ? aws_kms_key.dr[0].arn : null

  tags = {
    Name        = "${var.project_name}-backups-primary"
    Region      = "primary"
    Environment = var.environment
  }
}

# RDS Database for Metadata
resource "aws_db_instance" "metadata" {
  identifier     = "${var.project_name}-metadata-${var.environment}"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.r6i.xlarge"

  allocated_storage     = 100
  max_allocated_storage = 1000
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.main.arn

  db_name  = "backupmetadata"
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = module.vpc.database_subnet_group_name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  backup_retention_period = 30
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  performance_insights_enabled    = true
  performance_insights_kms_key_id = aws_kms_key.main.arn

  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "${var.project_name}-metadata-final-${formatdate("YYYY-MM-DD-hhmm", timestamp())}"

  tags = {
    Name        = "${var.project_name}-metadata"
    Environment = var.environment
  }
}

# RDS Security Group
resource "aws_security_group" "rds" {
  name_prefix = "${var.project_name}-rds-"
  description = "Security group for RDS metadata database"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [module.eks.cluster_security_group_id]
    description     = "PostgreSQL from EKS"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }

  tags = {
    Name        = "${var.project_name}-rds-sg"
    Environment = var.environment
  }
}

# EFS for Shared Storage
resource "aws_efs_file_system" "backup_storage" {
  creation_token = "${var.project_name}-backup-storage"
  encrypted      = true
  kms_key_id     = aws_kms_key.main.arn

  lifecycle_policy {
    transition_to_ia = "AFTER_30_DAYS"
  }

  lifecycle_policy {
    transition_to_primary_storage_class = "AFTER_1_ACCESS"
  }

  tags = {
    Name        = "${var.project_name}-backup-storage"
    Environment = var.environment
  }
}

resource "aws_efs_mount_target" "backup_storage" {
  count = length(module.vpc.private_subnet_ids)

  file_system_id  = aws_efs_file_system.backup_storage.id
  subnet_id       = module.vpc.private_subnet_ids[count.index]
  security_groups = [aws_security_group.efs.id]
}

resource "aws_security_group" "efs" {
  name_prefix = "${var.project_name}-efs-"
  description = "Security group for EFS"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port       = 2049
    to_port         = 2049
    protocol        = "tcp"
    security_groups = [module.eks.cluster_security_group_id]
    description     = "NFS from EKS"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }

  tags = {
    Name        = "${var.project_name}-efs-sg"
    Environment = var.environment
  }
}

# Kubernetes Provider Configuration
provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
  token                  = module.eks.cluster_token
}

provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
    token                  = module.eks.cluster_token
  }
}

# Deploy Application via Helm
resource "helm_release" "db_backup" {
  name       = "db-backup"
  namespace  = "db-backup"
  chart      = "../../helm/db-backup"
  version    = var.app_version
  wait       = true
  timeout    = 600

  create_namespace = true

  values = [
    templatefile("${path.module}/values.yaml", {
      image_tag                = var.app_version
      replicas                 = 3
      s3_bucket                = module.s3_backup_primary.bucket_id
      rds_endpoint             = aws_db_instance.metadata.endpoint
      rds_database             = aws_db_instance.metadata.db_name
      efs_filesystem_id        = aws_efs_file_system.backup_storage.id
      aws_region               = var.aws_region
      environment              = var.environment
      enable_istio             = var.enable_istio
      enable_autoscaling       = true
      min_replicas             = 3
      max_replicas             = 20
      monitoring_enabled       = true
      backup_node_selector_key = "role"
      backup_node_selector_val = "backup"
    })
  ]

  depends_on = [
    module.eks,
    aws_db_instance.metadata,
    aws_efs_mount_target.backup_storage
  ]
}

# CloudWatch Log Group for Application Logs
resource "aws_cloudwatch_log_group" "app" {
  name              = "/aws/eks/${module.eks.cluster_name}/db-backup"
  retention_in_days = var.log_retention_days
  kms_key_id        = aws_kms_key.main.arn

  tags = {
    Name        = "${var.project_name}-app-logs"
    Environment = var.environment
  }
}

# Disaster Recovery Resources (Conditional)
resource "aws_kms_key" "dr" {
  count = var.enable_cross_region_replication ? 1 : 0

  provider = aws.dr_region

  description             = "KMS key for db-backup DR encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name        = "db-backup-dr"
    Environment = var.environment
    Region      = "dr"
  }
}

resource "aws_kms_alias" "dr" {
  count = var.enable_cross_region_replication ? 1 : 0

  provider = aws.dr_region

  name          = "alias/db-backup-dr"
  target_key_id = aws_kms_key.dr[0].key_id
}

module "s3_backup_dr" {
  count = var.enable_cross_region_replication ? 1 : 0

  source = "../../modules/aws/s3"

  providers = {
    aws = aws.dr_region
  }

  bucket_name = "${var.project_name}-backups-${var.environment}-${var.dr_region}"
  environment = "${var.environment}-dr"
  kms_key_arn = aws_kms_key.dr[0].arn

  versioning_enabled              = true
  enable_lifecycle_rules          = true
  transition_to_glacier_days      = var.glacier_transition_days
  transition_to_deep_archive_days = var.deep_archive_transition_days
  backup_retention_days           = var.backup_retention_days

  enable_logging   = true
  enable_inventory = true

  # No replication back to primary (unidirectional)
  enable_replication = false

  tags = {
    Name        = "${var.project_name}-backups-dr"
    Region      = "dr"
    Environment = var.environment
  }
}

# DR Region Provider
provider "aws" {
  alias  = "dr_region"
  region = var.dr_region

  default_tags {
    tags = {
      Project     = "db-backup"
      Environment = "production-dr"
      ManagedBy   = "Terraform"
      Owner       = var.owner
      CostCenter  = var.cost_center
      Region      = "dr"
    }
  }
}
