package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewCamera_Valid(t *testing.T) {
	now := time.Now()
	c, err := NewCamera("cam-1", "org-1", "venue-1", "GoPro Hero 12 Black", "user-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.VenueID != "venue-1" || c.Model != "GoPro Hero 12 Black" || c.RegisteredAt != now {
		t.Fatalf("unexpected camera: %+v", c)
	}
}

func TestNewCamera_EmptyVenueIDRejected(t *testing.T) {
	_, err := NewCamera("cam-1", "org-1", "", "GoPro Hero 12 Black", "user-1", time.Now())
	if !errors.Is(err, ErrVenueIDEmpty) {
		t.Fatalf("expected ErrVenueIDEmpty, got %v", err)
	}
}

func TestNewCamera_EmptyModelRejected(t *testing.T) {
	_, err := NewCamera("cam-1", "org-1", "venue-1", "", "user-1", time.Now())
	if !errors.Is(err, ErrCameraModelEmpty) {
		t.Fatalf("expected ErrCameraModelEmpty, got %v", err)
	}
}

func TestNewCalibrationProfile_Valid(t *testing.T) {
	now := time.Now()
	p, err := NewCalibrationProfile("cam-1", "org-1", [3]float64{0.1, 0.2, 0.3}, [3]float64{1, 2, 3}, 0.8, "user-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Valid() {
		t.Errorf("expected profile with reprojection error 0.8 <= %v to be valid", MaxReprojectionErrorPx)
	}
}

func TestNewCalibrationProfile_NegativeErrorRejected(t *testing.T) {
	_, err := NewCalibrationProfile("cam-1", "org-1", [3]float64{}, [3]float64{}, -0.1, "user-1", time.Now())
	if !errors.Is(err, ErrInvalidCalibration) {
		t.Fatalf("expected ErrInvalidCalibration, got %v", err)
	}
}

func TestCalibrationProfile_ValidThreshold(t *testing.T) {
	tests := []struct {
		name      string
		errorPx   float64
		wantValid bool
	}{
		{"well within threshold", 0.5, true},
		{"exactly at threshold", MaxReprojectionErrorPx, true},
		{"just over threshold", MaxReprojectionErrorPx + 0.01, false},
		{"far over threshold", 50.0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewCalibrationProfile("cam-1", "org-1", [3]float64{}, [3]float64{}, tt.errorPx, "user-1", time.Now())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := p.Valid(); got != tt.wantValid {
				t.Errorf("Valid() with error %v = %v, want %v", tt.errorPx, got, tt.wantValid)
			}
		})
	}
}

func TestLensProfileFor_Known(t *testing.T) {
	profile, err := LensProfileFor("GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Model != "GoPro Hero 12 Black" || profile.FocalLengthPxX <= 0 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestLensProfileFor_Unknown(t *testing.T) {
	_, err := LensProfileFor("Some Unsupported Camera")
	if !errors.Is(err, ErrCameraModelUnknown) {
		t.Fatalf("expected ErrCameraModelUnknown, got %v", err)
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

func TestCanManageCalibration(t *testing.T) {
	if !CanManageCalibration(RoleOrganizerAdmin) {
		t.Error("expected organizer_admin to be able to manage calibration")
	}
	for _, r := range []Role{RolePlayer, RoleCoach, RoleUmpire, RoleBoardAdmin, RoleFan} {
		if CanManageCalibration(r) {
			t.Errorf("expected %q to not be able to manage calibration", r)
		}
	}
}
