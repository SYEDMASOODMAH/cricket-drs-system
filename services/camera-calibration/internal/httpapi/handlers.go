// Package httpapi is camera-calibration's transport layer: a chi router
// and handlers that decode requests, call into internal/service, and
// encode responses. Per rules.md Section 1, no business logic or
// persistence concern lives here.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
	"github.com/cricketdrs/services/camera-calibration/internal/service"
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

func (a *API) handleRegisterCamera(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	var req registerCameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	camera, err := a.svc.RegisterCamera(r.Context(), caller, orgID, domain.VenueID(req.VenueID), domain.CameraModel(req.Model))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCameraResponse(camera))
}

func (a *API) handleGetCamera(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	cameraID := domain.CameraID(chi.URLParam(r, "cameraID"))

	camera, err := a.svc.GetCamera(r.Context(), caller, orgID, cameraID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCameraResponse(camera))
}

func (a *API) handleListCameras(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	list, err := a.svc.ListCameras(r.Context(), caller, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]cameraResponse, len(list))
	for i, c := range list {
		out[i] = toCameraResponse(c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleStoreCalibration(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	cameraID := domain.CameraID(chi.URLParam(r, "cameraID"))

	var req storeCalibrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := a.svc.StoreCalibrationProfile(r.Context(), caller, orgID, cameraID, req.Rotation, req.Translation, req.ReprojectionErrorPx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCalibrationProfileResponse(profile))
}

func (a *API) handleGetCalibrationStatus(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	cameraID := domain.CameraID(chi.URLParam(r, "cameraID"))

	profile, err := a.svc.GetCalibrationStatus(r.Context(), caller, orgID, cameraID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCalibrationProfileResponse(profile))
}

// handleGetLensProfile is not tenant-scoped — a camera model's
// intrinsic lens profile is reference data, not org-owned state (see
// domain/lens_profile.go) — so it only needs a valid authenticated caller
// (enforced by requireAuth in the router), not any org/role check.
func (a *API) handleGetLensProfile(w http.ResponseWriter, r *http.Request) {
	model := domain.CameraModel(chi.URLParam(r, "model"))

	profile, err := a.svc.GetLensProfile(model)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toLensProfileResponse(profile))
}
