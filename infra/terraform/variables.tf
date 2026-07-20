variable "aws_region" {
  description = <<-EOT
    AWS region for all resources. Defaults to ca-central-1 (Montreal) —
    the initial target market is Montreal-area club/league cricket
    (prd.md Section 14, e.g. the Montreal Cricket Association), so this is
    a latency- and data-residency-conscious default, not an arbitrary one.
    Override if that changes.
  EOT
  type        = string
  default     = "ca-central-1"
}

variable "project_name" {
  description = "Short name used to prefix/tag every resource this config creates."
  type        = string
  default     = "cricket-drs"
}

variable "environment" {
  description = "Deployment environment name (dev/staging/prod). Phase 1 only ever runs one."
  type        = string
  default     = "dev"
}
