package service

import (
	"context"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

// StoreCalibrationProfile records a camera's already-computed extrinsic
// calibration (rotation, translation, reprojection error) — the actual
// pose-estimation math runs externally in ml-pipeline/camera-calibration
// (docs/adr/0005); this service only stores and evaluates the result.
// Requires the same tenant-isolation and CanManageCalibration check as
// RegisterCamera, plus confirming the camera is actually registered in
// this org first (no orphan profiles for a camera nobody registered).
func (s *Service) StoreCalibrationProfile(ctx context.Context, caller Caller, orgID domain.OrganizationID, cameraID domain.CameraID, rotation, translation [3]float64, reprojectionErrorPx float64) (domain.CalibrationProfile, error) {
	if caller.OrganizationID != orgID {
		return domain.CalibrationProfile{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageCalibration(caller.Role) {
		return domain.CalibrationProfile{}, domain.ErrPermissionDenied
	}

	if _, err := s.cameras.Get(ctx, orgID, cameraID); err != nil {
		return domain.CalibrationProfile{}, err
	}

	profile, err := domain.NewCalibrationProfile(cameraID, orgID, rotation, translation, reprojectionErrorPx, caller.UserID, s.now())
	if err != nil {
		return domain.CalibrationProfile{}, err
	}

	if err := s.profiles.Store(ctx, profile); err != nil {
		return domain.CalibrationProfile{}, err
	}
	return profile, nil
}

// GetCalibrationStatus only enforces tenant isolation — any authenticated
// org member can check whether a camera is calibrated. This is the
// "pre-match camera detection and calibration/health check" prd.md
// Section 5.4 describes: callers read CalibrationProfile.Valid() to
// decide if the camera is ready for a match.
func (s *Service) GetCalibrationStatus(ctx context.Context, caller Caller, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.CalibrationProfile, error) {
	if caller.OrganizationID != orgID {
		return domain.CalibrationProfile{}, domain.ErrCrossTenantAccess
	}
	return s.profiles.Get(ctx, orgID, cameraID)
}

// GetLensProfile looks up a camera model's standardized intrinsic
// profile (architecture.md Section 9). Not tenant-scoped — a model's lens
// profile is reference data, not org-owned state — so this only requires
// a valid authenticated caller, no organization or role check.
func (s *Service) GetLensProfile(model domain.CameraModel) (domain.LensProfile, error) {
	return domain.LensProfileFor(model)
}
