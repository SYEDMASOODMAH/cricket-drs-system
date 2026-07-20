package service

import (
	"context"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

// RegisterCamera requires the caller to belong to orgID (tenant
// isolation) and hold CanManageCalibration (organizer_admin — see
// domain/role.go's doc comment on why this reuses the human-persona role
// rather than a distinct technician/edge-device credential for now).
func (s *Service) RegisterCamera(ctx context.Context, caller Caller, orgID domain.OrganizationID, venueID domain.VenueID, model domain.CameraModel) (domain.Camera, error) {
	if caller.OrganizationID != orgID {
		return domain.Camera{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanManageCalibration(caller.Role) {
		return domain.Camera{}, domain.ErrPermissionDenied
	}

	cameraID := domain.CameraID(newID("camera"))
	camera, err := domain.NewCamera(cameraID, orgID, venueID, model, caller.UserID, s.now())
	if err != nil {
		return domain.Camera{}, err
	}

	if err := s.cameras.Create(ctx, camera); err != nil {
		return domain.Camera{}, err
	}
	return camera, nil
}

// GetCamera and ListCameras only enforce tenant isolation — any
// authenticated org member can read (mirrors media-ingest-gateway's
// GetClip and ListClips).
func (s *Service) GetCamera(ctx context.Context, caller Caller, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.Camera, error) {
	if caller.OrganizationID != orgID {
		return domain.Camera{}, domain.ErrCrossTenantAccess
	}
	return s.cameras.Get(ctx, orgID, cameraID)
}

func (s *Service) ListCameras(ctx context.Context, caller Caller, orgID domain.OrganizationID) ([]domain.Camera, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.cameras.ListByOrganization(ctx, orgID)
}
