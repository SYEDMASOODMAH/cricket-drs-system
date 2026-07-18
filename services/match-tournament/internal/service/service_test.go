package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
	"github.com/cricketdrs/services/match-tournament/internal/memstore"
)

// fakeConsentChecker lets service-layer tests exercise the consent gate
// (and its error path) without a real Identity & Access instance —
// rules.md Section 1: domain/use-case logic must be unit-testable without
// a network call.
type fakeConsentChecker struct {
	eligible map[domain.UserID]bool
	err      error
}

func (f *fakeConsentChecker) IsEligibleForDRS(_ context.Context, _ string, userID domain.UserID) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.eligible[userID], nil
}

type fakeTokenVerifier struct {
	claims Claims
	err    error
}

func (f *fakeTokenVerifier) Verify(_ string) (Claims, error) {
	return f.claims, f.err
}

func newTestService(consentEligible map[domain.UserID]bool) *Service {
	return New(
		memstore.NewTournamentStore(),
		memstore.NewTeamStore(),
		memstore.NewMatchStore(),
		memstore.NewRosterStore(),
		&fakeTokenVerifier{},
		&fakeConsentChecker{eligible: consentEligible},
	)
}

var orgAAdmin = Caller{OrganizationID: "org-a", UserID: "admin-a", Role: domain.RoleOrganizerAdmin}
var orgBAdmin = Caller{OrganizationID: "org-b", UserID: "admin-b", Role: domain.RoleOrganizerAdmin}
var orgAPlayer = Caller{OrganizationID: "org-a", UserID: "player-a", Role: domain.RolePlayer}

func TestCreateTournament_Success(t *testing.T) {
	svc := newTestService(nil)
	conditions := PlayingConditionsInput{ReviewQuotaPerInnings: 2, DecisionTypesEnabled: []domain.DecisionType{domain.DecisionTypeLBW}, CameraTier: domain.CameraTierAccessible}

	tour, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "MCA Summer League", conditions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tour.Name != "MCA Summer League" || tour.OrganizationID != "org-a" {
		t.Fatalf("unexpected tournament: %+v", tour)
	}
}

