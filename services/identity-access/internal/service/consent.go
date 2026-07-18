package service

import (
	"context"
	"errors"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

// CaptureConsent grants or revokes a single consent type for userID. Per
// prd.md Section 5.6, the caller must be one of: the player themself, the
// guardian on file (for a minor), or an organizer/club admin capturing
// initial consent during onboarding (Section 5.6.1) — never an unrelated
// party. isMinor/guardianUserID are only used the first time a record is
// created for userID; changing minor/guardian status afterward is a
// separate, not-yet-modeled use-case.
func (s *Service) CaptureConsent(ctx context.Context, caller Caller, orgID domain.OrganizationID, userID domain.UserID, isMinor bool, guardianUserID *domain.UserID, consentType domain.ConsentType, grant bool) (domain.ConsentRecord, error) {
	if caller.OrganizationID != orgID {
		return domain.ConsentRecord{}, domain.ErrCrossTenantAccess
	}

	authorized := caller.UserID == userID ||
		caller.Role == domain.RoleOrganizerAdmin ||
		(guardianUserID != nil && caller.UserID == *guardianUserID)
	if !authorized {
		return domain.ConsentRecord{}, domain.ErrPermissionDenied
	}

	rec, err := s.consents.Get(ctx, orgID, userID)
	if err != nil {
		if !errors.Is(err, domain.ErrConsentNotFound) {
			return domain.ConsentRecord{}, err
		}
		rec, err = domain.NewConsentRecord(userID, orgID, isMinor, guardianUserID, caller.UserID, s.now())
		if err != nil {
			return domain.ConsentRecord{}, err
		}
	}

	if grant {
		rec.Grant(consentType, caller.UserID, s.now())
	} else {
		rec.Revoke(consentType, caller.UserID, s.now())
	}

	if err := s.consents.Upsert(ctx, rec); err != nil {
		return domain.ConsentRecord{}, err
	}
	return rec, nil
}

// GetConsent fetches userID's consent record. Readable by the player
// themself, their guardian on file, or org staff with a governance/
// management role (organizer admin, board admin).
func (s *Service) GetConsent(ctx context.Context, caller Caller, orgID domain.OrganizationID, userID domain.UserID) (domain.ConsentRecord, error) {
	if caller.OrganizationID != orgID {
		return domain.ConsentRecord{}, domain.ErrCrossTenantAccess
	}

	rec, err := s.consents.Get(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrConsentNotFound) {
			if caller.UserID == userID || caller.Role == domain.RoleOrganizerAdmin || caller.Role == domain.RoleBoardAdmin {
				return domain.ConsentRecord{}, err
			}
			return domain.ConsentRecord{}, domain.ErrPermissionDenied
		}
		return domain.ConsentRecord{}, err
	}

	authorized := caller.UserID == userID ||
		caller.Role == domain.RoleOrganizerAdmin ||
		caller.Role == domain.RoleBoardAdmin ||
		(rec.GuardianUserID != nil && caller.UserID == *rec.GuardianUserID)
	if !authorized {
		return domain.ConsentRecord{}, domain.ErrPermissionDenied
	}
	return rec, nil
}
