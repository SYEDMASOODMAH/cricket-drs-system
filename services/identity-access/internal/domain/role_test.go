package domain

import "testing"

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{"player can view analytics", RolePlayer, PermissionViewAnalytics, true},
		{"player cannot manage users", RolePlayer, PermissionManageUsers, false},
		{"player cannot manage matches", RolePlayer, PermissionManageMatches, false},
		{"umpire can trigger review", RoleUmpire, PermissionTriggerReview, true},
		{"umpire can confirm decision", RoleUmpire, PermissionConfirmDecision, true},
		{"umpire cannot manage users", RoleUmpire, PermissionManageUsers, false},
		{"organizer admin can manage users", RoleOrganizerAdmin, PermissionManageUsers, true},
		{"organizer admin can manage matches", RoleOrganizerAdmin, PermissionManageMatches, true},
		{"organizer admin cannot confirm decision", RoleOrganizerAdmin, PermissionConfirmDecision, false},
		{"board admin can view governance", RoleBoardAdmin, PermissionViewGovernance, true},
		{"board admin cannot trigger review", RoleBoardAdmin, PermissionTriggerReview, false},
		{"fan can view analytics", RoleFan, PermissionViewAnalytics, true},
		{"fan cannot manage consent", RoleFan, PermissionManageConsent, false},
		{"unknown role has no permission", Role("astronaut"), PermissionViewAnalytics, false},
		{"zero-value role has no permission", Role(""), PermissionViewAnalytics, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPermission(tt.role, tt.permission); got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
			}
		})
	}
}

func TestRoleValid(t *testing.T) {
	valid := []Role{RolePlayer, RoleCoach, RoleUmpire, RoleOrganizerAdmin, RoleBoardAdmin, RoleFan}
	for _, r := range valid {
		if !r.Valid() {
			t.Errorf("Role(%q).Valid() = false, want true", r)
		}
	}

	invalid := []Role{"", "admin", "superuser"}
	for _, r := range invalid {
		if r.Valid() {
			t.Errorf("Role(%q).Valid() = true, want false", r)
		}
	}
}
