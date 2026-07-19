package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// initMetrics registers Go runtime/process collectors (covers
// "saturation" per rules.md Section 4.3's golden signals) plus HTTP
// request count/duration, labeled by method, chi route *pattern*, and
// status. Metric names are deliberately generic (not service-prefixed):
// each service runs its own registry behind its own /metrics endpoint, so
// Prometheus's own job/instance scrape labels are what distinguish them —
// embedding the service name into the metric name itself isn't the
// idiomatic Prometheus pattern.
func (o *Observability) initMetrics() {
	o.registry = prometheus.NewRegistry()
	o.registry.MustRegister(collectors.NewGoCollector())
	o.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	o.requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed, labeled by method, route, and status.",
	}, []string{"method", "route", "status"})
	o.registry.MustRegister(o.requestsTotal)

	o.requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds, labeled by method, route, and status.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
	o.registry.MustRegister(o.requestDuration)
}

// MetricsMiddleware records the metrics initMetrics registered. The route
// label is the chi route *pattern* (e.g.
// "/v1/organizations/{orgID}/matches"), read from the routing context
// *after* next.ServeHTTP returns — chi has fully resolved the pattern by
// then regardless of where in the middleware chain this sits, since
// routing/dispatch happens deep inside the chain this wraps. Never the
// resolved path with real IDs in it, which would be unbounded cardinality.
func (o *Observability) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(ww.Status())

		o.requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		o.requestDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
	})
}

// MetricsHandler exposes the registry in Prometheus text format — mount at
// GET /metrics.
func (o *Observability) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(o.registry, promhttp.HandlerOpts{})
}
