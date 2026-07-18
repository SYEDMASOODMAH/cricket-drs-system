package service

import (
	"context"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

func (s *Service) CreateTeam(ctx context.Context, caller Caller, orgID domain.OrganizationID, name string) (domain.Team, error) {
	if caller.OrganizationID != orgID {
		return domain.Team{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageMatches(caller.Role) {
		return domain.Team{}, domain.ErrPermissionDenied
	}

	t, err := domain.NewTeam(domain.TeamID(newID("team")), orgID, name)
	if err != nil {
		return domain.Team{}, err
	}
	if err := s.teams.Create(ctx, t); err != nil {
		return domain.Team{}, err
	}
	return t, nil
}

func (s *Service) GetTeam(ctx context.Context, caller Caller, orgID domain.OrganizationID, id domain.TeamID) (domain.Team, error) {
	if caller.OrganizationID != orgID {
		return domain.Team{}, domain.ErrCrossTenantAccess
	}
	return s.teams.Get(ctx, orgID, id)
}

func (s *Service) ListTeams(ctx context.Context, caller Caller, orgID domain.OrganizationID) ([]domain.Team, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.teams.ListByOrganization(ctx, orgID)
}
