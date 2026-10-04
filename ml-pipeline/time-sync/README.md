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
- `audio_extraction.py` — `extract_audio(video_path, sample_rate)` decodes real audio out of an actual
  video file via the `ffmpeg` binary (shelled out via `subprocess`, not a new pip dependency — see Setup
  below), and `find_offset_from_videos(reference_path, target_path, sample_rate)` chains that straight
  into `find_offset`. This is the real clips-in/offset-out path that used to not exist (see below).

**`find_offset` is proven against real audio, not just synthetic noise.** `tests/fixtures/real_sample.wav`
is a short clip trimmed from a real capture against the DJI Action 5 Pro's microphone
(`edge-agent/internal/capture.OpenMicrophone`) — `tests/test_real_audio.py` shifts it by a known amount
and confirms `find_offset` recovers that shift, the same technique the synthetic tests use but applied
to genuine signal content (transients, correlated harmonics, silence gaps) instead of Gaussian noise.
See `docs/adr/0006`'s "Revisit if" clause, which named this exact moment.

**`find_offset_from_videos` is proven against two real video files, not two pre-decoded arrays.**
`tests/fixtures/real_sample_video.mp4` is a ~3s/low-resolution trim of a real DJI Action 5 Pro recording
(video quality is irrelevant to this test — only the real audio track matters, which is why it was
re-encoded small rather than kept at source quality). `tests/test_audio_extraction.py` builds a second,
genuinely shifted copy via `ffmpeg`'s `adelay` filter (real silence prepended to a real audio track, not
a synthetic numpy shift) and confirms the whole real-file-in → offset-out path recovers it.

## No live caller yet

Pure, directly-tested function library — no FastAPI server, matching every other still-unwired
`ml-pipeline/` package. `media-ingest-gateway`'s sync endpoint still expects an offset + confidence score
to already be computed and submitted, by hand or a future tool that calls `find_offset_from_videos`
directly — the *extraction* half of that gap is closed (see above), but no service wires
capture→extract→correlate→submit together automatically yet; that stays deferred per
`docs/adr/0006`'s still-open sync-vs-async decision.

## Test

```bash
cd ml-pipeline
pytest time-sync/tests -v --cov=time_sync
```

`test_audio_correlation.py` projects a known reference signal through a deliberately chosen shift (with
and without added noise) to build a synthetic "second camera" track with a known ground-truth offset,
then checks `find_offset` recovers it — plus sanity checks that two genuinely unrelated signals score low
and a self-correlation is exactly zero-offset/maximum-confidence. `test_real_audio.py` does the same
shift-and-recover check against the real captured fixture instead. `test_audio_extraction.py` goes one
level further — real video files in, not pre-decoded arrays — plus negative-path tests (missing file,
non-video file) for `extract_audio`. Together this is `docs/phases.md`'s Phase 2 testing intent —
validating sync accuracy — exercised without any real venue or multi-camera match.

## Setup

```bash
cd ml-pipeline
pip install -e ".[dev]"
```

Uses this level's shared `numpy` dependency only — no new *pip* dependency was added for this package.
`audio_extraction.py` does need the **`ffmpeg` binary on `PATH`**, though — a genuinely new kind of
dependency for this codebase (an external tool, not something `pip install` manages). Already present on
this dev machine and pre-installed on GitHub Actions' `ubuntu-latest` runners (this repo's CI), so no
extra install step was needed either locally or in CI — but if you're setting this up somewhere else,
install `ffmpeg` separately and make sure it resolves on `PATH` before running `test_audio_extraction.py`.
