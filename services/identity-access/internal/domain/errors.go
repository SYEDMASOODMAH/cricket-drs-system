// Package domain holds identity-access's core types and business rules:
// organizations (tenants), users, RBAC, and consent. Per rules.md Section 1
// (Clean Architecture / Hexagonal boundaries), nothing here imports a
// transport, persistence, or framework package — it must be unit-testable
// with no database or network call.
package domain

import "errors"

// Sentinel errors let callers distinguish failure classes via errors.Is,
// per rules.md Section 4.2 (no generic errors bubbling to the API boundary).
var (
	ErrOrganizationNotFound    = errors.New("organization not found")
	ErrOrganizationNameEmpty   = errors.New("organization name must not be empty")
	ErrUserNotFound            = errors.New("user not found")
	ErrUserExists              = errors.New("user already exists")
	ErrUserEmailEmpty          = errors.New("user email must not be empty")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidRole             = errors.New("invalid role")
	ErrPermissionDenied        = errors.New("permission denied")
	ErrCrossTenantAccess       = errors.New("cross-tenant access denied")
	ErrConsentNotFound         = errors.New("consent record not found")
	ErrGuardianConsentRequired = errors.New("guardian consent is required for a minor")
)
