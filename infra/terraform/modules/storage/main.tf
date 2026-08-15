# S3 bucket for match video clips, plus the IAM role (IRSA) that would let
# media-ingest-gateway's pod read/write it — implements the "object
# storage integration with lifecycle policies" line from phases.md Phase
# 2's task list, and the commented `module "storage"` placeholder in the
# root config. Same security posture as bootstrap/'s state bucket
# (versioned, encrypted, public access blocked).

resource "aws_s3_bucket" "clips" {
  bucket = var.bucket_name
  tags   = var.tags
}

resource "aws_s3_bucket_versioning" "clips" {
  bucket = aws_s3_bucket.clips.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "clips" {
  bucket = aws_s3_bucket.clips.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "clips" {
  bucket = aws_s3_bucket.clips.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Tiered lifecycle policy (architecture.md Section 10): hot for the active
# season, transitioning to cheaper storage classes rather than deleting —
# the audit/dispute retention need (prd.md Section 5.7) is in direct
# tension with storage cost (architecture.md Section 19's cost model), so
# this is a deliberate cost-vs-retention tradeoff, not "keep everything at
# full price forever."
resource "aws_s3_bucket_lifecycle_configuration" "clips" {
  bucket = aws_s3_bucket.clips.id

  rule {
    id     = "tier-down-over-time"
    status = "Enabled"

    # Empty filter = applies to every object in the bucket. The provider
    # now requires this explicitly (an unqualified rule is deprecated).
    filter {}

    transition {
      days          = 90
      storage_class = "STANDARD_IA"
    }

    transition {
      days          = 365
      storage_class = "GLACIER"
    }

    noncurrent_version_transition {
      noncurrent_days = 30
      storage_class   = "STANDARD_IA"
    }
  }
}

# Standard IRSA trust policy — same pattern as modules/secrets, scoped to
# this specific namespace/ServiceAccount pair so no other workload in the
# cluster can assume this role.
data "aws_iam_policy_document" "irsa_trust" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [var.oidc_provider_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${replace(var.cluster_oidc_issuer_url, "https://", "")}:sub"
      values   = ["system:serviceaccount:${var.service_account_namespace}:${var.service_account_name}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${replace(var.cluster_oidc_issuer_url, "https://", "")}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "clips_access" {
  name               = "${var.project_name}-${var.environment}-clips-access"
  assume_role_policy = data.aws_iam_policy_document.irsa_trust.json
  tags               = var.tags
}

# Least privilege: read/write objects in this bucket only, nothing else.
data "aws_iam_policy_document" "clips_access" {
  statement {
    effect    = "Allow"
    actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
    resources = ["${aws_s3_bucket.clips.arn}/*"]
  }
  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.clips.arn]
  }
}

resource "aws_iam_role_policy" "clips_access" {
  name   = "clips-read-write"
  role   = aws_iam_role.clips_access.id
  policy = data.aws_iam_policy_document.clips_access.json
}
