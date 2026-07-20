package httpapi_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/httpapi"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/memstore"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/objectstore"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/service"
	"github.com/cricketdrs/services/observability"
)

// fakeVerifier maps fixed token strings to fixed claims, standing in for
// tokens Identity & Access would have issued — same pattern as
// match-tournament's httpapi tests.
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
	svc := service.New(memstore.NewClipStore(), objectstore.NewMemoryStore(), verifier)

	obs, err := observability.New("media-ingest-gateway-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return httpapi.New(svc, obs).Router()
}

func doRaw(t *testing.T, h http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	h := newTestAPI(t)
	rec := doRaw(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestProtectedEndpoint_MissingTokenReturns401(t *testing.T) {
	h := newTestAPI(t)
	rec := doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func uploadClip(t *testing.T, h http.Handler, token, orgID, matchID, cameraID string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	path := "/v1/organizations/" + orgID + "/matches/" + matchID + "/clips?camera_id=" + cameraID
	return doRaw(t, h, http.MethodPost, path, token, content)
}

func TestUploadClip_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("video-bytes"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}
	var clip struct {
		ID          string `json:"id"`
		SizeBytes   int64  `json:"size_bytes"`
		ContentHash string `json:"content_hash"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &clip); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if clip.ID == "" || clip.SizeBytes != int64(len("video-bytes")) || clip.ContentHash == "" {
		t.Fatalf("unexpected clip response: %+v", clip)
	}
}

func TestUploadClip_PermissionDeniedReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgAPlayerToken, "org-a", "match-1", "cam-1", []byte("video-bytes"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestUploadClip_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgBAdminToken, "org-a", "match-1", "cam-1", []byte("video-bytes"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestUploadClip_EmptyContentReturns400(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetClip_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("video-bytes"))
	var uploaded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	rec = doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips/"+uploaded.ID, orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetClip_NotFoundReturns404(t *testing.T) {
	h := newTestAPI(t)
	rec := doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips/does-not-exist", orgAAdminToken, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestListClips_EndToEnd(t *testing.T) {
	h := newTestAPI(t)
	uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("clip-one"))
	uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("clip-two"))

	rec := doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 clips, got %d", len(list))
	}
}

func TestDownloadClip_RoundTrip(t *testing.T) {
	h := newTestAPI(t)
	content := []byte("the-actual-video-bytes")
	rec := uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", content)
	var uploaded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	rec = doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips/"+uploaded.ID+"/download", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), content) {
		t.Fatalf("downloaded content mismatch: got %q, want %q", rec.Body.Bytes(), content)
	}
	if rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("unexpected content type: %q", rec.Header().Get("Content-Type"))
	}
}

func TestDownloadClip_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(t)
	rec := uploadClip(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("video-bytes"))
	var uploaded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	rec = doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips/"+uploaded.ID+"/download", orgBAdminToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}
