# ADR-0005: Camera Calibration Service split across `services/` (Go) and `ml-pipeline/` (Python)

**Status:** Accepted
**Date:** 2026-07-20

## Context

`phases.md`'s Phase 2 task list names "Camera Calibration Service: standardized GoPro-class
lens-distortion profile (built once) plus per-venue extrinsic calibration and per-camera-unit
registration/ID" as a deliverable. Where this service's code should live is genuinely ambiguous from
existing docs:

- `architecture.md` Section 5's Major Components table lists "Camera Calibration Service" as a row
  alongside Match & Tournament, Identity & Access, Media Ingest Gateway, etc. — all of which are Go,
  per Section 7's "core application services: Go" recommendation.
- `architecture.md` Section 7 also states "CV/ML pipeline: Python is non-negotiable here — the ML/CV
  ecosystem (PyTorch, OpenCV, NumPy) is Python-first."
- `architecture.md` Section 9 ("Computer Vision Pipeline") places calibration — "intrinsic (lens
  distortion) and extrinsic (camera position relative to known pitch geometry) calibration" — as item 1
  of that pipeline, i.e. inside the Python-owned CV layer.
- No existing ADR resolves this. ADR-0001 settles Go vs. Java for core services and just notes Python is
  "ecosystem-mandated" for ML without discussing calibration specifically. ADR-0003 (GoPro hardware)
  discusses calibration only as a hardware consequence, not a service-boundary decision.

So the same feature name ("Camera Calibration Service") appears once as a peer of Go application
services and once as an item inside the Python CV pipeline. This ADR resolves that by splitting the
feature along the same line `architecture.md` Section 7 already draws for every other kind of work:
structured state/API vs. numerical CV computation.

Separately: `architecture.md` Section 9's own revision note observes that since the platform now
provides a fixed GoPro-class kit (ADR-0003) rather than arbitrary phones, **intrinsic** calibration
(lens distortion) is a profile "built once per supported camera model, not re-derived per unit" — i.e.
reference data, not a live per-request computation. Only **extrinsic** calibration (a given physical
camera's position/orientation relative to the pitch at a given venue) is computed per venue setup.

## Options Considered

1. **All-Go** — model calibration profiles as opaque data the Go service stores; never implement the
   actual pose-estimation math in this repo, deferring it entirely to a future slice.
2. **All-Python** — build the whole service (including HTTP API, tenant-scoped registration, storage) in
   `ml-pipeline/`, mirroring the other five CV/ML sub-packages, even though its registration/storage half
   has none of their GPU/ML characteristics and none of them currently run their own service process.
3. **Split: Go owns registration/profile storage, Python owns the extrinsic-pose algorithm** — a new
   `services/camera-calibration/` Go service (registration, calibration-profile storage, calibration
   status/validity), plus a new `ml-pipeline/camera-calibration/` Python module (pure function library:
   given known pitch-geometry reference points and their observed image coordinates, compute extrinsic
   pose and reprojection error via OpenCV). No live call between the two in this slice — the Go service
   just stores whatever extrinsic values are submitted to it, the same way `media-ingest-gateway` stores
   an opaque `matchID` without validating it against `match-tournament`.

## Decision

**Option 3.** This is the only option consistent with both halves of `architecture.md` Section 7's
language guidance simultaneously, rather than picking one guideline and ignoring the other:

- Registration ("which camera is this, which venue is it at, is it currently calibrated") is
  indistinguishable in shape from every other tenant-scoped CRUD+status feature already built this
  session (Match & Tournament's rosters, Media Ingest Gateway's clip metadata) — it belongs in Go, in
  the same hexagonal shape as those services.
- The extrinsic-pose computation is a numerical CV problem (camera resectioning via `solvePnP`-style
  pose estimation) — squarely inside "the ML/CV ecosystem (PyTorch, OpenCV, NumPy)" Section 7 already
  calls non-negotiably Python, and `ml-pipeline/` already declares `opencv-python`/`numpy` as
  dependencies for exactly this kind of work.
- Because intrinsic calibration is now reference data ("built once per camera model"), it doesn't need a
  live computation step at all — it's represented as a small seed table in the Go service's domain
  package, not a third component.

No live integration between the two halves is built in this slice: there is no real venue-setup
workflow yet to be the caller, so wiring one now would be coupling built ahead of a concrete need — the
same reasoning `media-ingest-gateway`'s deferred `matchID` cross-validation already established as this
project's convention for exactly this situation.

## Consequences

**Easier:** each half stays consistent with its layer's established language convention — no special
case, no service that's "Go except for one Python-shaped corner" or vice versa. The extrinsic-pose
algorithm is testable in complete isolation (synthetic pitch-geometry fixtures with a known ground-truth
pose), independent of the Go service's own test suite.

**Harder:** the two halves currently have no automated integration test proving they'd actually work
together end-to-end (a technician's tool or venue-setup workflow calling the Python function and
submitting its output through the Go API) — because that caller doesn't exist yet. This is a known gap,
not an oversight; closing it is future work once a real invocation path exists.

**Revisit if:** a real venue-setup/calibration workflow is built and needs to call the Python
computation synchronously from a request path — at that point, decide whether that's an HTTP call to a
newly-wrapped FastAPI service (`ml-pipeline/` already depends on `fastapi`/`uvicorn` for this eventuality)
or a batch/offline job feeding results into the Go service's storage API.

## Related

- `docs/architecture.md` Section 5 (Major Components), Section 7 (language recommendations), Section 9
  (Computer Vision Pipeline, calibration item)
- `docs/prd.md` Section 13.5 (camera kit registration, calibration profile reuse per camera/venue),
  Section 8 (AI-assisted auto-calibration, explicitly deferred — Advanced/Post-MVP)
- `docs/adr/0001-go-for-core-services.md`, `docs/adr/0003-gopro-hardware.md`
- `services/media-ingest-gateway/internal/domain/ids.go` (`CameraID`'s deferred-cross-validation note,
  which this ADR's service now exists to eventually satisfy)
