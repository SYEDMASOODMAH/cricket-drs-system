variable "project_name" {
  type = string
}

variable "environment" {
  type = string
}

variable "bucket_name" {
  description = <<-EOT
    S3 bucket name for match video clips. Globally unique across all AWS
    accounts, not just yours — the scaffold default in the root config
    will collide. Override before applying, same as
    bootstrap/'s state_bucket_name.
  EOT
  type        = string
}

variable "oidc_provider_arn" {
  description = "From modules/cluster's output — federates this module's IRSA trust policy."
  type        = string
}

variable "cluster_oidc_issuer_url" {
  type = string
}

variable "service_account_namespace" {
  type    = string
  default = "cricket-drs"
}

variable "service_account_name" {
  description = <<-EOT
    K8s ServiceAccount name allowed to assume this IAM role (IRSA) —
    distinct from modules/secrets' service account, since that role only
    needs to read one secret and this one needs to read/write the clips
    bucket. No such ServiceAccount exists yet — creating and annotating it
    is part of the deferred K8s Deployment work (see the implementation
    plan's "explicitly deferred" section), not this module.
  EOT
  type        = string
  default     = "media-ingest-gateway"
}

variable "tags" {
  type    = map(string)
  default = {}
}
