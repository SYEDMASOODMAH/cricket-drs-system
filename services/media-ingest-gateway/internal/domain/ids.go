package domain

// OrganizationID and UserID reference Identity & Access's tenant and user
// identifiers. MatchID references Match & Tournament's Match. This
// service does not own any of that data — it stores foreign references
// only (see the implementation plan's "no cross-service matchID
// validation" note).
type OrganizationID string
type UserID string
type MatchID string

// CameraID identifies a physical camera. No Camera Calibration Service
// exists yet (a separate, later Phase 2 slice), so this is currently just
// an opaque string supplied by the uploader — not yet cross-validated
// against a registered camera record.
type CameraID string

type ClipID string
