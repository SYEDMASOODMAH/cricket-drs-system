package memstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

func TestClipStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewClipStore()
	clip := domain.Clip{ID: "clip-1", OrganizationID: "org-a", MatchID: "match-1", CameraID: "cam-1", UploadedAt: time.Now()}

	if err := s.Create(ctx, clip); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := s.Get(ctx, "org-a", "match-1", "clip-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "clip-1" {
		t.Fatalf("unexpected clip: %+v", got)
	}
}

func TestClipStore_GetNotFound(t *testing.T) {
	s := NewClipStore()
	_, err := s.Get(context.Background(), "org-a", "match-1", "missing")
	if !errors.Is(err, domain.ErrClipNotFound) {
		t.Fatalf("expected ErrClipNotFound, got %v", err)
	}
}

func TestClipStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewClipStore()
	clipA := domain.Clip{ID: "clip-1", OrganizationID: "org-a", MatchID: "match-1", CameraID: "cam-1"}
	clipB := domain.Clip{ID: "clip-1", OrganizationID: "org-b", MatchID: "match-1", CameraID: "cam-1"}
	if err := s.Create(ctx, clipA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, clipB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := s.Get(ctx, "org-b", "match-1", "clip-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listA, err := s.ListByMatch(ctx, "org-a", "match-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 {
		t.Fatalf("expected org-a to see exactly its own clip, got %+v", listA)
	}
}

func TestClipStore_ListByMatch_ScopesByMatchToo(t *testing.T) {
	ctx := context.Background()
	s := NewClipStore()
	if err := s.Create(ctx, domain.Clip{ID: "clip-1", OrganizationID: "org-a", MatchID: "match-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, domain.Clip{ID: "clip-2", OrganizationID: "org-a", MatchID: "match-2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := s.ListByMatch(ctx, "org-a", "match-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != "clip-1" {
		t.Fatalf("expected only match-1's clip, got %+v", list)
	}
}
