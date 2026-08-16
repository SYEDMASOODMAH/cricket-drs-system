package platformauth

import "testing"

func TestRole_Valid(t *testing.T) {
	valid := []Role{RolePlayer, RoleCoach, RoleUmpire, RoleOrganizerAdmin, RoleBoardAdmin, RoleFan}
	for _, r := range valid {
		if !r.Valid() {
			t.Errorf("Role(%q).Valid() = false, want true", r)
		}
	}
	if Role("astronaut").Valid() {
		t.Error(`Role("astronaut").Valid() = true, want false`)
	}
	if Role("").Valid() {
		t.Error(`Role("").Valid() = true, want false`)
	}
}
