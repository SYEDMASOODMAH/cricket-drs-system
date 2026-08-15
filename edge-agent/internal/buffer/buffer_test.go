package buffer

import (
	"image"
	"testing"
	"time"
)

func testFrame(t time.Time) Frame {
	return Frame{Image: image.NewRGBA(image.Rect(0, 0, 1, 1)), CapturedAt: t}
}

func TestRingBuffer_RetainsWithinWindow(t *testing.T) {
	b := NewRingBuffer(10 * time.Second)
	base := time.Now()

	b.Add(testFrame(base))
	b.Add(testFrame(base.Add(5 * time.Second)))

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 frames retained, got %d", len(snap))
	}
}

func TestRingBuffer_PrunesOlderThanWindow(t *testing.T) {
	b := NewRingBuffer(10 * time.Second)
	base := time.Now()

	b.Add(testFrame(base))
	b.Add(testFrame(base.Add(5 * time.Second)))
	// This frame is 15s after the first — the first frame (0s) is now
	// 15s old, outside the 10s window, and should be pruned.
	b.Add(testFrame(base.Add(15 * time.Second)))

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 frames retained after pruning, got %d", len(snap))
	}
	if !snap[0].CapturedAt.Equal(base.Add(5 * time.Second)) {
		t.Errorf("expected oldest retained frame at +5s, got %v", snap[0].CapturedAt)
	}
}

func TestRingBuffer_SnapshotOrderedOldestFirst(t *testing.T) {
	b := NewRingBuffer(time.Minute)
	base := time.Now()
	for i := 0; i < 5; i++ {
		b.Add(testFrame(base.Add(time.Duration(i) * time.Second)))
	}

	snap := b.Snapshot()
	if len(snap) != 5 {
		t.Fatalf("expected 5 frames, got %d", len(snap))
	}
	for i := 1; i < len(snap); i++ {
		if snap[i].CapturedAt.Before(snap[i-1].CapturedAt) {
			t.Fatalf("expected non-decreasing timestamps, got %v then %v", snap[i-1].CapturedAt, snap[i].CapturedAt)
		}
	}
}

func TestRingBuffer_SnapshotIsACopy(t *testing.T) {
	b := NewRingBuffer(time.Minute)
	base := time.Now()
	b.Add(testFrame(base))

	snap := b.Snapshot()
	snap[0].CapturedAt = base.Add(time.Hour) // mutate the copy

	again := b.Snapshot()
	if !again[0].CapturedAt.Equal(base) {
		t.Fatalf("expected internal state unaffected by mutating a snapshot, got %v", again[0].CapturedAt)
	}
}

func TestRingBuffer_EmptySnapshot(t *testing.T) {
	b := NewRingBuffer(time.Minute)
	snap := b.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot, got %d frames", len(snap))
	}
}
