package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewClip_Valid(t *testing.T) {
	now := time.Now()
	c, err := NewClip("clip-1", "org-1", "match-1", "cam-1", "org-1/match-1/clip-1.mp4", "deadbeef", 1024, "user-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.MatchID != "match-1" || c.CameraID != "cam-1" || c.SizeBytes != 1024 {
		t.Fatalf("unexpected clip: %+v", c)
	}
}

func TestNewClip_EmptyMatchIDRejected(t *testing.T) {
	_, err := NewClip("clip-1", "org-1", "", "cam-1", "key", "hash", 1024, "user-1", time.Now())
	if !errors.Is(err, ErrMatchIDEmpty) {
		t.Fatalf("expected ErrMatchIDEmpty, got %v", err)
	}
}

func TestNewClip_EmptyCameraIDRejected(t *testing.T) {
	_, err := NewClip("clip-1", "org-1", "match-1", "", "key", "hash", 1024, "user-1", time.Now())
	if !errors.Is(err, ErrCameraIDEmpty) {
		t.Fatalf("expected ErrCameraIDEmpty, got %v", err)
	}
}

func TestNewClip_NonPositiveSizeRejected(t *testing.T) {
	for _, size := range []int64{0, -1} {
		_, err := NewClip("clip-1", "org-1", "match-1", "cam-1", "key", "hash", size, "user-1", time.Now())
		if !errors.Is(err, ErrEmptyContent) {
			t.Fatalf("size=%d: expected ErrEmptyContent, got %v", size, err)
		}
	}
}

func TestRoleValid(t *testing.T) {
	valid := []Role{RolePlayer, RoleCoach, RoleUmpire, RoleOrganizerAdmin, RoleBoardAdmin, RoleFan}
	for _, r := range valid {
		if !r.Valid() {
			t.Errorf("Role(%q).Valid() = false, want true", r)
		}
	}
	if Role("astronaut").Valid() {
		t.Error("expected unknown role to be invalid")
	}
}

func TestCanUploadClips(t *testing.T) {
	if !CanUploadClips(RoleOrganizerAdmin) {
		t.Error("expected organizer_admin to be able to upload clips")
	}
	for _, r := range []Role{RolePlayer, RoleCoach, RoleUmpire, RoleBoardAdmin, RoleFan} {
		if CanUploadClips(r) {
			t.Errorf("expected %q to not be able to upload clips", r)
		}
	}
}
