# edge-agent

**Status:** Phase 2 slice implemented — captures real video from a USB-C tethered camera, buffers it,
and uploads a triggered window to Media Ingest Gateway. See `/docs/architecture.md` Sections 2, 9a, and
10 for this component's overall responsibilities, `docs/adr/0003` and `docs/adr/0007` for the hardware
decisions this build targets, and `/docs/phases.md` Phase 2 for what's still ahead.

## First real hardware this session

Every other service built this session (Media Ingest Gateway, Camera Calibration, time sync) tested
against synthetic fixtures because no real camera existed yet. This one is different: it was built and
verified against a real **DJI Action 5 Pro**, tethered via USB-C in Webcam Mode, confirmed via OBS to
present as a standard UVC device at 1920x1080 @ 30fps — see `docs/adr/0007` for why that's accepted here
instead of the `architecture.md` 120fps+ target, what that tradeoff costs in trajectory precision, and
its addendum for two more gaps found while actually building against it: `pion/mediadevices`' Windows
driver can't open above 1280x720 at all, and 1280x720 itself stalls after exactly one frame (a real,
reproduced, resolution-specific sustained-capture failure — see "Known simplifications"). **The
confirmed, sustained-working configuration in this environment is 640x480@30.**

## Architecture

```
internal/
  capture/      wraps github.com/pion/mediadevices — Open()/OpenMicrophone() start UVC video/audio
                streams, Read() returns an owned buffer.Frame/AudioChunk, Close() releases the device
  buffer/       RingBuffer + AudioRingBuffer: thread-safe, time-window retention (Add/Snapshot)
  clipformat/   Encode/Decode — a simple length-prefixed JPEG-sequence container, not a real
                video format (see "Known simplifications" below)
  wav/          minimal hand-written WAV encoder — the audio export/interop format (see "Audio capture")
  uploader/     HTTP client posting encoded clips to Media Ingest Gateway's existing upload endpoint
  webrtcupload/ WebRTC data-channel client, an alternative transport to uploader/ (see "Upload transport")
  config/       env-var driven runtime configuration
cmd/main.go     wires config → capture loops (goroutines feeding the ring buffers) → a tiny local HTTP
                server (GET /healthz, POST /trigger, GET /audio-snapshot)
```

Deliberately a separate Go module from `/services` — it ships to different hardware with a different
deployment lifecycle (`architecture.md` Section 14, edge compute).

## Why `pion/mediadevices`, not "v4l2/OpenCV"

The docs' original phrasing implied `gocv` (Go's OpenCV bindings). On Windows, `gocv` needs a from-source
OpenCV build (MinGW-W64 + CMake, over an hour) — impractical for iterating against real hardware.
`pion/mediadevices` is cross-platform: its Linux backend is V4L2 (the actual thing the docs named, and
what the real Jetson-class edge box will use), while Windows/Mac get native backends for dev-time
testing on a normal machine. Raw frame capture needs no video-codec library.

**One real cost of this choice, discovered while building it:** the Windows camera driver
(`camera_windows.go`) wraps Windows Media Foundation through C++ (`camera_windows.cpp`), which needs
**cgo and a C compiler** to build — not the "zero native dependency" situation initially assumed. This
environment had `CGO_ENABLED=0` and no C compiler at all; getting this to build required installing
MinGW-w64 (`winget install BrechtSanders.WinLibs.POSIX.UCRT`) — a real, if much smaller than `gocv`'s,
native-toolchain requirement. The Linux driver (the actual production target) has no such requirement —
it's pure Go.

## Audio capture

`internal/capture.OpenMicrophone` proves the other half of `docs/adr/0006`'s "Revisit if" clause: real
audio capture now exists, and `ml-pipeline/time-sync`'s `find_offset` algorithm has been proven against
it (a real captured sample, synthetically shifted by a known amount — see that package's
`tests/test_real_audio.py`), not just synthetic Gaussian noise. `GET /audio-snapshot` exports the
current audio buffer as a `.wav` download for inspection or feeding into a Python script — a local
verification/export mechanism, deliberately separate from `/trigger`'s video-upload path since a live
Go→Python correlation bridge remains a deferred decision (see "Known simplifications").

