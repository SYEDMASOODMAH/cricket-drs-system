// Package service holds identity-access's use-cases: creating organizations,
// managing users and roles, capturing/checking consent, and authenticating.
// It depends only on the domain package and the port interfaces below —
// never directly on a database driver, HTTP framework, or crypto library
// implementation (rules.md Section 1).
package service

import (
	"context"
	"time"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

// OrganizationRepository persists organizations (tenants).
type OrganizationRepository interface {
	Create(ctx context.Context, org domain.Organization) error
	Get(ctx context.Context, id domain.OrganizationID) (domain.Organization, error)
}

// UserRepository persists users, always scoped by OrganizationID. There is
// deliberately no "get user by ID alone" method — every read requires the
// caller to state which organization it expects the user in, which is the
// mechanical enforcement of tenant isolation (rules.md Section 6.5) at the
// port boundary rather than trusting every call site to remember.
type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	Get(ctx context.Context, orgID domain.OrganizationID, userID domain.UserID) (domain.User, error)
	GetByEmail(ctx context.Context, orgID domain.OrganizationID, email string) (domain.User, error)
	Update(ctx context.Context, u domain.User) error
	ListByOrganization(ctx context.Context, orgID domain.OrganizationID) ([]domain.User, error)
}

// ConsentRepository persists consent records, likewise always scoped by
// OrganizationID.
type ConsentRepository interface {
	Get(ctx context.Context, orgID domain.OrganizationID, userID domain.UserID) (domain.ConsentRecord, error)
	Upsert(ctx context.Context, rec domain.ConsentRecord) error
}

// PasswordHasher wraps a vetted password-hashing algorithm. rules.md
// Section 3.4 prohibits hand-rolled auth crypto; this interface exists so
// the service layer never imports the concrete algorithm (bcrypt) directly,
// keeping that choice swappable behind one seam (internal/security).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenIssuer wraps a vetted JWT implementation, same rationale as
// PasswordHasher above.
type TokenIssuer interface {
	Issue(userID domain.UserID, orgID domain.OrganizationID, role domain.Role) (string, error)
	Verify(token string) (Claims, error)
}

// Claims is what a verified token proves about its holder.
type Claims struct {
	UserID         domain.UserID
	OrganizationID domain.OrganizationID
	Role           domain.Role
}

// Caller is the authenticated identity behind a service call, threaded
// explicitly through every method that needs a tenant-isolation or
// permission check — never read implicitly from a context key, so the
// isolation boundary is visible at every call site and in every test.
type Caller struct {
	OrganizationID domain.OrganizationID
	UserID         domain.UserID
	Role           domain.Role
}

// Clock abstracts time.Now for deterministic tests.
type Clock func() time.Time
