// Package service holds camera-calibration's use-cases: registering
// cameras and storing/evaluating their calibration profiles. It depends
// only on the domain package and the port interfaces below — never on a
// database driver, HTTP framework, or the JWT library implementation
// directly (rules.md Section 1).
package service

import (
	"context"
	"time"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

// CameraRepository persists camera registrations, always scoped by
// OrganizationID — the same tenant-isolation-at-the-port-boundary pattern
// every other service in this module uses.
type CameraRepository interface {
	Create(ctx context.Context, camera domain.Camera) error
	Get(ctx context.Context, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.Camera, error)
	ListByOrganization(ctx context.Context, orgID domain.OrganizationID) ([]domain.Camera, error)
}

// ProfileRepository persists calibration profiles, one per camera — a
// new submission for the same camera replaces the previous profile
// (Store is an upsert), matching how a real recalibration would supersede
// a stale one rather than accumulate history.
type ProfileRepository interface {
	Store(ctx context.Context, profile domain.CalibrationProfile) error
	Get(ctx context.Context, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.CalibrationProfile, error)
}

// TokenVerifier wraps the vetted JWT implementation (rules.md Section 3.4
// bans hand-rolled auth crypto) — the service layer never imports the JWT
// library directly.
type TokenVerifier interface {
	Verify(token string) (Claims, error)
}

// Claims is what a verified token proves about its holder — minted by
// Identity & Access, verified here (same as the other three services).
type Claims struct {
	UserID         domain.UserID
	OrganizationID domain.OrganizationID
	Role           domain.Role
}

// Caller is the authenticated identity behind a service call, threaded
// explicitly through every method that needs a tenant-isolation or
// permission check.
type Caller struct {
	OrganizationID domain.OrganizationID
	UserID         domain.UserID
	Role           domain.Role
}

// Clock abstracts time.Now for deterministic tests.
type Clock func() time.Time
