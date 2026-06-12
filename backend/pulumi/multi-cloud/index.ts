import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as azure from "@pulumi/azure-native";
import * as gcp from "@pulumi/gcp";

// Configuration
const config = new pulumi.Config();
const projectName = pulumi.getProject();
const stackName = pulumi.getStack();
const primaryCloud = config.get("primaryCloud") || "aws";
const enableMultiCloud = config.getBoolean("enableMultiCloud") || false;

// Tags/Labels for all resources
const commonTags = {
    Project: projectName,
    Stack: stackName,
    ManagedBy: "Pulumi",
    Environment: stackName,
};

// ==================== AWS Resources ====================
const awsProvider = new aws.Provider("aws-provider", {
    region: config.get("awsRegion") || "us-east-1",
});

const awsBackupBucket = new aws.s3.Bucket("aws-backup-bucket", {
    bucket: `${projectName}-${stackName}-aws-backups`,
    acl: "private",
    versioning: { enabled: true },
    serverSideEncryptionConfiguration: {
        rule: {
            applyServerSideEncryptionByDefault: {
                sseAlgorithm: "AES256",
            },
        },
    },
    tags: commonTags,
}, { provider: awsProvider });

// Block public access for AWS bucket
const awsBlockPublicAccess = new aws.s3.BucketPublicAccessBlock("aws-backup-bucket-block", {
    bucket: awsBackupBucket.id,
    blockPublicAcls: true,
    blockPublicPolicy: true,
    ignorePublicAcls: true,
    restrictPublicBuckets: true,
}, { provider: awsProvider });

// ==================== Azure Resources ====================
const azureProvider = new azure.Provider("azure-provider", {
    location: config.get("azureLocation") || "eastus",
});

const azureResourceGroup = new azure.resources.ResourceGroup("backup-rg", {
    resourceGroupName: `${projectName}-${stackName}-rg`,
    location: config.get("azureLocation") || "eastus",
    tags: commonTags,
}, { provider: azureProvider, ignoreChanges: enableMultiCloud ? [] : ["*"] });

const azureStorageAccount = new azure.storage.StorageAccount("backupstorage", {
    accountName: `${projectName}${stackName}storage`.replace(/-/g, "").toLowerCase().substring(0, 24),
    resourceGroupName: azureResourceGroup.name,
    location: azureResourceGroup.location,
    sku: {
        name: "Standard_GRS",
    },
    kind: "StorageV2",
    enableHttpsTrafficOnly: true,
    minimumTlsVersion: "TLS1_2",
    tags: commonTags,
}, { provider: azureProvider, dependsOn: [azureResourceGroup], ignoreChanges: enableMultiCloud ? [] : ["*"] });

const azureContainer = new azure.storage.BlobContainer("backups-container", {
    containerName: "backups",
    accountName: azureStorageAccount.name,
    resourceGroupName: azureResourceGroup.name,
    publicAccess: "None",
}, { provider: azureProvider, dependsOn: [azureStorageAccount], ignoreChanges: enableMultiCloud ? [] : ["*"] });

// ==================== GCP Resources ====================
const gcpProvider = new gcp.Provider("gcp-provider", {
    project: config.get("gcpProject"),
    region: config.get("gcpRegion") || "us-central1",
});

const gcpBackupBucket = new gcp.storage.Bucket("gcp-backup-bucket", {
    name: `${projectName}-${stackName}-gcp-backups`,
    location: config.get("gcpRegion") || "US",
    storageClass: "STANDARD",
    uniformBucketLevelAccess: true,
    versioning: {
        enabled: true,
    },
    lifecycleRules: [
        {
            action: {
                type: "SetStorageClass",
                storageClass: "NEARLINE",
            },
            condition: {
                age: 30,
            },
        },
        {
            action: {
                type: "SetStorageClass",
                storageClass: "COLDLINE",
            },
            condition: {
                age: 90,
            },
        },
        {
            action: {
                type: "Delete",
            },
            condition: {
                age: 365,
            },
        },
    ],
    labels: commonTags,
}, { provider: gcpProvider, ignoreChanges: enableMultiCloud ? [] : ["*"] });

// ==================== Multi-Cloud Routing Configuration ====================
// This would typically involve a service like AWS Global Accelerator,
// Azure Traffic Manager, or a third-party solution like Cloudflare

// Export outputs
export const primaryCloudProvider = primaryCloud;
export const multiCloudEnabled = enableMultiCloud;

// AWS Outputs
export const awsBackupBucketName = awsBackupBucket.id;
export const awsBackupBucketArn = awsBackupBucket.arn;
export const awsBackupBucketRegion = config.get("awsRegion") || "us-east-1";

// Azure Outputs
export const azureResourceGroupName = enableMultiCloud ? azureResourceGroup.name : pulumi.Output.create("not-enabled");
export const azureStorageAccountName = enableMultiCloud ? azureStorageAccount.name : pulumi.Output.create("not-enabled");
export const azureContainerName = enableMultiCloud ? azureContainer.name : pulumi.Output.create("not-enabled");

// GCP Outputs
export const gcpBackupBucketName = enableMultiCloud ? gcpBackupBucket.name : pulumi.Output.create("not-enabled");
export const gcpBackupBucketUrl = enableMultiCloud ? gcpBackupBucket.url : pulumi.Output.create("not-enabled");

// Connection strings and endpoints
export const endpoints = pulumi.all([
    awsBackupBucket.bucketRegionalDomainName,
    enableMultiCloud ? azureStorageAccount.primaryBlobEndpoint : pulumi.Output.create(""),
    enableMultiCloud ? gcpBackupBucket.url : pulumi.Output.create(""),
]).apply(([aws, azure, gcp]) => ({
    aws: aws,
    azure: enableMultiCloud ? azure : "not-enabled",
    gcp: enableMultiCloud ? gcp : "not-enabled",
}));
