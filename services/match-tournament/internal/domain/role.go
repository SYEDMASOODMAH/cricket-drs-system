package domain

// Role mirrors identity-access's persona enum (prd.md Section 4).
// Duplicated rather than imported — Go's internal/ visibility rules mean
// this service cannot import identity-access's internal/domain, and this
// is a small, stable, low-risk duplication versus introducing a new
// cross-service shared package on the strength of a single precedent. See
// the implementation plan's "Decision flagged" note for the full
// rationale and the revisit trigger (a third service needing the same
// thing).
type Role string

const (
	RolePlayer         Role = "player"
	RoleCoach          Role = "coach"
	RoleUmpire         Role = "umpire"
	RoleOrganizerAdmin Role = "organizer_admin"
	RoleBoardAdmin     Role = "board_admin"
	RoleFan            Role = "fan"
)

func (r Role) Valid() bool {
	switch r {
	case RolePlayer, RoleCoach, RoleUmpire, RoleOrganizerAdmin, RoleBoardAdmin, RoleFan:
		return true
	default:
		return false
	}
}

// CanManageMatches reports whether role may create/modify tournaments,
// teams, matches, and rosters. Mirrors identity-access's
// PermissionManageMatches grant (organizer_admin only) — see
// internal/domain/role.go in identity-access.
func CanManageMatches(role Role) bool {
	return role == RoleOrganizerAdmin
}
