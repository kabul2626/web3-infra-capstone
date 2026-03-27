# Terraform (AWS)

This provisions a low-cost EKS cluster in eu-west-2 with public subnets and a single spot node group.

## Steps
1) `terraform init`
2) `terraform apply`
3) `aws eks update-kubeconfig --region eu-west-2 --name web3-infra-capstone`

## Outputs
- ECR repository URLs for app images
- Kubeconfig command
