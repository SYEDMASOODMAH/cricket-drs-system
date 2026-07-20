package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

// UploadClip requires the caller to belong to orgID (tenant isolation)
// and hold CanUploadClips (organizer_admin — see domain/role.go's doc
// comment on why this reuses the human-persona role rather than a
// distinct edge-device credential for now).
//
// content is read fully into memory, hashed (SHA-256 — the basic
// anti-tampering measure described in the implementation plan), and only
// then written to the ObjectStore — not a true streaming upload. That's a
// deliberate Phase 2 "basic" simplification: it keeps the logic simple
// and correct (no partial-write cleanup on a failed streaming write to
// worry about) at the cost of buffering a whole clip in memory, which is
// fine for what's actually being tested here and not yet a real
// production upload path (see the implementation plan's "explicitly
// deferred" section — there's no edge-agent streaming client to receive
// from yet).
func (s *Service) UploadClip(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, cameraID domain.CameraID, content io.Reader) (domain.Clip, error) {
	if caller.OrganizationID != orgID {
		return domain.Clip{}, domain.ErrCrossTenantAccess
	}
	if !domain.CanUploadClips(caller.Role) {
		return domain.Clip{}, domain.ErrPermissionDenied
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return domain.Clip{}, fmt.Errorf("service: read clip content: %w", err)
	}

	sum := sha256.Sum256(data)
	contentHash := hex.EncodeToString(sum[:])

	clipID := domain.ClipID(newID("clip"))
	storageKey := fmt.Sprintf("%s/%s/%s", orgID, matchID, clipID)

	clip, err := domain.NewClip(clipID, orgID, matchID, cameraID, storageKey, contentHash, int64(len(data)), caller.UserID, s.now())
	if err != nil {
		return domain.Clip{}, err
	}

	if err := s.objects.Put(ctx, storageKey, bytes.NewReader(data)); err != nil {
		return domain.Clip{}, fmt.Errorf("service: store clip content: %w", err)
	}
	if err := s.clips.Create(ctx, clip); err != nil {
		return domain.Clip{}, fmt.Errorf("service: record clip metadata: %w", err)
	}

	return clip, nil
}

// GetClip and ListClips only enforce tenant isolation — any authenticated
// org member can read (mirrors identity-access's GetUser and
// match-tournament's GetMatch).
func (s *Service) GetClip(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, clipID domain.ClipID) (domain.Clip, error) {
	if caller.OrganizationID != orgID {
		return domain.Clip{}, domain.ErrCrossTenantAccess
	}
	return s.clips.Get(ctx, orgID, matchID, clipID)
}

func (s *Service) ListClips(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.Clip, error) {
	if caller.OrganizationID != orgID {
		return nil, domain.ErrCrossTenantAccess
	}
	return s.clips.ListByMatch(ctx, orgID, matchID)
}

// DownloadClip fetches the clip's metadata (enforcing the same
// tenant-isolation check as GetClip) and then streams its bytes from the
// ObjectStore.
func (s *Service) DownloadClip(ctx context.Context, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, clipID domain.ClipID) (domain.Clip, io.ReadCloser, error) {
	clip, err := s.GetClip(ctx, caller, orgID, matchID, clipID)
	if err != nil {
		return domain.Clip{}, nil, err
	}

	content, err := s.objects.Get(ctx, clip.StorageKey)
	if err != nil {
		return domain.Clip{}, nil, fmt.Errorf("service: fetch clip content: %w", err)
	}
	return clip, content, nil
}
