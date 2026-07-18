package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the full route tree. /healthz, org creation, and login are
// unauthenticated by necessity (login IS the auth entry point); everything
// else is mounted behind requireAuth.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(structuredLogger)

	r.Get("/healthz", a.handleHealthz)
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

// structuredLogger emits one JSON log line per request via log/slog
// (stdlib, no new dependency), per rules.md Section 4.1's structured-
// logging requirement — never an unstructured print/string-concatenated
// log line.
func structuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
