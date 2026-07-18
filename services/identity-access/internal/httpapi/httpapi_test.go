package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cricketdrs/services/identity-access/internal/httpapi"
	"github.com/cricketdrs/services/identity-access/internal/memstore"
	"github.com/cricketdrs/services/identity-access/internal/security"
	"github.com/cricketdrs/services/identity-access/internal/service"
)

func newTestAPI() http.Handler {
	svc := service.New(
		memstore.NewOrganizationStore(),
		memstore.NewUserStore(),
		memstore.NewConsentStore(),
		security.NewBcryptHasher(),
		security.NewJWTIssuer([]byte("test-signing-key-do-not-use-in-prod")),
	)
	return httpapi.New(svc).Router()
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// bootstrapViaHTTP creates an org and logs in as its admin through the
// actual HTTP handlers, returning the org ID, admin user ID, and a bearer
// token — the end-to-end path a real client exercises.
func bootstrapViaHTTP(t *testing.T, h http.Handler, name, email, password string) (orgID, adminID, token string) {
	t.Helper()

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations", "", map[string]string{
		"name":           name,
		"admin_email":    email,
		"admin_password": password,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create organization: status %d, body %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
		Admin struct {
			ID string `json:"id"`
		} `json:"admin"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create-organization response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"organization_id": created.Organization.ID,
		"email":           email,
		"password":        password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	return created.Organization.ID, created.Admin.ID, loginResp.Token
}

func TestHealthz(t *testing.T) {
	h := newTestAPI()
	rec := doJSON(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateOrganizationAndLogin(t *testing.T) {
	h := newTestAPI()
	orgID, adminID, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")
	if orgID == "" || adminID == "" || token == "" {
		t.Fatalf("expected non-empty org/admin/token, got %q %q %q", orgID, adminID, token)
	}
}

func TestLogin_WrongPasswordReturns401(t *testing.T) {
	h := newTestAPI()
	orgID, _, _ := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"organization_id": orgID,
		"email":           "admin@mca.example",
		"password":        "wrong",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestProtectedEndpoint_MissingTokenReturns401(t *testing.T) {
	h := newTestAPI()
	orgID, _, _ := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users", "", map[string]string{
		"email":    "player@mca.example",
		"password": "password123",
		"role":     "player",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestInviteUser_EndToEnd(t *testing.T) {
	h := newTestAPI()
	orgID, _, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users", token, map[string]string{
		"email":    "player@mca.example",
		"password": "password123",
		"role":     "player",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}
	var u struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if u.Role != "player" || u.ID == "" {
		t.Fatalf("unexpected user response: %+v", u)
	}
}

func TestInviteUser_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI()
	_, _, tokenA := bootstrapViaHTTP(t, h, "Org A", "admin-a@example.com", "password123")
	orgBID, _, _ := bootstrapViaHTTP(t, h, "Org B", "admin-b@example.com", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgBID+"/users", tokenA, map[string]string{
		"email":    "intruder@example.com",
		"password": "password123",
		"role":     "player",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestConsentFlow_CaptureAndGet(t *testing.T) {
	h := newTestAPI()
	orgID, _, adminToken := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users", adminToken, map[string]string{
		"email":    "player@mca.example",
		"password": "password123",
		"role":     "player",
	})
	var player struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &player); err != nil {
		t.Fatalf("decode invite-user response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/users/"+player.ID+"/consent", adminToken, map[string]any{
		"consent_type": "video_capture",
		"grant":        true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("capture consent: status %d, body %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/v1/users/"+player.ID+"/consent", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get consent: status %d, body %s", rec.Code, rec.Body.String())
	}
	var consent struct {
		Grants map[string]bool `json:"grants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &consent); err != nil {
		t.Fatalf("decode consent response: %v", err)
	}
	if !consent.Grants["video_capture"] {
		t.Fatalf("expected video_capture granted, got %+v", consent.Grants)
	}
}

func TestInviteUser_InvalidBodyReturns400(t *testing.T) {
	h := newTestAPI()
	orgID, _, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID+"/users", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}
