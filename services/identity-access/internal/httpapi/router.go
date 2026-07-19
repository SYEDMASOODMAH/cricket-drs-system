package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the full route tree. /healthz, /metrics, org creation, and
// login are unauthenticated by necessity (login IS the auth entry point;
// /metrics is normally scraped same-network, not internet-facing);
// everything else is mounted behind requireAuth.
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
	r.Post("/v1/auth/login", a.handleLogin)
	r.Post("/v1/organizations", a.handleCreateOrganization)

	r.Group(func(r chi.Router) {
		r.Use(requireAuth(a.svc))
		r.Post("/v1/organizations/{orgID}/users", a.handleInviteUser)
		r.Get("/v1/organizations/{orgID}/users/{userID}", a.handleGetUser)
		r.Post("/v1/organizations/{orgID}/users/{userID}/roles", a.handleAssignRole)
		r.Post("/v1/users/{userID}/consent", a.handleCaptureConsent)
		r.Get("/v1/users/{userID}/consent", a.handleGetConsent)
	})

	return r
}
