output "cluster_name" {
  value = module.eks.cluster_name
}

output "region" {
  value = var.region
}

output "ecr_repository_urls" {
  value = { for k, v in aws_ecr_repository.repos : k => v.repository_url }
}

output "kubeconfig_command" {
  value = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}
