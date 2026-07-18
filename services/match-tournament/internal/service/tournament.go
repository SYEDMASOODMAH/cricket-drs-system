package service

import (
	"context"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

// CreateTournament requires the caller to belong to orgID (tenant
// isolation) and hold CanManageMatches (organizer_admin per prd.md
// Section 4). defaults is validated only after both checks pass — see
// PlayingConditionsInput's doc comment.
func (s *Service) CreateTournament(ctx context.Context, caller Caller, orgID domain.OrganizationID, name string, defaults PlayingConditionsInput) (domain.Tournament, error) {
	if caller.OrganizationID != orgID {
		return domain.Tournament{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageMatches(caller.Role) {
		return domain.Tournament{}, domain.ErrPermissionDenied
	}

	conditions, err := domain.NewPlayingConditions(defaults.ReviewQuotaPerInnings, defaults.DecisionTypesEnabled, defaults.CameraTier)
	if err != nil {
		return domain.Tournament{}, err
	}

	t, err := domain.NewTournament(domain.TournamentID(newID("tournament")), orgID, name, conditions)
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.tournaments.Create(ctx, t); err != nil {
		return domain.Tournament{}, err
	}
	return t, nil
}

// GetTournament and ListTournaments only enforce tenant isolation — any
// authenticated org member can read (mirrors identity-access's GetUser).
func (s *Service) GetTournament(ctx context.Context, caller Caller, orgID domain.OrganizationID, id domain.TournamentID) (domain.Tournament, error) {
	if caller.OrganizationID != orgID {
		return domain.Tournament{}, domain.ErrCrossTenantAccess
	}
	return s.tournaments.Get(ctx, orgID, id)
}

func (s *Service) ListTournaments(ctx context.Context, caller Caller, orgID domain.OrganizationID) ([]domain.Tournament, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.tournaments.ListByOrganization(ctx, orgID)
}
