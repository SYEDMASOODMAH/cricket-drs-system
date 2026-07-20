package domain

import "time"

// MaxReprojectionErrorPx is the accuracy threshold below which a
// calibration profile is considered valid for match use. 2.0px is a
// conventional "good calibration" bar for this kind of pose estimation —
// a placeholder pending real accuracy targets from Phase 0's field
// validation (phases.md Phase 2 testing requirement: "calibration
// accuracy validated against measured pitch geometry"), not a measured or
// contractually-committed number yet.
const MaxReprojectionErrorPx = 2.0

// CalibrationProfile is a camera's extrinsic calibration for one venue:
// its position and orientation relative to known pitch geometry
// (architecture.md Section 9), expressed as an OpenCV-style Rodrigues
// rotation vector plus a translation vector. The values themselves are
// computed externally (ml-pipeline/camera-calibration — see
// docs/adr/0005) and submitted here as already-computed numbers; this
// service stores and evaluates them, it does not compute them.
type CalibrationProfile struct {
	CameraID            CameraID
	OrganizationID      OrganizationID
	RotationVector      [3]float64
	TranslationVector   [3]float64
	ReprojectionErrorPx float64
	CalibratedAt        time.Time
	CalibratedBy        UserID
}

func NewCalibrationProfile(cameraID CameraID, orgID OrganizationID, rotation, translation [3]float64, reprojectionErrorPx float64, calibratedBy UserID, now time.Time) (CalibrationProfile, error) {
	if reprojectionErrorPx < 0 {
		return CalibrationProfile{}, ErrInvalidCalibration
	}
	return CalibrationProfile{
		CameraID:            cameraID,
		OrganizationID:      orgID,
		RotationVector:      rotation,
		TranslationVector:   translation,
		ReprojectionErrorPx: reprojectionErrorPx,
		CalibratedAt:        now,
		CalibratedBy:        calibratedBy,
	}, nil
}

// Valid reports whether this profile's accuracy is good enough for match
// use — the "pre-match camera detection and calibration/health check"
// prd.md Section 5.4 describes.
func (p CalibrationProfile) Valid() bool {
	return p.ReprojectionErrorPx <= MaxReprojectionErrorPx
}
