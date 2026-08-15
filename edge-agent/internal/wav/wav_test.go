package wav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncode_HeaderFieldsMatchSpec(t *testing.T) {
	samples := []int16{1, 2, 3, 4}
	data := Encode(samples, 8000, 1)

	const headerLen = 44
	dataLen := len(samples) * 2
	if len(data) != headerLen+dataLen {
		t.Fatalf("expected total length %d, got %d", headerLen+dataLen, len(data))
	}

	if string(data[0:4]) != "RIFF" {
		t.Errorf("expected RIFF tag, got %q", data[0:4])
	}
	if got := binary.LittleEndian.Uint32(data[4:8]); got != uint32(36+dataLen) {
		t.Errorf("expected RIFF chunk size %d, got %d", 36+dataLen, got)
	}
	if string(data[8:12]) != "WAVE" {
		t.Errorf("expected WAVE tag, got %q", data[8:12])
	}
	if string(data[12:16]) != "fmt " {
		t.Errorf("expected 'fmt ' tag, got %q", data[12:16])
	}
	if got := binary.LittleEndian.Uint32(data[16:20]); got != 16 {
		t.Errorf("expected fmt subchunk size 16, got %d", got)
	}
	if got := binary.LittleEndian.Uint16(data[20:22]); got != 1 {
		t.Errorf("expected PCM audio format 1, got %d", got)
	}
	if got := binary.LittleEndian.Uint16(data[22:24]); got != 1 {
		t.Errorf("expected 1 channel, got %d", got)
	}
	if got := binary.LittleEndian.Uint32(data[24:28]); got != 8000 {
		t.Errorf("expected sample rate 8000, got %d", got)
	}
	wantByteRate := uint32(8000 * 1 * 16 / 8)
	if got := binary.LittleEndian.Uint32(data[28:32]); got != wantByteRate {
		t.Errorf("expected byte rate %d, got %d", wantByteRate, got)
	}
	if got := binary.LittleEndian.Uint16(data[32:34]); got != 2 {
		t.Errorf("expected block align 2, got %d", got)
	}
	if got := binary.LittleEndian.Uint16(data[34:36]); got != 16 {
		t.Errorf("expected 16 bits per sample, got %d", got)
	}
	if string(data[36:40]) != "data" {
		t.Errorf("expected 'data' tag, got %q", data[36:40])
	}
	if got := binary.LittleEndian.Uint32(data[40:44]); got != uint32(dataLen) {
		t.Errorf("expected data subchunk size %d, got %d", dataLen, got)
	}
}

func TestEncode_SampleDataRoundTrips(t *testing.T) {
	samples := []int16{-32768, -1, 0, 1, 32767}
	data := Encode(samples, 44100, 1)

	got := make([]int16, len(samples))
	if err := binary.Read(bytes.NewReader(data[44:]), binary.LittleEndian, &got); err != nil {
		t.Fatalf("unexpected error reading back sample data: %v", err)
	}
	for i, want := range samples {
		if got[i] != want {
			t.Errorf("sample %d: got %d, want %d", i, got[i], want)
		}
	}
}

func TestEncode_StereoBlockAlignAndByteRate(t *testing.T) {
	samples := []int16{1, 2, 3, 4} // 2 frames of 2 channels
	data := Encode(samples, 48000, 2)

	if got := binary.LittleEndian.Uint16(data[22:24]); got != 2 {
		t.Errorf("expected 2 channels, got %d", got)
	}
	wantBlockAlign := uint16(2 * 16 / 8)
	if got := binary.LittleEndian.Uint16(data[32:34]); got != wantBlockAlign {
		t.Errorf("expected block align %d, got %d", wantBlockAlign, got)
	}
	wantByteRate := uint32(48000 * 2 * 16 / 8)
	if got := binary.LittleEndian.Uint32(data[28:32]); got != wantByteRate {
		t.Errorf("expected byte rate %d, got %d", wantByteRate, got)
	}
}

func TestEncode_EmptySamples(t *testing.T) {
	data := Encode(nil, 8000, 1)
	if len(data) != 44 {
		t.Fatalf("expected exactly a 44-byte header for empty samples, got %d bytes", len(data))
	}
	if got := binary.LittleEndian.Uint32(data[40:44]); got != 0 {
		t.Errorf("expected data subchunk size 0, got %d", got)
	}
}
