# EKS cluster. Uses the community terraform-aws-modules/eks module rather
# than hand-rolled aws_eks_cluster/aws_eks_node_group/IAM resources — same
# rationale as modules/network: this is a well-trodden path, and EKS's IAM
# and OIDC wiring has enough sharp edges that using the standard module is
# the "avoid inventing your own" choice (rules.md Section 3.3).
#
# One managed node group, not Fargate: Fargate can never run GPU
# workloads, and architecture.md Section 14 plans a GPU node pool addition
# in Phase 3 (see root main.tf's commented gpu_pool placeholder) — starting
# with an EC2-backed managed node group now avoids a forced migration off
# Fargate later. Sized small (var.node_instance_types defaults to a single
# t3.medium type, 1-3 nodes) since there's nothing GPU- or load-heavy to
# run yet.
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = var.kubernetes_version

  vpc_id     = var.vpc_id
  subnet_ids = var.private_subnet_ids

  # No public endpoint access by default — cluster API reachable only from
  # inside the VPC (e.g. a bastion or VPN) until there's a real reason to
  # open it up, which is a decision for whoever actually applies this, not
  # a scaffold default.
  cluster_endpoint_public_access = false

  # Creates the IAM OIDC identity provider for the cluster, which IRSA
  # (used by modules/secrets to let pods read the JWT_SIGNING_KEY secret)
  # depends on.
  enable_irsa = true

  eks_managed_node_groups = {
    default = {
      instance_types = var.node_instance_types
      min_size       = var.node_min_size
      max_size       = var.node_max_size
      desired_size   = var.node_desired_size
    }
  }

  tags = var.tags
}
