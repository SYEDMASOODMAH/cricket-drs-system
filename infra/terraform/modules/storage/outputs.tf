output "bucket_name" {
  value = aws_s3_bucket.clips.bucket
}

output "bucket_arn" {
  value = aws_s3_bucket.clips.arn
}

output "iam_role_arn" {
  description = "Annotate media-ingest-gateway's K8s ServiceAccount (var.service_account_name, once it exists) with eks.amazonaws.com/role-arn set to this value to complete the IRSA wiring."
  value       = aws_iam_role.clips_access.arn
}
