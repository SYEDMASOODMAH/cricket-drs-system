package service

import (
	"context"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

// CreateMatch validates that both teams (and the tournament, if given)
// exist within orgID before creating the match — a match can never
// reference another tenant's team or tournament, even if the caller
// somehow knew its ID. If conditions is nil and tournamentID is set, the
// tournament's DefaultConditions are copied in; an explicit conditions
// always wins. Copied rather than inherited by reference, per the
// implementation plan — keeps read-time logic simple for Phase 1.
func (s *Service) CreateMatch(ctx context.Context, caller Caller, orgID domain.OrganizationID, tournamentID *domain.TournamentID, homeTeamID, awayTeamID domain.TeamID, venue string, scheduledAt time.Time, conditionsInput *PlayingConditionsInput) (domain.Match, error) {
	if caller.OrganizationID != orgID {
		return domain.Match{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageMatches(caller.Role) {
		return domain.Match{}, domain.ErrPermissionDenied
	}

	if _, err := s.teams.Get(ctx, orgID, homeTeamID); err != nil {
		return domain.Match{}, err
	}
	if _, err := s.teams.Get(ctx, orgID, awayTeamID); err != nil {
		return domain.Match{}, err
	}

	var resolvedConditions domain.PlayingConditions
	if tournamentID != nil {
		tour, err := s.tournaments.Get(ctx, orgID, *tournamentID)
		if err != nil {
			return domain.Match{}, err
		}
		resolvedConditions = tour.DefaultConditions
	}
	// conditionsInput is validated only after the checks and lookups
	// above — see PlayingConditionsInput's doc comment.
	if conditionsInput != nil {
		pc, err := domain.NewPlayingConditions(conditionsInput.ReviewQuotaPerInnings, conditionsInput.DecisionTypesEnabled, conditionsInput.CameraTier)
		if err != nil {
			return domain.Match{}, err
		}
		resolvedConditions = pc
	}

	m, err := domain.NewMatch(domain.MatchID(newID("match")), orgID, tournamentID, homeTeamID, awayTeamID, venue, scheduledAt, resolvedConditions)
	if err != nil {
		return domain.Match{}, err
	}
	if err := s.matches.Create(ctx, m); err != nil {
		return domain.Match{}, err
	}
	return m, nil
}

func (s *Service) GetMatch(ctx context.Context, caller Caller, orgID domain.OrganizationID, id domain.MatchID) (domain.Match, error) {
	if caller.OrganizationID != orgID {
		return domain.Match{}, domain.ErrCrossTenantAccess
	}
	return s.matches.Get(ctx, orgID, id)
}

func (s *Service) ListMatches(ctx context.Context, caller Caller, orgID domain.OrganizationID) ([]domain.Match, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.matches.ListByOrganization(ctx, orgID)
}
