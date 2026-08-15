# camera-calibration

**Status:** extrinsic-pose computation implemented and tested (Phase 2, second slice — docs/phases.md).
See `docs/adr/0005-camera-calibration-language-split.md` for why this package holds the calibration
*math* while camera/venue registration and profile storage live in `services/camera-calibration` (Go).

## What's here

- `pitch_geometry.py` — the standard cricket pitch's stump/crease dimensions (Law 6) as 3D reference
  points: 3 stump bases + 1 stump top at each end (8 points total), the "22-yard length, stump
  dimensions" reference set `prd.md` Section 8 names for AI-assisted calibration.
- `extrinsic.py` — `solve_pose(world_points, image_points, intrinsics)`, wrapping `cv2.solvePnP` +
  `cv2.projectPoints`, to recover a camera's position/orientation relative to the pitch from known
  point correspondences, plus the reprojection-error accuracy metric.

**Intrinsic (lens-distortion) calibration is not computed here** — per `architecture.md` Section 9, it's
now a standardized profile built once per camera model, not a live per-request computation; that's a
static seed table in `services/camera-calibration`'s Go domain package instead.

## No live caller yet

This is a pure, directly-tested function library — no FastAPI server, no HTTP endpoint, matching every
other still-unwired `ml-pipeline/` package. There's no real venue-setup workflow yet to be the caller;
`services/camera-calibration`'s `PUT .../calibration` endpoint currently expects rotation/translation/
reprojection-error values to already be computed and submitted, by hand or a future tool that calls
`solve_pose` directly. See `docs/adr/0005`'s "Revisit if" note.

## Test

```bash
cd ml-pipeline
pytest camera-calibration/tests -v --cov=camera_calibration
```

`tests/test_extrinsic.py` is the substantive one: it projects the known pitch reference points through a
deliberately-chosen ground-truth camera pose (using OpenCV's own `projectPoints`, independent of
`solve_pose`) to get synthetic "observed" image coordinates, then checks `solve_pose` recovers that same
pose and reports a near-zero reprojection error — and that deliberately corrupting a couple of
correspondences produces a detectably worse error. This is `docs/phases.md`'s Phase 2 testing
requirement — "calibration accuracy validated against measured pitch geometry" — exercised without any
real camera or venue.

## Setup

```bash
cd ml-pipeline
pip install -e ".[dev]"
```

Uses this level's shared `opencv-python`/`numpy` dependencies — no new dependencies were added for this
package.
