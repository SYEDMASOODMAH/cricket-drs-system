package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
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
// rules.md Section 4.2 — same pattern as identity-access's and
// match-tournament's internal/httpapi/respond.go.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrClipNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrMatchIDEmpty),
		errors.Is(err, domain.ErrCameraIDEmpty),
		errors.Is(err, domain.ErrEmptyContent),
		errors.Is(err, domain.ErrSyncSelfReference),
		errors.Is(err, domain.ErrInvalidCorrelation),
		errors.Is(err, domain.ErrInvalidRole):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrPermissionDenied),
		errors.Is(err, domain.ErrCrossTenantAccess):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		slog.Error("httpapi: unmapped service error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
