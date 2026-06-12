# Infrastructure as Code (IaC) Guide

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Terraform Modules](#terraform-modules)
- [Environment Setup](#environment-setup)
- [Deployment Guide](#deployment-guide)
- [Disaster Recovery](#disaster-recovery)
- [Configuration Management](#configuration-management)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

This document describes the complete Infrastructure as Code (IaC) setup for the Database Backup Utility. The infrastructure is managed using:

- **Terraform**: Infrastructure provisioning and management
- **Ansible**: Configuration management and deployment automation
- **Helm**: Kubernetes application packaging and deployment

### Key Features

- Multi-cloud support (AWS, GCP, Azure)
- High availability and auto-scaling
- Disaster recovery with cross-region replication
- Automated deployments with CI/CD integration
- Infrastructure security and compliance
- Cost optimization with intelligent tiering

## Architecture

### Infrastructure Components

```
┌─────────────────────────────────────────────────────────────┐
│                     Production Environment                   │
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌─────────────────┐ │
│  │     VPC      │   │  EKS Cluster │   │   S3 Primary    │ │
│  │              │   │              │   │   (Backups)     │ │
│  │  Public      │───│  Node Groups │───│                 │ │
│  │  Private     │   │  • General   │   │  Replication    │ │
│  │  Database    │   │  • Backup    │   │       ↓         │ │
│  └──────────────┘   └──────────────┘   └─────────────────┘ │
│         │                   │                               │
│         └───────────┬───────┘                               │
│                     │                                       │
│         ┌───────────┴──────────┐                           │
│         │  RDS PostgreSQL      │                           │
│         │  (Metadata)          │                           │
│         │  • Multi-AZ          │                           │
│         │  • Auto Backup       │                           │
│         │  • Encryption        │                           │
│         └──────────────────────┘                           │
└─────────────────────────────────────────────────────────────┘
                              │
                    Cross-Region Replication
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                 Disaster Recovery Environment                │
│                                                              │
│                 ┌─────────────────┐                         │
│                 │   S3 DR Region  │                         │
│                 │   (Backups)     │                         │
│                 │                 │                         │
│                 │  Same retention │                         │
│                 │  Same lifecycle │                         │
│                 └─────────────────┘                         │
└─────────────────────────────────────────────────────────────┘
```

### Network Architecture

- **VPC CIDR**: 10.0.0.0/16
- **Public Subnets**: 10.0.0.0/24, 10.0.1.0/24, 10.0.2.0/24
- **Private Subnets**: 10.0.100.0/24, 10.0.101.0/24, 10.0.102.0/24
- **Database Subnets**: 10.0.200.0/24, 10.0.201.0/24, 10.0.202.0/24
- **NAT Gateways**: One per AZ for high availability
- **VPC Endpoints**: S3, ECR for cost optimization

## Terraform Modules

### Module Structure

```
terraform/
├── modules/
│   ├── aws/
│   │   ├── vpc/          # VPC with subnets, NAT, IGW
│   │   ├── eks/          # EKS cluster with node groups
│   │   ├── s3/           # S3 buckets with lifecycle
│   │   └── rds/          # RDS databases
│   ├── gcp/
│   │   ├── vpc/
│   │   ├── gke/
│   │   └── gcs/
│   ├── azure/
│   │   ├── vnet/
│   │   ├── aks/
│   │   └── storage/
│   └── common/
│       ├── monitoring/
│       └── security/
└── environments/
    ├── development/
    ├── staging/
    └── production/
```

### AWS VPC Module

Creates a production-ready VPC with:
- Public, private, and database subnets across multiple AZs
- NAT Gateways for private subnet internet access
- VPC endpoints for S3 and ECR
- VPC Flow Logs for security monitoring
- Proper tagging for cost allocation

**Usage:**

```hcl
module "vpc" {
  source = "../../modules/aws/vpc"

  vpc_name           = "db-backup-production"
  vpc_cidr           = "10.0.0.0/16"
  availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
  cluster_name       = "db-backup-eks"
  aws_region         = "us-east-1"

  enable_nat_gateway      = true
  single_nat_gateway      = false  # HA setup
  enable_database_subnets = true
  enable_s3_endpoint      = true
  enable_flow_logs        = true

  tags = {
    Environment = "production"
    ManagedBy   = "Terraform"
  }
}
```

### AWS EKS Module

Creates a production-ready EKS cluster with:
- Managed node groups with auto-scaling
- IRSA (IAM Roles for Service Accounts)
- EKS addons (VPC CNI, CoreDNS, kube-proxy)
- CSI drivers for EBS and EFS
- Secrets encryption with KMS
- Comprehensive logging

**Usage:**

```hcl
module "eks" {
  source = "../../modules/aws/eks"

  cluster_name       = "db-backup-production"
  kubernetes_version = "1.28"
  vpc_id             = module.vpc.vpc_id
  subnet_ids         = module.vpc.private_subnet_ids
  kms_key_arn        = aws_kms_key.main.arn

  node_groups = {
    general = {
      desired_size    = 3
      max_size        = 10
      min_size        = 3
      instance_types  = ["t3.xlarge"]
      capacity_type   = "ON_DEMAND"
      disk_size       = 100
      max_unavailable = 1
      labels          = { role = "general" }
      taints          = []
      tags            = {}
    }
    backup = {
      desired_size    = 2
      max_size        = 8
      min_size        = 2
      instance_types  = ["r6i.2xlarge"]
      capacity_type   = "ON_DEMAND"
      disk_size       = 200
      max_unavailable = 1
      labels          = { role = "backup" }
      taints = [{
        key    = "workload"
        value  = "backup"
        effect = "NoSchedule"
      }]
      tags = {}
    }
  }

  enable_ebs_csi_driver = true
  enable_efs_csi_driver = true
}
```

### AWS S3 Module

Creates S3 buckets with:
- Server-side encryption (KMS or AES256)
- Versioning for data protection
- Lifecycle policies (Glacier, Deep Archive)
- Cross-region replication for DR
- Access logging and inventory
- Intelligent tiering (optional)

**Usage:**

```hcl
module "s3_backup" {
  source = "../../modules/aws/s3"

  bucket_name = "db-backup-production-us-east-1"
  environment = "production"
  kms_key_arn = aws_kms_key.main.arn

  versioning_enabled              = true
  transition_to_glacier_days      = 90
  transition_to_deep_archive_days = 180
  backup_retention_days           = 0  # Never delete

  enable_replication        = true
  replication_bucket_arn    = module.s3_backup_dr.bucket_arn
  replication_storage_class = "STANDARD_IA"
}
```

## Environment Setup

### Prerequisites

1. **Tools Installation:**
```bash
# Terraform
brew install terraform

# AWS CLI
brew install awscli

# kubectl
brew install kubectl

# Helm
brew install helm

# Ansible
pip3 install ansible
ansible-galaxy collection install kubernetes.core
```

2. **AWS Credentials:**
```bash
aws configure
# Enter Access Key ID, Secret Access Key, Region
```

3. **Terraform State Backend:**
```bash
# Create S3 bucket for Terraform state
aws s3 mb s3://db-backup-terraform-state --region us-east-1

# Create DynamoDB table for state locking
aws dynamodb create-table \
  --table-name terraform-state-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

### Environment Variables

Create a `.tfvars` file for each environment:

**production.tfvars:**
```hcl
project_name  = "db-backup"
environment   = "production"
aws_region    = "us-east-1"
dr_region     = "us-west-2"
owner         = "platform-team"
cost_center   = "engineering"

vpc_cidr              = "10.0.0.0/16"
kubernetes_version    = "1.28"
app_version          = "v1.0.0"

# Security
allowed_cidr_blocks = ["203.0.113.0/24"]

# Storage
glacier_transition_days      = 90
deep_archive_transition_days = 180
backup_retention_days        = 0

# DR
enable_cross_region_replication = true

# Monitoring
log_retention_days = 90

# Service Mesh
enable_istio = true
```

## Deployment Guide

### Step 1: Initialize Terraform

```bash
cd terraform/environments/production

# Initialize Terraform
terraform init

# Validate configuration
terraform validate

# Review planned changes
terraform plan -var-file=production.tfvars -out=plan.out
```

### Step 2: Deploy Infrastructure

```bash
# Apply changes
terraform apply plan.out

# Save outputs for later use
terraform output > outputs.txt
```

### Step 3: Configure kubectl

```bash
# Update kubeconfig
aws eks update-kubeconfig \
  --region us-east-1 \
  --name db-backup-production

# Verify cluster access
kubectl cluster-info
kubectl get nodes
```

### Step 4: Deploy Application with Ansible

```bash
cd ../../ansible

# Create inventory file
cat > inventory/production.ini <<EOF
[kubernetes]
localhost ansible_connection=local

[kubernetes:vars]
deploy_environment=production
aws_region=us-east-1
eks_cluster_name=db-backup-production
image_repository=ghcr.io/your-org/db-backup
image_tag=v1.0.0
EOF

# Deploy application
ansible-playbook -i inventory/production.ini \
  playbooks/deploy.yml \
  --extra-vars "@vars/production.yml"

# Configure application
ansible-playbook -i inventory/production.ini \
  playbooks/configure.yml \
  --extra-vars "@vars/production.yml"
```

### Step 5: Verify Deployment

```bash
# Check pods
kubectl get pods -n db-backup

# Check services
kubectl get svc -n db-backup

# Check ingress
kubectl get ingress -n db-backup

# View logs
kubectl logs -n db-backup deployment/db-backup
```

## Disaster Recovery

### Architecture

The DR setup provides:
- **Cross-Region Replication**: Automatic S3 replication to DR region
- **RTO**: < 4 hours (Recovery Time Objective)
- **RPO**: < 15 minutes (Recovery Point Objective)

### DR Failover Procedure

#### 1. Assess Primary Region

```bash
# Check primary region status
aws health describe-events --region us-east-1

# Verify latest backup replication
aws s3api list-objects-v2 \
  --bucket db-backup-production-us-west-2 \
  --query 'sort_by(Contents, &LastModified)[-1]'
```

#### 2. Activate DR Environment

```bash
cd terraform/environments/dr

# Initialize DR environment
terraform init

# Deploy DR infrastructure
terraform plan -var-file=dr.tfvars -out=dr.out
terraform apply dr.out

# Update kubeconfig for DR cluster
aws eks update-kubeconfig \
  --region us-west-2 \
  --name db-backup-dr
```

#### 3. Restore Application State

```bash
# Deploy application to DR cluster
ansible-playbook -i inventory/dr.ini \
  playbooks/deploy.yml \
  --extra-vars "s3_bucket=db-backup-production-us-west-2"

# Restore database metadata
./scripts/restore-metadata.sh \
  --from-backup s3://db-backup-production-us-west-2/metadata/latest
```

#### 4. Update DNS

```bash
# Update Route53 to point to DR region
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://dr-dns-change.json
```

#### 5. Verify DR System

```bash
# Run smoke tests
ansible-playbook -i inventory/dr.ini \
  playbooks/smoke-tests.yml

# Test backup operations
kubectl exec -n db-backup deployment/db-backup -- \
  /app/db-backup backup --database test --dry-run
```

### DR Testing Schedule

- **Monthly**: Automated DR drill (automated failover test)
- **Quarterly**: Full DR exercise with team involvement
- **Annual**: Comprehensive DR audit and documentation review

## Configuration Management

### Ansible Playbooks

#### deploy.yml

Handles complete deployment:
- Updates kubeconfig
- Creates Kubernetes secrets and ConfigMaps
- Deploys application via Helm
- Runs smoke tests
- Sends notifications

**Usage:**
```bash
ansible-playbook playbooks/deploy.yml \
  -e "environment=production" \
  -e "image_tag=v1.0.0"
```

#### configure.yml

Post-deployment configuration:
- Sets up CronJobs for scheduled backups
- Configures monitoring alerts
- Applies network policies
- Sets resource quotas
- Configures retention policies

**Usage:**
```bash
ansible-playbook playbooks/configure.yml \
  -e "environment=production"
```

### Variables

Create environment-specific variable files in `ansible/vars/`:

**production.yml:**
```yaml
# Application
image_repository: "ghcr.io/your-org/db-backup"
image_tag: "v1.0.0"
replica_count: 3

# Autoscaling
min_replicas: 3
max_replicas: 20

# Resources
cpu_request: "500m"
memory_request: "512Mi"
cpu_limit: "2000m"
memory_limit: "2Gi"

# Storage
storage_class: "efs-sc"
storage_size: "500Gi"

# Ingress
enable_ingress: true
ingress_host: "db-backup.production.example.com"

# Monitoring
enable_monitoring: true
enable_istio: true

# Database
db_username: "{{ vault_db_username }}"
db_password: "{{ vault_db_password }}"

# Backup Schedules
backup_schedules:
  - name: "postgres-nightly"
    database: "production-postgres"
    type: "full"
    schedule: "0 2 * * *"
    host: "postgres.prod.example.com"
    port: "5432"
  - name: "mysql-nightly"
    database: "production-mysql"
    type: "full"
    schedule: "0 3 * * *"
    host: "mysql.prod.example.com"
    port: "3306"

# Retention
daily_retention: 7
weekly_retention: 4
monthly_retention: 12
yearly_retention: 3
```

## Best Practices

### Infrastructure Management

1. **State Management:**
   - Use remote state (S3) with locking (DynamoDB)
   - Never commit state files to Git
   - Use workspaces for environment isolation

2. **Security:**
   - Enable encryption at rest for all data stores
   - Use KMS for key management
   - Implement least privilege IAM policies
   - Enable VPC Flow Logs and CloudTrail

3. **Cost Optimization:**
   - Use S3 Intelligent-Tiering for unknown access patterns
   - Implement lifecycle policies for old backups
   - Use spot instances for non-critical workloads
   - Enable S3 Transfer Acceleration only when needed

4. **High Availability:**
   - Deploy across multiple AZs
   - Use Multi-AZ for RDS
   - Configure PodDisruptionBudgets
   - Implement health checks and auto-recovery

### Terraform Best Practices

1. **Module Design:**
   - Keep modules focused and reusable
   - Use variables for all configurable values
   - Provide meaningful outputs
   - Document module usage

2. **Code Organization:**
   - Separate modules from environments
   - Use consistent naming conventions
   - Group related resources together
   - Comment complex logic

3. **Change Management:**
   - Always run `terraform plan` before apply
   - Review plans carefully
   - Use `-target` for surgical changes
   - Tag all resources appropriately

### Ansible Best Practices

1. **Playbook Organization:**
   - Use roles for reusable functionality
   - Separate concerns (deploy, configure, test)
   - Use tags for selective execution
   - Implement idempotency

2. **Security:**
   - Use Ansible Vault for secrets
   - Never commit plain-text credentials
   - Use `no_log` for sensitive tasks
   - Implement proper SSH key management

3. **Error Handling:**
   - Check command return codes
   - Use `failed_when` and `changed_when`
   - Implement retries for flaky operations
   - Provide meaningful error messages

## Troubleshooting

### Common Issues

#### Terraform

**Issue**: State lock timeout
```bash
# Solution: Force unlock (use with caution)
terraform force-unlock <LOCK_ID>
```

**Issue**: Provider version conflict
```bash
# Solution: Upgrade providers
terraform init -upgrade
```

**Issue**: Resource already exists
```bash
# Solution: Import existing resource
terraform import module.vpc.aws_vpc.main vpc-12345678
```

#### EKS

**Issue**: Unable to connect to cluster
```bash
# Solution: Update kubeconfig
aws eks update-kubeconfig --name db-backup-production --region us-east-1

# Verify AWS credentials
aws sts get-caller-identity
```

**Issue**: Pods stuck in Pending
```bash
# Check node resources
kubectl describe nodes

# Check pod events
kubectl describe pod <pod-name> -n db-backup

# Check autoscaler logs
kubectl logs -n kube-system deployment/cluster-autoscaler
```

#### S3

**Issue**: Replication not working
```bash
# Check replication status
aws s3api get-bucket-replication --bucket db-backup-production

# Check IAM role permissions
aws iam get-role --role-name replication-role

# Verify destination bucket policy
aws s3api get-bucket-policy --bucket db-backup-dr
```

### Debugging Commands

```bash
# Terraform
terraform show                    # Show current state
terraform state list              # List resources
terraform state show <resource>   # Show resource details
terraform console                 # Interactive console

# kubectl
kubectl get events --sort-by='.lastTimestamp' -n db-backup
kubectl logs -f deployment/db-backup -n db-backup
kubectl exec -it <pod> -n db-backup -- /bin/sh

# AWS
aws logs tail /aws/eks/db-backup-production/cluster --follow
aws cloudformation describe-stacks --region us-east-1
```

## Support

- **Documentation**: https://backup.example.com/docs/iac
- **IaC Issues**: https://github.com/your-org/db-backup/labels/infrastructure
- **Slack**: #db-backup-infrastructure
