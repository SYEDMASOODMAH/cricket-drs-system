package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("httpapi: failed to encode response body", "error", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// writeServiceError maps a domain sentinel error to an HTTP status, per
// rules.md Section 4.2 (callers distinguish failure classes via
// errors.Is/As, not string matching) — same pattern as identity-access's
// internal/httpapi/respond.go.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTournamentNotFound),
		errors.Is(err, domain.ErrTeamNotFound),
		errors.Is(err, domain.ErrMatchNotFound),
		errors.Is(err, domain.ErrRosterEntryNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrRosterEntryExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrNameEmpty),
		errors.Is(err, domain.ErrVenueEmpty),
		errors.Is(err, domain.ErrUserIDEmpty),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidDecisionType),
		errors.Is(err, domain.ErrInvalidCameraTier),
		errors.Is(err, domain.ErrInvalidReviewQuota),
		errors.Is(err, domain.ErrInvalidTeamPairing):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrPermissionDenied),
		errors.Is(err, domain.ErrCrossTenantAccess),
		errors.Is(err, domain.ErrConsentRequired):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("httpapi: unmapped service error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
