# ML Pipeline (Python)

CV/ML components per `docs/architecture.md` Section 8/9. Python is used here specifically because the
CV/ML ecosystem (PyTorch, OpenCV) is Python-first — see `docs/architecture.md` Section 7 for why this
is the one deliberate exception to the Go-everywhere backend rule.

| Module | Responsibility |
|---|---|
| `ball-tracking` | Ball detection + trajectory reconstruction (2D/3D) |
| `edge-detection` | Audio+visual fusion for bat-ball contact |
| `lbw-engine` | Physics + ML residual LBW trajectory model |
| `runout-detection` | Bail-motion + pose-based run-out/stumping analysis |
| `model-registry` | Model versioning, evaluation, promotion/rollback |
| `camera-calibration` | Extrinsic camera-pose estimation against known pitch geometry (Phase 2) |
| `time-sync` | Multi-camera time sync via audio cross-correlation (Phase 2) |

All seven share one `pyproject.toml` at this level — a single dependency set and environment for now,
since they're tightly coupled by shared preprocessing (calibration, frame sync) and will likely run in
the same GPU-backed service pool early on. Split into independent packages if/when they need
independent deployment or dependency versions.

## Setup

```bash
cd ml-pipeline
python -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"
```

## Conventions (see `docs/rules.md` Section 6.3)

- No model reaches production without a documented accuracy report per tier (accessible vs. broadcast).
- Every model version is registered via `model-registry` with metrics tracked before promotion.
- Regression benchmark sets (real + adversarial/edge-case footage) are run against every new model
  version — never just against the training set.

## Test / lint

```bash
pytest
ruff check .
mypy .
```
