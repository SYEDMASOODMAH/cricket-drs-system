package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func createTournament(t *testing.T, h http.Handler, token, orgID, name string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/tournaments", token, map[string]any{
		"name": name,
		"default_conditions": map[string]any{
			"review_quota_per_innings": 2,
			"camera_tier":              "accessible",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tournament: status %d, body %s", rec.Code, rec.Body.String())
	}
	var tour struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tour); err != nil {
		t.Fatalf("decode tournament response: %v", err)
	}
	return tour.ID
}

func TestGetTournament_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	tourID := createTournament(t, h, orgAAdminToken, "org-a", "MCA Summer League")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/tournaments/"+tourID, orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestGetTournament_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(nil)
	tourID := createTournament(t, h, orgAAdminToken, "org-a", "MCA Summer League")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/tournaments/"+tourID, orgBAdminToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestListTournaments_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	createTournament(t, h, orgAAdminToken, "org-a", "League One")
	createTournament(t, h, orgAAdminToken, "org-a", "League Two")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/tournaments", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(list))
	}
}

func TestListTeams_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	createTeam(t, h, orgAAdminToken, "org-a", "Team One")
	createTeam(t, h, orgAAdminToken, "org-a", "Team Two")

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/teams", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(list))
	}
}

func createMatch(t *testing.T, h http.Handler, token, orgID, home, away string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/"+orgID+"/matches", token, map[string]any{
		"home_team_id": home, "away_team_id": away, "venue": "Ground", "scheduled_at": "2026-08-01T14:00:00Z",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create match: status %d, body %s", rec.Code, rec.Body.String())
	}
	var match struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &match); err != nil {
		t.Fatalf("decode match response: %v", err)
	}
	return match.ID
}

func TestGetMatch_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")
	matchID := createMatch(t, h, orgAAdminToken, "org-a", home, away)

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/matches/"+matchID, orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestListMatches_EndToEnd(t *testing.T) {
	h := newTestAPI(nil)
	home := createTeam(t, h, orgAAdminToken, "org-a", "Home CC")
	away := createTeam(t, h, orgAAdminToken, "org-a", "Away CC")
	createMatch(t, h, orgAAdminToken, "org-a", home, away)

	rec := doJSON(t, h, http.MethodGet, "/v1/organizations/org-a/matches", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 match, got %d", len(list))
	}
}

func TestCreateTournament_InvalidConditionsReturns400(t *testing.T) {
	h := newTestAPI(nil)
	rec := doJSON(t, h, http.MethodPost, "/v1/organizations/org-a/tournaments", orgAAdminToken, map[string]any{
		"name": "League",
		"default_conditions": map[string]any{
			"camera_tier": "imax",
		},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}
