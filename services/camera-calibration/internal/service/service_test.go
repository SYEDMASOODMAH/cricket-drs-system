package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
	"github.com/cricketdrs/services/camera-calibration/internal/memstore"
)

type fakeTokenVerifier struct {
	claims Claims
	err    error
}

func (f *fakeTokenVerifier) Verify(_ string) (Claims, error) {
	return f.claims, f.err
}

func newTestService() *Service {
	return New(memstore.NewCameraStore(), memstore.NewProfileStore(), &fakeTokenVerifier{})
}

var orgAAdmin = Caller{OrganizationID: "org-a", UserID: "admin-a", Role: domain.RoleOrganizerAdmin}
var orgBAdmin = Caller{OrganizationID: "org-b", UserID: "admin-b", Role: domain.RoleOrganizerAdmin}
var orgAPlayer = Caller{OrganizationID: "org-a", UserID: "player-a", Role: domain.RolePlayer}

func TestRegisterCamera_Success(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if camera.VenueID != "venue-1" || camera.Model != "GoPro Hero 12 Black" {
		t.Fatalf("unexpected camera: %+v", camera)
	}
}

func TestRegisterCamera_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-b", "venue-1", "GoPro Hero 12 Black")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestRegisterCamera_PermissionDenied(t *testing.T) {
	svc := newTestService()
	_, err := svc.RegisterCamera(context.Background(), orgAPlayer, "org-a", "venue-1", "GoPro Hero 12 Black")
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestRegisterCamera_EmptyVenueRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "", "GoPro Hero 12 Black")
	if !errors.Is(err, domain.ErrVenueIDEmpty) {
		t.Fatalf("expected ErrVenueIDEmpty, got %v", err)
	}
}

func TestGetCamera_Success(t *testing.T) {
	svc := newTestService()
	registered, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetCamera(context.Background(), orgAAdmin, "org-a", registered.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != registered.ID {
		t.Fatalf("unexpected camera: %+v", got)
	}
}

func TestGetCamera_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	registered, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetCamera(context.Background(), orgBAdmin, "org-a", registered.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestListCameras_TenantIsolation(t *testing.T) {
	svc := newTestService()
	if _, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.RegisterCamera(context.Background(), orgBAdmin, "org-b", "venue-1", "GoPro Hero 12 Black"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := svc.ListCameras(context.Background(), orgAAdmin, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see exactly its own camera, got %+v", listA)
	}
}

func TestListCameras_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.ListCameras(context.Background(), orgAAdmin, "org-b")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestStoreCalibrationProfile_Success(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	profile, err := svc.StoreCalibrationProfile(context.Background(), orgAAdmin, "org-a", camera.ID, [3]float64{0.1, 0.2, 0.3}, [3]float64{1, 2, 3}, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !profile.Valid() {
		t.Errorf("expected profile with error 0.5 to be valid")
	}
}

func TestStoreCalibrationProfile_UnregisteredCameraRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.StoreCalibrationProfile(context.Background(), orgAAdmin, "org-a", "does-not-exist", [3]float64{}, [3]float64{}, 0.5)
	if !errors.Is(err, domain.ErrCameraNotFound) {
		t.Fatalf("expected ErrCameraNotFound, got %v", err)
	}
}

func TestStoreCalibrationProfile_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.StoreCalibrationProfile(context.Background(), orgBAdmin, "org-a", camera.ID, [3]float64{}, [3]float64{}, 0.5)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestStoreCalibrationProfile_PermissionDenied(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.StoreCalibrationProfile(context.Background(), orgAPlayer, "org-a", camera.ID, [3]float64{}, [3]float64{}, 0.5)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestStoreCalibrationProfile_InvalidDataRejected(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.StoreCalibrationProfile(context.Background(), orgAAdmin, "org-a", camera.ID, [3]float64{}, [3]float64{}, -1.0)
	if !errors.Is(err, domain.ErrInvalidCalibration) {
		t.Fatalf("expected ErrInvalidCalibration, got %v", err)
	}
}

func TestGetCalibrationStatus_ValidAndInvalid(t *testing.T) {
	svc := newTestService()
	camera, err := svc.RegisterCamera(context.Background(), orgAAdmin, "org-a", "venue-1", "GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.StoreCalibrationProfile(context.Background(), orgAAdmin, "org-a", camera.ID, [3]float64{}, [3]float64{}, 50.0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, err := svc.GetCalibrationStatus(context.Background(), orgAAdmin, "org-a", camera.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Valid() {
		t.Errorf("expected a 50px reprojection error to be invalid, got valid profile: %+v", status)
	}
}

func TestGetCalibrationStatus_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetCalibrationStatus(context.Background(), orgAAdmin, "org-b", "cam-1")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestGetCalibrationStatus_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetCalibrationStatus(context.Background(), orgAAdmin, "org-a", "does-not-exist")
	if !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestGetLensProfile_KnownAndUnknown(t *testing.T) {
	svc := newTestService()
	profile, err := svc.GetLensProfile("GoPro Hero 12 Black")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Model != "GoPro Hero 12 Black" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	_, err = svc.GetLensProfile("Unsupported Camera")
	if !errors.Is(err, domain.ErrCameraModelUnknown) {
		t.Fatalf("expected ErrCameraModelUnknown, got %v", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	claims := Claims{UserID: "user-1", OrganizationID: "org-a", Role: domain.RoleOrganizerAdmin}
	svc := New(memstore.NewCameraStore(), memstore.NewProfileStore(), &fakeTokenVerifier{claims: claims})
	caller, err := svc.Authenticate("some-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.UserID != "user-1" || caller.OrganizationID != "org-a" || caller.Role != domain.RoleOrganizerAdmin {
		t.Fatalf("unexpected caller: %+v", caller)
	}
}

func TestAuthenticate_InvalidTokenRejected(t *testing.T) {
	svc := New(memstore.NewCameraStore(), memstore.NewProfileStore(), &fakeTokenVerifier{err: errors.New("bad token")})
	_, err := svc.Authenticate("garbage")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
