# edge-agent

**Status:** scaffold only (Phase 1). Implementation begins Phase 2 — Video Ingestion & Camera Management
(see `/docs/phases.md`).

Runs on venue capture hardware. Responsibilities (per `/docs/architecture.md` Sections 2, 9a, and 10;
hardware model per ADR-0003):

- **Capture cameras via USB-C tethered connection** (UVC webcam mode, read via `v4l2`/OpenCV; HDMI +
  capture card as a documented fallback for camera models without webcam mode) — not a network/WiFi
  stream. Each connected camera corresponds to a platform-provided GoPro-class unit registered with a
  known ID (`docs/prd.md` Section 13.5).
- Maintain a rolling buffer (target: last 20–30s) per connected camera.
- Time-synchronize multi-camera feeds (software/audio-fingerprint sync for accessible tier).
- Measure uplink bandwidth and select cloud vs. local-inference mode (`docs/architecture.md` Section 9a).
- On review trigger, push the relevant buffered window to the cloud Media Ingest Gateway via SRT/WebRTC
  — this is the *only* leg of the pipeline exposed to venue internet quality; camera-to-edge-box capture
  is wired and has no WiFi dependency.

Deliberately a separate Go module from `/services` — it ships to different hardware with a different
deployment lifecycle (see `/docs/architecture.md` Section 14, edge compute).

## Run locally

\`\`\`
go run ./cmd
\`\`\`
