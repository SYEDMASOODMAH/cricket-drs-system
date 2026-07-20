// Package httpapi is media-ingest-gateway's transport layer: a chi router
// and handlers that decode requests, call into internal/service, and
// encode responses. Per rules.md Section 1, no business logic or
// persistence concern lives here.
package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/service"
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

// handleUploadClip takes the clip's raw bytes as the request body (no
// multipart parsing — see the implementation plan's transport note) and
// the capturing camera's ID as a query parameter.
func (a *API) handleUploadClip(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))
	cameraID := domain.CameraID(r.URL.Query().Get("camera_id"))

	defer func() { _ = r.Body.Close() }()
	clip, err := a.svc.UploadClip(r.Context(), caller, orgID, matchID, cameraID, r.Body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toClipResponse(clip))
}

func (a *API) handleGetClip(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))
	clipID := domain.ClipID(chi.URLParam(r, "clipID"))

	clip, err := a.svc.GetClip(r.Context(), caller, orgID, matchID, clipID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toClipResponse(clip))
}

func (a *API) handleListClips(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))

	list, err := a.svc.ListClips(r.Context(), caller, orgID, matchID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]clipResponse, len(list))
	for i, c := range list {
		out[i] = toClipResponse(c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleDownloadClip(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))
	clipID := domain.ClipID(chi.URLParam(r, "clipID"))

	clip, content, err := a.svc.DownloadClip(r.Context(), caller, orgID, matchID, clipID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	defer func() { _ = content.Close() }()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(clip.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, content); err != nil {
		slog.Error("httpapi: failed to stream clip content", "error", err)
	}
}
