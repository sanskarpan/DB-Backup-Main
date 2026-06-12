import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as eks from "@pulumi/eks";

// Configuration
const config = new pulumi.Config();
const projectName = pulumi.getProject();
const stackName = pulumi.getStack();

// VPC Configuration
const vpc = new aws.ec2.Vpc("db-backup-vpc", {
    cidrBlock: "10.0.0.0/16",
    enableDnsHostnames: true,
    enableDnsSupport: true,
    tags: {
        Name: `${projectName}-${stackName}-vpc`,
        Project: projectName,
        ManagedBy: "Pulumi",
    },
});

// Public Subnets
const publicSubnet1 = new aws.ec2.Subnet("public-subnet-1", {
    vpcId: vpc.id,
    cidrBlock: "10.0.1.0/24",
    availabilityZone: "us-east-1a",
    mapPublicIpOnLaunch: true,
    tags: {
        Name: `${projectName}-${stackName}-public-1`,
        "kubernetes.io/role/elb": "1",
    },
});

const publicSubnet2 = new aws.ec2.Subnet("public-subnet-2", {
    vpcId: vpc.id,
    cidrBlock: "10.0.2.0/24",
    availabilityZone: "us-east-1b",
    mapPublicIpOnLaunch: true,
    tags: {
        Name: `${projectName}-${stackName}-public-2`,
        "kubernetes.io/role/elb": "1",
    },
});

// Private Subnets
const privateSubnet1 = new aws.ec2.Subnet("private-subnet-1", {
    vpcId: vpc.id,
    cidrBlock: "10.0.10.0/24",
    availabilityZone: "us-east-1a",
    tags: {
        Name: `${projectName}-${stackName}-private-1`,
        "kubernetes.io/role/internal-elb": "1",
    },
});

const privateSubnet2 = new aws.ec2.Subnet("private-subnet-2", {
    vpcId: vpc.id,
    cidrBlock: "10.0.11.0/24",
    availabilityZone: "us-east-1b",
    tags: {
        Name: `${projectName}-${stackName}-private-2`,
        "kubernetes.io/role/internal-elb": "1",
    },
});

// Internet Gateway
const igw = new aws.ec2.InternetGateway("igw", {
    vpcId: vpc.id,
    tags: {
        Name: `${projectName}-${stackName}-igw`,
    },
});

// Public Route Table
const publicRouteTable = new aws.ec2.RouteTable("public-rt", {
    vpcId: vpc.id,
    routes: [
        {
            cidrBlock: "0.0.0.0/0",
            gatewayId: igw.id,
        },
    ],
    tags: {
        Name: `${projectName}-${stackName}-public-rt`,
    },
});

// Associate public subnets with public route table
const publicRtAssoc1 = new aws.ec2.RouteTableAssociation("public-rt-assoc-1", {
    subnetId: publicSubnet1.id,
    routeTableId: publicRouteTable.id,
});

const publicRtAssoc2 = new aws.ec2.RouteTableAssociation("public-rt-assoc-2", {
    subnetId: publicSubnet2.id,
    routeTableId: publicRouteTable.id,
});

// NAT Gateway
const eip = new aws.ec2.Eip("nat-eip", {
    vpc: true,
    tags: {
        Name: `${projectName}-${stackName}-nat-eip`,
    },
});

const natGateway = new aws.ec2.NatGateway("nat-gateway", {
    allocationId: eip.id,
    subnetId: publicSubnet1.id,
    tags: {
        Name: `${projectName}-${stackName}-nat`,
    },
});

// Private Route Table
const privateRouteTable = new aws.ec2.RouteTable("private-rt", {
    vpcId: vpc.id,
    routes: [
        {
            cidrBlock: "0.0.0.0/0",
            natGatewayId: natGateway.id,
        },
    ],
    tags: {
        Name: `${projectName}-${stackName}-private-rt`,
    },
});

// Associate private subnets with private route table
const privateRtAssoc1 = new aws.ec2.RouteTableAssociation("private-rt-assoc-1", {
    subnetId: privateSubnet1.id,
    routeTableId: privateRouteTable.id,
});

const privateRtAssoc2 = new aws.ec2.RouteTableAssociation("private-rt-assoc-2", {
    subnetId: privateSubnet2.id,
    routeTableId: privateRouteTable.id,
});

// EKS Cluster
const cluster = new eks.Cluster("db-backup-eks", {
    vpcId: vpc.id,
    publicSubnetIds: [publicSubnet1.id, publicSubnet2.id],
    privateSubnetIds: [privateSubnet1.id, privateSubnet2.id],
    instanceType: "t3.medium",
    desiredCapacity: 3,
    minSize: 2,
    maxSize: 10,
    enabledClusterLogTypes: [
        "api",
        "audit",
        "authenticator",
        "controllerManager",
        "scheduler",
    ],
    tags: {
        Name: `${projectName}-${stackName}-eks`,
        Project: projectName,
        ManagedBy: "Pulumi",
    },
});

// S3 Bucket for backups
const backupBucket = new aws.s3.Bucket("backup-bucket", {
    bucket: `${projectName}-${stackName}-backups`,
    acl: "private",
    versioning: {
        enabled: true,
    },
    serverSideEncryptionConfiguration: {
        rule: {
            applyServerSideEncryptionByDefault: {
                sseAlgorithm: "AES256",
            },
        },
    },
    lifecycleRules: [
        {
            enabled: true,
            transitions: [
                {
                    days: 30,
                    storageClass: "STANDARD_IA",
                },
                {
                    days: 90,
                    storageClass: "GLACIER",
                },
            ],
            expiration: {
                days: 365,
            },
        },
    ],
    tags: {
        Name: `${projectName}-${stackName}-backups`,
        Project: projectName,
        ManagedBy: "Pulumi",
    },
});

// Block public access
const blockPublicAccess = new aws.s3.BucketPublicAccessBlock("backup-bucket-block-public", {
    bucket: backupBucket.id,
    blockPublicAcls: true,
    blockPublicPolicy: true,
    ignorePublicAcls: true,
    restrictPublicBuckets: true,
});

// Export outputs
export const vpcId = vpc.id;
export const clusterName = cluster.eksCluster.name;
export const kubeconfig = cluster.kubeconfig;
export const backupBucketName = backupBucket.id;
export const backupBucketArn = backupBucket.arn;
