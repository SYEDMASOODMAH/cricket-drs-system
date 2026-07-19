package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router builds the full route tree. Every route except /healthz and
// /metrics requires auth — unlike identity-access, this service issues no
// tokens of its own, so there's no unauthenticated write path at all.
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
