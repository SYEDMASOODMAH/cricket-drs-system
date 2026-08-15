package httpapi

import (
	"time"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

type registerCameraRequest struct {
	VenueID string `json:"venue_id"`
	Model   string `json:"model"`
}

type cameraResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	VenueID        string    `json:"venue_id"`
	Model          string    `json:"model"`
	RegisteredAt   time.Time `json:"registered_at"`
	RegisteredBy   string    `json:"registered_by"`
}

func toCameraResponse(c domain.Camera) cameraResponse {
	return cameraResponse{
		ID:             string(c.ID),
		OrganizationID: string(c.OrganizationID),
		VenueID:        string(c.VenueID),
		Model:          string(c.Model),
		RegisteredAt:   c.RegisteredAt,
		RegisteredBy:   string(c.RegisteredBy),
	}
}

type storeCalibrationRequest struct {
	Rotation            [3]float64 `json:"rotation"`
	Translation         [3]float64 `json:"translation"`
	ReprojectionErrorPx float64    `json:"reprojection_error_px"`
}

type calibrationProfileResponse struct {
	CameraID            string     `json:"camera_id"`
	OrganizationID      string     `json:"organization_id"`
	Rotation            [3]float64 `json:"rotation"`
	Translation         [3]float64 `json:"translation"`
	ReprojectionErrorPx float64    `json:"reprojection_error_px"`
	Valid               bool       `json:"valid"`
	CalibratedAt        time.Time  `json:"calibrated_at"`
	CalibratedBy        string     `json:"calibrated_by"`
}

func toCalibrationProfileResponse(p domain.CalibrationProfile) calibrationProfileResponse {
	return calibrationProfileResponse{
		CameraID:            string(p.CameraID),
		OrganizationID:      string(p.OrganizationID),
		Rotation:            p.RotationVector,
		Translation:         p.TranslationVector,
		ReprojectionErrorPx: p.ReprojectionErrorPx,
		Valid:               p.Valid(),
		CalibratedAt:        p.CalibratedAt,
		CalibratedBy:        string(p.CalibratedBy),
	}
}

type lensProfileResponse struct {
	Model             string     `json:"model"`
	FocalLengthPxX    float64    `json:"focal_length_px_x"`
	FocalLengthPxY    float64    `json:"focal_length_px_y"`
	PrincipalPointPxX float64    `json:"principal_point_px_x"`
	PrincipalPointPxY float64    `json:"principal_point_px_y"`
	DistortionCoeffs  [5]float64 `json:"distortion_coeffs"`
}

func toLensProfileResponse(p domain.LensProfile) lensProfileResponse {
	return lensProfileResponse{
		Model:             string(p.Model),
		FocalLengthPxX:    p.FocalLengthPxX,
		FocalLengthPxY:    p.FocalLengthPxY,
		PrincipalPointPxX: p.PrincipalPointPxX,
		PrincipalPointPxY: p.PrincipalPointPxY,
		DistortionCoeffs:  p.DistortionCoeffs,
	}
}
