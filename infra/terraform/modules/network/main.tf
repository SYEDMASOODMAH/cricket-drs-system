# VPC for the EKS cluster. Uses the community terraform-aws-modules/vpc
# module rather than hand-rolled subnet/route-table/NAT resources — that
# module is widely used and battle-tested; a hand-written equivalent would
# almost certainly get an edge case (route propagation, AZ failure
# handling) subtly wrong somewhere rules.md Section 3.3 wouldn't catch
# until it mattered.
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = var.name
  cidr = var.vpc_cidr

  azs = var.availability_zones
  # /20 subnets (4096 addresses each) — plenty for Phase 1's 2 small node
  # groups, leaves headroom without pre-allocating a huge address space.
  private_subnets = [for i, az in var.availability_zones : cidrsubnet(var.vpc_cidr, 4, i)]
  public_subnets  = [for i, az in var.availability_zones : cidrsubnet(var.vpc_cidr, 4, i + 8)]

  enable_nat_gateway     = true
  single_nat_gateway     = true # one NAT, not one per AZ — architecture.md Section 19 cost caution
  one_nat_gateway_per_az = false

  enable_dns_hostnames = true
  enable_dns_support   = true

  # EKS uses these tags to discover subnets for the cluster's own ENIs and
  # for any LoadBalancer-type Services created later.
  public_subnet_tags = {
    "kubernetes.io/role/elb"            = "1"
    "kubernetes.io/cluster/${var.name}" = "shared"
  }
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb"   = "1"
    "kubernetes.io/cluster/${var.name}" = "shared"
  }

  tags = var.tags
}
