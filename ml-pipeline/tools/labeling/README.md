# labeling tool

**Dev-only utility — not a shipped `ml-pipeline` component.** Turns raw clips in
`ml-pipeline/dataset/raw-footage/` into labeled `(frame, ball position)` pairs under
`ml-pipeline/dataset/labels/` — the actual input a future `ball-tracking` model would train on. Neither
`dataset/raw-footage/` nor `dataset/labels/` nor `dataset/.frame-cache/` are committed (see
`.gitignore`'s `ml-pipeline/dataset/` entry) — these are large personal video files and per-clip
annotation state, not source code.

## Why this exists

Raw video isn't training data. A model learns from `(frame, correct answer)` pairs; nobody had produced
any of those for this project's footage until this tool existed. See `docs/phases.md`'s Phase 0
completion criteria (≥500 labeled clips) and Phase 3's data-scaling task for where this fits — this tool
produces the labels, it doesn't train anything itself (separate, later, GPU-dependent step).

## Run locally

Requires `ffmpeg`/`ffprobe` on `PATH` (same requirement `ml-pipeline/time-sync` already documents).

```bash
cd ml-pipeline
.venv\Scripts\activate
python -m uvicorn tools.labeling.server:app --port 8090
```

Then open `http://localhost:8090`. Pick a clip from the sidebar; click on the ball's position in each
frame (the frame auto-advances), or press `N` to explicitly mark a frame as "ball not visible" — the
tool distinguishes that from "not yet reviewed," since both are meaningfully different states.

| Action | How |
|---|---|
| Place the ball + advance | Click on it |
| Resize the label box | Scroll wheel over the canvas |
| Next / previous frame (no label change) | `→`/`Space` / `←` |
| Mark "not visible" + advance | `N` |
| Clear this frame's label | `Backspace` |

## Label format

One JSON file per clip, `dataset/labels/<clip-stem>.json`:

```json
{
  "clip": "DJI_20261001193458_0001_D-00.00.29.826-00.00.36.455-seg04.LRF",
  "fps": 29.97,
  "frame_count": 124,
  "labels": {
    "37": {"x": 640, "y": 310, "w": 24, "h": 24, "visible": true},
    "52": {"visible": false}
  }
}
```

`labels` is keyed by 1-based frame index (string, since JSON object keys are always strings). A frame
with no entry is "not yet reviewed." `{"visible": false}` is a deliberate negative label (the ball was
looked for and not found in that frame) — real signal for training, not the same as an unreviewed frame.
`x`/`y`/`w`/`h` are pixel coordinates in the *original* frame's resolution (not the browser's scaled
display size).

## Test

```bash
cd ml-pipeline
pytest tools/labeling/tests -v --cov=tools.labeling
```

Tests cover the pure logic (frame-cache population/reuse, label save/load/clear round-trips, the
"not visible" vs. "not yet reviewed" distinction) against a real video file (reusing `time-sync`'s
checked-in fixture) — not a canvas/UI test, which is verified manually in a browser instead.

## Known simplifications (tracked, not accidental)

- **No auto-detection/assist** — fully manual labeling; a pretrained generic detector suggesting
  candidate positions was considered and deliberately deferred.
- **Single local user, no concurrent-write handling** — each save rewrites the whole label file; fine
  for one person on one machine, not designed for multiple simultaneous labelers.
- **No export to a specific training framework's format** (COCO, YOLO, etc.) — the JSON format above is
  the source of truth; converting it is a separate, later step once there's an actual training pipeline
  to feed.
