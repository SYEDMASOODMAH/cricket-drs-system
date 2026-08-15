variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "oidc_provider_arn" {
  description = "From modules/cluster's output — the EKS cluster's IAM OIDC identity provider, which IRSA's trust policy federates against."
  type        = string
}

variable "cluster_oidc_issuer_url" {
  description = "From modules/cluster's output. Used to scope the trust policy to this specific cluster."
  type        = string
}

variable "service_account_namespace" {
  description = "K8s namespace the app's ServiceAccount lives in — must match infra/k8s/base/namespace.yaml."
  type        = string
  default     = "cricket-drs"
}

variable "service_account_name" {
  description = <<-EOT
    K8s ServiceAccount name allowed to assume this IAM role (IRSA). No
    ServiceAccount with this name exists yet — creating it, and annotating
    it with this role's ARN, is part of the deferred K8s Deployment work
    (see the implementation plan's "Explicitly deferred" section), not
    this module.
  EOT
  type        = string
  default     = "cricket-drs-app"
}

variable "tags" {
  type    = map(string)
  default = {}
}
