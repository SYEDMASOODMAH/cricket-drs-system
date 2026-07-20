// Package domain holds media-ingest-gateway's core types and business
// rules: clips and their upload authorization. Per rules.md Section 1,
// nothing here imports a transport, persistence, or framework package —
// it must be unit-testable with no database, object store, or network
// call.
package domain

import "errors"

var (
	ErrClipNotFound        = errors.New("clip not found")
	ErrMatchIDEmpty        = errors.New("match id must not be empty")
	ErrCameraIDEmpty       = errors.New("camera id must not be empty")
	ErrEmptyContent        = errors.New("clip content must not be empty")
	ErrContentHashMismatch = errors.New("stored content hash does not match recorded hash")
	ErrInvalidRole         = errors.New("invalid role")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrCrossTenantAccess   = errors.New("cross-tenant access denied")
	ErrInvalidToken        = errors.New("invalid or expired token")
)
