# Remote state, backed by the bucket/table infra/terraform/bootstrap/
# creates (run that first — see its README). Values below match that
# config's variable defaults; if you override those, update these too.
#
# NOT active until `terraform init` is actually run against a real AWS
# account — this repo has never applied either config (see
# infra/terraform/README.md).
terraform {
  backend "s3" {
    bucket         = "cricket-drs-terraform-state"
    key            = "cricket-drs/terraform.tfstate"
    region         = "ca-central-1"
    dynamodb_table = "cricket-drs-terraform-lock"
    encrypt        = true
  }
}
