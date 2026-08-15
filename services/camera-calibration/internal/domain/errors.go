package domain

import "errors"

var (
	ErrCameraNotFound     = errors.New("camera not found")
	ErrProfileNotFound    = errors.New("calibration profile not found")
	ErrVenueIDEmpty       = errors.New("venue id must not be empty")
	ErrCameraModelEmpty   = errors.New("camera model must not be empty")
	ErrCameraModelUnknown = errors.New("camera model has no registered lens profile")
	ErrInvalidCalibration = errors.New("calibration data invalid")
	ErrInvalidRole        = errors.New("invalid role")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrCrossTenantAccess  = errors.New("cross-tenant access denied")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
