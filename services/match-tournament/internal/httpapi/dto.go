package httpapi

import (
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type playingConditionsDTO struct {
	ReviewQuotaPerInnings int      `json:"review_quota_per_innings"`
	DecisionTypesEnabled  []string `json:"decision_types_enabled"`
	CameraTier            string   `json:"camera_tier"`
}

func toPlayingConditionsDTO(pc domain.PlayingConditions) playingConditionsDTO {
	types := make([]string, len(pc.DecisionTypesEnabled))
	for i, dt := range pc.DecisionTypesEnabled {
		types[i] = string(dt)
	}
	return playingConditionsDTO{
		ReviewQuotaPerInnings: pc.ReviewQuotaPerInnings,
		DecisionTypesEnabled:  types,
		CameraTier:            string(pc.CameraTier),
	}
}

func (d playingConditionsDTO) toDomain() ([]domain.DecisionType, domain.CameraTier) {
	types := make([]domain.DecisionType, len(d.DecisionTypesEnabled))
	for i, dt := range d.DecisionTypesEnabled {
		types[i] = domain.DecisionType(dt)
	}
	return types, domain.CameraTier(d.CameraTier)
}

type tournamentResponse struct {
	ID                string               `json:"id"`
	OrganizationID    string               `json:"organization_id"`
	Name              string               `json:"name"`
	DefaultConditions playingConditionsDTO `json:"default_conditions"`
}

func toTournamentResponse(t domain.Tournament) tournamentResponse {
	return tournamentResponse{
		ID:                string(t.ID),
		OrganizationID:    string(t.OrganizationID),
		Name:              t.Name,
		DefaultConditions: toPlayingConditionsDTO(t.DefaultConditions),
	}
}

type createTournamentRequest struct {
	Name              string               `json:"name"`
	DefaultConditions playingConditionsDTO `json:"default_conditions"`
}

type teamResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

func toTeamResponse(t domain.Team) teamResponse {
	return teamResponse{ID: string(t.ID), OrganizationID: string(t.OrganizationID), Name: t.Name}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type matchResponse struct {
	ID                string               `json:"id"`
	OrganizationID    string               `json:"organization_id"`
	TournamentID      *string              `json:"tournament_id,omitempty"`
	HomeTeamID        string               `json:"home_team_id"`
	AwayTeamID        string               `json:"away_team_id"`
	Venue             string               `json:"venue"`
	ScheduledAt       time.Time            `json:"scheduled_at"`
	PlayingConditions playingConditionsDTO `json:"playing_conditions"`
}

func toMatchResponse(m domain.Match) matchResponse {
	var tournamentID *string
	if m.TournamentID != nil {
		s := string(*m.TournamentID)
		tournamentID = &s
	}
	return matchResponse{
		ID:                string(m.ID),
		OrganizationID:    string(m.OrganizationID),
		TournamentID:      tournamentID,
		HomeTeamID:        string(m.HomeTeamID),
		AwayTeamID:        string(m.AwayTeamID),
		Venue:             m.Venue,
		ScheduledAt:       m.ScheduledAt,
		PlayingConditions: toPlayingConditionsDTO(m.PlayingConditions),
	}
}

type createMatchRequest struct {
	TournamentID      *string               `json:"tournament_id,omitempty"`
	HomeTeamID        string                `json:"home_team_id"`
	AwayTeamID        string                `json:"away_team_id"`
	Venue             string                `json:"venue"`
	ScheduledAt       time.Time             `json:"scheduled_at"`
	PlayingConditions *playingConditionsDTO `json:"playing_conditions,omitempty"`
}

type rosterEntryResponse struct {
	MatchID string    `json:"match_id"`
	TeamID  string    `json:"team_id"`
	UserID  string    `json:"user_id"`
	AddedAt time.Time `json:"added_at"`
	AddedBy string    `json:"added_by"`
}

func toRosterEntryResponse(e domain.RosterEntry) rosterEntryResponse {
	return rosterEntryResponse{
		MatchID: string(e.MatchID),
		TeamID:  string(e.TeamID),
		UserID:  string(e.UserID),
		AddedAt: e.AddedAt,
		AddedBy: string(e.AddedBy),
	}
}

type addRosterEntryRequest struct {
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
}
