# infra

Infrastructure as code for Cricket DRS, per `docs/architecture.md` Section 14.

- `terraform/` — cloud resources (Kubernetes cluster, GPU node pools, object storage, networking).
- `k8s/` — Kubernetes manifests for application services.

**Status:** Phase 1 scaffold only — see `docs/phases.md` for what's provisioned in each phase.
Nothing here should be applied to a real cloud account yet.
