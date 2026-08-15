package domain

// OrganizationID and UserID reference Identity & Access's tenant and user
// identifiers. MatchID references Match & Tournament's Match. This
// service does not own any of that data — it stores foreign references
// only (see the implementation plan's "no cross-service matchID
// validation" note).
type OrganizationID string
type UserID string
type MatchID string

// CameraID identifies a physical camera. Unlike MatchID above, this one IS
// cross-validated: internal/service/clip.go's UploadClip calls
// internal/cameracalibration.Client.IsRegistered against
// services/camera-calibration (docs/adr/0005) before accepting an upload,
// rejecting camera_id values that were never registered there.
type CameraID string

type ClipID string
