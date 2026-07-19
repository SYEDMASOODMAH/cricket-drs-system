package observability

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newTracerProvider builds an OTel tracer provider for serviceName.
// Exports to stdout, so tracing is visible with zero setup — no trace
// backend exists in this environment yet (see
// docs/adr/0004-shared-observability-package.md). An OTLP exporter (for a
// real collector once one exists) is a deliberately deferred follow-up,
// not built here: go.opentelemetry.io/otel/exporters/otlp/otlptrace pulls
// in a gRPC dependency chain whose latest compatible versions require a
// newer Go than this module targets (go 1.23.0) — not worth fighting that
// version pressure for an exporter with nothing to export to yet.
func newTracerProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	res := resource.NewSchemaless(attribute.String("service.name", serviceName))

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("observability: build stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

// TracingMiddleware creates a span per incoming request via otelhttp, the
// standard OTel HTTP server instrumentation. Explicitly bound to this
// Observability's own tracer provider (not the global one New also sets)
// so each instance is self-contained — real code paths behave the same
// either way, but tests can build an Observability around an in-memory
// exporter without mutating global state. Mount this *before*
// MetricsMiddleware and LoggingMiddleware (i.e. register it first via
// r.Use, so it wraps them) — LoggingMiddleware reads the active span from
// the request context to attach trace_id, which only works if a span
// already exists by the time it runs.
func (o *Observability) TracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, o.serviceName,
		otelhttp.WithTracerProvider(o.tracerProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
	)
}

// HTTPClientTransport wraps base (nil -> http.DefaultTransport) so
// outgoing requests propagate trace context and create a child span of
// whatever request triggered them — used by match-tournament's
// internal/identityaccess client for the consent-gate call, so it shows
// up as a child span of the roster-add request that triggered it.
func (o *Observability) HTTPClientTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base,
		otelhttp.WithTracerProvider(o.tracerProvider),
		otelhttp.WithPropagators(propagation.TraceContext{}),
	)
}
