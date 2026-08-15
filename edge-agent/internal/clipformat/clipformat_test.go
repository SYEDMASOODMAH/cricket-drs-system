package clipformat

import (
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/cricketdrs/edge-agent/internal/buffer"
)

// solidFrame builds a small synthetic frame filled with a single
// distinguishable color, so a round trip can check real pixel content
// survived (approximately — JPEG is lossy) rather than just structure.
func solidFrame(t time.Time, c color.RGBA) buffer.Frame {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return buffer.Frame{Image: img, CapturedAt: t}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	base := time.Now().Truncate(time.Second) // JPEG/nanosecond-safe truncation not needed, but keeps comparisons tidy
	frames := []buffer.Frame{
		solidFrame(base, color.RGBA{R: 200, G: 20, B: 20, A: 255}),
		solidFrame(base.Add(33*time.Millisecond), color.RGBA{R: 20, G: 200, B: 20, A: 255}),
		solidFrame(base.Add(66*time.Millisecond), color.RGBA{R: 20, G: 20, B: 200, A: 255}),
	}

	encoded, err := Encode(frames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) != len(frames) {
		t.Fatalf("expected %d frames, got %d", len(frames), len(decoded))
	}

	for i, want := range frames {
		got := decoded[i]
		if !got.CapturedAt.Equal(want.CapturedAt) {
			t.Errorf("frame %d: timestamp mismatch: got %v, want %v", i, got.CapturedAt, want.CapturedAt)
		}
		if got.Image.Bounds() != want.Image.Bounds() {
			t.Errorf("frame %d: bounds mismatch: got %v, want %v", i, got.Image.Bounds(), want.Image.Bounds())
		}
		// JPEG is lossy, so check the decoded center pixel is close to
		// the original solid color rather than exactly equal.
		wantR, wantG, wantB, _ := want.Image.At(16, 16).RGBA()
		gotR, gotG, gotB, _ := got.Image.At(16, 16).RGBA()
		const tolerance = 10 << 8 // RGBA() returns 16-bit-scaled components
		if absDiff(gotR, wantR) > tolerance || absDiff(gotG, wantG) > tolerance || absDiff(gotB, wantB) > tolerance {
			t.Errorf("frame %d: pixel color drifted too far: got (%d,%d,%d), want (%d,%d,%d)", i, gotR, gotG, gotB, wantR, wantG, wantB)
		}
	}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestEncodeDecode_EmptyFrameList(t *testing.T) {
	encoded, err := Encode(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) != 0 {
		t.Fatalf("expected 0 frames, got %d", len(decoded))
	}
}

func TestDecode_RejectsBadMagic(t *testing.T) {
	_, err := Decode([]byte("not a clip file at all"))
	if !errors.Is(err, ErrBadMagic) {
		t.Fatalf("expected ErrBadMagic, got %v", err)
	}
}

func TestDecode_RejectsTruncatedData(t *testing.T) {
	frames := []buffer.Frame{solidFrame(time.Now(), color.RGBA{R: 100, G: 100, B: 100, A: 255})}
	encoded, err := Encode(frames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = Decode(encoded[:len(encoded)-5])
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("expected ErrTruncated, got %v", err)
	}
}

func TestDecode_RejectsCorruptJPEGPayload(t *testing.T) {
	frames := []buffer.Frame{solidFrame(time.Now(), color.RGBA{R: 100, G: 100, B: 100, A: 255})}
	encoded, err := Encode(frames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mangle bytes well inside the JPEG payload (after the 8-byte magic,
	// 1-byte version, 4-byte frame count, 8-byte timestamp, and 4-byte
	// length header) without changing the container framing, so Decode
	// gets as far as jpeg.Decode and fails there.
	for i := 30; i < 40 && i < len(encoded); i++ {
		encoded[i] = 0xFF
	}

	_, err = Decode(encoded)
	if err == nil {
		t.Fatal("expected an error decoding a corrupted jpeg payload")
	}
}

func TestDecode_RejectsUnsupportedVersion(t *testing.T) {
	frames := []buffer.Frame{solidFrame(time.Now(), color.RGBA{R: 100, G: 100, B: 100, A: 255})}
	encoded, err := Encode(frames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	encoded[8] = 99 // corrupt the version byte (right after the 8-byte magic)

	_, err = Decode(encoded)
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected ErrUnsupportedVersion, got %v", err)
	}
}
