# DB Backup - Pulumi Infrastructure as Code

This directory contains Pulumi programs for deploying DB Backup infrastructure across multiple cloud providers.

## Directory Structure

```
pulumi/
├── aws/              # AWS-specific infrastructure
├── azure/            # Azure-specific infrastructure
├── gcp/              # GCP-specific infrastructure
└── multi-cloud/      # Multi-cloud deployment
```

## Prerequisites

1. Install Pulumi CLI:
   ```bash
   curl -fsSL https://get.pulumi.com | sh
   ```

2. Install Node.js dependencies:
   ```bash
   cd <cloud-provider-directory>
   npm install
   ```

3. Configure cloud provider credentials:
   - AWS: `aws configure`
   - Azure: `az login`
   - GCP: `gcloud auth login`

## AWS Deployment

Deploy infrastructure to AWS:

```bash
cd aws
pulumi login
pulumi stack init dev
pulumi config set aws:region us-east-1
pulumi up
```

### Configuration Options

- `aws:region`: AWS region (default: us-east-1)
- `db-backup-aws:clusterSize`: Number of EKS worker nodes (default: 3)
- `db-backup-aws:instanceType`: EC2 instance type (default: t3.medium)

## Multi-Cloud Deployment

Deploy to multiple cloud providers simultaneously:

```bash
cd multi-cloud
pulumi login
pulumi stack init production

# Configure primary cloud
pulumi config set db-backup-multi-cloud:primaryCloud aws

# Enable multi-cloud
pulumi config set db-backup-multi-cloud:enableMultiCloud true

# Configure each provider
pulumi config set db-backup-multi-cloud:awsRegion us-east-1
pulumi config set db-backup-multi-cloud:azureLocation eastus
pulumi config set db-backup-multi-cloud:gcpProject my-project-id
pulumi config set db-backup-multi-cloud:gcpRegion us-central1

# Deploy
pulumi up
```

### Multi-Cloud Architecture

When `enableMultiCloud` is true, the stack will deploy:

1. **AWS**: S3 bucket with versioning and lifecycle policies
2. **Azure**: Storage Account with blob container and GRS replication
3. **GCP**: Cloud Storage bucket with multi-regional storage

Primary cloud provider is always deployed, others are optional.

## Outputs

After deployment, you'll receive outputs including:

- Cluster endpoints and credentials
- Storage bucket names and endpoints
- Network configuration details
- Multi-cloud routing information

## Managing Stacks

### List stacks
```bash
pulumi stack ls
```

### Switch stacks
```bash
pulumi stack select <stack-name>
```

### View outputs
```bash
pulumi stack output
```

### Destroy infrastructure
```bash
pulumi destroy
```

## Best Practices

1. **Use separate stacks for environments** (dev, staging, production)
2. **Store sensitive configuration as secrets**:
   ```bash
   pulumi config set --secret dbPassword mySecretPassword
   ```
3. **Review changes before applying**:
   ```bash
   pulumi preview
   ```
4. **Tag resources appropriately** - All resources are automatically tagged with project, stack, and managed-by labels

## Troubleshooting

### Authentication Issues

If you encounter authentication errors:

```bash
# AWS
aws sts get-caller-identity

# Azure
az account show

# GCP
gcloud auth list
```

### State Management

Pulumi state is stored in Pulumi Cloud by default. To use a different backend:

```bash
pulumi login s3://my-pulumi-state-bucket
# or
pulumi login azblob://my-container
# or
pulumi login gs://my-pulumi-bucket
```

## Cost Optimization

- Use lifecycle policies to move old backups to cheaper storage tiers
- Enable auto-scaling for compute resources
- Review and right-size instance types
- Use spot/preemptible instances for non-critical workloads

## Security

- All storage buckets have public access blocked
- Encryption at rest is enabled by default
- Network policies restrict inbound traffic
- IAM roles follow principle of least privilege

## Support

For issues or questions:
- Check the [Pulumi documentation](https://www.pulumi.com/docs/)
- Review cloud provider documentation
- Open an issue in the project repository
