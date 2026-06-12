# AWS EKS Module Variables

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
}

variable "kubernetes_version" {
  description = "Kubernetes version to use for the EKS cluster"
  type        = string
  default     = "1.28"
}

variable "vpc_id" {
  description = "VPC ID where the cluster will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs for the EKS cluster"
  type        = list(string)
}

variable "endpoint_private_access" {
  description = "Enable private API server endpoint"
  type        = bool
  default     = true
}

variable "endpoint_public_access" {
  description = "Enable public API server endpoint"
  type        = bool
  default     = true
}

variable "public_access_cidrs" {
  description = "List of CIDR blocks that can access the public API server endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "allowed_cidr_blocks" {
  description = "List of CIDR blocks allowed to access the cluster"
  type        = list(string)
  default     = []
}

variable "kms_key_arn" {
  description = "ARN of KMS key for secrets encryption"
  type        = string
}

variable "cluster_log_types" {
  description = "List of control plane logging types to enable"
  type        = list(string)
  default     = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
}

variable "node_groups" {
  description = "Map of node group configurations"
  type = map(object({
    desired_size     = number
    max_size         = number
    min_size         = number
    instance_types   = list(string)
    capacity_type    = string
    disk_size        = number
    max_unavailable  = number
    labels           = map(string)
    taints           = list(object({
      key    = string
      value  = string
      effect = string
    }))
    tags             = map(string)
  }))
  default = {
    general = {
      desired_size    = 2
      max_size        = 10
      min_size        = 2
      instance_types  = ["t3.large"]
      capacity_type   = "ON_DEMAND"
      disk_size       = 50
      max_unavailable = 1
      labels          = {}
      taints          = []
      tags            = {}
    }
  }
}

variable "vpc_cni_version" {
  description = "VPC CNI addon version"
  type        = string
  default     = null
}

variable "kube_proxy_version" {
  description = "Kube-proxy addon version"
  type        = string
  default     = null
}

variable "coredns_version" {
  description = "CoreDNS addon version"
  type        = string
  default     = null
}

variable "enable_ebs_csi_driver" {
  description = "Enable EBS CSI driver addon"
  type        = bool
  default     = true
}

variable "ebs_csi_driver_version" {
  description = "EBS CSI driver addon version"
  type        = string
  default     = null
}

variable "ebs_csi_irsa_role_arn" {
  description = "IAM role ARN for EBS CSI driver IRSA"
  type        = string
  default     = null
}

variable "enable_efs_csi_driver" {
  description = "Enable EFS CSI driver addon"
  type        = bool
  default     = false
}

variable "efs_csi_driver_version" {
  description = "EFS CSI driver addon version"
  type        = string
  default     = null
}

variable "efs_csi_irsa_role_arn" {
  description = "IAM role ARN for EFS CSI driver IRSA"
  type        = string
  default     = null
}

variable "vpc_cni_irsa_role_arn" {
  description = "IAM role ARN for VPC CNI IRSA"
  type        = string
  default     = null
}

variable "additional_roles" {
  description = "Additional IAM roles to add to aws-auth ConfigMap"
  type = list(object({
    rolearn  = string
    username = string
    groups   = list(string)
  }))
  default = []
}

variable "additional_users" {
  description = "Additional IAM users to add to aws-auth ConfigMap"
  type = list(object({
    userarn  = string
    username = string
    groups   = list(string)
  }))
  default = []
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
