package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newTestObservability builds an Observability wired to an in-memory span
// exporter instead of the real stdout one, so tests can assert on
// recorded spans without parsing process stdout. Internal test (package
// observability, not observability_test) specifically to construct this
// directly — the tracerProvider field is otherwise unexported by design.
func newTestObservability(t *testing.T) (*Observability, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	o := &Observability{serviceName: "test-service", tracerProvider: tp}
	o.initMetrics()
	return o, exporter
}

func TestTracingMiddleware_CreatesSpan(t *testing.T) {
	o, exporter := newTestObservability(t)

	handler := o.TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if err := o.tracerProvider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("unexpected error flushing spans: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span, got %d", len(spans))
	}
}

func TestHTTPClientTransport_PropagatesTraceContext(t *testing.T) {
	o, exporter := newTestObservability(t)

	var receivedTraceparent string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Simulate an incoming request creating a parent span, and make an
	// outgoing call within that span's context via HTTPClientTransport —
	// exactly what match-tournament's identityaccess client does for the
	// consent-gate check.
	handler := o.TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{Transport: o.HTTPClientTransport(nil)}
		// Must carry r's context (not client.Get, which uses
		// context.Background()) so the outgoing call is a child of the
		// incoming request's span — exactly what
		// internal/identityaccess.Client.IsEligibleForDRS does with the
		// context it's handed.
		outReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream.URL, nil)
		if err != nil {
			t.Errorf("unexpected error building request: %v", err)
			return
		}
		resp, err := client.Do(outReq)
		if err != nil {
			t.Errorf("unexpected error calling upstream: %v", err)
			return
		}
		_ = resp.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if receivedTraceparent == "" {
		t.Fatal("expected the outgoing request to carry a traceparent header")
	}

	if err := o.tracerProvider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("unexpected error flushing spans: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans (incoming request + outgoing call), got %d", len(spans))
	}
	if spans[0].SpanContext.TraceID() != spans[1].SpanContext.TraceID() {
		t.Fatal("expected both spans to share the same trace ID — context propagation is broken")
	}
}

func TestHTTPClientTransport_NilBaseDefaultsToDefaultTransport(t *testing.T) {
	o, _ := newTestObservability(t)
	transport := o.HTTPClientTransport(nil)
	if transport == nil {
		t.Fatal("expected a non-nil transport")
	}
}
