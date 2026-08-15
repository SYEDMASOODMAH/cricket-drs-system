# ADR-0007: DJI Action 5 Pro replaces the GoPro-class assumption; 1080p30 accepted as interim accessible-tier capture spec

**Status:** Accepted
**Date:** 2026-07-22

## Context

`docs/adr/0003-gopro-hardware.md` decided on a platform-provided "GoPro-class kit (120fps+), USB-C
tethered capture (webcam mode) to the edge box, with HDMI+capture-card as fallback." That ADR's own
"Revisit if" clause anticipated exactly this situation: "...or a different action-camera vendor become
the fallback options to re-evaluate."

The actual hardware purchased is a **DJI Action 5 Pro**, not a GoPro. Testing against it directly (OBS,
connected via USB-C in the camera's Webcam Mode) confirms:

- It does expose a standard **UVC (USB Video Class)** device when tethered — the core mechanism
  ADR-0003's architecture depends on works, no vendor-specific protocol needed.
- The advertised capture mode is **1920x1080 @ 30fps** — not the 120fps+ `architecture.md`'s
  Tier-Capability Matrix specifies for the accessible tier. This matches the general pattern flagged
  when researching this decision: action-camera "webcam modes" are built for video calls, not high-speed
  capture, and are typically capped well below the camera's native SD-card recording specs (which do
  include faster modes — e.g. up to 120fps at lower resolutions like 1080p or 2.7K — that aren't reachable
  through the USB-C webcam-mode path).

This is a real, material gap against a documented target, not a rounding error: at 30fps a ball
delivered at ~140km/h moves roughly 1.3m between frames; at 120fps that drops to ~0.32m. Trajectory
reconstruction accuracy (LBW line/impact-point estimation, edge/nick timing) degrades with the coarser
sampling.

## Options Considered

1. **Chase higher frame rates now** — check for a hidden webcam-mode quality setting, or move to an
   HDMI + capture-card path (ADR-0003's documented fallback) to try to reach the camera's higher native
   recording frame rates via a different capture path.
2. **Accept 1920x1080@30fps and build the capture pipeline against it now**, as an explicit, documented
   interim limitation — revisit the frame-rate question once the rest of the pipeline (buffer, sync,
   upload, and eventually CV processing) has something real to run against.

## Decision

**Option 2**, per your explicit choice. Building `edge-agent`'s real capture pipeline against confirmed,
working hardware now is more valuable than continuing to block on chasing a higher frame rate — every
other Phase 2 slice this session (Media Ingest Gateway, Camera Calibration, time sync) has been able to
move forward exactly because it stopped waiting for ideal conditions and built against what's honestly
available, with the gap documented rather than hidden.

`ADR-0003`'s underlying architecture — USB-C tethered capture, webcam-mode primary path, HDMI+capture-card
fallback — is **unchanged and still correct**. Only two specifics change: the vendor (GoPro → DJI Action
5 Pro) and the accepted frame rate (120fps+ target → 1080p30 actual, for now). `ADR-0003`'s Status line
is updated to point here for the current numbers.

## Consequences

**Easier:** `edge-agent`'s capture pipeline can be built and tested against real, physically-present
hardware today, rather than staying blocked indefinitely on procurement or a higher-fps path that hasn't
been validated yet.

**Harder:** the accessible tier's actual trajectory-reconstruction accuracy is materially worse than
`architecture.md`'s 120fps+ target implies, until this is revisited. Any accuracy claims made about the
accessible tier before that revisit should account for 30fps sampling, not 120fps.

**Revisit if:** a webcam-mode quality setting is found that unlocks a higher frame rate; the HDMI +
capture-card fallback is tried and confirmed to reach the camera's faster native recording modes; or
real accuracy testing (once the CV pipeline exists) shows 30fps is materially insufficient for reliable
LBW/edge-detection decisions, forcing the capture-path question sooner than otherwise planned.

## Addendum (2026-08-15): Windows dev-testing further capped at 640x480 for sustained capture

Building `edge-agent`'s real capture pipeline against this hardware surfaced two more gaps, both
specific to `pion/mediadevices`' Windows driver, not the camera or the architecture:

1. **Resolution ceiling.** The driver only exposes 640x480 and 1280x720 as exact-matchable modes for
   this camera — 1920x1080, the mode OBS itself shows as available, cannot be opened at all (every
   resolution above 1280x720 fails with "no driver fits the constraints"), including with
   `frame.FormatMJPEG` explicitly requested. Most likely this driver only enumerates a subset of the
   camera's raw/uncompressed modes, not the higher-resolution (likely MJPEG-compressed) ones OBS
   negotiates directly.
2. **1280x720 stalls after exactly one frame.** Even though it opens successfully, sustained reads at
   1280x720 reliably deliver exactly one frame and then hang indefinitely on the second — reproduced
   multiple times, independent of the reader's `copyFrame` setting. **640x480 sustains cleanly at a
   steady ~30fps with no such stall.** The most likely explanation is a USB bandwidth ceiling for
   uncompressed video at 1280x720 over this connection, not a code bug — see `internal/capture.Open`'s
   `NewReader` comment for where this was isolated.

Both are Windows-*driver* limitations specific to this library, not known to apply to the eventual
Linux/Jetson-class edge box, which uses `pion/mediadevices`' V4L2 driver instead — a different code
path. `edge-agent` requests 640x480@30 as an **ideal**, not exact, constraint precisely so it isn't
hardcoded to this Windows-specific ceiling; a Linux deployment will request the same ideal value but
isn't bound to what this Windows driver happens to expose or sustain.

**Revisit if:** this becomes worth chasing on Windows specifically (unlikely — Windows was always the
dev/test environment, not the production target); or Linux/V4L2 testing (once available) turns out to
have a similar unexpected ceiling, which would be a more serious finding worth its own investigation.

## Related

- `docs/adr/0003-gopro-hardware.md` (architecture this ADR builds on, superseded here only for
  vendor/fps specifics)
- `docs/architecture.md` Section 1a (Tier-Capability Matrix — 120fps+ target), Section 9 (CV Pipeline)
- `docs/phases.md` Phase 2 (edge capture/buffer agent task, completion criteria)
- `edge-agent/README.md`
