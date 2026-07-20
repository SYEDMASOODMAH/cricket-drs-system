# infra

Infrastructure as code for Cricket DRS, per `docs/architecture.md` Section 14. Cloud provider: **AWS**
(`ca-central-1` by default — see `terraform/variables.tf`).

- `terraform/` — cloud resources: VPC + EKS cluster + a Secrets Manager secret for `JWT_SIGNING_KEY`
  (Phase 1). GPU node pools (Phase 3) and object storage (Phase 2) are still unimplemented, as
  commented in `terraform/main.tf`.
- `k8s/` — Kubernetes manifests. Only the shared `Namespace` exists so far — Deployment/Service
  manifests for identity-access and match-tournament are a deferred follow-up (see
  `terraform/README.md`).

**Status:** the Terraform in this directory is written and `terraform validate`-clean, but **has never
been applied** — this environment has no AWS credentials, and provisioning a real EKS cluster costs
real recurring money (control plane + NAT gateway + node instances). Applying it against a real AWS
account, with real credentials, is a deliberate action for whoever holds that account — see
`terraform/README.md` for the exact sequence.
