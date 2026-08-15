variable "name" {
  description = "Prefix for all resources this module creates (e.g. \"cricket-drs-dev\")."
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "AZs to spread subnets across. Phase 1 modesty: 2, not 3 (architecture.md Section 19)."
  type        = list(string)
}

variable "tags" {
  description = "Tags applied to every resource this module creates."
  type        = map(string)
  default     = {}
}
