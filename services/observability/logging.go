package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// LoggingMiddleware emits one JSON log line per request via log/slog
// (stdlib, no new dependency), per rules.md Section 4.1's structured-
// logging requirement — including trace_id when a span is active, per
// that same section's explicit ask, so a log line and its trace can be
// correlated. Must be mounted after TracingMiddleware (registered later
// via r.Use, so it's wrapped inside it) so a span already exists in the
// request context by the time this reads it.
func (o *Observability) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
			attrs = append(attrs, "trace_id", sc.TraceID().String())
		}

		slog.Info("http_request", attrs...)
	})
}
