package domain

import "github.com/cricketdrs/services/platformauth"

// Role aliases platformauth.Role, the shared identity vocabulary every
// service in this module now verifies against (docs/adr/0008) — this used
// to be a hand-copied definition (the 4th instance of the exact
// duplication that ADR triggered on); see that ADR for why the enum
// itself was safe to share while CanManageCalibration below (a real,
// service-specific authorization decision) was not.
type Role = platformauth.Role

const (
	RolePlayer         = platformauth.RolePlayer
	RoleCoach          = platformauth.RoleCoach
	RoleUmpire         = platformauth.RoleUmpire
	RoleOrganizerAdmin = platformauth.RoleOrganizerAdmin
	RoleBoardAdmin     = platformauth.RoleBoardAdmin
	RoleFan            = platformauth.RoleFan
)

// CanManageCalibration reports whether role may register cameras and
// submit/view calibration profiles. organizer_admin only, same rationale
// as media-ingest-gateway's CanUploadClips — until a distinct
// technician/edge-device credential exists, whoever manages venue setup
// on the platform's behalf authenticates as this persona.
func CanManageCalibration(role Role) bool {
	return role == RoleOrganizerAdmin
}
