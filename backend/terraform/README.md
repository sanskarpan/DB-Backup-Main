# Terraform Infrastructure

This directory contains Terraform modules and environment configurations for deploying the Database Backup Utility infrastructure.

## Quick Start

### 1. Prerequisites

- Terraform >= 1.6.0
- AWS CLI configured with appropriate credentials
- kubectl installed
- Helm >= 3.0

### 2. Initialize Backend

```bash
# Create S3 bucket for Terraform state
aws s3 mb s3://db-backup-terraform-state --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket db-backup-terraform-state \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for state locking
aws dynamodb create-table \
  --table-name terraform-state-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

### 3. Deploy Production Environment

```bash
cd environments/production

# Copy and customize variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Initialize Terraform
terraform init

# Plan deployment
terraform plan -var-file=terraform.tfvars -out=plan.out

# Apply changes
terraform apply plan.out
```

### 4. Access the Cluster

```bash
# Update kubeconfig
aws eks update-kubeconfig \
  --region $(terraform output -raw aws_region) \
  --name $(terraform output -raw eks_cluster_name)

# Verify access
kubectl get nodes
```

## Directory Structure

```
terraform/
├── modules/              # Reusable Terraform modules
│   ├── aws/
│   │   ├── vpc/         # VPC with subnets, NAT, endpoints
│   │   ├── eks/         # EKS cluster with node groups
│   │   ├── s3/          # S3 buckets with lifecycle policies
│   │   └── rds/         # RDS databases (optional)
│   ├── gcp/             # GCP modules (future)
│   ├── azure/           # Azure modules (future)
│   └── common/          # Cross-cloud common modules
└── environments/        # Environment configurations
    ├── development/
    ├── staging/
    └── production/
```

## Modules

### AWS VPC Module

**Location**: `modules/aws/vpc/`

Creates a production-ready VPC with:
- Public, private, and database subnets across 3 AZs
- NAT Gateways for high availability
- VPC endpoints for S3 and ECR
- VPC Flow Logs
- Proper Kubernetes tagging

**Example:**
```hcl
module "vpc" {
  source = "../../modules/aws/vpc"

  vpc_name           = "db-backup-prod"
  vpc_cidr           = "10.0.0.0/16"
  availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
  cluster_name       = "db-backup-eks"
  aws_region         = "us-east-1"
}
```

### AWS EKS Module

**Location**: `modules/aws/eks/`

Creates an EKS cluster with:
- Managed node groups with auto-scaling
- IRSA for pod-level IAM permissions
- EKS addons (VPC CNI, CoreDNS, kube-proxy)
- EBS and EFS CSI drivers
- Secrets encryption with KMS

**Example:**
```hcl
module "eks" {
  source = "../../modules/aws/eks"

  cluster_name       = "db-backup-prod"
  kubernetes_version = "1.28"
  vpc_id             = module.vpc.vpc_id
  subnet_ids         = module.vpc.private_subnet_ids
  kms_key_arn        = aws_kms_key.main.arn

  node_groups = {
    general = {
      desired_size   = 3
      max_size       = 10
      min_size       = 3
      instance_types = ["t3.xlarge"]
      // ... additional configuration
    }
  }
}
```

### AWS S3 Module

**Location**: `modules/aws/s3/`

Creates S3 buckets with:
- Server-side encryption (KMS or AES256)
- Versioning
- Lifecycle policies (Glacier, Deep Archive)
- Cross-region replication for DR
- Access logging

**Example:**
```hcl
module "s3_backup" {
  source = "../../modules/aws/s3"

  bucket_name = "db-backup-prod-us-east-1"
  environment = "production"
  kms_key_arn = aws_kms_key.main.arn

  versioning_enabled         = true
  transition_to_glacier_days = 90
  enable_replication         = true
}
```

## Environments

### Development

Cost-optimized configuration for development:
- Single NAT Gateway
- Smaller instance types
- Reduced node counts
- No cross-region replication

```bash
cd environments/development
terraform init
terraform plan -var-file=development.tfvars
terraform apply
```

### Staging

Pre-production environment:
- Similar to production configuration
- Smaller scale
- Optional cross-region replication

```bash
cd environments/staging
terraform init
terraform plan -var-file=staging.tfvars
terraform apply
```

### Production

Full production environment:
- Multi-AZ deployment
- High availability NAT Gateways
- Cross-region replication
- Production-grade security

```bash
cd environments/production
terraform init
terraform plan -var-file=production.tfvars
terraform apply
```

## State Management

### Remote State

Terraform state is stored in S3 with DynamoDB locking:

```hcl
terraform {
  backend "s3" {
    bucket         = "db-backup-terraform-state"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}
```

### State Commands

```bash
# List resources in state
terraform state list

# Show specific resource
terraform state show module.vpc.aws_vpc.main

# Move resource in state
terraform state mv module.old.resource module.new.resource

# Remove resource from state
terraform state rm module.old.resource
```

## Security Best Practices

1. **Never commit sensitive data:**
   - Use `.tfvars` files for sensitive values
   - Add `*.tfvars` to `.gitignore` (except `.example` files)
   - Use AWS Secrets Manager or Parameter Store

2. **Enable encryption:**
   - Use KMS for all encryption
   - Enable at-rest encryption for all data stores
   - Enable in-transit encryption (TLS/HTTPS)

3. **Least privilege:**
   - Use minimal IAM permissions
   - Implement IRSA for pod-level permissions
   - Regularly audit IAM policies

4. **Network security:**
   - Use private subnets for workloads
   - Implement security groups with minimal rules
   - Enable VPC Flow Logs

## Cost Optimization

1. **Use appropriate instance types:**
   - Development: t3.medium
   - Production: t3.xlarge or larger

2. **Enable S3 Intelligent-Tiering:**
   - Automatic cost optimization
   - No retrieval fees

3. **Use spot instances for non-critical workloads:**
```hcl
node_groups = {
  spot = {
    capacity_type = "SPOT"
    // ...
  }
}
```

4. **Implement lifecycle policies:**
   - Move old backups to Glacier
   - Delete very old backups

## Disaster Recovery

### Cross-Region Replication

Enable S3 cross-region replication:

```hcl
module "s3_backup_primary" {
  source = "../../modules/aws/s3"
  // ...
  enable_replication     = true
  replication_bucket_arn = module.s3_backup_dr.bucket_arn
}
```

### DR Failover

1. Deploy DR infrastructure in secondary region
2. Verify replication is working
3. Update DNS to point to DR region
4. Deploy application to DR cluster

See `docs/INFRASTRUCTURE_AS_CODE.md` for detailed DR procedures.

## Troubleshooting

### Common Issues

**Issue**: State lock timeout
```bash
terraform force-unlock <LOCK_ID>
```

**Issue**: Provider version conflict
```bash
terraform init -upgrade
```

**Issue**: Resource already exists
```bash
terraform import module.vpc.aws_vpc.main vpc-12345678
```

**Issue**: Plan shows many changes
```bash
# Format code
terraform fmt -recursive

# Validate configuration
terraform validate

# Show detailed diff
terraform plan -detailed-exitcode
```

## Documentation

- [Infrastructure as Code Guide](../docs/INFRASTRUCTURE_AS_CODE.md)
- [Deployment Guide](../docs/DEPLOYMENT_GUIDE.md)
- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)

## Support

- GitHub Issues: https://github.com/your-org/db-backup/issues
- Slack: #db-backup-infrastructure
