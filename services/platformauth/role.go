// Package platformauth is the shared identity/token vocabulary for every
// service in this module: the Role enum and JWT issuing/verification
// logic, extracted from 4 hand-copied per-service implementations once
// direct comparison confirmed they were byte-for-byte identical (see
// docs/adr/0008-platformauth-shared-package.md). Sits outside every
// service's internal/ tree, same as services/observability
// (docs/adr/0004), specifically so it's importable across service
// boundaries.
//
// What does NOT live here: per-service authorization decisions
// (CanUploadClips, CanManageMatches, CanManageCalibration,
// identity-access's Permission/HasPermission table) — those carry real,
// service-specific business meaning and stay in each service's own
// internal/domain package.
package platformauth

// Role mirrors identity-access's persona enum (prd.md Section 4) — the
// single source of truth every service's domain.Role now aliases.
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
