// Package domain holds match-tournament's core types and business rules:
// tournaments, teams, matches, playing conditions, and match rosters. Per
// rules.md Section 1, nothing here imports a transport, persistence, or
// framework package — it must be unit-testable with no database or network
// call.
package domain

import "errors"

var (
	ErrTournamentNotFound  = errors.New("tournament not found")
	ErrTeamNotFound        = errors.New("team not found")
	ErrMatchNotFound       = errors.New("match not found")
	ErrNameEmpty           = errors.New("name must not be empty")
	ErrVenueEmpty          = errors.New("venue must not be empty")
	ErrUserIDEmpty         = errors.New("user id must not be empty")
	ErrInvalidRole         = errors.New("invalid role")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrCrossTenantAccess   = errors.New("cross-tenant access denied")
	ErrRosterEntryExists   = errors.New("player already on roster")
	ErrRosterEntryNotFound = errors.New("roster entry not found")
	ErrConsentRequired     = errors.New("player has not granted required DRS consent")
	ErrInvalidDecisionType = errors.New("invalid decision type")
	ErrInvalidCameraTier   = errors.New("invalid camera tier")
	ErrInvalidReviewQuota  = errors.New("review quota per innings must not be negative")
	ErrInvalidTeamPairing  = errors.New("home and away teams must both be set and different")
	ErrInvalidToken        = errors.New("invalid or expired token")
)
