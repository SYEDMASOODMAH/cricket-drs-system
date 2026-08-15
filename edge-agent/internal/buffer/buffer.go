// Package buffer holds the rolling window of recently captured frames
// edge-agent keeps per camera (docs/architecture.md Sections 2, 9a, 10:
// "maintain a rolling buffer, target: last 20-30s"). Pure in-memory
// logic — no camera, network, or file I/O — so it's fully unit-testable
// without real hardware.
package buffer

import (
	"image"
	"sync"
	"time"
)

// Frame is one captured video frame with its capture timestamp.
type Frame struct {
	Image      image.Image
	CapturedAt time.Time
}

// RingBuffer retains frames added within the last Window duration,
// dropping older ones as new frames arrive. Safe for concurrent use: one
// goroutine typically calls Add continuously from the capture loop while
// another calls Snapshot on trigger (cmd/main.go).
type RingBuffer struct {
	mu     sync.Mutex
	window time.Duration
	frames []Frame
}

func NewRingBuffer(window time.Duration) *RingBuffer {
	return &RingBuffer{window: window}
}

// Add appends a frame and prunes any frames now older than Window
// relative to this frame's timestamp. Assumes frames arrive in
// non-decreasing CapturedAt order, true for a live sequential capture
// loop.
func (b *RingBuffer) Add(frame Frame) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.frames = append(b.frames, frame)

	cutoff := frame.CapturedAt.Add(-b.window)
	i := 0
	for i < len(b.frames) && b.frames[i].CapturedAt.Before(cutoff) {
		i++
	}
	if i > 0 {
		b.frames = b.frames[i:]
	}
}

// Snapshot returns a copy of the frames currently retained, oldest
// first. A copy, not a slice view, so the caller can safely encode it
// (clipformat.Encode) while Add continues running concurrently.
func (b *RingBuffer) Snapshot() []Frame {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]Frame, len(b.frames))
	copy(out, b.frames)
	return out
}
