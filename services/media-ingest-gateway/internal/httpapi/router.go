package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the full route tree. Every route except /healthz and
// /metrics requires auth — this service issues no tokens of its own,
// same as match-tournament.
//
// Middleware order matters: TracingMiddleware must wrap MetricsMiddleware
// and LoggingMiddleware (registered first, via r.Use) so a span already
// exists in the request context by the time LoggingMiddleware reads it to
// attach trace_id — see services/observability's doc comments.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(a.obs.TracingMiddleware)
	r.Use(a.obs.MetricsMiddleware)
	r.Use(a.obs.LoggingMiddleware)

	r.Get("/healthz", a.handleHealthz)
	r.Method(http.MethodGet, "/metrics", a.obs.MetricsHandler())

	r.Group(func(r chi.Router) {
		r.Use(requireAuth(a.svc))

		r.Post("/v1/organizations/{orgID}/matches/{matchID}/clips", a.handleUploadClip)
		r.Get("/v1/organizations/{orgID}/matches/{matchID}/clips", a.handleListClips)
		r.Get("/v1/organizations/{orgID}/matches/{matchID}/clips/{clipID}", a.handleGetClip)
		r.Get("/v1/organizations/{orgID}/matches/{matchID}/clips/{clipID}/download", a.handleDownloadClip)
	})

	return r
}
