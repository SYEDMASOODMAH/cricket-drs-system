package domain

// OrganizationID and UserID reference Identity & Access's tenant and user
// identifiers. MatchID references Match & Tournament's Match. This
// service does not own any of that data — it stores foreign references
// only (see the implementation plan's "no cross-service matchID
// validation" note).
type OrganizationID string
type UserID string
type MatchID string

// CameraID identifies a physical camera. The registry this was waiting on
// now exists at services/camera-calibration (docs/adr/0005), but this
// field is still just an opaque string supplied by the uploader — live
// cross-validation against a registered camera record is a documented
// follow-up, not wired in yet (same "trusted foreign reference"
// simplification already used for MatchID above).
type CameraID string

type ClipID string
