# identity-access

**Status:** Phase 1 slice implemented — multi-tenant org/user/RBAC, consent capture, and
password/JWT auth. See `/docs/architecture.md` for this service's overall responsibilities and
`/docs/phases.md` for what's still ahead (Postgres persistence, real secrets management).

## Architecture

Hexagonal, per `rules.md` Section 1 — `internal/domain` has zero framework/database imports and is
fully unit-testable on its own:

```
internal/
  domain/      Organization, User, Role/Permission (RBAC), ConsentRecord — pure business rules
  service/     use-cases (CreateOrganization, InviteUser, AssignRole, Login, CaptureConsent, ...),
               repository port interfaces
  memstore/    thread-safe in-memory implementation of the repository ports (see "Persistence" below)
  security/    bcrypt password hashing + JWT issuance/verification, adapting to the service ports
  httpapi/     chi router + handlers, the only layer that imports net/http
```

`cmd/main.go` is the only place these are wired together (`rules.md` Section 2.2).

## Observability

Metrics, tracing, and structured-request logging come from `services/observability` (shared with
match-tournament — see `docs/adr/0004-shared-observability-package.md`), not this service's own code:

- `GET /metrics` — Prometheus text format (request count/duration by method/route/status, plus Go
  runtime/process collectors).
- Traces export to stdout by default (no trace backend exists in this environment yet — deferred to
  when cloud infrastructure exists). Every request gets a span; `internal/httpapi`'s structured log
  lines include `trace_id` when a span is active, per `rules.md` Section 4.1.

## Persistence

This is currently backed by an **in-memory store** (`internal/memstore`), not Postgres. There's no
Docker/Postgres available in the environment this was built in, and `rules.md` Section 1 requires
domain logic to be unit-testable without a database — so persistence is a swappable port
(`service.OrganizationRepository` / `UserRepository` / `ConsentRepository`), and a Postgres adapter
implementing the same interfaces is a follow-up, not a rewrite. **Data does not survive a restart.**

## Run locally

```
go run ./cmd
```

Health check: `GET http://localhost:8080/healthz`

### Configuration (environment variables)

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | |
| `JWT_SIGNING_KEY` | shared insecure dev-only key | If unset, falls back to a fixed, committed dev-only constant (`insecureDevSigningKey` in `cmd/main.go`) — identical to match-tournament's own fallback, so the two services can talk to each other with zero config for solo local preview (e.g. this repo's `.claude/launch.json`, whose format can't inject an env var). **Not a real secret** — logged loudly on use, and anything beyond solo local preview must set a real value. In production this is injected by a secrets manager (`architecture.md` Section 15); wiring that is deferred until a cloud provider is chosen. |

### Example walkthrough

```bash
# Create an organization + its first admin user
curl -s localhost:8080/v1/organizations -d '{
  "name": "Montreal Cricket Association",
  "admin_email": "admin@mca.example",
  "admin_password": "correct-horse-battery-staple"
}'

# Log in as the admin
curl -s localhost:8080/v1/auth/login -d '{
  "organization_id": "<org id from above>",
  "email": "admin@mca.example",
  "password": "correct-horse-battery-staple"
}'

# Add a player (Authorization: Bearer <token from login>)
curl -s localhost:8080/v1/organizations/<org id>/users \
  -H "Authorization: Bearer <token>" \
  -d '{"email": "player@mca.example", "password": "password123", "role": "player"}'

# Capture the player's consent
curl -s localhost:8080/v1/users/<player id>/consent \
  -H "Authorization: Bearer <token>" \
  -d '{"consent_type": "video_capture", "grant": true}'
```

Full endpoint set: `openapi.yaml`.

## Test

```
go test ./... -cover
```

All packages are at or above the 80% line-coverage target in `rules.md` Section 6.1. Tenant-isolation
is exercised directly in `internal/service` and `internal/httpapi` tests (`rules.md` Section 6.5) —
e.g. verifying a caller from one organization can never read, invite, or manage-consent-for another
organization's users, at both the service-method and HTTP-handler level.

## Known Phase 1 simplifications (tracked, not accidental)

- **No Postgres adapter yet** — see "Persistence" above.
- **No real secrets manager wiring** — `JWT_SIGNING_KEY` is env-var based; a cloud-provider decision
  (AWS vs GCP) is needed before wiring real secrets-manager injection.
- **`CreateOrganization` is unauthenticated** — there's no platform-superadmin role modeled yet, so
  org creation is effectively signup rather than an RBAC-gated action. Revisit alongside self-serve
  onboarding (`phases.md` Phase 10) or a platform-admin persona.
- **RBAC is a fixed, hand-rolled role→permission table** (`internal/domain/role.go`), not a policy
  engine — sufficient for the fixed persona list in `prd.md` Section 4. See
  `docs/adr/0001-go-for-core-services.md`'s "revisit if" clause.
