package domain

import "time"

// MinSyncCorrelationScore is the confidence threshold above which a
// submitted sync offset is trusted. No document (architecture.md,
// prd.md, phases.md) specifies an actual number — phases.md's Phase 2
// completion criteria says only "within a defined tolerance," which is
// aspirational/TBD language, not a spec. 0.5 is a structural placeholder,
// same honesty as camera-calibration's MaxReprojectionErrorPx, pending
// real accuracy targets from field validation.
const MinSyncCorrelationScore = 0.5

// ApplySyncOffset records an already-computed time offset between this
// clip and a reference clip — the correlation itself runs externally in
// ml-pipeline/time-sync (see docs/adr/0006); this only validates and
// stores the result, it never computes it.
func ApplySyncOffset(clip Clip, referenceClipID ClipID, offsetMs int64, correlationScore float64, now time.Time) (Clip, error) {
	if referenceClipID == clip.ID {
		return Clip{}, ErrSyncSelfReference
	}
	if correlationScore < -1.0 || correlationScore > 1.0 {
		return Clip{}, ErrInvalidCorrelation
	}

	clip.SyncOffsetMs = &offsetMs
	clip.SyncReferenceClipID = &referenceClipID
	clip.SyncCorrelationScore = &correlationScore
	clip.SyncedAt = &now
	return clip, nil
}

// SyncConfident reports whether this clip's sync offset (if any) meets
// the confidence threshold for trusted use — the sync-side counterpart to
// services/camera-calibration's CalibrationProfile.Valid().
func (c Clip) SyncConfident() bool {
	return c.SyncCorrelationScore != nil && *c.SyncCorrelationScore >= MinSyncCorrelationScore
}
