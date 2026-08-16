package domain

import "github.com/cricketdrs/services/platformauth"

// Role aliases platformauth.Role, the shared identity vocabulary every
// service in this module now verifies against (docs/adr/0008) — this
// service is the source of that vocabulary (prd.md Section 4), so the
// alias just makes explicit that identity-access, platformauth, and every
// verify-only service now agree on one definition instead of 4 hand-copied
// ones. Permission/HasPermission below (this service's own authorization
// model) stay local — platformauth only owns the identity/token
// vocabulary, not per-service authorization decisions.
type Role = platformauth.Role

const (
	RolePlayer         = platformauth.RolePlayer
	RoleCoach          = platformauth.RoleCoach
	RoleUmpire         = platformauth.RoleUmpire
	RoleOrganizerAdmin = platformauth.RoleOrganizerAdmin
	RoleBoardAdmin     = platformauth.RoleBoardAdmin
	RoleFan            = platformauth.RoleFan
)

// Permission is a coarse-grained action gate. Per rules.md Section 1
// (Open/Closed), new permissions can be added and assigned to roles below
// without changing the HasPermission check itself.
type Permission string

const (
	PermissionManageUsers     Permission = "manage_users"
	PermissionManageMatches   Permission = "manage_matches"
	PermissionTriggerReview   Permission = "trigger_review"
	PermissionConfirmDecision Permission = "confirm_decision"
	PermissionViewAnalytics   Permission = "view_analytics"
	PermissionViewGovernance  Permission = "view_governance"
	PermissionManageConsent   Permission = "manage_consent"
)

// rolePermissions is the static role -> permission-set table. Board-admin
// governance/audit scope and organizer-admin match/user management are
// deliberately kept within a single Organization for this Phase 1 slice —
// cross-club board aggregation is an Analytics & Reporting Service concern
// (architecture.md Section 5), not an Identity & Access one.
var rolePermissions = map[Role]map[Permission]struct{}{
	RolePlayer: {
		PermissionViewAnalytics: {},
		PermissionManageConsent: {},
	},
	RoleCoach: {
		PermissionViewAnalytics: {},
		PermissionManageConsent: {},
	},
	RoleUmpire: {
		PermissionTriggerReview:   {},
		PermissionConfirmDecision: {},
		PermissionManageConsent:   {},
	},
	RoleOrganizerAdmin: {
		PermissionManageUsers:   {},
		PermissionManageMatches: {},
		PermissionViewAnalytics: {},
		PermissionManageConsent: {}, // may capture initial consent per prd.md Section 5.6.1
	},
	RoleBoardAdmin: {
		PermissionViewGovernance: {},
		PermissionViewAnalytics:  {},
		PermissionManageUsers:    {},
	},
	RoleFan: {
		PermissionViewAnalytics: {},
	},
}

// HasPermission reports whether role grants permission. An unknown role
// (e.g. the zero value) never has any permission — fail closed.
func HasPermission(role Role, permission Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[permission]
	return ok
}
