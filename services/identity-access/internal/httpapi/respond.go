package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cricketdrs/services/identity-access/internal/domain"
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
// errors.Is/As, not string matching).
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrganizationNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrConsentNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrUserExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrOrganizationNameEmpty),
		errors.Is(err, domain.ErrUserEmailEmpty),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrGuardianConsentRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrPermissionDenied),
		errors.Is(err, domain.ErrCrossTenantAccess):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("httpapi: unmapped service error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
