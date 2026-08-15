// Package wav writes a minimal canonical WAV (RIFF/fmt/data) file from
// raw 16-bit PCM samples. Chosen as the interop format for exporting
// captured audio (internal/buffer.AudioChunk) because it's a simple,
// fixed, well-documented binary layout — hand-writable with
// encoding/binary and zero new dependencies, and readable by anything
// (Python's stdlib wave module, Audacity, ffmpeg) with zero dependencies
// on that side either. See edge-agent's README for why this was chosen
// over reusing internal/clipformat's bespoke container (that one only
// ever needs to round-trip within this codebase; audio needs to cross
// the Go/Python boundary and ideally be independently inspectable).
package wav

import (
	"bytes"
	"encoding/binary"
)

const bitsPerSample = 16

// Encode serializes mono or multi-channel 16-bit PCM samples
// (interleaved if channels > 1) into a canonical 44-byte-header WAV file.
func Encode(samples []int16, sampleRate, channels int) []byte {
	dataSize := len(samples) * 2 // 2 bytes per int16 sample
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	var buf bytes.Buffer

	// RIFF chunk descriptor
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize)) //nolint:errcheck // bytes.Buffer.Write never errors
	buf.WriteString("WAVE")

	// fmt subchunk
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16)) // subchunk1 size, 16 for PCM
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // audio format, 1 = PCM
	_ = binary.Write(&buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	// data subchunk
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	_ = binary.Write(&buf, binary.LittleEndian, samples)

	return buf.Bytes()
}
