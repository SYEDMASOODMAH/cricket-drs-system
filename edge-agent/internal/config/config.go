// Package config loads edge-agent's runtime configuration from
// environment variables — same zero-config-friendly-defaults-where-safe
// pattern as every services/*/cmd/main.go's jwtSigningKey()/PORT
// handling, adapted for a device agent instead of an HTTP service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultBufferSeconds   = 20
	defaultPort            = "9090"
	defaultUploadTransport = "http"
)

// Config holds everything cmd/main.go needs to wire the capture loop,
// buffer, and uploader together.
type Config struct {
	// GatewayURL is Media Ingest Gateway's base URL (e.g.
	// http://localhost:8080) that captured clips are uploaded to.
	GatewayURL string
	// BearerToken authenticates uploads as an organizer_admin — there is
	// no distinct edge-device credential yet (see this package's README
	// note / the implementation plan's "explicitly deferred" section),
	// so this must be a token minted for a real organizer_admin user.
	BearerToken string
	OrgID       string
	MatchID     string
	CameraID    string
	// CameraDeviceLabel, if set, pins internal/capture.Open to the OS-level
	// video device whose label contains this substring (see
	// capture.ListDevices/DeviceInfo.Label) — distinct from CameraID,
	// which is just the opaque label uploaded clips are tagged with.
	// Optional: empty leaves device selection automatic, which is enough
	// on a machine with exactly one real camera but not when a
	// virtual/software camera is also registered (see edge-agent's
	// README). Matched by Label, not DeviceID: mediadevices regenerates
	// DeviceID every process run, so it can't be persisted in config.
	CameraDeviceLabel string
	// BufferWindow is how much recently captured video edge-agent keeps
	// in memory before a trigger arrives (architecture.md: "target: last
	// 20-30s").
	BufferWindow time.Duration
	// Port is what edge-agent's own local HTTP server (healthz +
	// trigger) listens on.
	Port string
	// UploadTransport selects how handleTrigger pushes an encoded clip to
	// GatewayURL: "http" (internal/uploader, a plain authenticated POST)
	// or "webrtc" (internal/webrtcupload, docs/adr/0009). Defaults to
	// "http" so existing behavior is unchanged unless explicitly opted in.
	UploadTransport string
}

// Load reads configuration from the environment. GATEWAY_URL,
// BEARER_TOKEN, ORG_ID, MATCH_ID, and CAMERA_ID are required — there's no
// sane default for "which gateway, which match, which camera," unlike
// PORT or BUFFER_SECONDS.
func Load() (Config, error) {
	cfg := Config{
		GatewayURL:        os.Getenv("GATEWAY_URL"),
		BearerToken:       os.Getenv("BEARER_TOKEN"),
		OrgID:             os.Getenv("ORG_ID"),
		MatchID:           os.Getenv("MATCH_ID"),
		CameraID:          os.Getenv("CAMERA_ID"),
		CameraDeviceLabel: os.Getenv("CAMERA_DEVICE_LABEL"),
		Port:              defaultPort,
	}

	for name, val := range map[string]string{
		"GATEWAY_URL":  cfg.GatewayURL,
		"BEARER_TOKEN": cfg.BearerToken,
		"ORG_ID":       cfg.OrgID,
		"MATCH_ID":     cfg.MatchID,
		"CAMERA_ID":    cfg.CameraID,
	} {
		if val == "" {
			return Config{}, fmt.Errorf("config: %s must be set", name)
		}
	}

	bufferSeconds := defaultBufferSeconds
	if raw := os.Getenv("BUFFER_SECONDS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("config: BUFFER_SECONDS must be a positive integer, got %q", raw)
		}
		bufferSeconds = parsed
	}
	cfg.BufferWindow = time.Duration(bufferSeconds) * time.Second

	if raw := os.Getenv("PORT"); raw != "" {
		cfg.Port = raw
	}

	cfg.UploadTransport = defaultUploadTransport
	if raw := os.Getenv("UPLOAD_TRANSPORT"); raw != "" {
		if raw != "http" && raw != "webrtc" {
			return Config{}, fmt.Errorf("config: UPLOAD_TRANSPORT must be %q or %q, got %q", "http", "webrtc", raw)
		}
		cfg.UploadTransport = raw
	}

	return cfg, nil
}
