output "state_bucket_name" {
  description = "Feed this into the root config's backend.tf `bucket`."
  value       = aws_s3_bucket.terraform_state.bucket
}

output "lock_table_name" {
  description = "Feed this into the root config's backend.tf `dynamodb_table`."
  value       = aws_dynamodb_table.terraform_lock.name
}
