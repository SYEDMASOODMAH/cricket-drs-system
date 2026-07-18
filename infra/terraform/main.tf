# Root Terraform config — scaffold only (Phase 1).
# Per docs/architecture.md Section 14: Kubernetes (EKS/GKE) for application
# services, managed GPU node pools for ML inference, object storage for
# video/clips. Modules land as each is needed rather than all at once.

terraform {
  required_version = ">= 1.7.0"
}

# module "cluster" { source = "./modules/cluster" }   # Phase 1
# module "gpu_pool" { source = "./modules/gpu_pool" } # Phase 3
# module "storage" { source = "./modules/storage" }   # Phase 2
