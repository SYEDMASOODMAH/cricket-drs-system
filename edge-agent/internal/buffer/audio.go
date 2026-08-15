package buffer

import (
	"sync"
	"time"
)

// AudioChunk is one captured slice of PCM audio samples with its capture
// timestamp — the audio counterpart to Frame. A separate type (not a
// generic Frame[T]) since its shape genuinely differs (raw samples +
// sample rate, not an image); see internal/capture.Microphone.Read.
type AudioChunk struct {
	Samples    []int16
	SampleRate int
	CapturedAt time.Time
}

// AudioRingBuffer retains audio chunks added within the last Window
// duration, the same time-window Add/Snapshot shape as RingBuffer
// (duplicated rather than generic-ified — see the implementation plan's
// "Decisions being made" section for why).
type AudioRingBuffer struct {
	mu     sync.Mutex
	window time.Duration
	chunks []AudioChunk
}

func NewAudioRingBuffer(window time.Duration) *AudioRingBuffer {
	return &AudioRingBuffer{window: window}
}

// Add appends a chunk and prunes any chunks now older than Window
// relative to this chunk's timestamp. Assumes chunks arrive in
// non-decreasing CapturedAt order, true for a live sequential capture
// loop.
func (b *AudioRingBuffer) Add(chunk AudioChunk) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chunks = append(b.chunks, chunk)

	cutoff := chunk.CapturedAt.Add(-b.window)
	i := 0
	for i < len(b.chunks) && b.chunks[i].CapturedAt.Before(cutoff) {
		i++
	}
	if i > 0 {
		b.chunks = b.chunks[i:]
	}
}

// Snapshot returns a copy of the chunks currently retained, oldest
// first.
func (b *AudioRingBuffer) Snapshot() []AudioChunk {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]AudioChunk, len(b.chunks))
	copy(out, b.chunks)
	return out
}
