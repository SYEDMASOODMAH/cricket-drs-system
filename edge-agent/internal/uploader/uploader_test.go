package uploader

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cricketdrs/edge-agent/internal/transport"
)

func TestUpload_Success(t *testing.T) {
	var gotPath, gotAuth, gotContentType string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "clip_abc123"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	clipID, err := client.Upload(context.Background(), "test-token", "org-1", "match-1", "cam-1", []byte("clip-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clipID != "clip_abc123" {
		t.Fatalf("unexpected clip id: %q", clipID)
	}
	if gotPath != "/v1/organizations/org-1/matches/match-1/clips?camera_id=cam-1" {
		t.Fatalf("unexpected request path: %q", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected Authorization header: %q", gotAuth)
	}
	if gotContentType != "application/octet-stream" {
		t.Fatalf("unexpected Content-Type header: %q", gotContentType)
	}
	if string(gotBody) != "clip-bytes" {
		t.Fatalf("unexpected uploaded body: %q", gotBody)
	}
}

func TestUpload_GatewayErrorPropagated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "permission denied"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Upload(context.Background(), "test-token", "org-1", "match-1", "cam-1", []byte("clip-bytes"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected error to mention the gateway's message, got: %v", err)
	}

	var rejected *transport.RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a *transport.RejectedError (a gateway rejection should never be retried), got %T", err)
	}
	if rejected.StatusCode != http.StatusForbidden {
		t.Fatalf("expected StatusCode 403, got %d", rejected.StatusCode)
	}
}

func TestUpload_MissingClipIDInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Upload(context.Background(), "test-token", "org-1", "match-1", "cam-1", []byte("clip-bytes"))
	if err == nil {
		t.Fatal("expected an error for a response missing the clip id")
	}
}

func TestUpload_NetworkErrorPropagated(t *testing.T) {
	client := NewClient("http://127.0.0.1:0") // nothing listens here
	_, err := client.Upload(context.Background(), "test-token", "org-1", "match-1", "cam-1", []byte("clip-bytes"))
	if err == nil {
		t.Fatal("expected a network error")
	}

	var rejected *transport.RejectedError
	if errors.As(err, &rejected) {
		t.Fatal("a network error is transient and must not be classified as a RejectedError (callers should retry it)")
	}
}
