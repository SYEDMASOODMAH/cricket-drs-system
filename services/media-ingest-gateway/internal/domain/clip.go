package domain

import "time"

// Clip is one uploaded video clip's metadata — the object bytes
// themselves live in the ObjectStore port (internal/service/ports.go),
// keyed by StorageKey; this is the record of what was uploaded, when, by
// whom, and its integrity hash.
type Clip struct {
	ID             ClipID
	OrganizationID OrganizationID
	MatchID        MatchID
	CameraID       CameraID
	StorageKey     string
	// ContentHash is a hex-encoded SHA-256 of the clip's bytes, computed
	// server-side during upload (never trusted from the client) — the
	// basic anti-tampering measure described in the implementation plan:
	// full replay/liveness detection is deferred, but a stored clip's
	// integrity can always be re-verified against this hash.
	ContentHash string
	SizeBytes   int64
	UploadedAt  time.Time
	UploadedBy  UserID
}

func NewClip(id ClipID, orgID OrganizationID, matchID MatchID, cameraID CameraID, storageKey, contentHash string, sizeBytes int64, uploadedBy UserID, now time.Time) (Clip, error) {
	if matchID == "" {
		return Clip{}, ErrMatchIDEmpty
	}
	if cameraID == "" {
		return Clip{}, ErrCameraIDEmpty
	}
	if sizeBytes <= 0 {
		return Clip{}, ErrEmptyContent
	}
	return Clip{
		ID:             id,
		OrganizationID: orgID,
		MatchID:        matchID,
		CameraID:       cameraID,
		StorageKey:     storageKey,
		ContentHash:    contentHash,
		SizeBytes:      sizeBytes,
		UploadedAt:     now,
		UploadedBy:     uploadedBy,
	}, nil
}
