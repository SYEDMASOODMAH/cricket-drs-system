// Package clipformat encodes a buffered sequence of frames
// (internal/buffer.Frame) into bytes suitable for uploading to Media
// Ingest Gateway, and decodes them back.
//
// This is deliberately not a real video container — no H.264/mp4 muxing,
// just a length-prefixed sequence of JPEG images (stdlib image/jpeg
// only, no codec dependency). It exists to prove the capture → buffer →
// upload → storage → retrieval mechanism round-trips real frame content
// correctly; a production-grade encoded format is future work (see
// edge-agent's README "Known simplifications").
package clipformat

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"time"

	"github.com/cricketdrs/edge-agent/internal/buffer"
)

// magic identifies this container format at the start of every encoded
// clip, so a malformed or unrelated byte blob is rejected early rather
// than partially decoded.
var magic = [8]byte{'C', 'D', 'R', 'S', 'C', 'L', 'I', 'P'}

const formatVersion = 1

var (
	ErrBadMagic           = errors.New("clipformat: not a recognized clip container (bad magic bytes)")
	ErrUnsupportedVersion = errors.New("clipformat: unsupported container version")
	ErrTruncated          = errors.New("clipformat: truncated or corrupt clip data")
)

// jpegQuality trades size for fidelity — 85 is a conventional
// "visually good, meaningfully smaller than 100" default; not tuned
// against any real accuracy requirement yet.
const jpegQuality = 85

// Encode serializes frames into this package's container format, oldest
// frame first (the order buffer.RingBuffer.Snapshot already returns them
// in).
func Encode(frames []buffer.Frame) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(magic[:])
	if err := binary.Write(&buf, binary.BigEndian, uint8(formatVersion)); err != nil {
		return nil, fmt.Errorf("clipformat: write version: %w", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(frames))); err != nil {
		return nil, fmt.Errorf("clipformat: write frame count: %w", err)
	}

	for i, frame := range frames {
		if err := binary.Write(&buf, binary.BigEndian, frame.CapturedAt.UnixNano()); err != nil {
			return nil, fmt.Errorf("clipformat: write frame %d timestamp: %w", i, err)
		}

		var jpegBuf bytes.Buffer
		if err := jpeg.Encode(&jpegBuf, frame.Image, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, fmt.Errorf("clipformat: jpeg-encode frame %d: %w", i, err)
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(jpegBuf.Len())); err != nil {
			return nil, fmt.Errorf("clipformat: write frame %d length: %w", i, err)
		}
		buf.Write(jpegBuf.Bytes())
	}

	return buf.Bytes(), nil
}

// Decode parses this package's container format back into frames.
func Decode(data []byte) ([]buffer.Frame, error) {
	r := bytes.NewReader(data)

	var gotMagic [8]byte
	if _, err := r.Read(gotMagic[:]); err != nil || gotMagic != magic {
		return nil, ErrBadMagic
	}

	var version uint8
	if err := binary.Read(r, binary.BigEndian, &version); err != nil {
		return nil, ErrTruncated
	}
	if version != formatVersion {
		return nil, ErrUnsupportedVersion
	}

	var frameCount uint32
	if err := binary.Read(r, binary.BigEndian, &frameCount); err != nil {
		return nil, ErrTruncated
	}

	frames := make([]buffer.Frame, 0, frameCount)
	for i := uint32(0); i < frameCount; i++ {
		var capturedAtNano int64
		if err := binary.Read(r, binary.BigEndian, &capturedAtNano); err != nil {
			return nil, fmt.Errorf("%w: frame %d timestamp", ErrTruncated, i)
		}

		var jpegLen uint32
		if err := binary.Read(r, binary.BigEndian, &jpegLen); err != nil {
			return nil, fmt.Errorf("%w: frame %d length", ErrTruncated, i)
		}

		jpegBytes := make([]byte, jpegLen)
		if _, err := io.ReadFull(r, jpegBytes); err != nil {
			return nil, fmt.Errorf("%w: frame %d data", ErrTruncated, i)
		}

		img, err := jpeg.Decode(bytes.NewReader(jpegBytes))
		if err != nil {
			return nil, fmt.Errorf("clipformat: decode frame %d jpeg: %w", i, err)
		}

		frames = append(frames, buffer.Frame{
			Image:      img,
			CapturedAt: time.Unix(0, capturedAtNano),
		})
	}

	return frames, nil
}
