// Package httpapi is identity-access's transport layer: a chi router and
// handlers that decode requests, call into internal/service, and encode
// responses. Per rules.md Section 1, no business logic or persistence
// concern lives here — every handler is a thin adapter over a service.Service
// method.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cricketdrs/services/identity-access/internal/domain"
	"github.com/cricketdrs/services/identity-access/internal/service"
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

func (a *API) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req createOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	org, admin, err := a.svc.CreateOrganization(r.Context(), req.Name, req.AdminEmail, req.AdminPassword)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, createOrganizationResponse{
		Organization: toOrganizationResponse(org),
		Admin:        toUserResponse(admin),
	})
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := a.svc.Login(r.Context(), domain.OrganizationID(req.OrganizationID), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func (a *API) handleInviteUser(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))

	var req inviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := a.svc.InviteUser(r.Context(), caller, orgID, req.Email, req.Password, domain.Role(req.Role))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

func (a *API) handleGetUser(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	userID := domain.UserID(chi.URLParam(r, "userID"))

	u, err := a.svc.GetUser(r.Context(), caller, orgID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (a *API) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	userID := domain.UserID(chi.URLParam(r, "userID"))

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := a.svc.AssignRole(r.Context(), caller, orgID, userID, domain.Role(req.Role))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

// handleCaptureConsent and handleGetConsent take their OrganizationID from
// the authenticated caller, not the URL — a user only ever manages consent
// within their own tenant context (see internal/service/consent.go).

func (a *API) handleCaptureConsent(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	userID := domain.UserID(chi.URLParam(r, "userID"))

	var req consentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var guardianUserID *domain.UserID
	if req.GuardianUserID != nil {
		g := domain.UserID(*req.GuardianUserID)
		guardianUserID = &g
	}

	rec, err := a.svc.CaptureConsent(r.Context(), caller, caller.OrganizationID, userID, req.IsMinor, guardianUserID, domain.ConsentType(req.ConsentType), req.Grant)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toConsentResponse(rec))
}

func (a *API) handleGetConsent(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	userID := domain.UserID(chi.URLParam(r, "userID"))

	rec, err := a.svc.GetConsent(r.Context(), caller, caller.OrganizationID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toConsentResponse(rec))
}
