output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnet_ids" {
  description = "Where the EKS cluster and its node group live."
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Where public-facing load balancers (if any) live."
  value       = module.vpc.public_subnets
}
