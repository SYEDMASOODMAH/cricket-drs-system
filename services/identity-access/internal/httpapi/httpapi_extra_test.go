package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetUser_EndToEnd(t *testing.T) {
	h := newTestAPI()
	orgID, adminID, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/"+orgID+"/users/"+adminID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var u struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if u.ID != adminID || u.Role != "organizer_admin" {
		t.Fatalf("unexpected user response: %+v", u)
	}
}

func TestGetUser_NotFoundReturns404(t *testing.T) {
	h := newTestAPI()
	orgID, _, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/"+orgID+"/users/does-not-exist", token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestAssignRole_EndToEnd(t *testing.T) {
	h := newTestAPI()
	orgID, _, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users", token, map[string]string{
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

	rec = doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users/"+player.ID+"/roles", token, map[string]string{
		"role": "coach",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var u struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if u.Role != "coach" {
		t.Fatalf("expected role coach, got %q", u.Role)
	}
}

func TestAssignRole_InvalidRoleReturns400(t *testing.T) {
	h := newTestAPI()
	orgID, adminID, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users/"+adminID+"/roles", token, map[string]string{
		"role": "astronaut",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestAssignRole_PermissionDeniedReturns403(t *testing.T) {
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

	rec = doJSON(t, h, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"organization_id": orgID,
		"email":           "player@mca.example",
		"password":        "password123",
	})
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users/"+player.ID+"/roles", loginResp.Token, map[string]string{
		"role": "coach",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateOrganization_InvalidBodyReturns400(t *testing.T) {
	h := newTestAPI()
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations", "", map[string]string{
		"name":           "",
		"admin_email":    "admin@example.com",
		"admin_password": "password123",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty org name, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_InvalidBodyReturns400(t *testing.T) {
	h := newTestAPI()
	rec := doJSON(t, h, http.MethodPost, "/v1/auth/login", "", "not-an-object")
	// A JSON string decodes fine into loginRequest's zero value under
	// encoding/json's loose typing for a non-object top-level value only
	// if the target were `any`; against a struct it fails to unmarshal.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetConsent_NotFoundReturns404(t *testing.T) {
	h := newTestAPI()
	_, adminID, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodGet, "/v1/users/"+adminID+"/consent", token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCaptureConsent_MinorWithoutGuardianReturns400(t *testing.T) {
	h := newTestAPI()
	orgID, _, token := bootstrapViaHTTP(t, h, "MCA", "admin@mca.example", "password123")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/users", token, map[string]string{
		"email":    "minor@mca.example",
		"password": "password123",
		"role":     "player",
	})
	var minor struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &minor); err != nil {
		t.Fatalf("decode invite-user response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/users/"+minor.ID+"/consent", token, map[string]any{
		"consent_type": "video_capture",
		"grant":        true,
		"is_minor":     true,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}
