package domain

import "github.com/cricketdrs/services/platformauth"

// Role aliases platformauth.Role, the shared identity vocabulary every
// service in this module now verifies against (docs/adr/0008) — this used
// to be a hand-copied definition; see that ADR for why the enum itself was
// safe to share while CanManageMatches below (a real, service-specific
// authorization decision) was not.
type Role = platformauth.Role

const (
	RolePlayer         = platformauth.RolePlayer
	RoleCoach          = platformauth.RoleCoach
	RoleUmpire         = platformauth.RoleUmpire
	RoleOrganizerAdmin = platformauth.RoleOrganizerAdmin
	RoleBoardAdmin     = platformauth.RoleBoardAdmin
	RoleFan            = platformauth.RoleFan
)

// CanManageMatches reports whether role may create/modify tournaments,
// teams, matches, and rosters. Mirrors identity-access's
// PermissionManageMatches grant (organizer_admin only) — see
// internal/domain/role.go in identity-access.
func CanManageMatches(role Role) bool {
	return role == RoleOrganizerAdmin
}
