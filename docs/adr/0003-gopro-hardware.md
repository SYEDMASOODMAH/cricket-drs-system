# ADR-0003: Platform-provided GoPro kit (USB-C tethered) as accessible-tier hardware

**Status:** Accepted
**Date:** 2026-07-18

## Context

The original accessible-tier hardware assumption in `prd.md`/`architecture.md` was "bring your own
commodity smartphone," with the club/venue owning or renting the camera. The actual business model is a
**subscription** where the platform owns and provides the camera hardware as part of the service — this
changes both the product economics and what's architecturally sensible.

## Options Considered

1. **Continue with bring-your-own-phone.** No hardware cost to the platform, but inconsistent camera
   specs per venue (frame rate, lens, FOV), harder calibration, and doesn't fit a hardware-inclusive
   subscription model.
2. **Platform-provided GoPro-class kit, native WiFi livestream to the edge box.** Consistent hardware,
   but GoPro's built-in RTMP/WiFi livestreaming is consumer-grade — capped resolution, requires a phone
   as bridge, known reliability issues over sustained sessions. Reintroduces the wireless-reliability
   risk the Section 9a local-inference fallback was designed to route around.
3. **Platform-provided GoPro-class kit, USB-C tethered capture (webcam mode) to the edge box, with
   HDMI+capture-card as fallback for models without webcam mode.** Consistent hardware, wired and
   reliable venue-local capture, same cable delivers power (solving multi-hour battery/thermal limits),
   standard UVC/HDMI capture integration — no GoPro-specific protocol work needed in the edge agent.

## Decision

**Option 3:** platform-provided GoPro-class kit (2–4 cameras, 120fps+), USB-C tethered capture to the
edge box, bundled into the subscription.

## Consequences

**Easier:** consistent, known camera specs simplify calibration (one lens-distortion profile per
supported camera model instead of arbitrary phone lenses); higher frame rate (120fps+ vs. 30–60fps)
genuinely improves edge-detection and run-out timing precision; wired capture removes an entire class of
WiFi-related dropped-frame/reconnect failure; USB-C power solves the multi-hour battery/thermal problem
in principle.

**Harder:** the platform now carries hardware cost, logistics (kit shipping/provisioning per venue), and
maintenance/replacement responsibility — this must be reflected in subscription pricing (see
`architecture.md` Section 19 directional cost model) and is a genuine operational function the business
didn't have before (hardware fleet management, not just software).

**Revisit if:** Phase 0/2 field validation shows USB-C tethered capture is less reliable in practice
than expected (e.g., specific GoPro models drop webcam-mode connection under heat), or if the per-kit
cost makes the subscription price point untenable against Phase 0's business-model validation findings —
in which case HDMI+capture-card or a different action-camera vendor become the fallback options to
re-evaluate.

## Related

- `prd.md` Section 10 (NFRs — Portability), Section 13.5 (Hardware Kit Provisioning), Section 14
  (Key Assumptions).
- `architecture.md` Section 1a (Tier-Capability Matrix), Section 9 (CV Pipeline — calibration),
  Section 10 (Video Processing Pipeline — transport), Section 14 (Cloud Architecture — edge compute).
- `phases.md` Phase 0 (GoPro kit field validation), Phase 2 (USB-C tethered capture implementation).
