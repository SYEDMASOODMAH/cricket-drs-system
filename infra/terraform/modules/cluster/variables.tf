variable "cluster_name" {
  description = "EKS cluster name."
  type        = string
}

variable "kubernetes_version" {
  description = "EKS Kubernetes version."
  type        = string
  default     = "1.31"
}

variable "vpc_id" {
  type = string
}

variable "private_subnet_ids" {
  description = "The cluster and its node group both live in private subnets — no direct internet ingress to nodes."
  type        = list(string)
}

variable "node_instance_types" {
  description = "Phase 1 modesty: small general-purpose instances. GPU node pools are a separate, Phase 3 module (see root main.tf's commented gpu_pool placeholder) — not this one."
  type        = list(string)
  default     = ["t3.medium"]
}

variable "node_min_size" {
  type    = number
  default = 1
}

variable "node_max_size" {
  type    = number
  default = 3
}

variable "node_desired_size" {
  type    = number
  default = 2
}

variable "tags" {
  type    = map(string)
  default = {}
}
