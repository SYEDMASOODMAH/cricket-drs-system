package buffer

import (
	"testing"
	"time"
)

func testAudioChunk(t time.Time) AudioChunk {
	return AudioChunk{Samples: []int16{1, 2, 3}, SampleRate: 8000, CapturedAt: t}
}

func TestAudioRingBuffer_RetainsWithinWindow(t *testing.T) {
	b := NewAudioRingBuffer(10 * time.Second)
	base := time.Now()

	b.Add(testAudioChunk(base))
	b.Add(testAudioChunk(base.Add(5 * time.Second)))

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 chunks retained, got %d", len(snap))
	}
}

func TestAudioRingBuffer_PrunesOlderThanWindow(t *testing.T) {
	b := NewAudioRingBuffer(10 * time.Second)
	base := time.Now()

	b.Add(testAudioChunk(base))
	b.Add(testAudioChunk(base.Add(5 * time.Second)))
	b.Add(testAudioChunk(base.Add(15 * time.Second)))

	snap := b.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 chunks retained after pruning, got %d", len(snap))
	}
	if !snap[0].CapturedAt.Equal(base.Add(5 * time.Second)) {
		t.Errorf("expected oldest retained chunk at +5s, got %v", snap[0].CapturedAt)
	}
}

func TestAudioRingBuffer_SnapshotOrderedOldestFirst(t *testing.T) {
	b := NewAudioRingBuffer(time.Minute)
	base := time.Now()
	for i := 0; i < 5; i++ {
		b.Add(testAudioChunk(base.Add(time.Duration(i) * time.Second)))
	}

	snap := b.Snapshot()
	if len(snap) != 5 {
		t.Fatalf("expected 5 chunks, got %d", len(snap))
	}
	for i := 1; i < len(snap); i++ {
		if snap[i].CapturedAt.Before(snap[i-1].CapturedAt) {
			t.Fatalf("expected non-decreasing timestamps, got %v then %v", snap[i-1].CapturedAt, snap[i].CapturedAt)
		}
	}
}

func TestAudioRingBuffer_SnapshotIsACopy(t *testing.T) {
	b := NewAudioRingBuffer(time.Minute)
	base := time.Now()
	b.Add(testAudioChunk(base))

	snap := b.Snapshot()
	snap[0].CapturedAt = base.Add(time.Hour)

	again := b.Snapshot()
	if !again[0].CapturedAt.Equal(base) {
		t.Fatalf("expected internal state unaffected by mutating a snapshot, got %v", again[0].CapturedAt)
	}
}

func TestAudioRingBuffer_EmptySnapshot(t *testing.T) {
	b := NewAudioRingBuffer(time.Minute)
	snap := b.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot, got %d chunks", len(snap))
	}
}
