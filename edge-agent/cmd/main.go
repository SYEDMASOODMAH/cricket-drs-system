// Command edge-agent runs on venue capture hardware. It captures video
// from a USB-C tethered UVC camera (Webcam Mode; see docs/adr/0003 and
// docs/adr/0007 for the hardware decisions), maintains a rolling buffer,
// and on trigger (currently a manual HTTP call — see this package's
// README "explicitly deferred") pushes the buffered window to the cloud
// Media Ingest Gateway. Per rules.md Section 2.2, this is the only place
// wiring happens; the handlers below are thin glue over
// internal/clipformat and internal/uploader, which hold the actual logic.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cricketdrs/edge-agent/internal/buffer"
	"github.com/cricketdrs/edge-agent/internal/capture"
	"github.com/cricketdrs/edge-agent/internal/clipformat"
	"github.com/cricketdrs/edge-agent/internal/config"
	"github.com/cricketdrs/edge-agent/internal/transport"
	"github.com/cricketdrs/edge-agent/internal/uploader"
	"github.com/cricketdrs/edge-agent/internal/wav"
	"github.com/cricketdrs/edge-agent/internal/webrtcupload"
)

// maxUploadAttempts and uploadRetryBackoff bound handleTrigger's retry
// loop for transient upload failures (network blips, a briefly
// unreachable gateway) — a definitive rejection (transport.RejectedError)
// short-circuits immediately instead, since retrying it can never
// succeed. A fixed backoff is simple and sufficient for 3 attempts; not
// worth tuning further without real venue network data to justify it.
const maxUploadAttempts = 3

// uploadRetryBackoff is a var, not a const, purely so tests can shrink it
// — production always runs with the 2s default set here.
var uploadRetryBackoff = 2 * time.Second

// Uploader is the transport-agnostic contract handleTrigger pushes an
// encoded clip through — internal/uploader.Client (plain HTTP, the
// default) and internal/webrtcupload.Client (docs/adr/0009) both satisfy
// it, selected by UPLOAD_TRANSPORT (internal/config) at startup. Nothing
// downstream of this interface needs to know or care which one is active.
type Uploader interface {
	Upload(ctx context.Context, token, orgID, matchID, cameraID string, clipBytes []byte) (string, error)
}

// Capture resolution/frame rate: requested as ideal, not exact,
// constraints (internal/capture.Open), so this same binary requests
// sensible values against whatever camera is actually connected and gets
// the closest match available. 640x480 is not the DJI Action 5 Pro's
// full confirmed Webcam Mode capability (1920x1080 — docs/adr/0007) — see
// that ADR's addendum: on this Windows dev machine, 1280x720 opens but
// stalls after exactly one frame (a real, reproduced, resolution-specific
// sustained-capture failure, most likely a USB bandwidth ceiling for
// uncompressed video at that size), while 640x480 sustains reliably at a
// steady ~30fps. A Linux deployment using the same ideal constraint isn't
// bound to this same ceiling — different driver, and likely a wired
// connection with more predictable bandwidth.
const (
	captureWidth     = 640
	captureHeight    = 480
	captureFrameRate = 30
	// captureSampleRate is likewise an ideal target
	// (internal/capture.OpenMicrophone) — proving real audio capture and
	// feeding it into ml-pipeline/time-sync's find_offset (see the
	// implementation plan) is what matters here, not a specific rate.
	captureSampleRate = 48000
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("edge-agent: failed to load config", "error", err)
		os.Exit(1)
	}

	devices := capture.ListDevices()
	slog.Info("edge-agent: available video devices", "devices", devices)

	cam, err := capture.Open(captureWidth, captureHeight, captureFrameRate, cfg.CameraDeviceLabel)
	if err != nil {
		slog.Error("edge-agent: failed to open camera", "error", err)
		os.Exit(1)
	}
	defer func() { _ = cam.Close() }()

	ring := buffer.NewRingBuffer(cfg.BufferWindow)
	go captureLoop(cam, ring)

	// Microphone failure is not fatal: video capture (already proven end
	// to end) stands on its own, and audio is new, additive capability —
	// a missing/denied mic shouldn't take down the whole agent. See the
	// implementation plan: this is specifically about proving audio
	// capture and the time-sync algorithm against real signal
	// characteristics, not a hard production requirement yet.
	var audioRing *buffer.AudioRingBuffer
	mic, err := capture.OpenMicrophone(captureSampleRate)
	if err != nil {
		slog.Error("edge-agent: failed to open microphone, continuing without audio capture", "error", err)
	} else {
		defer func() { _ = mic.Close() }()
		audioRing = buffer.NewAudioRingBuffer(cfg.BufferWindow)
		go audioCaptureLoop(mic, audioRing)
	}

	var uploadClient Uploader
	switch cfg.UploadTransport {
	case "webrtc":
		uploadClient = webrtcupload.NewClient(cfg.GatewayURL)
	default:
		uploadClient = uploader.NewClient(cfg.GatewayURL)
	}

	srv := &server{cfg: cfg, ring: ring, audioRing: audioRing, uploader: uploadClient}

	slog.Info("edge-agent starting", "port", cfg.Port, "buffer_window", cfg.BufferWindow, "org_id", cfg.OrgID, "match_id", cfg.MatchID, "camera_id", cfg.CameraID, "upload_transport", cfg.UploadTransport)
	if err := http.ListenAndServe(":"+cfg.Port, srv.router()); err != nil {
		slog.Error("edge-agent server exited", "error", err)
		os.Exit(1)
	}
}

