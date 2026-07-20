# AWS Secrets Manager secret for JWT_SIGNING_KEY, plus the IAM role (IRSA)
# that would let a pod read it — the mechanism, not the full sync
# pipeline. Nothing here changes services/identity-access or
# services/match-tournament's Go code: both already read JWT_SIGNING_KEY
# from the environment and only fall back to the insecure dev-only
# constant when it's unset (see each service's cmd/main.go), so once a
# real deployment injects this secret's value as that env var — via
# whatever K8s-side mechanism (Secrets Store CSI driver, External
# Secrets Operator, etc.) — the fallback simply never triggers. Wiring
# that K8s-side sync is deferred; see the implementation plan.

resource "random_password" "jwt_signing_key" {
  length  = 64
  special = false # keep it a safe literal in a shell env var / K8s Secret
}

resource "aws_secretsmanager_secret" "jwt_signing_key" {
  name = "${var.project_name}/${var.environment}/jwt-signing-key"
  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "jwt_signing_key" {
  secret_id     = aws_secretsmanager_secret.jwt_signing_key.id
  secret_string = random_password.jwt_signing_key.result
}

# Standard IRSA trust policy: federates against this cluster's OIDC
# provider, scoped to one specific namespace/ServiceAccount pair so any
# other workload in the cluster can't assume this role.
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

resource "aws_iam_role" "app_secrets_reader" {
  name               = "${var.project_name}-${var.environment}-secrets-reader"
  assume_role_policy = data.aws_iam_policy_document.irsa_trust.json
  tags               = var.tags
}

# Least privilege: read access to exactly this one secret, nothing else.
data "aws_iam_policy_document" "read_secret" {
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.jwt_signing_key.arn]
  }
}

resource "aws_iam_role_policy" "read_secret" {
  name   = "read-jwt-signing-key"
  role   = aws_iam_role.app_secrets_reader.id
  policy = data.aws_iam_policy_document.read_secret.json
}
