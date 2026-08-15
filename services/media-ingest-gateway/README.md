# media-ingest-gateway

**Status:** Phase 2 slices implemented — accepts uploaded match video clips, stores them, and stores
multi-camera time-sync offsets computed against them. See `/docs/architecture.md` Section 5 for this
service's overall responsibilities and `/docs/phases.md` Phase 2 for what's still ahead (the edge-agent's
actual camera capture, SRT/WebRTC transport).

## Architecture

Hexagonal, same shape as identity-access and match-tournament — see identity-access's README for the
general pattern. Two ports here instead of one persistence port: `ClipRepository` (metadata) and
`ObjectStore` (raw bytes), because "where a clip's *record* lives" and "where its *bytes* live" are
different concerns with different storage technologies.

```
internal/
  domain/          Clip (+ sync fields, ApplySyncOffset, SyncConfident), Role enum + CanUploadClips,
                    sentinel errors
  service/         UploadClip / GetClip / ListClips / DownloadClip / SubmitSyncOffset use-cases,
                    ClipRepository + ObjectStore ports
  memstore/         in-memory ClipRepository (metadata: org, match, camera, hash, size, time, sync)
  objectstore/      ObjectStore's two adapters: memory.go (in-memory bytes, tests/local dev)
                    and s3.go (real AWS SDK v2, unit-tested against a fake — see "Object storage" below)
  security/         JWT *verify-only* adapter — this service never issues tokens, only validates
                    ones Identity & Access minted (see "Shared auth" below)
  httpapi/          chi router + handlers
```

## Multi-camera time sync

`architecture.md` Section 5 assigns "time-synchronizes multi-camera feeds" directly to this service
(unlike Camera Calibration, which got its own service — see `docs/adr/0006` for why sync didn't). A
clip's `Sync*` fields (`internal/domain/clip.go`) record an offset relative to another clip in the same
match, submitted via `PUT .../clips/{clipID}/sync`. **This service never computes the offset itself** —
the actual audio cross-correlation runs in `ml-pipeline/time-sync` (`docs/adr/0006`), tested against
synthetic audio fixtures rather than real camera footage. `Clip.SyncConfident()` compares the submitted
`correlation_score` against `domain.MinSyncCorrelationScore` — a placeholder threshold (0.5), since no
document specifies a real one (`phases.md`'s Phase 2 completion criteria says only "within a defined
tolerance," never a number).

## Object storage

Two adapters behind the same `ObjectStore` port:

- **In-memory** (`objectstore.NewMemoryStore`) — the default. Uploaded clips **do not survive a
  restart**. Same reasoning as every other service's in-memory persistence: no Docker/Postgres/real AWS
  access in the environment this was built in.
- **S3** (`objectstore.NewS3Store`) — used automatically if `S3_BUCKET` is set (see Configuration
  below). The bucket itself is provisioned by `infra/terraform/modules/storage`, which is written and
  `terraform validate`-clean but **has never been applied** (no AWS credentials in this environment —
  see that module's README). The adapter's own logic is unit-tested against a fake implementing the
  three S3 client methods it uses (`internal/objectstore/s3_test.go`) — no real AWS call happens
  anywhere in this codebase.

## Shared auth with Identity & Access and Match & Tournament

This service duplicates the same small JWT-verification adapter and `Role` enum match-tournament does,
for the same reason (Go's `internal/` visibility rules — see match-tournament's README for the full
rationale). **This is now the third service with this exact duplication** — per the original "revisit
once a third service needs the same thing" note, extracting a shared auth package is worth actually
reconsidering next time any of these three needs a change, rather than continuing to hand-copy a fourth
time.

**Consequence: all three services must be started with the same `JWT_SIGNING_KEY`** (or all three left
unset, in which case they share the same committed insecure dev-only fallback — see identity-access's
README for why that fallback exists and what it's for).

## Upload authorization

Clip upload is gated to the `organizer_admin` persona — same as match-tournament's match-management
writes. There's no distinct edge-device/machine credential yet (a real venue's edge-agent arguably needs
its own identity, per `architecture.md` Section 15's "authenticated/signed camera sessions"), which is
deferred until the edge-agent itself is being built — see the implementation plan's "explicitly
deferred" section.

## Run locally

```bash
go run ./cmd
```

Health check: `GET http://localhost:8080/healthz`

### Configuration (environment variables)

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | |
| `JWT_SIGNING_KEY` | shared insecure dev-only key | Must match identity-access's and match-tournament's — see "Shared auth" above |
| `S3_BUCKET` | *(unset — uses in-memory storage)* | If set, clips are stored in this S3 bucket instead of in-memory. Requires real AWS credentials to actually work (`config.LoadDefaultConfig`'s standard credential chain) |

### Example walkthrough (assumes Identity & Access is running and you have a token — see its README)

```bash
# Upload a clip (raw bytes as the request body)
curl -s -X POST "localhost:8080/v1/organizations/<org id>/matches/<match id>/clips?camera_id=cam-1" \
  -H "Authorization: Bearer <token>" \
  --data-binary @clip.mp4

# List clips for a match
curl -s "localhost:8080/v1/organizations/<org id>/matches/<match id>/clips" \
  -H "Authorization: Bearer <token>"

# Download a clip back
curl -s "localhost:8080/v1/organizations/<org id>/matches/<match id>/clips/<clip id>/download" \
  -H "Authorization: Bearer <token>" -o downloaded.mp4

# Submit a computed sync offset (offset_ms/correlation_score from ml-pipeline/time-sync's find_offset)
curl -s -X PUT "localhost:8080/v1/organizations/<org id>/matches/<match id>/clips/<clip id>/sync" \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"reference_clip_id":"<other clip id>","offset_ms":250,"correlation_score":0.92}'
```

Full endpoint set: `openapi.yaml`.

## Test

```
go test ./... -cover
```

All packages are at or above the 80% line-coverage target in `rules.md` Section 6.1. Tenant isolation
is exercised directly in `internal/service` and `internal/httpapi`, same as the other two services.

## Known Phase 2 simplifications (tracked, not accidental)

- **Plain HTTP upload, not SRT/WebRTC** — `architecture.md` Section 10 specifies SRT/WebRTC for the real
  venue-to-cloud leg; there's no edge-agent streaming client to talk to yet, so a straightforward
  authenticated upload endpoint is what's actually being tested now.
- **Whole clip buffered in memory during upload**, not true streaming — keeps the upload logic simple
  and correct (no partial-write cleanup to worry about) at the cost of memory use proportional to clip
  size. Not yet a real production upload path either way.
- **Basic anti-tampering only** — authenticated uploads plus a server-computed SHA-256 hash, not full
  replay/liveness detection (a hard, real-footage-dependent problem).
- **No cross-service `matchID` validation** against match-tournament — treated as an opaque, trusted
  foreign reference for this Phase 2 "basic" slice.
- **JWT verification and the `Role` enum are duplicated a third time** — see "Shared auth" above; worth
  actually revisiting now.
- **No live Go↔Python wiring for sync** — `find_offset`'s output must currently be submitted by hand (or
  a future tool) via `PUT .../clips/{clipID}/sync`; see "Multi-camera time sync" above.
- **Placeholder sync confidence threshold** — `domain.MinSyncCorrelationScore` is structurally correct,
  not measured; no document specifies a real tolerance.
- **No real audio extraction from uploaded clips** — `ml-pipeline/time-sync` operates on already-decoded
  sample arrays; extracting audio from an actual `.mp4` needs a dependency (ffmpeg/PyAV) not added here.
