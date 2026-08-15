output "secret_arn" {
  value = aws_secretsmanager_secret.jwt_signing_key.arn
}

output "iam_role_arn" {
  description = "Annotate the K8s ServiceAccount (var.service_account_name, once it exists) with eks.amazonaws.com/role-arn set to this value to complete the IRSA wiring."
  value       = aws_iam_role.app_secrets_reader.arn
}
