package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the full route tree. Every route except /healthz requires
// auth — unlike identity-access, this service issues no tokens of its own,
// so there's no unauthenticated write path at all.
func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(structuredLogger)

	r.Get("/healthz", a.handleHealthz)

	r.Group(func(r chi.Router) {
		r.Use(requireAuth(a.svc))

		r.Post("/v1/organizations/{orgID}/tournaments", a.handleCreateTournament)
		r.Get("/v1/organizations/{orgID}/tournaments", a.handleListTournaments)
		r.Get("/v1/organizations/{orgID}/tournaments/{tournamentID}", a.handleGetTournament)

		r.Post("/v1/organizations/{orgID}/teams", a.handleCreateTeam)
		r.Get("/v1/organizations/{orgID}/teams", a.handleListTeams)
		r.Get("/v1/organizations/{orgID}/teams/{teamID}", a.handleGetTeam)

		r.Post("/v1/organizations/{orgID}/matches", a.handleCreateMatch)
		r.Get("/v1/organizations/{orgID}/matches", a.handleListMatches)
		r.Get("/v1/organizations/{orgID}/matches/{matchID}", a.handleGetMatch)

		r.Post("/v1/organizations/{orgID}/matches/{matchID}/roster", a.handleAddRosterEntry)
		r.Get("/v1/organizations/{orgID}/matches/{matchID}/roster", a.handleListRoster)
		r.Delete("/v1/organizations/{orgID}/matches/{matchID}/roster/{userID}", a.handleRemoveRosterEntry)
	})

	return r
}

// structuredLogger emits one JSON log line per request via log/slog,
// same as identity-access's internal/httpapi/router.go.
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
