# time-sync

**Status:** correlation algorithm implemented and tested (Phase 2, third slice — docs/phases.md). See
`docs/adr/0006-time-sync-language-split.md` for why this package holds the sync *math* while storing a
computed offset against a clip lives in `services/media-ingest-gateway` (Go), which
`architecture.md` Section 5 assigns "time-synchronizes multi-camera feeds" to directly.

## What's here

- `audio_correlation.py` — `find_offset(reference, target, sample_rate)`, FFT-based normalized
  cross-correlation of two 1-D audio signals, returning a signed time offset (ms) and a confidence score
  (`services/media-ingest-gateway`'s `CalibrationProfile.Valid()`-style threshold check happens on the Go
  side against this score). No new dependency: plain NumPy (`fft`/`ifft`), no `scipy`.

**Real audio extraction from an actual uploaded video clip is not implemented here** — that needs a
decode dependency (ffmpeg/PyAV) this package doesn't declare. `find_offset` operates on already-decoded
1-D sample arrays; producing those from a real `.mp4` is deferred, same as
`ml-pipeline/camera-calibration` defers real checkerboard/pitch photography.

**`find_offset` is proven against real audio, not just synthetic noise.** `tests/fixtures/real_sample.wav`
is a short clip trimmed from a real capture against the DJI Action 5 Pro's microphone
(`edge-agent/internal/capture.OpenMicrophone`) — `tests/test_real_audio.py` shifts it by a known amount
and confirms `find_offset` recovers that shift, the same technique the synthetic tests use but applied
to genuine signal content (transients, correlated harmonics, silence gaps) instead of Gaussian noise.
See `docs/adr/0006`'s "Revisit if" clause, which named this exact moment.

## No live caller yet

Pure, directly-tested function library — no FastAPI server, matching every other still-unwired
`ml-pipeline/` package. There's no real audio-extraction pipeline yet to be the caller;
`media-ingest-gateway`'s sync endpoint currently expects an offset + confidence score to already be
computed and submitted, by hand or a future tool that calls `find_offset` directly.

## Test

```bash
cd ml-pipeline
pytest time-sync/tests -v --cov=time_sync
```

`test_audio_correlation.py` projects a known reference signal through a deliberately chosen shift (with
and without added noise) to build a synthetic "second camera" track with a known ground-truth offset,
then checks `find_offset` recovers it — plus sanity checks that two genuinely unrelated signals score low
and a self-correlation is exactly zero-offset/maximum-confidence. `test_real_audio.py` does the same
shift-and-recover check against the real captured fixture instead. Together this is `docs/phases.md`'s
Phase 2 testing intent — validating sync accuracy — exercised without any real venue or multi-camera
match.

## Setup

```bash
cd ml-pipeline
pip install -e ".[dev]"
```

Uses this level's shared `numpy` dependency only — no new dependencies were added for this package.
