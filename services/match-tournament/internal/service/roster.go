package service

import (
	"context"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

// AddPlayerToRoster is the consent gate itself: prd.md Section 5.6.2
// requires that a non-consenting player never appear on a DRS-enabled
// match roster. callerToken is the caller's own bearer token, forwarded
// unchanged to Identity & Access so its own GetConsent authorization rules
// apply — this service never impersonates the caller or bypasses that
// check with an elevated credential.
func (s *Service) AddPlayerToRoster(ctx context.Context, caller Caller, callerToken string, orgID domain.OrganizationID, matchID domain.MatchID, teamID domain.TeamID, userID domain.UserID) (domain.RosterEntry, error) {
	if caller.OrganizationID != orgID {
		return domain.RosterEntry{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageMatches(caller.Role) {
		return domain.RosterEntry{}, domain.ErrPermissionDenied
	}

	if _, err := s.matches.Get(ctx, orgID, matchID); err != nil {
		return domain.RosterEntry{}, err
	}
	if _, err := s.teams.Get(ctx, orgID, teamID); err != nil {
		return domain.RosterEntry{}, err
	}

	eligible, err := s.consent.IsEligibleForDRS(ctx, callerToken, userID)
	if err != nil {
		return domain.RosterEntry{}, err
	}
	if !eligible {
		return domain.RosterEntry{}, domain.ErrConsentRequired
	}

	entry, err := domain.NewRosterEntry(orgID, matchID, teamID, userID, caller.UserID, s.now())
	if err != nil {
		return domain.RosterEntry{}, err
	}
	if err := s.rosters.Add(ctx, entry); err != nil {
		return domain.RosterEntry{}, err
	}
	return entry, nil
}

func (s *Service) RemovePlayerFromRoster(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, userID domain.UserID) error {
	if caller.OrganizationID != orgID {
		return domain.ErrCrossTenantAccess
	}
	if !domain.CanManageMatches(caller.Role) {
		return domain.ErrPermissionDenied
	}
	return s.rosters.Remove(ctx, orgID, matchID, userID)
}

func (s *Service) ListRoster(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.RosterEntry, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.rosters.ListByMatch(ctx, orgID, matchID)
}
