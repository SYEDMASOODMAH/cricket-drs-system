package domain

import "time"

// Match deliberately has no camera/calibration fields yet — Venue is a
// plain string. Camera registration and calibration profiles land in
// Phase 2 (phases.md) once the Camera Calibration Service exists.
type Match struct {
	ID             MatchID
	OrganizationID OrganizationID
	// TournamentID is optional: a standalone club friendly doesn't need to
	// belong to a tournament.
	TournamentID      *TournamentID
	HomeTeamID        TeamID
	AwayTeamID        TeamID
	Venue             string
	ScheduledAt       time.Time
	PlayingConditions PlayingConditions
}

func NewMatch(id MatchID, orgID OrganizationID, tournamentID *TournamentID, homeTeamID, awayTeamID TeamID, venue string, scheduledAt time.Time, conditions PlayingConditions) (Match, error) {
	if venue == "" {
		return Match{}, ErrVenueEmpty
	}
	if homeTeamID == "" || awayTeamID == "" || homeTeamID == awayTeamID {
		return Match{}, ErrInvalidTeamPairing
	}
	return Match{
		ID:                id,
		OrganizationID:    orgID,
		TournamentID:      tournamentID,
		HomeTeamID:        homeTeamID,
		AwayTeamID:        awayTeamID,
		Venue:             venue,
		ScheduledAt:       scheduledAt,
		PlayingConditions: conditions,
	}, nil
}
