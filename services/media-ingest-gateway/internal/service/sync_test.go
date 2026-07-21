package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

func uploadTestClip(t *testing.T, svc *Service, caller Caller, orgID domain.OrganizationID, matchID domain.MatchID, cameraID domain.CameraID, content string) domain.Clip {
	t.Helper()
	clip, err := svc.UploadClip(context.Background(), caller, orgID, matchID, cameraID, bytes.NewReader([]byte(content)))
	if err != nil {
		t.Fatalf("unexpected error uploading clip: %v", err)
	}
	return clip
}

func TestSubmitSyncOffset_Success(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	synced, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", target.ID, reference.ID, 250, 0.9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if synced.SyncOffsetMs == nil || *synced.SyncOffsetMs != 250 {
		t.Fatalf("unexpected sync offset: %+v", synced.SyncOffsetMs)
	}
	if !synced.SyncConfident() {
		t.Error("expected a 0.9 correlation score to be sync-confident")
	}

	// Confirm the update was actually persisted, not just returned.
	got, err := svc.GetClip(context.Background(), orgAAdmin, "org-a", "match-1", target.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SyncOffsetMs == nil || *got.SyncOffsetMs != 250 {
		t.Fatalf("expected sync offset to be persisted, got: %+v", got.SyncOffsetMs)
	}
}

func TestSubmitSyncOffset_LowConfidenceNotConfident(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	synced, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", target.ID, reference.ID, 250, 0.1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if synced.SyncConfident() {
		t.Error("expected a 0.1 correlation score to not be sync-confident")
	}
}

func TestSubmitSyncOffset_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	_, err := svc.SubmitSyncOffset(context.Background(), orgBAdmin, "org-a", "match-1", target.ID, reference.ID, 250, 0.9)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestSubmitSyncOffset_PermissionDenied(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	_, err := svc.SubmitSyncOffset(context.Background(), orgAPlayer, "org-a", "match-1", target.ID, reference.ID, 250, 0.9)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestSubmitSyncOffset_TargetClipNotFound(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")

	_, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", "does-not-exist", reference.ID, 250, 0.9)
	if !errors.Is(err, domain.ErrClipNotFound) {
		t.Fatalf("expected ErrClipNotFound, got %v", err)
	}
}

func TestSubmitSyncOffset_ReferenceClipNotFound(t *testing.T) {
	svc := newTestService()
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	_, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", target.ID, "does-not-exist", 250, 0.9)
	if !errors.Is(err, domain.ErrClipNotFound) {
		t.Fatalf("expected ErrClipNotFound, got %v", err)
	}
}

func TestSubmitSyncOffset_SelfReferenceRejected(t *testing.T) {
	svc := newTestService()
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	_, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", target.ID, target.ID, 0, 1.0)
	if !errors.Is(err, domain.ErrSyncSelfReference) {
		t.Fatalf("expected ErrSyncSelfReference, got %v", err)
	}
}

func TestSubmitSyncOffset_InvalidCorrelationScoreRejected(t *testing.T) {
	svc := newTestService()
	reference := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-1", "reference")
	target := uploadTestClip(t, svc, orgAAdmin, "org-a", "match-1", "cam-2", "target")

	_, err := svc.SubmitSyncOffset(context.Background(), orgAAdmin, "org-a", "match-1", target.ID, reference.ID, 250, 1.5)
	if !errors.Is(err, domain.ErrInvalidCorrelation) {
		t.Fatalf("expected ErrInvalidCorrelation, got %v", err)
	}
}
