package service

import (
	"context"
	"fmt"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

// SubmitSyncOffset records an already-computed time offset between
// clipID and referenceClipID — the correlation itself runs externally in
// ml-pipeline/time-sync (docs/adr/0006); this only validates and
// persists the result. Requires the same tenant-isolation and
// CanUploadClips check as UploadClip (same persona responsible for
// match-day operations submits sync results, same as clip upload
// itself).
func (s *Service) SubmitSyncOffset(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, clipID, referenceClipID domain.ClipID, offsetMs int64, correlationScore float64) (domain.Clip, error) {
	if caller.OrganizationID != orgID {
		return domain.Clip{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanUploadClips(caller.Role) {
		return domain.Clip{}, domain.ErrPermissionDenied
	}

	clip, err := s.clips.Get(ctx, orgID, matchID, clipID)
	if err != nil {
		return domain.Clip{}, err
	}
	// Confirm the reference clip is a real clip in this same match — a
	// sync offset relative to a nonexistent or foreign-match clip is
	// meaningless.
	if _, err := s.clips.Get(ctx, orgID, matchID, referenceClipID); err != nil {
		return domain.Clip{}, err
	}

	synced, err := domain.ApplySyncOffset(clip, referenceClipID, offsetMs, correlationScore, s.now())
	if err != nil {
		return domain.Clip{}, err
	}

	// ClipRepository.Create is a plain upsert keyed by (org, match, clip)
	// id — see internal/memstore/clips.go — so reusing it here updates
	// the existing record in place rather than needing a separate Update
	// method on the port.
	if err := s.clips.Create(ctx, synced); err != nil {
		return domain.Clip{}, fmt.Errorf("service: record sync offset: %w", err)
	}
	return synced, nil
}
