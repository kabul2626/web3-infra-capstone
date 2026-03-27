// Terraform configuration for AWS infrastructure provisioning.
// Provisions EKS cluster, VPC, ECR repositories, and related AWS resources.

provider "aws" {
  region = var.region
}

// Local variables for consistent naming and tagging across resources
locals {
  name = var.cluster_name
  tags = {
    Project = "web3-infra-capstone"
  }
}

// Fetch available AZs in the region for multi-AZ deployment
data "aws_availability_zones" "available" {}

// VPC module: creates networking infrastructure with public subnets in 2 AZs
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = local.name
  cidr = "10.0.0.0/16"

  azs            = slice(data.aws_availability_zones.available.names, 0, 2)
  public_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  map_public_ip_on_launch = true

  enable_nat_gateway = false
  single_nat_gateway = false
  enable_dns_hostnames = true

  tags = local.tags
}

// EKS module: creates managed Kubernetes cluster with auto-scaling node groups
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = local.name
  cluster_version = "1.29"
  vpc_id          = module.vpc.vpc_id
  subnet_ids      = module.vpc.public_subnets

  cluster_endpoint_public_access = true
  enable_cluster_creator_admin_permissions = true

  eks_managed_node_groups = {
    default = {
      desired_size = var.node_desired_size
      min_size     = var.node_min_size
      max_size     = var.node_max_size

      instance_types = [var.node_instance_type]
      capacity_type  = "SPOT"  // Use spot instances to reduce costs
      subnet_ids     = module.vpc.public_subnets
    }
  }

  tags = local.tags
}

// ECR repositories for storing Docker images of the project's services
resource "aws_ecr_repository" "repos" {
  for_each = toset([
    "web3-infra-capstone-agent",
    "web3-infra-capstone-anvil",
    "web3-infra-capstone-monitor"
  ])

  name                 = each.value
  image_tag_mutability = "MUTABLE"  // Allow image tag overwrites
  force_delete         = true        // Allow deletion even with images

  // Enable automatic vulnerability scanning on image push
  image_scanning_configuration {
    scan_on_push = true
  }

  tags = local.tags
}
