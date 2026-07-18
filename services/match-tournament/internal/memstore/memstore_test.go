package memstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

func TestTournamentStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewTournamentStore()

	tourA := domain.Tournament{ID: "t1", OrganizationID: "org-a", Name: "League A"}
	tourB := domain.Tournament{ID: "t1", OrganizationID: "org-b", Name: "League B"}
	if err := s.Create(ctx, tourA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, tourB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "org-a", "t1")
	if err != nil || got.Name != "League A" {
		t.Fatalf("expected org-a's tournament, got %+v, err %v", got, err)
	}

	listA, err := s.ListByOrganization(ctx, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 || listA[0].Name != "League A" {
		t.Fatalf("expected only org-a's tournament in list, got %+v", listA)
	}
}

func TestTournamentStore_GetNotFound(t *testing.T) {
	s := NewTournamentStore()
	_, err := s.Get(context.Background(), "org-a", "missing")
	if !errors.Is(err, domain.ErrTournamentNotFound) {
		t.Fatalf("expected ErrTournamentNotFound, got %v", err)
	}
}

func TestTeamStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewTeamStore()
	teamA := domain.Team{ID: "team1", OrganizationID: "org-a", Name: "Overseas CC"}
	teamB := domain.Team{ID: "team1", OrganizationID: "org-b", Name: "Rivals CC"}
	if err := s.Create(ctx, teamA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, teamB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "org-b", "team1")
	if err != nil || got.Name != "Rivals CC" {
		t.Fatalf("expected org-b's team, got %+v, err %v", got, err)
	}
}

func TestTeamStore_GetNotFound(t *testing.T) {
	s := NewTeamStore()
	_, err := s.Get(context.Background(), "org-a", "missing")
	if !errors.Is(err, domain.ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
}

func TestMatchStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewMatchStore()
	matchA := domain.Match{ID: "m1", OrganizationID: "org-a", Venue: "Ground A"}
	matchB := domain.Match{ID: "m1", OrganizationID: "org-b", Venue: "Ground B"}
	if err := s.Create(ctx, matchA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, matchB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listB, err := s.ListByOrganization(ctx, "org-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listB) != 1 || listB[0].Venue != "Ground B" {
		t.Fatalf("expected only org-b's match in list, got %+v", listB)
	}
}

func TestMatchStore_GetNotFound(t *testing.T) {
	s := NewMatchStore()
	_, err := s.Get(context.Background(), "org-a", "missing")
	if !errors.Is(err, domain.ErrMatchNotFound) {
		t.Fatalf("expected ErrMatchNotFound, got %v", err)
	}
}

func TestRosterStore_AddListRemove(t *testing.T) {
	ctx := context.Background()
	s := NewRosterStore()
	entry := domain.RosterEntry{OrganizationID: "org-a", MatchID: "m1", TeamID: "team1", UserID: "user1", AddedAt: time.Now()}

	if err := s.Add(ctx, entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := s.ListByMatch(ctx, "org-a", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].UserID != "user1" {
		t.Fatalf("unexpected roster list: %+v", list)
	}

	if err := s.Remove(ctx, "org-a", "m1", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err = s.ListByMatch(ctx, "org-a", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty roster after removal, got %+v", list)
	}
}

func TestRosterStore_AddDuplicateFails(t *testing.T) {
	ctx := context.Background()
	s := NewRosterStore()
	entry := domain.RosterEntry{OrganizationID: "org-a", MatchID: "m1", TeamID: "team1", UserID: "user1"}
	if err := s.Add(ctx, entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Add(ctx, entry); !errors.Is(err, domain.ErrRosterEntryExists) {
		t.Fatalf("expected ErrRosterEntryExists, got %v", err)
	}
}

func TestRosterStore_RemoveNotFound(t *testing.T) {
	s := NewRosterStore()
	err := s.Remove(context.Background(), "org-a", "m1", "user1")
	if !errors.Is(err, domain.ErrRosterEntryNotFound) {
		t.Fatalf("expected ErrRosterEntryNotFound, got %v", err)
	}
}

func TestRosterStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewRosterStore()
	entryA := domain.RosterEntry{OrganizationID: "org-a", MatchID: "m1", TeamID: "team1", UserID: "user1"}
	entryB := domain.RosterEntry{OrganizationID: "org-b", MatchID: "m1", TeamID: "team1", UserID: "user1"}
	if err := s.Add(ctx, entryA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Add(ctx, entryB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := s.ListByMatch(ctx, "org-a", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see only its own roster entry, got %+v", listA)
	}

	// Removing org-a's entry must not affect org-b's identically-keyed one.
	if err := s.Remove(ctx, "org-a", "m1", "user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	listB, err := s.ListByMatch(ctx, "org-b", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listB) != 1 {
		t.Fatalf("expected org-b's roster entry to survive org-a's removal, got %+v", listB)
	}
}
