package memstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

func TestCameraStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewCameraStore()
	camera := domain.Camera{ID: "cam-1", OrganizationID: "org-a", VenueID: "venue-1", Model: "GoPro Hero 12 Black", RegisteredAt: time.Now()}

	if err := s.Create(ctx, camera); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := s.Get(ctx, "org-a", "cam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "cam-1" {
		t.Fatalf("unexpected camera: %+v", got)
	}
}

func TestCameraStore_GetNotFound(t *testing.T) {
	s := NewCameraStore()
	_, err := s.Get(context.Background(), "org-a", "missing")
	if !errors.Is(err, domain.ErrCameraNotFound) {
		t.Fatalf("expected ErrCameraNotFound, got %v", err)
	}
}

func TestCameraStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewCameraStore()
	camA := domain.Camera{ID: "cam-1", OrganizationID: "org-a", VenueID: "venue-1", Model: "GoPro Hero 12 Black"}
	camB := domain.Camera{ID: "cam-1", OrganizationID: "org-b", VenueID: "venue-1", Model: "GoPro Hero 12 Black"}
	if err := s.Create(ctx, camA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, camB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := s.ListByOrganization(ctx, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see exactly its own camera, got %+v", listA)
	}
}

func TestProfileStore_StoreAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewProfileStore()
	profile := domain.CalibrationProfile{CameraID: "cam-1", OrganizationID: "org-a", ReprojectionErrorPx: 0.5}

	if err := s.Store(ctx, profile); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := s.Get(ctx, "org-a", "cam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReprojectionErrorPx != 0.5 {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestProfileStore_GetNotFound(t *testing.T) {
	s := NewProfileStore()
	_, err := s.Get(context.Background(), "org-a", "missing")
	if !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestProfileStore_StoreUpserts(t *testing.T) {
	ctx := context.Background()
	s := NewProfileStore()
	if err := s.Store(ctx, domain.CalibrationProfile{CameraID: "cam-1", OrganizationID: "org-a", ReprojectionErrorPx: 5.0}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Store(ctx, domain.CalibrationProfile{CameraID: "cam-1", OrganizationID: "org-a", ReprojectionErrorPx: 0.5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "org-a", "cam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReprojectionErrorPx != 0.5 {
		t.Fatalf("expected the later submission to replace the earlier one, got %+v", got)
	}
}

func TestProfileStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewProfileStore()
	if err := s.Store(ctx, domain.CalibrationProfile{CameraID: "cam-1", OrganizationID: "org-a", ReprojectionErrorPx: 1.0}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Store(ctx, domain.CalibrationProfile{CameraID: "cam-1", OrganizationID: "org-b", ReprojectionErrorPx: 2.0}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "org-b", "cam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReprojectionErrorPx != 2.0 {
		t.Fatalf("unexpected profile for org-b: %+v", got)
	}
}
