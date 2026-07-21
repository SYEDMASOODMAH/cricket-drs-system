package domain

import (
	"errors"
	"testing"
	"time"
)

func TestApplySyncOffset_Valid(t *testing.T) {
	clip := Clip{ID: "clip-2", OrganizationID: "org-1"}
	now := time.Now()
	synced, err := ApplySyncOffset(clip, "clip-1", 150, 0.9, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if synced.SyncOffsetMs == nil || *synced.SyncOffsetMs != 150 {
		t.Fatalf("unexpected offset: %+v", synced.SyncOffsetMs)
	}
	if synced.SyncReferenceClipID == nil || *synced.SyncReferenceClipID != "clip-1" {
		t.Fatalf("unexpected reference clip id: %+v", synced.SyncReferenceClipID)
	}
	if synced.SyncCorrelationScore == nil || *synced.SyncCorrelationScore != 0.9 {
		t.Fatalf("unexpected correlation score: %+v", synced.SyncCorrelationScore)
	}
	if synced.SyncedAt == nil || !synced.SyncedAt.Equal(now) {
		t.Fatalf("unexpected synced at: %+v", synced.SyncedAt)
	}
}

func TestApplySyncOffset_SelfReferenceRejected(t *testing.T) {
	clip := Clip{ID: "clip-1", OrganizationID: "org-1"}
	_, err := ApplySyncOffset(clip, "clip-1", 0, 1.0, time.Now())
	if !errors.Is(err, ErrSyncSelfReference) {
		t.Fatalf("expected ErrSyncSelfReference, got %v", err)
	}
}

func TestApplySyncOffset_CorrelationScoreOutOfRangeRejected(t *testing.T) {
	clip := Clip{ID: "clip-2", OrganizationID: "org-1"}
	for _, score := range []float64{1.01, -1.01, 2.0, -5.0} {
		_, err := ApplySyncOffset(clip, "clip-1", 0, score, time.Now())
		if !errors.Is(err, ErrInvalidCorrelation) {
			t.Fatalf("score=%v: expected ErrInvalidCorrelation, got %v", score, err)
		}
	}
}

func TestApplySyncOffset_CorrelationScoreBoundariesAccepted(t *testing.T) {
	clip := Clip{ID: "clip-2", OrganizationID: "org-1"}
	for _, score := range []float64{-1.0, 1.0, 0.0} {
		if _, err := ApplySyncOffset(clip, "clip-1", 0, score, time.Now()); err != nil {
			t.Fatalf("score=%v: unexpected error: %v", score, err)
		}
	}
}

func TestClip_SyncConfident(t *testing.T) {
	unsynced := Clip{ID: "clip-2"}
	if unsynced.SyncConfident() {
		t.Error("expected an unsynced clip to not be sync-confident")
	}

	confident, err := ApplySyncOffset(Clip{ID: "clip-2"}, "clip-1", 100, MinSyncCorrelationScore, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confident.SyncConfident() {
		t.Errorf("expected a clip with score exactly at the threshold (%v) to be confident", MinSyncCorrelationScore)
	}

	unconfident, err := ApplySyncOffset(Clip{ID: "clip-2"}, "clip-1", 100, MinSyncCorrelationScore-0.01, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unconfident.SyncConfident() {
		t.Error("expected a clip with score just below the threshold to not be confident")
	}
}
