package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

func TestGetTournament_Success(t *testing.T) {
	svc := newTestService(nil)
	tour, err := svc.CreateTournament(context.Background(), orgAAdmin, "org-a", "League", PlayingConditionsInput{CameraTier: domain.CameraTierAccessible})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetTournament(context.Background(), orgAAdmin, "org-a", tour.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != tour.ID {
		t.Fatalf("unexpected tournament: %+v", got)
	}
}

func TestGetTeam_Success(t *testing.T) {
	svc := newTestService(nil)
	home, _ := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	got, err := svc.GetTeam(context.Background(), orgAAdmin, "org-a", home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != home {
		t.Fatalf("unexpected team: %+v", got)
	}
}

func TestListTeams_TenantIsolation(t *testing.T) {
	svc := newTestService(nil)
	setupTwoTeams(t, svc, orgAAdmin, "org-a")
	if _, err := svc.CreateTeam(context.Background(), orgBAdmin, "org-b", "Org B Team"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := svc.ListTeams(context.Background(), orgAAdmin, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 2 {
		t.Fatalf("expected org-a to see exactly its own 2 teams, got %+v", listA)
	}
}

func TestGetMatch_Success(t *testing.T) {
	svc := newTestService(nil)
	matchID, _ := setupMatch(t, svc, orgAAdmin, "org-a")
	got, err := svc.GetMatch(context.Background(), orgAAdmin, "org-a", matchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != matchID {
		t.Fatalf("unexpected match: %+v", got)
	}
}

func TestListMatches_TenantIsolation(t *testing.T) {
	svc := newTestService(nil)
	setupMatch(t, svc, orgAAdmin, "org-a")
	setupMatch(t, svc, orgBAdmin, "org-b")

	listA, err := svc.ListMatches(context.Background(), orgAAdmin, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see exactly 1 match, got %+v", listA)
	}
}

func TestRemovePlayerFromRoster_PermissionDenied(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")
	if _, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := svc.RemovePlayerFromRoster(context.Background(), orgAPlayer, "org-a", matchID, "player-1")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestRemovePlayerFromRoster_CrossTenantRejected(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	matchID, teamID := setupMatch(t, svc, orgAAdmin, "org-a")
	if _, err := svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", matchID, teamID, "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := svc.RemovePlayerFromRoster(context.Background(), orgBAdmin, "org-a", matchID, "player-1")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestAddPlayerToRoster_UnknownTeamRejected(t *testing.T) {
	svc := newTestService(map[domain.UserID]bool{"player-1": true})
	home, away := setupTwoTeams(t, svc, orgAAdmin, "org-a")
	m, err := svc.CreateMatch(context.Background(), orgAAdmin, "org-a", nil, home, away, "Ground", time.Now(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.AddPlayerToRoster(context.Background(), orgAAdmin, "caller-token", "org-a", m.ID, "does-not-exist", "player-1")
	if !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
}
