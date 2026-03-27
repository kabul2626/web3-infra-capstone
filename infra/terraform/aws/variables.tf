# AWS region for all infrastructure
variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-2"
}

# EKS cluster identifier and naming
variable "cluster_name" {
  description = "EKS cluster name"
  type        = string
  default     = "web3-infra-capstone"
}

# EC2 instance type for Kubernetes nodes (spot instances for cost savings)
variable "node_instance_type" {
  description = "EKS node instance type"
  type        = string
  default     = "t3.small"
}

# Target number of nodes (scales to desired count)
variable "node_desired_size" {
  description = "Desired number of nodes"
  type        = number
  default     = 1
}

# Minimum nodes to maintain for availability
variable "node_min_size" {
  description = "Minimum number of nodes"
  type        = number
  default     = 1
}

# Maximum nodes allowed by auto-scaler
variable "node_max_size" {
  description = "Maximum number of nodes"
  type        = number
  default     = 2
}
