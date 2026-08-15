package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/memstore"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/objectstore"
)

type fakeTokenVerifier struct {
	claims Claims
	err    error
}

func (f *fakeTokenVerifier) Verify(_ string) (Claims, error) {
	return f.claims, f.err
}

func newTestService() *Service {
	return New(memstore.NewClipStore(), objectstore.NewMemoryStore(), &fakeTokenVerifier{})
}

var orgAAdmin = Caller{OrganizationID: "org-a", UserID: "admin-a", Role: domain.RoleOrganizerAdmin}
var orgBAdmin = Caller{OrganizationID: "org-b", UserID: "admin-b", Role: domain.RoleOrganizerAdmin}
var orgAPlayer = Caller{OrganizationID: "org-a", UserID: "player-a", Role: domain.RolePlayer}

func TestUploadClip_Success(t *testing.T) {
	svc := newTestService()
	clip, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("video-bytes")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clip.SizeBytes != int64(len("video-bytes")) || clip.ContentHash == "" {
		t.Fatalf("unexpected clip: %+v", clip)
	}
}

func TestUploadClip_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.UploadClip(context.Background(), orgAAdmin, "org-b", "match-1", "cam-1", bytes.NewReader([]byte("x")))
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestUploadClip_PermissionDenied(t *testing.T) {
	svc := newTestService()
	_, err := svc.UploadClip(context.Background(), orgAPlayer, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("x")))
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestUploadClip_EmptyContentRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader(nil))
	if !errors.Is(err, domain.ErrEmptyContent) {
		t.Fatalf("expected ErrEmptyContent, got %v", err)
	}
}

func TestGetClip_Success(t *testing.T) {
	svc := newTestService()
	uploaded, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("video-bytes")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetClip(context.Background(), orgAAdmin, "org-a", "match-1", uploaded.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != uploaded.ID {
		t.Fatalf("unexpected clip: %+v", got)
	}
}

func TestGetClip_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	uploaded, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("video-bytes")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.GetClip(context.Background(), orgBAdmin, "org-a", "match-1", uploaded.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestListClips_TenantIsolation(t *testing.T) {
	svc := newTestService()
	if _, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("a"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.UploadClip(context.Background(), orgBAdmin, "org-b", "match-1", "cam-1", bytes.NewReader([]byte("b"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := svc.ListClips(context.Background(), orgAAdmin, "org-a", "match-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see exactly its own clip, got %+v", listA)
	}
}

func TestListClips_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.ListClips(context.Background(), orgAAdmin, "org-b", "match-1")
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestDownloadClip_RoundTrip(t *testing.T) {
	svc := newTestService()
	uploaded, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("video-bytes")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clip, rc, err := svc.DownloadClip(context.Background(), orgAAdmin, "org-a", "match-1", uploaded.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "video-bytes" {
		t.Fatalf("unexpected downloaded content: %q", data)
	}
	if clip.ID != uploaded.ID {
		t.Fatalf("unexpected clip metadata: %+v", clip)
	}
}

func TestDownloadClip_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	uploaded, err := svc.UploadClip(context.Background(), orgAAdmin, "org-a", "match-1", "cam-1", bytes.NewReader([]byte("video-bytes")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _, err = svc.DownloadClip(context.Background(), orgBAdmin, "org-a", "match-1", uploaded.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestDownloadClip_NotFound(t *testing.T) {
	svc := newTestService()
	_, _, err := svc.DownloadClip(context.Background(), orgAAdmin, "org-a", "match-1", "does-not-exist")
	if !errors.Is(err, domain.ErrClipNotFound) {
		t.Fatalf("expected ErrClipNotFound, got %v", err)
	}
}

func TestAuthenticate_Success(t *testing.T) {
	claims := Claims{UserID: "user-1", OrganizationID: "org-a", Role: domain.RoleOrganizerAdmin}
	svc := New(memstore.NewClipStore(), objectstore.NewMemoryStore(), &fakeTokenVerifier{claims: claims})
	caller, err := svc.Authenticate("some-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.UserID != "user-1" || caller.OrganizationID != "org-a" || caller.Role != domain.RoleOrganizerAdmin {
		t.Fatalf("unexpected caller: %+v", caller)
	}
}

func TestAuthenticate_InvalidTokenRejected(t *testing.T) {
	svc := New(memstore.NewClipStore(), objectstore.NewMemoryStore(), &fakeTokenVerifier{err: errors.New("bad token")})
	_, err := svc.Authenticate("garbage")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
