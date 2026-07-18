# ADR-0001: Go for core application services and edge agent

**Status:** Accepted
**Date:** 2026-07-18

## Context

`docs/architecture.md` originally proposed Java (Spring Boot) as the default for core application
services (Review Orchestration, Match & Tournament, Identity & Access, Analytics & Reporting), with Go
noted as an alternative. The venue edge capture/buffer agent was always going to be Go (or a similarly
lightweight compiled language), since it runs on constrained hardware (Jetson-class devices) where a
JVM is a poor fit.

## Options Considered

1. **Java (Spring Boot) for core services, Go for the edge agent.** Two backend languages/ecosystems.
   Spring's RBAC/multi-tenancy/ORM tooling is more mature out of the box for the governance-heavy
   Identity & Access and Match & Tournament domains.
2. **Go for both core services and the edge agent.** One backend language (plus Python for ML, which
   is ecosystem-mandated regardless). Better fit for the bursty, I/O-fan-out-heavy review orchestration
   workload; native alignment with the Go-based infra stack (Kubernetes, Terraform, Prometheus); lighter
   operational footprint (no JVM tuning); faster cold starts for autoscaling under concurrent-match load.

## Decision

**Go**, for both core services and the edge agent.

The deciding factors, in order:
- The edge agent needed Go regardless, so a Java core meant running two backend ecosystems for no
  strong offsetting benefit.
- The review orchestration workload's concurrency profile (fan-out to ML services, message-bus
  consumption, real-time client push) fits Go's goroutine/channel model well.
- Operational simplicity: no JVM heap/GC tuning, single static binaries, native fit with the Go-based
  cloud-native tooling already in the stack.

## Consequences

**Easier:** one language across the non-ML backend (hiring, tooling, idioms), simpler container images
and autoscaling behavior, less operational tuning surface.

**Harder:** RBAC and multi-tenant domain modeling (Identity & Access, Match & Tournament) has less
mature off-the-shelf tooling in Go than Spring Security/Spring Data provide in Java — more of this will
be hand-rolled or assembled from smaller libraries (e.g., `ory`, `casbin`, `ent`).

**Revisit this decision if:**
- The team's actual hiring pool ends up skewing heavily toward Spring/Java expertise with little Go
  depth.
- Identity & Access / governance complexity grows to a point where Go's thinner ecosystem here becomes
  a measurable drag on delivery speed (track this qualitatively during Phase 1).

See `docs/architecture.md` Section 7 and Section 17 for the updated technology stack reflecting this
decision.
