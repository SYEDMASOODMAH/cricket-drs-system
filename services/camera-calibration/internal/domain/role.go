package domain

// Role mirrors identity-access's persona enum (prd.md Section 4).
// Duplicated rather than imported — this is the 4th instance of this
// exact duplication (identity-access, match-tournament,
// media-ingest-gateway, now this service). A shared services/platformauth
// package was considered for this slice and explicitly deferred (see
// docs/adr/0005 and this service's README) — extraction stays its own
// dedicated, reviewable change, not bundled into a new service's PR.
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

// CanManageCalibration reports whether role may register cameras and
// submit/view calibration profiles. organizer_admin only, same rationale
// as media-ingest-gateway's CanUploadClips — until a distinct
// technician/edge-device credential exists, whoever manages venue setup
// on the platform's behalf authenticates as this persona.
func CanManageCalibration(role Role) bool {
	return role == RoleOrganizerAdmin
}
