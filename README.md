# Cricket DRS

AI-powered Cricket Decision Review System — see `docs/prd.md` for product scope and `docs/architecture.md` for the full technical design.

## Foundation documents

| Doc | Purpose |
|---|---|
| [docs/prd.md](docs/prd.md) | Product vision, personas, features, requirements |
| [docs/architecture.md](docs/architecture.md) | System architecture, tech stack, data flow |
| [docs/rules.md](docs/rules.md) | Engineering standards and AI boundaries |
| [docs/phases.md](docs/phases.md) | Implementation roadmap (Phase 0–10) |
| [docs/design.md](docs/design.md) | UI/UX design system |
| [docs/adr/](docs/adr) | Architecture Decision Records |

## Repository layout

```
/services        Go — core application/orchestration services (modular monolith)
/edge-agent       Go — venue camera capture/buffer/sync agent
/ml-pipeline      Python — CV/ML: ball tracking, edge detection, LBW engine, run-out detection
/apps             web-app + umpire-console (React/TypeScript), mobile-app (Flutter/Dart)
/infra            Terraform + Kubernetes manifests
/research         Phase 0 feasibility notebooks/spikes — not production code
/docs             Foundation docs and ADRs
```

This monorepo layout follows the "monolith-first" principle in `docs/architecture.md` Section 7:
core services are structured as separate modules within one deployable unit until team size and
independent-deployment needs justify splitting them out. The CV/ML pipeline and edge agent are
separately deployable from day one, since they have fundamentally different scaling/runtime needs.

## Getting started

Each subproject has its own README with setup instructions:

- [`services/README.md`](services/README.md) — Go core services (start here for Phase 1)
- [`edge-agent/README.md`](edge-agent/README.md)
- [`ml-pipeline/README.md`](ml-pipeline/README.md)
- [`apps/web-app/README.md`](apps/web-app/README.md)
- [`apps/umpire-console/README.md`](apps/umpire-console/README.md)
- [`apps/mobile-app/README.md`](apps/mobile-app/README.md)
- [`infra/README.md`](infra/README.md)

## Current phase

**Phase 1 — Foundation & Architecture** (see `docs/phases.md`). No feature code has been written yet;
this scaffold establishes structure, tooling, and conventions so subsequent phases build on a
consistent foundation.
