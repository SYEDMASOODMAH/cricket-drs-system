# ADR-0009: WebRTC, not SRT, for the edge-box-to-cloud upload leg

**Status:** Accepted
**Date:** 2026-08-16

## Context

`docs/architecture.md` Section 10 specifies a transport for the edge-box-to-cloud leg: SRT preferred
for its resilience on variable club-venue networks (the accessible tier this project targets), WebRTC
named as an alternative "for lowest-latency broadcast-tier scenarios with good connectivity." Section 4
confirms this is a *transport* choice only — the trigger→push-the-buffered-window model stays the same;
nothing here asks for continuous live streaming. `docs/phases.md`'s Phase 2 task list names this
explicitly, and both `edge-agent` and `media-ingest-gateway`'s READMEs have flagged "plain HTTP, not
SRT/WebRTC" as a tracked simplification since those services first shipped.

## What was actually verified

- `github.com/pion/webrtc/v4` was **already a transitive dependency** of `edge-agent` (pulled in by
  `pion/mediadevices`, confirmed in `go.sum` before this change) and is **pure Go** — no cgo, no native
  toolchain requirement.
- A real SRT path means either a cgo binding to the native `libsrt` C library (not installed anywhere in
  this environment — a heavier ask than the MinGW-w64 compiler this session already needed just to build
  `pion/mediadevices`' Windows camera driver, ADR-0007), or a pure-Go reimplementation
  (`datarhei/gosrt`) whose current maturity/API this environment has no way to verify (no internet
  access from these tools).
- No prior ADR had decided this either way — searched all of `docs/adr/` for "SRT"/"WebRTC"/"transport";
  the choice was fully open.

## Decision

**Build WebRTC, not SRT, for this slice**, as an **additional** transport alongside the existing plain
HTTP upload path (`internal/uploader`), not a replacement — selected via edge-agent's
`UPLOAD_TRANSPORT` env var, defaulting to `http` so existing behavior is unchanged unless explicitly
opted in.

This is an explicit, acknowledged deviation from architecture.md's *stated preference* (SRT is meant to
be primary for the accessible/club-venue tier). The deviation is justified by buildability, not by
disagreeing with the architectural reasoning: WebRTC was buildable and verifiably testable in this
environment today (pure Go, real end-to-end tests against real `pion/webrtc` peer connections and data
channels — see `edge-agent/internal/webrtcupload` and
`services/media-ingest-gateway/internal/httpapi/webrtc.go`'s test suites), while a real SRT
implementation would have either repeated this session's cgo/native-toolchain cost or required trusting
an unverifiable dependency.

Concretely: a single-round-trip HTTP signaling exchange (SDP offer/answer) against a new
`POST /v1/organizations/{orgID}/matches/{matchID}/clips/webrtc-offer` endpoint, followed by the clip
bytes flowing directly between edge-agent and Media Ingest Gateway over an ICE-negotiated, ordered/
reliable WebRTC data channel, acked back to the sender once the server has stored the clip via the exact
same `service.UploadClip` call the HTTP path already uses.

## Consequences

**Easier:** a second, real, tested transport option exists without touching the proven HTTP path at
all; `pion/webrtc` needed no new native toolchain, unlike every other hardware/media dependency this
session has touched.

**Harder:** `pion/webrtc/v4` becomes a **new direct dependency of `services/media-ingest-gateway`** —
previously that module had zero real-time-media dependencies. It's a substantial dependency (ICE, DTLS,
SCTP, the whole WebRTC stack) landing in the core services module for a feature only one client
(edge-agent) uses so far.

**Not done:** SRT itself is still unbuilt. If a real venue test surfaces WebRTC/ICE connectivity
problems (NAT traversal without STUN/TURN, which this slice deliberately doesn't configure — see
`edge-agent/README.md`'s "Known simplifications") that SRT's design specifically avoids, that's the
concrete signal to revisit this decision, not a hypothetical one.

## Related

- `docs/architecture.md` Section 10 (Video Processing Pipeline — transport), Section 4 (Data Flow)
- `docs/adr/0007-dji-action5-hardware.md` (the cgo/MinGW-w64 precedent this ADR's cost comparison rests
  on)
- `edge-agent/internal/webrtcupload/`, `services/media-ingest-gateway/internal/httpapi/webrtc.go`
