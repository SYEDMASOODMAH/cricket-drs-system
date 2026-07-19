# ADR-0004: Shared `services/observability` package for metrics/tracing/logging middleware

**Status:** Accepted
**Date:** 2026-07-18

## Context

`phases.md` lists "Observability stack setup (structured logging, metrics, tracing baseline)" as a
Phase 1 deliverable, and `rules.md` Section 4.3 requires golden-signal instrumentation (latency,
traffic, errors, saturation) for every service via a standard stack (OpenTelemetry → Prometheus, or a
managed equivalent).

Both `identity-access` and `match-tournament` already have a `structuredLogger` middleware
(`log/slog`-based), and it is byte-for-byte duplicated between them today. Adding metrics and tracing
raises the same question at larger scale: copy the same Prometheus/OpenTelemetry setup into every
service's `internal/httpapi`, or share it once.

This is a similar-shaped question to the one ADR-0001's Go decision and the Match & Tournament
implementation already answered for JWT verification and the `Role` enum — those were **deliberately
kept duplicated per-service** rather than shared, because Go's `internal/` visibility rules make direct
reuse impossible, and because that code carries real domain/security semantics where a divergence
between copies would be a meaningful bug, not just wasted lines. This ADR reaches the opposite
conclusion for observability middleware, and explains why the two aren't in tension.

## Options Considered

1. **Keep duplicating** — copy `structuredLogger`-equivalent metrics/tracing setup into each service's
   `internal/` tree, as already done for JWT verification.
2. **Extract a shared `services/observability` package** — a new top-level (non-`internal/`) package in
   the existing `services` Go module, importable by any service in the module regardless of its own
   `internal/` boundary.

## Decision

**Option 2.** Observability middleware (request-count/duration metrics, span creation, structured
request logging) carries **zero domain meaning** — recording "this route returned 200 in 40ms" is
identical boilerplate for a match-tournament request as it is for an identity-access one, and will be
identical again for Review Orchestration and Analytics & Reporting once those are built. Unlike RBAC
roles or JWT claims, there is no business vocabulary here that could legitimately diverge between
services. Continuing to hand-copy this is pure waste, not a safety margin.

`services/observability/` sits outside every service's `internal/` tree specifically so it's
importable across service boundaries — this is a deliberate, first use of that pattern in this repo,
which is why it gets an ADR per `rules.md` Section 2.3 rather than being introduced silently.

## Consequences

**Easier:** one place to get metrics/tracing/logging right, reused by every current and future core
service; adding a new service's observability wiring becomes "import and call `New()`," not "re-derive
Prometheus label cardinality rules from scratch again."

**Harder:** this is the first cross-service shared package in the repo, so it sets a precedent —
anyone tempted to add unrelated shared helpers here later should get their own ADR, not ride on this
one. `services/observability` should stay narrowly scoped to observability concerns.

**Revisit if:** the JWT/Role duplication (`match-tournament`'s implementation notes) ever needs
revisiting once a third service needs the same auth vocabulary — at that point, compare whether that
extraction should also live under `services/`, following the precedent set here, or whether auth's
domain-coupling still argues for a different shape.

## Related

- `docs/adr/0001-go-for-core-services.md` (the "duplicate vs. share" question for JWT/Role, decided the
  other way, for different reasons)
- `services/match-tournament/README.md` ("Shared auth with Identity & Access" section)
- `rules.md` Section 4.1 (structured logging), Section 4.3 (monitoring/golden signals)
