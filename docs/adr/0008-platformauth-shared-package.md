# ADR-0008: Shared `services/platformauth` package for the Role enum and JWT verification

**Status:** Accepted
**Date:** 2026-08-15

## Context

The `Role` enum and JWT verification logic were hand-copied across all 4 Go services
(`identity-access`, `match-tournament`, `media-ingest-gateway`, `camera-calibration`) since
match-tournament shipped, deliberately, per `docs/adr/0001-go-for-core-services.md`: Go's `internal/`
visibility rules block importing another service's package directly, and — at the time — that code
carried real domain/security semantics where a divergence between copies would be a meaningful bug, not
just wasted lines.

`docs/adr/0004-shared-observability-package.md` extracted metrics/tracing/logging middleware the same
way `platformauth` is extracted here, but explicitly left the JWT/Role question open in its "Revisit if"
clause: compare whether that duplication should also move under `services/`, "or whether auth's
domain-coupling still argues for a different shape." Each subsequent service's copy narrated the same
open question — match-tournament's said revisit once a third service needs it, media-ingest-gateway's
said "now the third data point, worth actually revisiting," camera-calibration's said "the 4th instance
of this exact duplication."

## What was actually verified

Before deciding, all 4 services' `internal/domain/role.go` and `internal/security/jwt.go` were read and
compared directly (not assumed identical from memory):

- **The `Role` type (6 constants + `Valid()`) and the JWT `tokenClaims` wire struct are byte-for-byte
  identical in all 4 services** — same string values, same JSON tags, same HMAC-only keyfunc check, same
  error strings.
- **The per-service `Can*(role Role) bool` functions are NOT uniform** — `CanUploadClips`,
  `CanManageMatches`, `CanManageCalibration` are three different names gating three different actions,
  each currently reducing to `role == RoleOrganizerAdmin` but for different, service-specific reasons.
  identity-access's `Permission`/`HasPermission` table is unique to that service entirely.

This is exactly the fact ADR-0004 required before sharing: "no business vocabulary here that could
legitimately diverge between services." It's true of `Role` and the JWT wire format. It is **not** true
of the `Can*`/`Permission` layer — that's real, service-specific authorization business logic.

## Decision

Extract `services/platformauth/` (top-level, non-`internal/`, same module as `services/observability`)
containing:

- `Role` (the enum + `Valid()`) — every service's `domain.Role` becomes a **type alias**
  (`type Role = platformauth.Role`), not a new type, so every existing call site keeps compiling
  unchanged.
- `Verifier`/`Claims` — the JWT verify-only logic every service except identity-access needs.
- `Issuer` — wraps `Verifier` and adds `Issue`; only identity-access constructs one, but it lives here
  (not duplicated into identity-access's own package) because `Issue` and `Verify` share the same wire
  struct and must not drift apart.
- `InsecureDevSigningKey` — the same category of duplication (an identical literal hand-copied into 4
  `cmd/main.go` files with a comment warning it must stay byte-identical), closed the same way.

**Explicitly NOT moved:** every service's `Can*` function, and identity-access's `Permission`/
`HasPermission` table. Those stay in each service's own `internal/domain` package — this is the
"domain-coupling" ADR-0004 flagged as the open question, resolved here as: shared identity/token
*vocabulary* lives in `platformauth`; per-service authorization *decisions* stay local.

## Consequences

**Easier:** one place to fix a JWT bug instead of four; `InsecureDevSigningKey` can't silently drift
between services; a new service only needs `platformauth.NewVerifier(signingKey)` instead of
re-deriving the HMAC keyfunc/error-handling logic from scratch.

**Harder:** none observed — the type-alias approach kept every existing call site
(`domain.RoleOrganizerAdmin`, `role.Valid()`, `service.Claims`) compiling unchanged; verified via a full
`go build ./... && go vet ./... && go test ./... -cover` across the module after the migration, with all
pre-existing tests passing.

**Test consolidation:** the expired/wrong-key/garbage-token negative-path tests, previously duplicated
near-verbatim across 3-4 services' `internal/security/security_test.go`, now live once in
`platformauth/verifier_test.go` and `issuer_test.go`. Each service keeps one adapter-level round-trip
test confirming its own domain-type conversion still works.

## Related

- `docs/adr/0004-shared-observability-package.md` (the precedent this resolves the open question from)
- `docs/adr/0001-go-for-core-services.md` (original duplicate-per-service rationale)
- `services/platformauth/` (this package)