// captureLoop reads frames continuously and feeds them into the ring
// buffer until the process exits. A single failed read is logged and
// skipped rather than treated as fatal — a real deployment would want
// reconnect/backoff logic once camera-disconnect scenarios are actually
// being handled (not yet — see this package's README).
func captureLoop(cam *capture.Camera, ring *buffer.RingBuffer) {
	frameCount := 0
	lastLog := time.Now()
	for {
		frame, err := cam.Read()
		if err != nil {
			slog.Error("edge-agent: frame read failed", "error", err)
			continue
		}
		ring.Add(frame)
		frameCount++

		if time.Since(lastLog) >= 5*time.Second {
			slog.Info("edge-agent: capturing", "frames_captured", frameCount, "buffered_frames", len(ring.Snapshot()))
			frameCount = 0
			lastLog = time.Now()
		}
	}
}

// audioCaptureLoop mirrors captureLoop for the microphone — same
// continue-on-error, periodic-log shape.
func audioCaptureLoop(mic *capture.Microphone, ring *buffer.AudioRingBuffer) {
	chunkCount := 0
	lastLog := time.Now()
	for {
		chunk, err := mic.Read()
		if err != nil {
			slog.Error("edge-agent: audio chunk read failed", "error", err)
			continue
		}
		ring.Add(chunk)
		chunkCount++

		if time.Since(lastLog) >= 5*time.Second {
			slog.Info("edge-agent: capturing audio", "chunks_captured", chunkCount, "buffered_chunks", len(ring.Snapshot()))
			chunkCount = 0
			lastLog = time.Now()
		}
	}
}

// server holds the tiny local HTTP API: a health check, the manual
// trigger stand-in for a real review-orchestration signal (docs/phases.md
// Phase 2, this package's implementation plan), and an audio export
// endpoint for verifying real microphone capture.
type server struct {
	cfg       config.Config
	ring      *buffer.RingBuffer
	audioRing *buffer.AudioRingBuffer
	uploader  Uploader
}

func (s *server) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /trigger", s.handleTrigger)
	mux.HandleFunc("GET /audio-snapshot", s.handleAudioSnapshot)
	return mux
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type triggerResponse struct {
	ClipID     string `json:"clip_id"`
	FrameCount int    `json:"frame_count"`
}

// handleTrigger snapshots the current buffer, encodes it
// (internal/clipformat), and uploads it (internal/uploader) to Media
// Ingest Gateway — the "on trigger, push the buffered window" mechanism
// docs/architecture.md describes, with the trigger source itself
// currently manual (see README).
func (s *server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	frames := s.ring.Snapshot()
	if len(frames) == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no frames buffered yet"})
		return
	}

	encoded, err := clipformat.Encode(frames)
	if err != nil {
		slog.Error("edge-agent: failed to encode clip", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to encode clip"})
		return
	}

	clipID, err := uploadWithRetry(r.Context(), s.uploader, s.cfg, encoded)
	if err != nil {
		var rejected *transport.RejectedError
		if errors.As(err, &rejected) {
			status := http.StatusBadGateway
			if rejected.StatusCode >= 400 && rejected.StatusCode < 500 {
				status = rejected.StatusCode
			}
			slog.Error("edge-agent: upload rejected, not retrying", "error", err)
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		slog.Error("edge-agent: upload failed after retries", "error", err, "attempts", maxUploadAttempts)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("upload failed after %d attempts: %s", maxUploadAttempts, err.Error())})
		return
	}

	slog.Info("edge-agent: clip uploaded", "clip_id", clipID, "frame_count", len(frames))
	writeJSON(w, http.StatusOK, triggerResponse{ClipID: clipID, FrameCount: len(frames)})
}

// uploadWithRetry calls uploader.Upload up to maxUploadAttempts times with
// uploadRetryBackoff between attempts, resending the exact same encoded
// bytes each time (the ring buffer keeps rolling independently, so a
// retry must resend what was actually triggered, not a newer window). A
// transport.RejectedError (the gateway definitively rejected the request)
// short-circuits immediately — retrying it can never succeed.
func uploadWithRetry(ctx context.Context, u Uploader, cfg config.Config, encoded []byte) (string, error) {
	var clipID string
	var err error
	for attempt := 1; attempt <= maxUploadAttempts; attempt++ {
		uploadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		clipID, err = u.Upload(uploadCtx, cfg.BearerToken, cfg.OrgID, cfg.MatchID, cfg.CameraID, encoded)
		cancel()
		if err == nil {
			return clipID, nil
		}

		var rejected *transport.RejectedError
		if errors.As(err, &rejected) {
			return "", err
		}

		if attempt < maxUploadAttempts {
			slog.Warn("edge-agent: upload attempt failed, retrying", "attempt", attempt, "error", err)
			time.Sleep(uploadRetryBackoff)
		}
	}
	return "", err
}

// handleAudioSnapshot returns the currently buffered audio as a .wav
// download — a local verification/export mechanism, deliberately
// separate from /trigger's video-upload path since production audio
// handling (e.g. a live Go->Python correlation bridge, per
// docs/adr/0006's still-deferred decision) isn't being designed here,
// just proven capturable. Chunks are concatenated in Snapshot's
// oldest-first order; the sample rate is taken from the first chunk
// (capture uses one microphone for the whole process lifetime, so this
// is not expected to vary chunk to chunk).
func (s *server) handleAudioSnapshot(w http.ResponseWriter, r *http.Request) {
	if s.audioRing == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no microphone available"})
		return
	}

	chunks := s.audioRing.Snapshot()
	if len(chunks) == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no audio buffered yet"})
		return
	}

	sampleRate := chunks[0].SampleRate
	var samples []int16
	for _, c := range chunks {
		samples = append(samples, c.Samples...)
	}

	data := wav.Encode(samples, sampleRate, 1)

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", `attachment; filename="audio-snapshot.wav"`)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		slog.Error("edge-agent: failed to write audio snapshot response", "error", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("edge-agent: failed to encode response body", "error", err)
	}
}
