// Package httpapi is match-tournament's transport layer: a chi router and
// handlers that decode requests, call into internal/service, and encode
// responses. Per rules.md Section 1, no business logic or persistence
// concern lives here.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
	"github.com/cricketdrs/services/match-tournament/internal/service"
	"github.com/cricketdrs/services/observability"
)

type API struct {
	svc *service.Service
	obs *observability.Observability
}

func New(svc *service.Service, obs *observability.Observability) *API {
	return &API{svc: svc, obs: obs}
}

func (a *API) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// --- Tournaments ---

func (a *API) handleCreateTournament(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	var req createTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	types, tier := req.DefaultConditions.toDomain()
	defaults := service.PlayingConditionsInput{
		ReviewQuotaPerInnings: req.DefaultConditions.ReviewQuotaPerInnings,
		DecisionTypesEnabled:  types,
		CameraTier:            tier,
	}

	t, err := a.svc.CreateTournament(r.Context(), caller, orgID, req.Name, defaults)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTournamentResponse(t))
}

func (a *API) handleGetTournament(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	id := domain.TournamentID(chi.URLParam(r, "tournamentID"))

	t, err := a.svc.GetTournament(r.Context(), caller, orgID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTournamentResponse(t))
}

func (a *API) handleListTournaments(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	list, err := a.svc.ListTournaments(r.Context(), caller, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]tournamentResponse, len(list))
	for i, t := range list {
		out[i] = toTournamentResponse(t)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Teams ---

func (a *API) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	team, err := a.svc.CreateTeam(r.Context(), caller, orgID, req.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTeamResponse(team))
}

func (a *API) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	id := domain.TeamID(chi.URLParam(r, "teamID"))

	team, err := a.svc.GetTeam(r.Context(), caller, orgID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTeamResponse(team))
}

func (a *API) handleListTeams(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	list, err := a.svc.ListTeams(r.Context(), caller, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]teamResponse, len(list))
	for i, t := range list {
		out[i] = toTeamResponse(t)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Matches ---

func (a *API) handleCreateMatch(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	var req createMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var tournamentID *domain.TournamentID
	if req.TournamentID != nil {
		tid := domain.TournamentID(*req.TournamentID)
		tournamentID = &tid
	}

	var conditionsInput *service.PlayingConditionsInput
	if req.PlayingConditions != nil {
		types, tier := req.PlayingConditions.toDomain()
		conditionsInput = &service.PlayingConditionsInput{
			ReviewQuotaPerInnings: req.PlayingConditions.ReviewQuotaPerInnings,
			DecisionTypesEnabled:  types,
			CameraTier:            tier,
		}
	}

	m, err := a.svc.CreateMatch(r.Context(), caller, orgID, tournamentID, domain.TeamID(req.HomeTeamID), domain.TeamID(req.AwayTeamID), req.Venue, req.ScheduledAt, conditionsInput)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMatchResponse(m))
}

func (a *API) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	id := domain.MatchID(chi.URLParam(r, "matchID"))

	m, err := a.svc.GetMatch(r.Context(), caller, orgID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMatchResponse(m))
}

func (a *API) handleListMatches(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	list, err := a.svc.ListMatches(r.Context(), caller, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]matchResponse, len(list))
	for i, m := range list {
		out[i] = toMatchResponse(m)
	}
	writeJSON(w, http.StatusOK, out)
}

// --- Roster ---

func (a *API) handleAddRosterEntry(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))

	var req addRosterEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// The consent check (internal/service/roster.go -> ConsentChecker)
	// calls Identity & Access on the caller's own behalf, so it needs the
	// caller's raw token, not just the parsed Caller.
	token := bearerToken(r)

	entry, err := a.svc.AddPlayerToRoster(r.Context(), caller, token, orgID, matchID, domain.TeamID(req.TeamID), domain.UserID(req.UserID))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toRosterEntryResponse(entry))
}

func (a *API) handleRemoveRosterEntry(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))
	userID := domain.UserID(chi.URLParam(r, "userID"))

	if err := a.svc.RemovePlayerFromRoster(r.Context(), caller, orgID, matchID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleListRoster(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))

	list, err := a.svc.ListRoster(r.Context(), caller, orgID, matchID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]rosterEntryResponse, len(list))
	for i, e := range list {
		out[i] = toRosterEntryResponse(e)
	}
	writeJSON(w, http.StatusOK, out)
}