func TestCreateTournament_CrossTenantRejected(t *testing.T) {
	svc := newTestService(nil)
	_, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-b", "Intruder League", PlayingConditionsInput{})
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestCreateTournament_PermissionDenied(t *testing.T) {
	svc := newTestService(nil)
	_, err := svc.CreateTournament(context.Background(), orgAPlayer, "org-a", "League", PlayingConditionsInput{})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestGetTournament_CrossTenantRejected(t *testing.T) {
	svc := newTestService(nil)
	tour, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "League", PlayingConditionsInput{CameraTier: domain.CameraTierAccessible})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = svc.GetTournament(context.Background(), orgBAdmin, "org-a", tour.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestListTournaments_TenantIsolation(t *testing.T) {
	svc := newTestService(nil)
	if _, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "League A", PlayingConditionsInput{CameraTier: domain.CameraTierAccessible}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.CreateTournament(context.Background(), orgBAdmin, "org-b", "League B", PlayingConditionsInput{CameraTier: domain.CameraTierAccessible}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := svc.ListTournaments(context.Background(), orgAAdmin, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 || listA[0].Name != "League A" {
		t.Fatalf("expected only org-a's tournament, got %+v", listA)
	}
}

func TestCreateTeam_Success(t *testing.T) {
	svc := newTestService(nil)
	team, err := svc.CreateTeam(context.Background(), orgAAdmin, "org-a", "Montreal Overseas CC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name != "Montreal Overseas CC" {
		t.Fatalf("unexpected team: %+v", team)
	}
}

func TestCreateTeam_PermissionDenied(t *testing.T) {
	svc := newTestService(nil)
	_, err := svc.CreateTeam(context.Background(), orgAPlayer, "org-a", "Team")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func setupTwoTeams(t *testing.T, svc *Service, caller Caller, orgID domain.OrganizationID) (domain.TeamID, domain.TeamID) {
	t.Helper()
	home, err := svc.CreateTeam(context.Background(), caller, orgID, "Home CC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	away, err := svc.CreateTeam(context.Background(), caller, orgID, "Away CC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return home.ID, away.ID
}

func TestCreateMatch_Success(t *testing.T) {
	svc := newTestService(nil)
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	conditions := PlayingConditionsInput{ReviewQuotaPerInnings: 2, DecisionTypesEnabled: []domain.DecisionType{domain.DecisionTypeLBW}, CameraTier: domain.CameraTierAccessible}

	m, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", nil, home, away, "MCA Ground 1", time.Now(), &conditions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.HomeTeamID != home || m.AwayTeamID != away || m.PlayingConditions.ReviewQuotaPerInnings != 2 {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestCreateMatch_InheritsTournamentDefaultConditions(t *testing.T) {
	svc := newTestService(nil)
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	defaults := PlayingConditionsInput{ReviewQuotaPerInnings: 3, DecisionTypesEnabled: []domain.DecisionType{domain.DecisionTypeRunOut}, CameraTier: domain.CameraTierBroadcast}
	tour, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "League", defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", &tour.ID, home, away, "MCA Ground 1", time.Now(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PlayingConditions.ReviewQuotaPerInnings != 3 || m.PlayingConditions.CameraTier != domain.CameraTierBroadcast {
		t.Fatalf("expected match to inherit tournament defaults, got %+v", m.PlayingConditions)
	}
}

func TestCreateMatch_ExplicitConditionsOverrideTournamentDefaults(t *testing.T) {
	svc := newTestService(nil)
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	defaults := PlayingConditionsInput{ReviewQuotaPerInnings: 3, CameraTier: domain.CameraTierBroadcast}
	tour, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "League", defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	override := PlayingConditionsInput{ReviewQuotaPerInnings: 1, CameraTier: domain.CameraTierAccessible}

	m, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", &tour.ID, home, away, "MCA Ground 1", time.Now(), &override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PlayingConditions.ReviewQuotaPerInnings != 1 || m.PlayingConditions.CameraTier != domain.CameraTierAccessible {
		t.Fatalf("expected explicit conditions to override tournament defaults, got %+v", m.PlayingConditions)
	}
}

func TestCreateMatch_UnknownTeamRejected(t *testing.T) {
	svc := newTestService(nil)
	_, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	_, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", nil, "does-not-exist", away, "Ground", time.Now(), nil)
	if !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
}

func TestCreateMatch_UnknownTournamentRejected(t *testing.T) {
	svc := newTestService(nil)
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	missing := domain.TournamentID("does-not-exist")
	_, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", &missing, home, away, "Ground", time.Now(), nil)
	if !errors.Is(err, domain.ErrTournamentNotFound) {
		t.Fatalf("expected ErrTournamentNotFound, got %v", err)
	}
}

func TestCreateMatch_CrossTenantTeamRejected(t *testing.T) {
	svc := newTestService(nil)
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	// org-b's caller can't reference org-a's teams even by exact ID.
	_, err := svc.CreateMatch(context.Background(), orgBAdmin, "org-b", nil, home, away, "Ground", time.Now(), nil)
	if !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound (org-b must not see org-a's teams), got %v", err)
	}
}

func setupMatch(t *testing.T, svc *Service, caller Caller, orgID domain.OrganizationID) (domain.MatchID, domain.TeamID) {
	t.Helper()
	home, away := setupTwoTeams(t, svc, caller, orgID)
	m, err := svc.CreateMatch(context.Background(), caller, orgID, nil, home, away, "Ground", time.Now(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return m.ID, home
}

func TestAddPlayerToRoster_EligiblePlayerSucceeds(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")

	entry, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.UserID != "player-1" {
		t.Fatalf("unexpected roster entry: %+v", entry)
	}
}

func TestAddPlayerToRoster_NonConsentingPlayerRejected(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": false})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")

	_, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1")
	if !errors.Is(err, domain.ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}
}

func TestAddPlayerToRoster_ConsentCheckerErrorPropagates(t *testing.T) {
	svc := New(
		memstore.NewTournamentStore(),
		memstore.NewTeamStore(),
		memstore.NewMatchStore(),
		memstore.NewRosterStore(),
		&fakeTokenVerifier{},
		&fakeConsentChecker{err: errors.New("identity-access unreachable")},
	)
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")

	_, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1")
	if err == nil || errors.Is(err, domain.ErrConsentRequired) {
		t.Fatalf("expected the underlying consent-checker error to propagate, got %v", err)
	}
}

func TestAddPlayerToRoster_UnknownMatchRejected(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	_, teamID := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	_, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", "does-not-exist", teamID, "player-1")
	if !errors.Is(err, domain.ErrMatchNotFound) {
		t.Fatalf("expected ErrMatchNotFound, got %v", err)
	}
}

func TestAddPlayerToRoster_CrossTenantRejected(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")

	_, err := svc.AddPlayerToRoster(context.Background(), orgBAdmin, "caller-token", "org-a", matchID, teamID, "player-1")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestAddPlayerToRoster_PermissionDenied(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")

	_, err := svc.AddPlayerToRoster(context.Background(), orgAPlayer, "caller-token", "org-a", matchID, teamID, "player-1")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestRemovePlayerFromRoster_Success(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")
	if _, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.RemovePlayerFromRoster(context.Background(), orgAAdmin, "org-a", matchID, "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err := svc.ListRoster(context.Background(), orgAAdmin, "org-a", matchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty roster after removal, got %+v", list)
	}
}

func TestListRoster_CrossTenantRejected(t *testing.T) {
	svc := newTestService(nil)
	matchID, _ := setupMatch(t, svc, orgAAdmin, "org-a")
	_, err := svc.ListRoster(context.Background(), orgBAdmin, "org-a", matchID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	claims := Claims{UserID: "user-1", OrganizationID: "org-a", Role: domain.RoleOrganizerAdmin}
	svc := New(
		memstore.NewTournamentStore(), memstore.NewTeamStore(), memstore.NewMatchStore(), memstore.NewRosterStore(),
		&fakeTokenVerifier{claims: claims}, &fakeConsentChecker{},
	)
	caller, err := svc.Authenticate("some-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.UserID != "user-1" || caller.OrganizationID != "org-a" || caller.Role != domain.RoleOrganizerAdmin {
		t.Fatalf("unexpected caller: %+v", caller)
	}
}

func TestAuthenticate_InvalidTokenRejected(t *testing.T) {
	svc := New(
		memstore.NewTournamentStore(), memstore.NewTeamStore(), memstore.NewMatchStore(), memstore.NewRosterStore(),
		&fakeTokenVerifier{err: errors.New("bad token")}, &fakeConsentChecker{},
	)
	_, err := svc.Authenticate("garbage")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
