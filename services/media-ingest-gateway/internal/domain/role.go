package domain

// Role mirrors identity-access's persona enum (prd.md Section 4).
// Duplicated rather than imported — same rationale as
// services/match-tournament/internal/domain/role.go: Go's internal/
// visibility rules block importing another service's package directly,
// and this is a small, stable, low-risk duplication. Revisit (shared
// services/platformauth-style package) once a third service needs the
// same JWT-verify + Role vocabulary — see match-tournament's README for
// the original "revisit trigger" note; this is now the third data point,
// worth actually revisiting on the next service that needs this.
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

// CanUploadClips reports whether role may upload/manage match video
// clips. organizer_admin only, for the same reason match-tournament gates
// match-management writes to that role (prd.md Section 5.4) — until a
// distinct edge-device credential exists (deferred, see the
// implementation plan), whoever uploads on the venue's behalf
// authenticates as this persona.
func CanUploadClips(role Role) bool {
	return role == RoleOrganizerAdmin
}
