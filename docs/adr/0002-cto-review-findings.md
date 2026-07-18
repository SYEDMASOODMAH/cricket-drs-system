# ADR-0002: Pre-Phase-1 CTO review — findings and document revisions

**Status:** Accepted
**Date:** 2026-07-18

## Context

Before starting Phase 1 implementation, a critical CTO/Principal Architect review was performed across
`prd.md`, `architecture.md`, `rules.md`, and `phases.md`. The review identified gaps across missing
requirements, unrealistic assumptions, technical/AI-ML risk, architecture weaknesses, scalability,
security, cost, and development-complexity risk — full findings recorded in the review itself (not
duplicated here; see project history / review document).

## Decision

All Critical and High priority findings, and most Medium findings, were incorporated directly into the
four foundation documents (each bumped to v1.1) rather than tracked as a separate backlog, since they
change the documents' actual content, not just implementation detail. Key changes:

- **`prd.md`:** added a Legal, Privacy & Consent Framework (Section 13), a dispute/appeal workflow and
  onboarding-consent journey, revised accessible-tier accuracy targets from asserted to
  validation-gated, added a hardware-cost baseline, and moved business-model validation from Phase 10 to
  Phase 0/1.
- **`architecture.md`:** added a local-inference fallback (Section 9a) resolving the accessible-tier
  connectivity-vs-latency contradiction, a tier-capability matrix (Section 1a), a specified
  hash-chaining integrity mechanism, secrets management and Umpire Console threat-model requirements, a
  partner-integration adapter boundary, and a directional cost model (Section 19). Also fixed a stale
  Java reference that survived the ADR-0001 Go migration.
- **`rules.md`:** moved tenant-isolation and secrets-management testing from Phase 9 to Phase 1, added a
  rigor-ramp clarification (Section 0), reclassified weather/lighting variation as core training
  distribution, and added class-imbalance and human-override-label-QA requirements.
- **`phases.md`:** expanded Phase 0 to include a connectivity assessment, directional cost model, and
  business-model validation, with an explicit exit gate; added exit gates with kill/pivot criteria to
  Phase 3, 5, and 9; added Phase 1 tasks for secrets management, tenant-isolation testing, and legal/
  consent groundwork.

## Consequences

**Easier:** the roadmap now has explicit stop/pivot points instead of assuming every phase succeeds as
scoped; security and privacy are Phase 1 concerns instead of Phase 9 retrofits; the connectivity
assumption underlying the core latency NFR is now architecturally resolved rather than silently false
for the target market.

**Harder:** Phase 0 now has more deliverables (connectivity data, cost model, business-model
conversations) before Phase 1 can start — a deliberately accepted delay, since the alternative (finding
these problems in Phase 5–9) is far more expensive.

**Revisit if:** the Phase 0 exit gate reveals the accessible-tier accuracy ceiling or connectivity
reality is materially worse than hoped — in which case the pivot options documented in the Phase 3 Exit
Gate (`phases.md`) apply.
