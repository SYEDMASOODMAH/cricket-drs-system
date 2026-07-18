package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
	"github.com/cricketdrs/services/match-tournament/internal/httpapi"
	"github.com/cricketdrs/services/match-tournament/internal/memstore"
	"github.com/cricketdrs/services/match-tournament/internal/service"
)

// fakeVerifier maps fixed token strings to fixed claims, standing in for
// tokens Identity & Access would have issued — this service only verifies,
// it never mints, so tests can't just "log in" the way identity-access's
// httpapi tests do.
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

type fakeConsentChecker struct {
	eligible map[domain.UserID]bool
}

func (f *fakeConsentChecker) IsEligibleForDRS(_ context.Context, _ string, userID domain.UserID) (bool, error) {
	return f.eligible[userID], nil
}

const (
	orgAAdminToken  = "org-a-admin-token"
	orgAPlayerToken = "org-a-player-token"
	orgBAdminToken  = "org-b-admin-token"
)

func newTestAPI(consentEligible map[domain.UserID]bool) http.Handler {
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		orgAAdminToken:  {UserID: "admin-a", OrganizationID: "org-a", Role: domain.RoleOrganizerAdmin},
		orgAPlayerToken: {UserID: "player-a", OrganizationID: "org-a", Role: domain.RolePlayer},
		orgBAdminToken:  {UserID: "admin-b", OrganizationID: "org-b", Role: domain.RoleOrganizerAdmin},
	}}
	svc := service.New(
		memstore.NewTournamentStore(),
		memstore.NewTeamStore(),
		memstore.NewMatchStore(),
		memstore.NewRosterStore(),
		verifier,
		&fakeConsentChecker{eligible: consentEligible},
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

func createTeam(t *testing.T, h http.Handler, token, orgID, name string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/teams", token, map[string]string{"name": name})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create team: status %d, body %s", rec.Code, rec.Body.String())
	}
	var team struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &team); err != nil {
		t.Fatalf("decode team response: %v", err)
	}
	return team.ID
}

func TestHealthz(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestProtectedEndpoint_MissingTokenReturns401(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/tournaments", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestProtectedEndpoint_InvalidTokenReturns401(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/tournaments", "not-a-real-token", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTournament_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/tournaments", orgAAdminToken, map[string]any{
		"name": "MCA Summer League",
		"default_conditions": map[string]any{
			"review_quota_per_innings": 2,
			"decision_types_enabled":   []string{"lbw", "edge"},
			"camera_tier":              "accessible",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}
	var tour struct {
		Name              string `json:"name"`
		DefaultConditions struct {
			ReviewQuotaPerInnings int `json:"review_quota_per_innings"`
		} `json:"default_conditions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tour); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if tour.Name != "MCA Summer League" || tour.DefaultConditions.ReviewQuotaPerInnings != 2 {
		t.Fatalf("unexpected tournament response: %+v", tour)
	}
}

func TestCreateTournament_PermissionDeniedReturns403(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/tournaments", orgAPlayerToken, map[string]any{
		"name": "League",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTeamAndGet_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	teamID := createTeam(t, h, orgAAdminToken, "org-a", "Montreal Overseas CC")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/teams/"+teamID, orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateMatch_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/matches", orgAAdminToken, map[string]any{
		"home_team_id": home,
		"away_team_id": away,
		"venue":        "MCA Ground 1",
		"scheduled_at": "2026-08-01T14:00:00Z",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCreateMatch_CrossTenantTeamReturns404(t *testing.T) {
	h := newTestAPI(nil)
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")

	// org-b's admin references org-a's team IDs — must not resolve.
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-b/matches", orgBAdminToken, map[string]any{
		"home_team_id": home,
		"away_team_id": away,
		"venue":        "Ground",
		"scheduled_at": "2026-08-01T14:00:00Z",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestRosterFlow_EligiblePlayerSucceeds(t *testing.T) {
	h := newTestAPI(map[domain.UserID]bool{"player-1": true})
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/matches", orgAAdminToken, map[string]any{
		"home_team_id": home, "away_team_id": away, "venue": "Ground", "scheduled_at": "2026-08-01T14:00:00Z",
	})
	var match struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &match); err != nil {
		t.Fatalf("decode match response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/matches/"+match.ID+"/roster", orgAAdminToken, map[string]string{
		"team_id": home, "user_id": "player-1",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/matches/"+match.ID+"/roster", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var roster []struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &roster); err != nil {
		t.Fatalf("decode roster response: %v", err)
	}
	if len(roster) != 1 || roster[0].UserID != "player-1" {
		t.Fatalf("unexpected roster: %+v", roster)
	}

	rec = doJSON(t, h, http.MethodDelete, "/v1/organizations/org-a/matches/"+match.ID+"/roster/player-1", orgAAdminToken, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestRosterFlow_NonConsentingPlayerReturns403(t *testing.T) {
	h := newTestAPI(map[domain.UserID]bool{"player-1": false})
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")

	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/matches", orgAAdminToken, map[string]any{
		"home_team_id": home, "away_team_id": away, "venue": "Ground", "scheduled_at": "2026-08-01T14:00:00Z",
	})
	var match struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &match); err != nil {
		t.Fatalf("decode match response: %v", err)
	}

	rec = doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/matches/"+match.ID+"/roster", orgAAdminToken, map[string]string{
		"team_id": home, "user_id": "player-1",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 (consent required), got %d, body %s", rec.Code, rec.Body.String())
	}
}
