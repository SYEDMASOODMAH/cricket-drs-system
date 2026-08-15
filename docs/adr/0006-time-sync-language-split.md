# ADR-0006: Multi-camera time sync split across `services/media-ingest-gateway` (Go) and `ml-pipeline/time-sync` (Python)

**Status:** Accepted
**Date:** 2026-07-21

## Context

`phases.md`'s Phase 2 task list names "Multi-camera time synchronization approach (software/
audio-fingerprint sync for accessible tier)" as a deliverable. Unlike Camera Calibration Service
(ADR-0005), this one isn't ambiguous about where it lives: `architecture.md` Section 5's Major
Components table assigns it directly to Media Ingest Gateway — "Accepts venue video streams, buffers,
**time-synchronizes** multi-camera feeds" — rather than giving it its own row. No document specifies an
algorithm beyond "software/audio-fingerprint sync," and Section 18 (Key Architectural Risks) flags it as
unsolved R&D needing dedicated time.

The actual cross-correlation computation is still a numerical signal-processing problem, squarely inside
`architecture.md` Section 7's "CV/ML pipeline: Python is non-negotiable" guidance — the same tension
ADR-0005 already resolved for Camera Calibration between "this is a Go-shaped registration/storage
feature" and "this is a Python-shaped computation."

## Options Considered

1. **All-Go** — implement cross-correlation directly in `media-ingest-gateway` (Go has no comparable
   signal-processing ecosystem to NumPy/SciPy; this would mean hand-rolling FFT or a naive O(n²)
   correlation with no test-proven numerical library backing it).
2. **All-Python service** — build a standalone sync service in `ml-pipeline/`, mirroring Camera
   Calibration's Option 2 (rejected there for the same reason: no other `ml-pipeline/` package runs as
   its own deployed service yet, and there's no live caller to justify one here either).
3. **Split, following ADR-0005's precedent exactly:** the correlation algorithm lives in
   `ml-pipeline/time-sync/` as a pure, synthetic-fixture-tested function library; `media-ingest-gateway`
   (which already owns `Clip`, per `architecture.md` Section 5) stores an already-computed offset and
   confidence score against a clip. No live call between the two.

## Decision

**Option 3**, for the same reasons ADR-0005 gives, not re-derived here: the storage half
("which clip is offset by how much relative to which other clip, and how confident is that estimate")
is indistinguishable in shape from every other tenant-scoped record this session's Go services already
manage, while the correlation math itself belongs with NumPy in `ml-pipeline/`. The one difference from
ADR-0005: there's no separate "registration" concept to stand up here — `Clip` already exists in
`media-ingest-gateway`, so this extends that domain type rather than creating a new service or even a
new top-level resource.

As with Camera Calibration, no dependency is added for real audio extraction from an uploaded video file
(would need ffmpeg/PyAV) — the algorithm is tested against synthetic audio signals (a known signal, a
deliberately time-shifted noisy copy) rather than real footage, the same "test the math, defer the real-
media plumbing" split ADR-0005 established for checkerboard/pitch imagery.

## Consequences

**Easier:** no new service to stand up or wire into `.claude/launch.json`; the sync feature rides on
`media-ingest-gateway`'s existing auth, tenant isolation, and `Clip` storage. The correlation algorithm
is independently testable with zero dependency on real audio or video files.

**Harder:** same gap as ADR-0005 — no automated test proves the two halves work together end-to-end
(a real caller extracting audio from an uploaded clip, running `find_offset`, and submitting the result)
because that caller doesn't exist yet.

**Revisit if:** real audio-from-video extraction is built and the correlation needs to run synchronously
as part of the upload path — at that point, decide whether `media-ingest-gateway` calls out to a newly
wrapped `ml-pipeline` HTTP endpoint or whether extraction+correlation becomes an async job.

## Related

- `docs/adr/0005-camera-calibration-language-split.md` (direct precedent for this decision)
- `docs/architecture.md` Section 5 (Major Components — sync assigned to Media Ingest Gateway), Section 7
  (language recommendations), Section 18 (Key Architectural Risks — sync flagged as unsolved R&D)
- `docs/phases.md` Phase 2 (task list, completion criteria's undefined "tolerance")
- `services/media-ingest-gateway/internal/domain/clip.go`
