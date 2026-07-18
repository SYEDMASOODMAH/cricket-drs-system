// Package service holds match-tournament's use-cases: tournaments, teams,
// matches, and roster management, including the consent gate (prd.md
// Section 5.6.2). It depends only on the domain package and the port
// interfaces below — never on a database driver, HTTP framework, or the
// JWT library implementation directly (rules.md Section 1).
package service

import (
	"context"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type TournamentRepository interface {
	Create(ctx context.Context, t domain.Tournament) error
	Get(ctx context.Context, orgID domain.OrganizationID, id domain.TournamentID) (domain.Tournament, error)
	ListByOrganization(ctx context.Context, orgID domain.OrganizationID) ([]domain.Tournament, error)
}

type TeamRepository interface {
	Create(ctx context.Context, t domain.Team) error
	Get(ctx context.Context, orgID domain.OrganizationID, id domain.TeamID) (domain.Team, error)
	ListByOrganization(ctx context.Context, orgID domain.OrganizationID) ([]domain.Team, error)
}

type MatchRepository interface {
	Create(ctx context.Context, m domain.Match) error
	Get(ctx context.Context, orgID domain.OrganizationID, id domain.MatchID) (domain.Match, error)
	ListByOrganization(ctx context.Context, orgID domain.OrganizationID) ([]domain.Match, error)
}

// RosterRepository is always scoped by OrganizationID, the same
// tenant-isolation-at-the-port-boundary pattern identity-access's
// UserRepository uses.
type RosterRepository interface {
	Add(ctx context.Context, entry domain.RosterEntry) error
	Remove(ctx context.Context, orgID domain.OrganizationID, matchID domain.MatchID, userID domain.UserID) error
	ListByMatch(ctx context.Context, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.RosterEntry, error)
}

// TokenVerifier wraps the vetted JWT implementation (rules.md Section 3.4
// bans hand-rolled auth crypto), same rationale as identity-access's
// TokenIssuer/PasswordHasher ports: the service layer never imports the
// JWT library directly.
type TokenVerifier interface {
	Verify(token string) (Claims, error)
}

// Claims is what a verified token proves about its holder — minted by
// Identity & Access, verified here.
type Claims struct {
	UserID         domain.UserID
	OrganizationID domain.OrganizationID
	Role           domain.Role
}

// ConsentChecker verifies a player's DRS consent eligibility via Identity &
// Access, per prd.md Section 5.6.2: a non-consenting player must never
// appear on a match roster. token is the caller's own bearer token,
// forwarded so Identity & Access's own GetConsent authorization rules
// (self/guardian/organizer_admin/board_admin) apply unchanged rather than
// this service impersonating a different caller. err is reserved for
// genuine failures (Identity & Access unreachable, unexpected response) —
// "no consent record yet" is a normal false, not an error.
type ConsentChecker interface {
	IsEligibleForDRS(ctx context.Context, token string, userID domain.UserID) (bool, error)
}

// PlayingConditionsInput carries raw, unvalidated playing-conditions
// fields in from the transport layer. Validation (domain.NewPlayingConditions)
// happens inside each use-case *after* its permission check — never
// before, so an unauthorized caller can't use a validation error as an
// oracle for what would have been valid.
type PlayingConditionsInput struct {
	ReviewQuotaPerInnings int
	DecisionTypesEnabled  []domain.DecisionType
	CameraTier            domain.CameraTier
}

// Caller is the authenticated identity behind a service call, threaded
// explicitly through every method that needs a tenant-isolation or
// permission check — same rationale as identity-access's Caller.
type Caller struct {
	OrganizationID domain.OrganizationID
	UserID         domain.UserID
	Role           domain.Role
}

// Clock abstracts time.Now for deterministic tests.
type Clock func() time.Time
