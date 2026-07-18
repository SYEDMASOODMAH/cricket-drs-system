package service

import (
	"context"
	"errors"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

// InviteUser adds a user to an organization with a role. The caller must
// belong to the target organization (tenant isolation) and hold
// PermissionManageUsers.
func (s *Service) InviteUser(ctx context.Context, caller Caller, orgID domain.OrganizationID, email, password string, role domain.Role) (domain.User, error) {
	if caller.OrganizationID != orgID {
		return domain.User{}, domain.ErrCrossTenantAccess
	}
	if !domain.HasPermission(caller.Role, domain.PermissionManageUsers) {
		return domain.User{}, domain.ErrPermissionDenied
	}
	if _, err := s.orgs.Get(ctx, orgID); err != nil {
		return domain.User{}, err
	}
	if _, err := s.users.GetByEmail(ctx, orgID, email); err == nil {
		return domain.User{}, domain.ErrUserExists
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}

	u, err := domain.NewUser(domain.UserID(newID("user")), orgID, email, hash, role)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.users.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// GetUser fetches a user, enforcing that the caller belongs to the same
// organization as the user being fetched.
func (s *Service) GetUser(ctx context.Context, caller Caller, orgID domain.OrganizationID, userID domain.UserID) (domain.User, error) {
	if caller.OrganizationID != orgID {
		return domain.User{}, domain.ErrCrossTenantAccess
	}
	return s.users.Get(ctx, orgID, userID)
}

// AssignRole changes a user's role. The caller must belong to the target
// organization and hold PermissionManageUsers.
func (s *Service) AssignRole(ctx context.Context, caller Caller, orgID domain.OrganizationID, userID domain.UserID, newRole domain.Role) (domain.User, error) {
	if caller.OrganizationID != orgID {
		return domain.User{}, domain.ErrCrossTenantAccess
	}
	if !domain.HasPermission(caller.Role, domain.PermissionManageUsers) {
		return domain.User{}, domain.ErrPermissionDenied
	}
	if !newRole.Valid() {
		return domain.User{}, domain.ErrInvalidRole
	}

	u, err := s.users.Get(ctx, orgID, userID)
	if err != nil {
		return domain.User{}, err
	}
	u.Role = newRole
	if err := s.users.Update(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// Login authenticates an email/password pair within an organization and
// issues a short-lived token on success. There is no Caller here — this is
// the entry point that produces one.
func (s *Service) Login(ctx context.Context, orgID domain.OrganizationID, email, password string) (string, error) {
	u, err := s.users.GetByEmail(ctx, orgID, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return "", domain.ErrInvalidCredentials
	}
	return s.tokens.Issue(u.ID, u.OrganizationID, u.Role)
}

// Authenticate verifies a bearer token and returns the Caller it proves.
// httpapi's auth middleware calls this so it never needs to import the JWT
// library directly.
func (s *Service) Authenticate(token string) (Caller, error) {
	claims, err := s.tokens.Verify(token)
	if err != nil {
		return Caller{}, domain.ErrInvalidCredentials
	}
	return Caller{
		OrganizationID: claims.OrganizationID,
		UserID:         claims.UserID,
		Role:           claims.Role,
	}, nil
}
