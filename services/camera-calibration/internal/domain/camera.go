package domain

import "time"

// Camera is one physical camera unit registered to an organization and
// venue at kit provisioning (prd.md Section 13.5) — registering it here
// once is what lets its calibration profile be reused across matches
// instead of recalibrated from scratch every time.
type Camera struct {
	ID             CameraID
	OrganizationID OrganizationID
	VenueID        VenueID
	Model          CameraModel
	RegisteredAt   time.Time
	RegisteredBy   UserID
}

func NewCamera(id CameraID, orgID OrganizationID, venueID VenueID, model CameraModel, registeredBy UserID, now time.Time) (Camera, error) {
	if venueID == "" {
		return Camera{}, ErrVenueIDEmpty
	}
	if model == "" {
		return Camera{}, ErrCameraModelEmpty
	}
	return Camera{
		ID:             id,
		OrganizationID: orgID,
		VenueID:        venueID,
		Model:          model,
		RegisteredAt:   now,
		RegisteredBy:   registeredBy,
	}, nil
}
