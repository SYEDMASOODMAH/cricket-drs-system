package domain

// LensProfile is a camera model's standardized intrinsic
// (lens-distortion) calibration — built once per supported camera model,
// not re-derived per physical unit (architecture.md Section 9). This is
// reference data, not per-tenant state, so it's a hardcoded table rather
// than a repository-backed resource.
//
// FocalLength/PrincipalPoint are in pixels, matching OpenCV's camera
// matrix convention; DistortionCoeffs follows OpenCV's standard 5-element
// (k1, k2, p1, p2, k3) plumb-bob model.
type LensProfile struct {
	Model             CameraModel
	FocalLengthPxX    float64
	FocalLengthPxY    float64
	PrincipalPointPxX float64
	PrincipalPointPxY float64
	DistortionCoeffs  [5]float64
}

// knownLensProfiles seeds the one GoPro model referenced elsewhere in
// this codebase's docs (docs/adr/0003). These specific numbers are
// structurally-shaped placeholders, not measured values — real
// coefficients depend on the exact resolution/FOV/lens mode the platform
// standardizes on, which is a Phase 0 field-validation output this
// environment has no access to. Replace before relying on this for a real
// match.
var knownLensProfiles = map[CameraModel]LensProfile{
	"GoPro Hero 12 Black": {
		Model:             "GoPro Hero 12 Black",
		FocalLengthPxX:    1420.0,
		FocalLengthPxY:    1420.0,
		PrincipalPointPxX: 960.0,
		PrincipalPointPxY: 540.0,
		DistortionCoeffs:  [5]float64{-0.28, 0.09, 0.0, 0.0, -0.01},
	},
}

// LensProfileFor looks up a camera model's standardized intrinsic
// profile.
func LensProfileFor(model CameraModel) (LensProfile, error) {
	profile, ok := knownLensProfiles[model]
	if !ok {
		return LensProfile{}, ErrCameraModelUnknown
	}
	return profile, nil
}
