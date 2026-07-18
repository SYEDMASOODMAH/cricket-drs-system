# Core Application Services (Go)

Modular monolith per `docs/architecture.md` Section 7: one Go module (`github.com/cricketdrs/services`),
one deployable unit for now, organized as independent packages per bounded context so a future split
into separately-deployed services is a low-friction move, not a rewrite.

| Service | Responsibility (see docs/architecture.md Section 5) |
|---|---|
| `review-orchestration` | End-to-end review workflow state machine |
| `match-tournament` | Match setup, playing conditions, review-quota enforcement |
| `identity-access` | Multi-tenant auth, RBAC |
| `analytics-reporting` | Post-match, season, and scouting analytics |

## Conventions

- Domain logic lives in each service's `internal/` package and must not import transport
  (`net/http`, gRPC) or persistence (`pgx`, `sqlc`-generated) types directly — see `docs/rules.md`
  Section 1. `cmd/` is the only place wiring happens.
- No ORM. Data access uses explicit SQL via `sqlc` or `pgx` once persistence lands (`docs/rules.md`
  Section 3.2/3.4).
- Errors are returned values, wrapped with `fmt.Errorf("...: %w", err)` for context — never discarded,
  never used via `panic` for normal control flow (`docs/rules.md` Section 4.2).
- Each service directory is self-contained: its own `cmd/`, `internal/`, `README.md`, and (for any
  service exposing a public/partner API) an `openapi.yaml` kept in sync with the implementation.

## Local development

```bash
cd services
go build ./...
go vet ./...
go test ./...

# run a single service
go run ./review-orchestration/cmd
```

## Adding a new service

1. Create the directory with `cmd/`, `internal/`, `README.md`, following the pattern of the existing
   four services.
2. Register it in this table.
3. If it's not purely internal, add an `openapi.yaml` contract stub.
4. Open an ADR in `docs/adr/` if it changes the architecture described in `docs/architecture.md`.
