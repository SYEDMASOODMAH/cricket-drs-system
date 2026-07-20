// Package service holds media-ingest-gateway's use-cases: uploading,
// listing, fetching, and downloading match video clips. It depends only
// on the domain package and the port interfaces below — never on a
// database driver, HTTP framework, or the S3/JWT library implementations
// directly (rules.md Section 1).
package service

import (
	"context"
	"io"
	"time"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

// ClipRepository persists clip metadata, always scoped by
// OrganizationID — the same tenant-isolation-at-the-port-boundary pattern
// identity-access's UserRepository and match-tournament's repositories
// use.
type ClipRepository interface {
	Create(ctx context.Context, clip domain.Clip) error
	Get(ctx context.Context, orgID domain.OrganizationID, matchID domain.MatchID, clipID domain.ClipID) (domain.Clip, error)
	ListByMatch(ctx context.Context, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.Clip, error)
}

// ObjectStore persists the clip's raw bytes, keyed by the storage key
// ClipRepository's records reference. Two adapters implement this:
// internal/objectstore's in-memory one (tests/local dev) and its S3 one
// (real AWS SDK, unit-tested against a fake — no real AWS call is made
// anywhere in this codebase, see that package's tests).
type ObjectStore interface {
	Put(ctx context.Context, key string, content io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// TokenVerifier wraps the vetted JWT implementation (rules.md Section 3.4
// bans hand-rolled auth crypto) — the service layer never imports the JWT
// library directly.
type TokenVerifier interface {
	Verify(token string) (Claims, error)
}

// Claims is what a verified token proves about its holder — minted by
// Identity & Access, verified here (same as match-tournament).
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
