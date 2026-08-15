terraform {
  required_version = ">= 1.7.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Deliberately no backend block: this config creates the S3 bucket and
  # DynamoDB table the root config's remote backend depends on, so it
  # can't use that backend itself (chicken-and-egg). Local state is
  # correct here, not a shortcut — see README.md.
}

provider "aws" {
  region = var.aws_region
}
