package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware_EmitsStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prevLogger)

	o := newTestObservability(t)
	defer func() { _ = o.Shutdown(context.Background()) }()

	handler := o.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/example", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
		t.Fatalf("failed to parse log line as JSON: %v\noutput: %s", err, buf.String())
	}

	if entry["method"] != "GET" || entry["path"] != "/v1/example" {
		t.Fatalf("unexpected log fields: %+v", entry)
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusTeapot {
		t.Fatalf("expected status 418, got %+v", entry["status"])
	}
	if _, ok := entry["trace_id"]; ok {
		t.Fatal("expected no trace_id field when no span is active in context")
	}
}

func TestLoggingMiddleware_IncludesTraceIDWhenSpanActive(t *testing.T) {
	var buf bytes.Buffer
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	defer slog.SetDefault(prevLogger)

	o := newTestObservability(t)
	defer func() { _ = o.Shutdown(context.Background()) }()

	// TracingMiddleware wraps LoggingMiddleware, matching the real mount
	// order in both services' routers — a span must exist in context
	// before LoggingMiddleware reads it.
	handler := o.TracingMiddleware(o.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/v1/example", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
		t.Fatalf("failed to parse log line as JSON: %v\noutput: %s", err, buf.String())
	}

	traceID, ok := entry["trace_id"].(string)
	if !ok || len(traceID) != 32 {
		t.Fatalf("expected a 32-hex-char trace_id when a span is active, got %+v", entry["trace_id"])
	}
}
