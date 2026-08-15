package config

import (
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GATEWAY_URL", "http://localhost:8080")
	t.Setenv("BEARER_TOKEN", "test-token")
	t.Setenv("ORG_ID", "org-1")
	t.Setenv("MATCH_ID", "match-1")
	t.Setenv("CAMERA_ID", "cam-1")
}

func TestLoad_DefaultsApplied(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BufferWindow != 20*time.Second {
		t.Errorf("expected default 20s buffer window, got %v", cfg.BufferWindow)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected default port 9090, got %q", cfg.Port)
	}
}

func TestLoad_OverridesRespected(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("BUFFER_SECONDS", "45")
	t.Setenv("PORT", "9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BufferWindow != 45*time.Second {
		t.Errorf("expected 45s buffer window, got %v", cfg.BufferWindow)
	}
	if cfg.Port != "9999" {
		t.Errorf("expected port 9999, got %q", cfg.Port)
	}
}

func TestLoad_MissingRequiredVarRejected(t *testing.T) {
	required := []string{"GATEWAY_URL", "BEARER_TOKEN", "ORG_ID", "MATCH_ID", "CAMERA_ID"}
	for _, missing := range required {
		t.Run(missing, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(missing, "")

			_, err := Load()
			if err == nil {
				t.Fatalf("expected an error when %s is missing", missing)
			}
		})
	}
}

func TestLoad_InvalidBufferSecondsRejected(t *testing.T) {
	setRequiredEnv(t)
	for _, invalid := range []string{"not-a-number", "0", "-5"} {
		t.Run(invalid, func(t *testing.T) {
			t.Setenv("BUFFER_SECONDS", invalid)
			_, err := Load()
			if err == nil {
				t.Fatalf("expected an error for BUFFER_SECONDS=%q", invalid)
			}
		})
	}
}
