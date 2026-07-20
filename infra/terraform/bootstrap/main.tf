# Bootstrap: creates the S3 bucket + DynamoDB table the root Terraform
# config's remote state backend (../backend.tf) depends on. Run this once,
# manually, before the root config's `terraform init` can succeed — see
# README.md for the exact sequence. Nothing in this repo applies this
# automatically; it's a deliberate one-time, human-run step.

resource "aws_s3_bucket" "terraform_state" {
  bucket = var.state_bucket_name

  # Guards against a `terraform destroy` run against this config from
  # accidentally deleting the bucket that holds every other config's state.
  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket_versioning" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_dynamodb_table" "terraform_lock" {
  name         = var.lock_table_name
  billing_mode = "PAY_PER_REQUEST" # no idle cost — matches Phase 1 modesty
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }
}
