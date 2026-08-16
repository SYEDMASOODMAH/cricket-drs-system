# camera-calibration

**Status:** Phase 2 slice implemented — registers physical cameras to venues and stores/evaluates their
calibration profiles. See `/docs/architecture.md` Section 5 for this service's overall responsibilities,
`/docs/adr/0005-camera-calibration-language-split.md` for why this service is split across Go (this
package) and Python (`ml-pipeline/camera-calibration`), and `/docs/phases.md` Phase 2 for what's still
ahead (the edge-agent's actual camera capture, real field validation).

## Architecture

Hexagonal, same shape as the other three services — see identity-access's README for the general
pattern.

```
internal/
  domain/       Camera, CalibrationProfile (+ Valid() threshold check), LensProfile seed table,
                Role enum + CanManageCalibration, sentinel errors
  service/      RegisterCamera / GetCamera / ListCameras / StoreCalibrationProfile /
                GetCalibrationStatus / GetLensProfile use-cases, CameraRepository + ProfileRepository ports
  memstore/     in-memory adapters for both repositories, tenant-scoped
  security/     JWT verify-only adapter — this service never issues tokens, only validates ones
                Identity & Access minted (see "Shared auth" below)
  httpapi/      chi router + handlers
```

## Go owns registration/storage; Python owns the calibration math

See `docs/adr/0005` for the full reasoning. In short: this service treats a calibration profile's
`rotation`/`translation`/`reprojection_error_px` as **already-computed values submitted to it** — it
never runs `cv2.solvePnP` or any CV computation itself. The actual pose-estimation algorithm lives in
`ml-pipeline/camera-calibration` as a standalone, synthetic-fixture-tested Python function library.
**No live call exists between the two in this slice** — there's no real venue-setup workflow yet to be
the caller, so wiring one now would be coupling built ahead of a concrete need (same reasoning
media-ingest-gateway's deferred `matchID` cross-validation already established for this codebase).

## Intrinsic vs. extrinsic calibration

- **Intrinsic** (lens distortion): a small hardcoded seed table in `internal/domain/lens_profile.go`,
  keyed by `CameraModel`. `architecture.md` Section 9 explains why this is reference data rather than a
  live computation — the platform now standardizes on specific GoPro-class hardware, so one profile per
  *model* is built once and reused, not re-derived per physical unit. **The seeded coefficients are
  structurally-shaped placeholders, not measured values** — pending real numbers from Phase 0's field
  validation.
- **Extrinsic** (camera position/orientation relative to the pitch): computed per venue, submitted via
  `PUT .../calibration`. `CalibrationProfile.Valid()` checks the submitted `reprojection_error_px`
  against `domain.MaxReprojectionErrorPx` (2.0px — also a placeholder pending real accuracy targets) to
  answer the "pre-match camera detection and calibration/health check" `prd.md` Section 5.4 describes.

## Shared auth with the other three services

The `Role` enum and JWT-verification logic used to be hand-copied a 4th time here — now they're a thin
wrapper around `services/platformauth`, a shared, non-`internal/` package importable across every
service in this module (`docs/adr/0008-platformauth-shared-package.md`, which resolves the deferral
`docs/adr/0005` originally noted). `CanManageCalibration` (this service's own authorization decision)
still lives locally — only the shared identity/token vocabulary moved.

**Consequence: all four services must be started with the same `JWT_SIGNING_KEY`** (or all four left
unset, in which case they share the same committed insecure dev-only fallback,
`platformauth.InsecureDevSigningKey` — see identity-access's README for why that fallback exists and
what it's for).

## Cross-service `CameraID` validation

`media-ingest-gateway`'s clip-upload path (`internal/cameracalibration.Client`) calls this service's
`GET .../cameras/{cameraID}` before accepting an upload, rejecting any `camera_id` that isn't registered
here — closing the gap this section used to describe as outstanding. The registration endpoint itself
didn't change: it was already org-scoped and already returned 404 for an unknown camera, which is exactly
what that check relies on.

## Run locally

```bash
go run ./cmd
```

Health check: `GET http://localhost:8080/healthz`

### Configuration (environment variables)

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | |
| `JWT_SIGNING_KEY` | shared insecure dev-only key | Must match the other three services' — see "Shared auth" above |

### Example walkthrough (assumes Identity & Access is running and you have a token — see its README)

```bash
# Register a camera
curl -s -X POST "localhost:8080/v1/organizations/<org id>/cameras" \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"venue_id":"venue-1","model":"GoPro Hero 12 Black"}'

# Submit a calibration profile (rotation/translation from ml-pipeline/camera-calibration's solve_pose)
curl -s -X PUT "localhost:8080/v1/organizations/<org id>/cameras/<camera id>/calibration" \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"rotation":[0.1,0.2,0.3],"translation":[1,2,3],"reprojection_error_px":0.6}'

# Check calibration status
curl -s "localhost:8080/v1/organizations/<org id>/cameras/<camera id>/calibration" \
  -H "Authorization: Bearer <token>"

# Look up a camera model's standardized lens profile
curl -s "localhost:8080/v1/camera-models/GoPro%20Hero%2012%20Black/lens-profile" \
  -H "Authorization: Bearer <token>"
```

Full endpoint set: `openapi.yaml`.

## Test

```
go test ./... -cover
```

All packages are at or above the 80% line-coverage target in `rules.md` Section 6.1. Tenant isolation and
the calibration validity threshold are both exercised directly in `internal/service` and
`internal/httpapi`.

For the extrinsic-calibration algorithm's own tests (synthetic pitch-geometry fixtures), see
`ml-pipeline/camera-calibration/README.md`.

## Known Phase 2 simplifications (tracked, not accidental)

- **No live Go↔Python wiring** — the calibration algorithm's output must currently be submitted by hand
  (or a future venue-setup tool) via `PUT .../calibration`; see "Go owns registration/storage" above.
- **Placeholder lens-distortion and validity-threshold numbers** — structurally correct, not measured;
  pending real Phase 0 field-validation data.
- **AI-assisted auto-calibration for ad-hoc setups** (`prd.md` Section 8) is explicitly Post-MVP, not
  attempted here.