Building this surfaced two real bugs, both found only by testing against the actual microphone:

- **Stereo, not mono.** `OpenMicrophone` requests `ChannelCount=1` as an ideal constraint, but this
  device delivers 2 channels regardless. `Microphone.Read` now averages all channels down to mono
  unconditionally, so the rest of the pipeline (`AudioChunk`, `AudioRingBuffer`, the WAV export) can stay
  simply mono — matching `find_offset`'s expected 1-D input — no matter what a given device actually
  provides.
- **Chunk timestamps from arrival time were wrong.** The microphone driver can deliver a backlog of
  already-captured audio in a rapid burst, which — when each chunk was stamped with `time.Now()` at the
  moment `Read()` happened to return — gave many chunks nearly-identical recent timestamps regardless of
  when their audio was actually captured, defeating `AudioRingBuffer`'s time-window pruning (a 20s
  window held 40s of real audio in initial testing). Fixed by deriving each chunk's timestamp from
  `streamStart` plus its position in the audio timeline (cumulative samples / sample rate) instead.

## Upload transport

`handleTrigger` pushes the encoded clip through whichever `Uploader` `cmd/main.go` wired up
(`UPLOAD_TRANSPORT`, see the config table below):

- **`http`** (`internal/uploader`, the default) — a plain authenticated `POST` of the whole clip.
- **`webrtc`** (`internal/webrtcupload`, `docs/adr/0009`) — `docs/architecture.md` Section 10's WebRTC
  option for this leg. Signaling is a single HTTP round-trip (an SDP offer/answer exchange against Media
  Ingest Gateway's `POST .../clips/webrtc-offer`); the clip bytes then flow directly between the two
  processes over an ICE-negotiated, ordered/reliable data channel, chunked and acked once the server has
  fully stored the clip. Chosen over SRT because `pion/webrtc` was already a transitive dependency (pure
  Go, no new native toolchain — unlike a real SRT implementation, which needs either `libsrt` via cgo or
  an unverified pure-Go library) — see the ADR for the full reasoning, including the explicit deviation
  from architecture.md's stated *preference* for SRT.

Both implement the same small `Upload(ctx, token, orgID, matchID, cameraID, clipBytes) (string, error)`
contract, so `handleTrigger` itself doesn't change based on which is active.

### Retry and rejection handling

Both transports classify their own failures via `internal/transport.RejectedError` — a definitive
rejection (permission denied, an unregistered `camera_id`, a malformed request) vs. everything else
(network blips, timeouts), which `handleTrigger` treats very differently:

- **`RejectedError`** short-circuits immediately — retrying can never succeed. If it carries an HTTP
  status code in the 4xx range (the HTTP transport always has one; the WebRTC transport only has one for
  a *signaling*-level rejection, not an ack-carried one — see `internal/webrtcupload`'s doc comments),
  that status is passed straight through as `/trigger`'s own response instead of a generic `502`.
- **Anything else** is retried up to 3 times with a 2s fixed backoff, resending the exact same
  already-encoded clip bytes each time (the ring buffer keeps rolling independently — a retry must not
  silently resend a *different*, newer window than what was actually triggered). Exhausting all attempts
  returns `502` with the attempt count in the message.

## Run locally

Requires a UVC camera connected in Webcam Mode, and `media-ingest-gateway` running (see its README for a
bearer token).

```bash
export GATEWAY_URL=http://localhost:8080
export BEARER_TOKEN=<organizer_admin token>
export ORG_ID=<org id>
export MATCH_ID=<match id>
export CAMERA_ID=dji-action5-01
go run ./cmd
```

### Configuration (environment variables)

| Variable | Default | Notes |
|---|---|---|
| `GATEWAY_URL` | *(required)* | Media Ingest Gateway base URL |
| `BEARER_TOKEN` | *(required)* | An `organizer_admin` token — no distinct edge-device credential exists yet |
| `ORG_ID` | *(required)* | |
| `MATCH_ID` | *(required)* | |
| `CAMERA_ID` | *(required)* | Must be registered with Camera Calibration Service — media-ingest-gateway now rejects uploads from an unregistered `camera_id` |
| `BUFFER_SECONDS` | `20` | Rolling buffer window |
| `PORT` | `9090` | edge-agent's own local HTTP server |
| `UPLOAD_TRANSPORT` | `http` | `http` or `webrtc` — see "Upload transport" above |

### Triggering a capture / exporting audio

```bash
# Video: snapshots the current buffer, encodes it, and uploads it to Media Ingest Gateway — returns
# the assigned clip ID and frame count. Stands in for the real review-trigger signal (see "Known
# simplifications").
curl -s -X POST "localhost:9090/trigger"

# Audio: downloads the current audio buffer as a .wav file.
curl -s "localhost:9090/audio-snapshot" -o snapshot.wav
```

## Test

```
go test ./... -cover
```

`buffer`, `clipformat`, `wav`, `transport`, and `uploader` are pure/network-mockable and are unit-tested
normally (79-100% coverage). `webrtcupload` is tested against a real (if loopback) `pion/webrtc` peer
connection and data channel — not a mock of the transport — proving the chunk-send/ack/close sequence
actually works over real WebRTC (~80% coverage; the uncovered lines are timeout paths that would need
real multi-second waits to trigger deterministically). `cmd`'s `uploadWithRetry` (the retry/rejection
logic — see "Retry and rejection handling" above) is unit-tested against a fake `Uploader`; the rest of
`cmd` (device wiring, capture loops) is untestable the same way `capture` is, below. `capture` wraps real
hardware I/O and isn't
meaningfully unit-testable the way the others are — both the camera and microphone were instead
exercised for real against the DJI Action 5 Pro during manual verification (see the implementation
plan), the first components this session whose real backend, not a fake, was actually run.

## Known simplifications (tracked, not accidental)

- **640x480@30, not 1080p30 or 120fps+** — the DJI Action 5 Pro's confirmed USB-C Webcam Mode capability
  is 1920x1080@30 (already a deviation from the 120fps+ target — see `docs/adr/0007`), but
  `pion/mediadevices`' Windows driver can't open above 1280x720 at all, and 1280x720 stalls after one
  frame in sustained capture (a real, reproduced, resolution-specific finding — see `docs/adr/0007`'s
  addendum and `internal/capture.Open`'s `NewReader` comment). 640x480 is what's confirmed to sustain
  reliably in this environment. Requested as an ideal, not exact, constraint, so a Linux deployment isn't
  bound to this same Windows-specific ceiling.
- **Custom JPEG-sequence container, not H.264/mp4** — `internal/clipformat` proves the capture → buffer →
  upload → storage → retrieval mechanism round-trips real frame content correctly; it is not a playable
  production video format.
- **No live Go→Python correlation bridge.** Audio is exported as `.wav` for local verification/scripting;
  `docs/adr/0006`'s deferred decision (HTTP-wrapped `find_offset`, or an async job) remains deferred —
  this proves the algorithm against real signal characteristics, it doesn't wire production
  capture→correlate→submit end-to-end.
- **Manual HTTP trigger (`POST /trigger`), not a real review-orchestration signal** — that service doesn't
  exist yet.
- **No distinct edge-device credential** — uploads authenticate as a pre-obtained `organizer_admin` token,
  same status as Media Ingest Gateway's own documented deferred decision on this.
- **Single camera, no device selection** — `capture.Open` uses whatever the OS considers the first UVC
  device; `capture.ListDevices()` logs what's available at startup for diagnostics, but there's no
  multi-camera selection logic yet (real accessible-tier deployments need 2-4 cameras).
- **Microphone failure is non-fatal but silent about which mic it picked** — `OpenMicrophone` has no
  label-based selection the way `Open` does for video; on a machine with multiple audio input devices,
  which one gets used is automatic, same caveat as video's device selection above.
- **No STUN/TURN for the WebRTC transport** — `internal/webrtcupload` uses host ICE candidates only,
  sufficient for localhost/same-LAN testing; real NAT traversal for actual venue deployments is a
  separate, later decision (`docs/adr/0009`).
- **SRT itself is not built** — WebRTC is the only new transport this slice adds, a deliberate deviation
  from architecture.md's stated SRT preference (`docs/adr/0009`).
- **No automatic transport fallback** — `UPLOAD_TRANSPORT` is a static operator choice, not a runtime
  negotiation between HTTP and WebRTC.
