variable "aws_region" {
  description = "AWS region for the Terraform state backend resources."
  type        = string
  default     = "ca-central-1" # Montreal — see root variables.tf for why.
}

variable "state_bucket_name" {
  description = <<-EOT
    S3 bucket name for Terraform state. S3 bucket names are globally
    unique across *all* AWS accounts, not just yours — the scaffold
    default below will collide with someone else's bucket. Override it
    (e.g. via -var or a .tfvars file, never committed if it encodes an
    account-specific choice) before applying.
  EOT
  type        = string
  default     = "cricket-drs-terraform-state"
}

variable "lock_table_name" {
  description = "DynamoDB table name for Terraform state locking."
  type        = string
  default     = "cricket-drs-terraform-lock"
}
