// Package domain holds camera-calibration's core types and business
// rules: camera registration and calibration-profile validity. Per
// rules.md Section 1, nothing here imports a transport, persistence, or
// framework package — it must be unit-testable with no database or
// network call.
package domain

// OrganizationID and UserID reference Identity & Access's tenant and user
// identifiers, duplicated the same way as every other service in this
// module (see this service's README).
type OrganizationID string
type UserID string

// VenueID identifies a physical venue (club ground). Opaque foreign
// reference only — no Venue entity is owned or validated here, the same
// "trusted foreign reference" simplification media-ingest-gateway already
// uses for MatchID.
type VenueID string

// CameraID identifies one physical camera, registered once at kit
// provisioning (prd.md Section 13.5) and reused across matches. This is
// the registry media-ingest-gateway's own CameraID type has been waiting
// on (see that service's internal/domain/ids.go) — cross-service
// validation between the two remains a deferred follow-up, not wired in
// this slice (docs/adr/0005).
type CameraID string

// CameraModel identifies a camera hardware model (e.g. "GoPro Hero 12
// Black"), not a specific physical unit — intrinsic lens-distortion
// profiles are keyed by this, one profile per model rather than per unit
// (architecture.md Section 9: "standardized... profile, built once per
// supported camera model, not re-derived per unit").
type CameraModel string
