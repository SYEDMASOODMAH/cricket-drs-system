# terraform

AWS infrastructure for Cricket DRS (`docs/architecture.md` Section 14). **Written and validated, never
applied** — see "Status" below before running anything here against a real AWS account.

## What exists

```
providers.tf     AWS + random provider requirements
variables.tf     aws_region (default ca-central-1 — see below), project_name, environment
backend.tf       S3 remote state — points at the bucket/table bootstrap/ creates
main.tf          wires modules/network -> modules/cluster -> modules/secrets

modules/
  network/       VPC (community terraform-aws-modules/vpc/aws), 2 AZs, single NAT gateway
  cluster/       EKS cluster (community terraform-aws-modules/eks/aws), one small managed node group
  secrets/       Secrets Manager secret for JWT_SIGNING_KEY + the IAM role (IRSA) to read it

bootstrap/       separate config: creates the S3 bucket + DynamoDB table backend.tf depends on
```

`region` defaults to `ca-central-1` (Montreal) rather than a generic US region — the initial target
market is Montreal-area club/league cricket (`prd.md` Section 14, e.g. the Montreal Cricket
Association), so this is a latency- and data-residency-conscious default. Override `var.aws_region` if
that changes.

## Status: not applied

This environment has no AWS credentials, and no cluster has ever been created from this config.
Provisioning a real EKS cluster is **not free**: directionally, the EKS control plane is ~$73/month,
a single NAT gateway is ~$32/month plus data-processing charges, and the default 2x `t3.medium` node
group is on top of that — call it roughly $150-200/month minimum before any real usage, per
`architecture.md` Section 19's directional cost model. `terraform apply` against a real account is a
deliberate step for whoever holds that account, with those numbers in mind — not something to run
casually or automatically.

## Applying this for real (when you're ready)

1. **Bootstrap the state backend once, manually**, from `bootstrap/`:
   ```bash
   cd bootstrap
   terraform init
   terraform apply   # creates the S3 bucket + DynamoDB lock table
   ```
   The default bucket name (`cricket-drs-terraform-state`) **will collide** — S3 bucket names are
   globally unique across all AWS accounts, not just yours. Override `var.state_bucket_name` (e.g.
   `-var="state_bucket_name=your-unique-name"`) before applying, and update `../backend.tf`'s `bucket`
   to match.

2. **Then the root config**:
   ```bash
   cd ..
   terraform init
   terraform plan    # review what this would actually create and roughly what it'll cost
   terraform apply
   ```

## What's deliberately not here yet

- **Dockerfiles** for `services/identity-access` and `services/match-tournament`, and **K8s
  Deployment/Service manifests** to actually run them on the cluster this config creates.
  `phases.md`'s Phase 1 ask is specifically "Kubernetes cluster **bootstrap**," not "deploy the
  services" — building deployment automation for a cluster that doesn't exist yet, for services
  `phases.md` still frames as an "empty-shell platform," is exactly the over-provisioning-ahead-of-need
  `architecture.md` Section 19 warns against.
- **The Secrets Store CSI driver** (or equivalent) that would actually sync `modules/secrets`' AWS
  secret into a running pod's environment. The IAM role (IRSA) this config creates is the mechanism;
  wiring a real workload to use it is part of the deferred K8s deployment work above. When that
  happens, **no Go code changes are needed** — both services already read `JWT_SIGNING_KEY` from the
  environment and only fall back to an insecure dev-only constant when it's unset (see each service's
  `cmd/main.go`), so a K8s-injected env var sourced from this secret just works.
- **GPU node pool** (Phase 3) and **object storage** (Phase 2) modules — still commented out in
  `main.tf`, exactly as in the original scaffold.

## Verifying without applying

```bash
terraform fmt -recursive -check   # from infra/terraform/
terraform init -backend=false     # downloads providers/modules, skips remote state
terraform validate                # checks syntax + that module wiring type-checks — no AWS calls
```

`bootstrap/` validates the same way, from within that directory (it uses local state, so a plain
`terraform init` works there without `-backend=false`).
