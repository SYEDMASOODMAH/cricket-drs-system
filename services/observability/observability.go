// Package observability provides shared HTTP metrics, tracing, and
// structured-logging middleware for every core service in this module. It
// deliberately sits outside any single service's internal/ tree so it can
// be imported across service boundaries — see
// docs/adr/0004-shared-observability-package.md for why this is shared
// rather than duplicated per-service, unlike the JWT/Role code in
// services/match-tournament/internal/security and internal/domain.
package observability

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Observability bundles one service's metrics registry and tracer
// provider. Construct one per service at startup via New, mount its
// middlewares (TracingMiddleware, then MetricsMiddleware, then
// LoggingMiddleware — see each method's doc comment for why that order
// matters), and defer Shutdown.
type Observability struct {
	serviceName string

	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec

	tracerProvider *sdktrace.TracerProvider
}

// New builds an Observability for serviceName: a Prometheus registry with
// Go runtime/process collectors plus HTTP request metrics (metrics.go),
// and an OTel tracer provider (tracing.go).
func New(serviceName string) (*Observability, error) {
	o := &Observability{serviceName: serviceName}
	o.initMetrics()

	tp, err := newTracerProvider(serviceName)
	if err != nil {
		return nil, err
	}
	o.tracerProvider = tp

	return o, nil
}

// Shutdown flushes any buffered spans. Call via defer in cmd/main.go.
func (o *Observability) Shutdown(ctx context.Context) error {
	return o.tracerProvider.Shutdown(ctx)
}
