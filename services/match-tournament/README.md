# match-tournament

**Status:** Phase 1 slice implemented — tournaments, teams, matches, playing conditions, and
consent-gated match rosters. See `/docs/architecture.md` for this service's overall
responsibilities and `/docs/phases.md` for what's still ahead (Postgres persistence, real secrets
management, camera/venue calibration in Phase 2).

## Architecture

Hexagonal, same shape as `services/identity-access` — see that service's README for the general
pattern. One addition here: `internal/identityaccess` is an HTTP adapter, not a persistence one — it
implements the consent-gate check (`service.ConsentChecker`) by calling Identity & Access over HTTP,
this service's one piece of real service-to-service communication (`architecture.md` Section 7).

```
internal/
  domain/          Tournament, Team, Match, PlayingConditions, RosterEntry, Role enum
  service/         use-cases + repository ports + ConsentChecker port
  memstore/         in-memory implementation of the repository ports (see "Persistence" below)
  identityaccess/   HTTP adapter calling Identity & Access's consent endpoint
  security/         JWT *verify-only* adapter — this service never issues tokens, only validates
                     ones Identity & Access minted (see "Shared auth" below)
  httpapi/          chi router + handlers
```

## Observability

Metrics, tracing, and structured-request logging come from `services/observability` (shared with
identity-access — see `docs/adr/0004-shared-observability-package.md`, and contrast with the "Shared
auth" section below, which deliberately *doesn't* share code, for different reasons):

- `GET /metrics` — Prometheus text format.
- Traces export to stdout by default. The consent-gate HTTP call to Identity & Access
  (`internal/identityaccess`) is wrapped so it's a proper child span of whatever request triggered it —
  the roster-add flow produces a 3-span trace across both services (incoming roster-add request →
  outgoing consent-check call → Identity & Access's incoming request), all sharing one trace ID.

## Shared auth with Identity & Access — read this before running both together

This service duplicates a small (~60-line) JWT-verification adapter and the `Role` enum rather than
importing Identity & Access's code — Go's `internal/` visibility rules make that literally impossible
across services, and a new shared package wasn't justified yet for two data points (see the
implementation plan's "Decision flagged" note; revisit once a third service needs the same thing).

**Consequence: both services must be started with the exact same `JWT_SIGNING_KEY`.** If unset, both
services fall back to `insecureDevSigningKey` — an identical, committed, obviously-fake constant
defined in each service's `cmd/main.go` (both log a loud `WARN` when they use it). This exists purely
so a zero-config single-command dev/preview run (this repo's `.claude/launch.json`, whose format has
no way to inject an env var) has the two services able to talk to each other without any manual setup.
It is **not a real secret** — anything beyond solo local preview (a shared dev environment, staging,
production, or running one service with an explicit `JWT_SIGNING_KEY` and the other without) must set
the same real value on both:

```bash
# both services need this to be the same value
export JWT_SIGNING_KEY="some-shared-dev-secret"
```

## Persistence

In-memory (`internal/memstore`), not Postgres — same reasoning as Identity & Access: no Docker/Postgres
available in this environment, and `rules.md` Section 1 requires domain logic to be unit-testable
without a database. **Data does not survive a restart.**

## Run locally

```bash
go run ./cmd   # or set JWT_SIGNING_KEY first if identity-access has one set too
```

Health check: `GET http://localhost:8080/healthz`

### Configuration (environment variables)

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | Change this if running alongside identity-access on the same machine — they both default to 8080 |
| `JWT_SIGNING_KEY` | shared insecure dev-only key | Falls back to a fixed, committed dev-only constant if unset — see "Shared auth" above. Must match the value Identity & Access was started with |
| `IDENTITY_ACCESS_URL` | `http://localhost:8080` | Base URL used for the consent-gate check |

### Example walkthrough (assumes Identity & Access is already running and you have a token — see its README)

```bash
# Create a team
curl -s localhost:8080/v1/organizations/<org id>/teams \
  -H "Authorization: Bearer <token>" -d '{"name": "Montreal Overseas CC"}'

# Create a match
curl -s localhost:8080/v1/organizations/<org id>/matches \
  -H "Authorization: Bearer <token>" \
  -d '{"home_team_id": "<team a>", "away_team_id": "<team b>", "venue": "MCA Ground 1", "scheduled_at": "2026-08-01T14:00:00Z"}'

# Add a player to the roster — this calls Identity & Access's consent
# endpoint first and rejects with 403 if they haven't granted video_capture
# + ai_analysis consent (prd.md Section 5.6.2)
curl -s localhost:8080/v1/organizations/<org id>/matches/<match id>/roster \
  -H "Authorization: Bearer <token>" \
  -d '{"team_id": "<team a>", "user_id": "<player id>"}'
```

Full endpoint set: `openapi.yaml`.

## Test

```
go test ./... -cover
```

All packages are at or above the 80% line-coverage target in `rules.md` Section 6.1. Tenant isolation
is exercised directly in `internal/service` and `internal/httpapi` (a caller from one org can never
read/write another org's tournaments, teams, matches, or rosters), and the consent gate is tested with
a fake `ConsentChecker` (`internal/service`) plus a real HTTP round-trip against an `httptest` server
(`internal/identityaccess`) — the real network call to a live Identity & Access is only exercised in
the manual walkthrough above.

## Known Phase 1 simplifications (tracked, not accidental)

- **No Postgres adapter yet** — see "Persistence" above.
- **JWT verification and the `Role` enum are duplicated from Identity & Access**, not shared — see
  "Shared auth" above.
- **Review-quota is a stored rule, not enforced consumption** — `PlayingConditions.ReviewQuotaPerInnings`
  is configuration; there's no review-triggering code yet to consume it against (Review Orchestration
  Service, `phases.md` Phase 7).
- **No camera/venue calibration modeling** — `Match.Venue` is a plain string; the Camera Calibration
  Service lands in Phase 2.
