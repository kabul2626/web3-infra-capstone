# EKS cluster name for kubectl context
output "cluster_name" {
  value = module.eks.cluster_name
}

# AWS region for AWS CLI commands
output "region" {
  value = var.region
}

# ECR repository URLs for pushing Docker images
output "ecr_repository_urls" {
  value = { for k, v in aws_ecr_repository.repos : k => v.repository_url }
}

# Command to configure local kubeconfig for cluster access
output "kubeconfig_command" {
  value = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}
