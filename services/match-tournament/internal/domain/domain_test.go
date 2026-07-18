package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewTournament(t *testing.T) {
	conditions, err := NewPlayingConditions(2, []DecisionType{DecisionTypeLBW}, CameraTierAccessible)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tour, err := NewTournament("t1", "org-1", "MCA Summer League", conditions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tour.Name != "MCA Summer League" || tour.OrganizationID != "org-1" {
		t.Fatalf("unexpected tournament: %+v", tour)
	}
}

func TestNewTournament_EmptyName(t *testing.T) {
	_, err := NewTournament("t1", "org-1", "", PlayingConditions{})
	if !errors.Is(err, ErrNameEmpty) {
		t.Fatalf("expected ErrNameEmpty, got %v", err)
	}
}

func TestNewTeam(t *testing.T) {
	team, err := NewTeam("team-1", "org-1", "Montreal Overseas CC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name != "Montreal Overseas CC" {
		t.Fatalf("unexpected team: %+v", team)
	}
}

func TestNewTeam_EmptyName(t *testing.T) {
	_, err := NewTeam("team-1", "org-1", "")
	if !errors.Is(err, ErrNameEmpty) {
		t.Fatalf("expected ErrNameEmpty, got %v", err)
	}
}

func TestNewPlayingConditions_Valid(t *testing.T) {
	pc, err := NewPlayingConditions(2, []DecisionType{DecisionTypeLBW, DecisionTypeEdge}, CameraTierAccessible)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.ReviewQuotaPerInnings != 2 || len(pc.DecisionTypesEnabled) != 2 {
		t.Fatalf("unexpected playing conditions: %+v", pc)
	}
}

func TestNewPlayingConditions_NegativeQuotaRejected(t *testing.T) {
	_, err := NewPlayingConditions(-1, nil, CameraTierAccessible)
	if !errors.Is(err, ErrInvalidReviewQuota) {
		t.Fatalf("expected ErrInvalidReviewQuota, got %v", err)
	}
}

func TestNewPlayingConditions_InvalidTierRejected(t *testing.T) {
	_, err := NewPlayingConditions(2, nil, CameraTier("imax"))
	if !errors.Is(err, ErrInvalidCameraTier) {
		t.Fatalf("expected ErrInvalidCameraTier, got %v", err)
	}
}

func TestNewPlayingConditions_InvalidDecisionTypeRejected(t *testing.T) {
	_, err := NewPlayingConditions(2, []DecisionType{"boundary_catch"}, CameraTierAccessible)
	if !errors.Is(err, ErrInvalidDecisionType) {
		t.Fatalf("expected ErrInvalidDecisionType, got %v", err)
	}
}

func TestNewMatch_Valid(t *testing.T) {
	m, err := NewMatch("m1", "org-1", nil, "team-a", "team-b", "MCA Ground 1", time.Now(), PlayingConditions{CameraTier: CameraTierAccessible})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.HomeTeamID != "team-a" || m.AwayTeamID != "team-b" {
		t.Fatalf("unexpected match: %+v", m)
	}
}

func TestNewMatch_EmptyVenueRejected(t *testing.T) {
	_, err := NewMatch("m1", "org-1", nil, "team-a", "team-b", "", time.Now(), PlayingConditions{})
	if !errors.Is(err, ErrVenueEmpty) {
		t.Fatalf("expected ErrVenueEmpty, got %v", err)
	}
}

func TestNewMatch_SameTeamRejected(t *testing.T) {
	_, err := NewMatch("m1", "org-1", nil, "team-a", "team-a", "MCA Ground 1", time.Now(), PlayingConditions{})
	if !errors.Is(err, ErrInvalidTeamPairing) {
		t.Fatalf("expected ErrInvalidTeamPairing, got %v", err)
	}
}

func TestNewMatch_EmptyTeamRejected(t *testing.T) {
	_, err := NewMatch("m1", "org-1", nil, "", "team-b", "MCA Ground 1", time.Now(), PlayingConditions{})
	if !errors.Is(err, ErrInvalidTeamPairing) {
		t.Fatalf("expected ErrInvalidTeamPairing, got %v", err)
	}
}

func TestNewRosterEntry(t *testing.T) {
	now := time.Now()
	entry, err := NewRosterEntry("org-1", "m1", "team-a", "user-1", "organizer-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.UserID != "user-1" || entry.AddedBy != "organizer-1" || !entry.AddedAt.Equal(now) {
		t.Fatalf("unexpected roster entry: %+v", entry)
	}
}

func TestNewRosterEntry_EmptyUserIDRejected(t *testing.T) {
	_, err := NewRosterEntry("org-1", "m1", "team-a", "", "organizer-1", time.Now())
	if !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("expected ErrUserIDEmpty, got %v", err)
	}
}

func TestRoleValid(t *testing.T) {
	valid := []Role{RolePlayer, RoleCoach, RoleUmpire, RoleOrganizerAdmin, RoleBoardAdmin, RoleFan}
	for _, r := range valid {
		if !r.Valid() {
			t.Errorf("Role(%q).Valid() = false, want true", r)
		}
	}
	if Role("astronaut").Valid() {
		t.Error("expected unknown role to be invalid")
	}
}

func TestCanManageMatches(t *testing.T) {
	if !CanManageMatches(RoleOrganizerAdmin) {
		t.Error("expected organizer_admin to manage matches")
	}
	for _, r := range []Role{RolePlayer, RoleCoach, RoleUmpire, RoleBoardAdmin, RoleFan} {
		if CanManageMatches(r) {
			t.Errorf("expected %q to not manage matches", r)
		}
	}
}
