package domain

// Role is a fixed persona, per prd.md Section 4. It is intentionally a
// closed set for Phase 1 rather than a dynamic policy engine — see
// docs/adr/0001-go-for-core-services.md's "revisit if" clause: hand-rolled
// RBAC is the deliberate choice while the persona list stays this small.
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
