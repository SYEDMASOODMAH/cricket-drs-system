package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
	"github.com/cricketdrs/services/camera-calibration/internal/httpapi"
	"github.com/cricketdrs/services/camera-calibration/internal/memstore"
	"github.com/cricketdrs/services/camera-calibration/internal/service"
	"github.com/cricketdrs/services/observability"
)

// fakeVerifier maps fixed token strings to fixed claims, standing in for
// tokens Identity & Access would have issued — same pattern as the other
// three services' httpapi tests.
type fakeVerifier struct {
	claims map[string]service.Claims
}

func (f *fakeVerifier) Verify(token string) (service.Claims, error) {
	c, ok := f.claims[token]
	if !ok {
		return service.Claims{}, errors.New("invalid token")
	}
	return c, nil
}

const (
	orgAAdminToken  = "org-a-admin-token"
	orgAPlayerToken = "org-a-player-token"
	orgBAdminToken  = "org-b-admin-token"
)

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		orgAAdminToken:  {UserID: "admin-a", OrganizationID: "org-a", Role: domain.RoleOrganizerAdmin},
		orgAPlayerToken: {UserID: "player-a", OrganizationID: "org-a", Role: domain.RolePlayer},
		orgBAdminToken:  {UserID: "admin-b", OrganizationID: "org-b", Role: domain.RoleOrganizerAdmin},
	}}
	svc := service.New(memstore.NewCameraStore(), memstore.NewProfileStore(), verifier)

	obs, err := observability.New("camera-calibration-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return httpapi.New(svc, obs).Router()
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	h := newTestAPI(t)
	rec := doJSON(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestProtectedEndpoint_MissingTokenReturns401(t *testing.T) {
	h := newTestAPI(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func registerCamera(t *testing.T, h http.Handler, token, orgID, venueID, model string) *httptest.ResponseRecorder {
	t.Helper()
	return doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/cameras", token, map[string]string{
		"venue_id": venueID,
		"model":    model,
	})
}

func TestRegisterCamera_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}
	var camera struct {
		ID      string `json:"id"`
		VenueID string `json:"venue_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &camera); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if camera.ID == "" || camera.VenueID != "venue-1" {
		t.Fatalf("unexpected camera response: %+v", camera)
	}
}

func TestRegisterCamera_PermissionDeniedReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAPlayerToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterCamera_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgBAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterCamera_EmptyVenueReturns400(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "", "GoPro Hero 12 Black")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetCamera_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	rec = doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras/"+registered.ID, orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetCamera_NotFoundReturns404(t *testing.T) {
	h := newTestAPI(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras/does-not-exist", orgAAdminToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestListCameras_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	registerCamera(t, h, orgAAdminToken, "org-a", "venue-2", "GoPro Hero 12 Black")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 cameras, got %d", len(list))
	}
}

func storeCalibration(t *testing.T, h http.Handler, token, orgID, cameraID string, reprojectionErrorPx float64) *httptest.ResponseRecorder {
	t.Helper()
	return doJSON(t, h, http.MethodPut, "/v1/organizations/"+orgID+"/cameras/"+cameraID+"/calibration", token, map[string]any{
		"rotation":              [3]float64{0.1, 0.2, 0.3},
		"translation":           [3]float64{1, 2, 3},
		"reprojection_error_px": reprojectionErrorPx,
	})
}

func TestStoreCalibration_EndToEnd_Valid(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	rec = storeCalibration(t, h, orgAAdminToken, "org-a", registered.ID, 0.5)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var profile struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !profile.Valid {
		t.Fatalf("expected profile with error 0.5px to be valid")
	}
}

func TestStoreCalibration_HighErrorMarkedInvalid(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	rec = storeCalibration(t, h, orgAAdminToken, "org-a", registered.ID, 50.0)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var profile struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if profile.Valid {
		t.Fatalf("expected profile with error 50px to be invalid")
	}
}

func TestStoreCalibration_UnregisteredCameraReturns404(t *testing.T) {
	h := newTestAPI(t)
	rec := storeCalibration(t, h, orgAAdminToken, "org-a", "does-not-exist", 0.5)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetCalibrationStatus_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	storeCalibration(t, h, orgAAdminToken, "org-a", registered.ID, 0.5)

	rec = doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras/"+registered.ID+"/calibration", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetCalibrationStatus_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := registerCamera(t, h, orgAAdminToken, "org-a", "venue-1", "GoPro Hero 12 Black")
	var registered struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	storeCalibration(t, h, orgAAdminToken, "org-a", registered.ID, 0.5)

	rec = doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/cameras/"+registered.ID+"/calibration", orgBAdminToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetLensProfile_Known(t *testing.T) {
	h := newTestAPI(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/camera-models/GoPro%20Hero%2012%20Black/lens-profile", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var profile struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if profile.Model != "GoPro Hero 12 Black" {
		t.Fatalf("unexpected model: %q", profile.Model)
	}
}

func TestGetLensProfile_Unknown(t *testing.T) {
	h := newTestAPI(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/camera-models/Unsupported%20Camera/lens-profile", orgAAdminToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}
