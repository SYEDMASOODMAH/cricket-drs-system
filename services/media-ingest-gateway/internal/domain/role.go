package domain

import "github.com/cricketdrs/services/platformauth"

// Role aliases platformauth.Role, the shared identity vocabulary every
// service in this module now verifies against (docs/adr/0008) — this used
// to be a hand-copied definition; see that ADR for why the enum itself was
// safe to share while CanUploadClips below (a real, service-specific
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

// CanUploadClips reports whether role may upload/manage match video
// clips. organizer_admin only, for the same reason match-tournament gates
// match-management writes to that role (prd.md Section 5.4) — until a
// distinct edge-device credential exists (deferred, see the
// implementation plan), whoever uploads on the venue's behalf
// authenticates as this persona.
func CanUploadClips(role Role) bool {
	return role == RoleOrganizerAdmin
}
