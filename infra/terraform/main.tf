# Root Terraform config (Phase 1/2). Per docs/architecture.md Section 14:
# Kubernetes (EKS/GKE) for application services, managed GPU node pools for
# ML inference, object storage for video/clips. Modules land as each is
# needed rather than all at once — gpu_pool (Phase 3) stays unimplemented
# below until that phase actually needs it.
#
# STATUS: written, not applied. See README.md — this environment has no
# AWS credentials, and provisioning a real EKS cluster costs real recurring
# money. `terraform init`/`plan`/`apply` against a real AWS account is a
# deliberate step for whoever holds that account, not something this repo
# does automatically.

locals {
  cluster_name = "${var.project_name}-${var.environment}"

  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

module "network" {
  source = "./modules/network"

  name               = local.cluster_name
  availability_zones = ["${var.aws_region}a", "${var.aws_region}b"]
  tags               = local.common_tags
}

module "cluster" {
  source = "./modules/cluster"

  cluster_name       = local.cluster_name
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  tags               = local.common_tags
}

module "secrets" {
  source = "./modules/secrets"

  project_name            = var.project_name
  environment             = var.environment
  oidc_provider_arn       = module.cluster.oidc_provider_arn
  cluster_oidc_issuer_url = module.cluster.cluster_oidc_issuer_url
  tags                    = local.common_tags
}

module "storage" {
  source = "./modules/storage"

  project_name            = var.project_name
  environment             = var.environment
  bucket_name             = var.clips_bucket_name
  oidc_provider_arn       = module.cluster.oidc_provider_arn
  cluster_oidc_issuer_url = module.cluster.cluster_oidc_issuer_url
  tags                    = local.common_tags
}

# module "gpu_pool" { source = "./modules/gpu_pool" } # Phase 3
