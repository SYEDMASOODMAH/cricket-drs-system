package service

import (
	"context"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

// CreateOrganization provisions a new tenant together with its first user,
// who becomes that org's OrganizerAdmin. It is deliberately unauthenticated
// for Phase 1: there is no platform-superadmin role yet (only roles that
// already belong to an org — see domain.Role), so this is effectively
// club/league onboarding/signup rather than an admin action gated by an
// existing tenant's RBAC. Revisit once self-serve onboarding (phases.md
// Phase 10) or a platform-admin persona is introduced.
//
// Creating the admin atomically with the org isn't a stylistic choice: any
// other user (InviteUser) requires a caller who already belongs to the
// target org and holds PermissionManageUsers — a brand-new org has no such
// caller yet, so without this, no one could ever add its first member.
func (s *Service) CreateOrganization(ctx context.Context, name, adminEmail, adminPassword string) (domain.Organization, domain.User, error) {
	org, err := domain.NewOrganization(domain.OrganizationID(newID("org")), name)
	if err != nil {
		return domain.Organization{}, domain.User{}, err
	}
	if err := s.orgs.Create(ctx, org); err != nil {
		return domain.Organization{}, domain.User{}, err
	}

	hash, err := s.hasher.Hash(adminPassword)
	if err != nil {
		return domain.Organization{}, domain.User{}, err
	}
	admin, err := domain.NewUser(domain.UserID(newID("user")), org.ID, adminEmail, hash, domain.RoleOrganizerAdmin)
	if err != nil {
		return domain.Organization{}, domain.User{}, err
	}
	if err := s.users.Create(ctx, admin); err != nil {
		return domain.Organization{}, domain.User{}, err
	}

	return org, admin, nil
}
