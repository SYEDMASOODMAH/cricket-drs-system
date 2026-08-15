// Package capture wraps github.com/pion/mediadevices to read frames from
// a USB-C tethered UVC camera (docs/adr/0003, docs/adr/0007). Chosen over
// the docs' original "v4l2/OpenCV" phrasing because gocv's Windows build
// needs a from-source OpenCV build (~1hr+); mediadevices is cross-platform
// (Linux backend is V4L2 — the actual eventual edge-box target — Windows/
// Mac get native backends for dev-time testing) and needs no codec
// libraries for raw frame capture. See edge-agent's README for the full
// rationale, including the Windows cgo/MinGW-w64 build requirement this
// package's driver import pulls in.
package capture

import (
	"fmt"
	"image"
	"image/draw"
	"strings"
	"time"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/io/audio"
	"github.com/pion/mediadevices/pkg/io/video"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/mediadevices/pkg/wave"

	// Registers the platform camera and microphone drivers as a side
	// effect — required by mediadevices' design (there is no default
	// media input registered).
	_ "github.com/pion/mediadevices/pkg/driver/camera"
	_ "github.com/pion/mediadevices/pkg/driver/microphone"

	"github.com/cricketdrs/edge-agent/internal/buffer"
)

// Camera is an open handle to one video source.
type Camera struct {
	track  *mediadevices.VideoTrack
	reader video.Reader
}

// Open starts capturing from a camera matching the given resolution as
// an ideal (not exact) constraint — mediadevices picks the closest match
// a connected device actually supports, which for the confirmed DJI
// Action 5 Pro Webcam Mode is exactly 1920x1080@30 (see docs/adr/0007).
// frameRate is also an ideal target for the same reason.
//
// labelSubstring, if non-empty, pins the device to open by matching it
// against DeviceInfo.Label (case-insensitive substring) — needed in
// practice: a machine can have multiple video-input devices registered
// (a real UVC camera, a virtual/software camera with no live source,
// etc.), and mediadevices' automatic constraint-fitness selection has no
// way to know which one is actually the intended physical camera.
//
// This resolves the match to a DeviceID freshly, on every call, rather
// than accepting a DeviceID directly: mediadevices regenerates each
// device's DeviceID on every EnumerateDevices() call (confirmed
// empirically — it is not a stable hardware identifier across process
// runs), while Label is stable, so Label is the only thing safe to
// persist in config (see internal/config's CameraDeviceLabel).
func Open(width, height int, frameRate float32, labelSubstring string) (*Camera, error) {
	deviceID, err := resolveDeviceID(labelSubstring)
	if err != nil {
		return nil, err
	}

	stream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(c *mediadevices.MediaTrackConstraints) {
			c.Width = prop.Int(width)
			c.Height = prop.Int(height)
			c.FrameRate = prop.Float(frameRate)
			if deviceID != "" {
				c.DeviceID = prop.StringExact(deviceID)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("capture: open camera: %w", err)
	}

	tracks := stream.GetVideoTracks()
	if len(tracks) == 0 {
		return nil, fmt.Errorf("capture: camera opened but produced no video track")
	}
	track, ok := tracks[0].(*mediadevices.VideoTrack)
	if !ok {
		return nil, fmt.Errorf("capture: unexpected video track type %T", tracks[0])
	}

	// true = force the broadcaster to hand back an independent copy per
	// frame — a small extra cost, but ruled out as a factor (tested both
	// ways) in a real stall found against actual hardware: at 1280x720
	// this driver reliably reads exactly one frame and then hangs
	// indefinitely on the second, while 640x480 sustains cleanly at a
	// steady ~30fps. Root-caused to resolution, not this flag — most
	// likely a USB bandwidth ceiling for uncompressed video at 1280x720
	// (see docs/adr/0007's addendum). Left at true since it's the safer
	// default regardless.
	reader := track.NewReader(true)

	return &Camera{track: track, reader: reader}, nil
}

// Read blocks until the next frame is available and returns it as an
// owned buffer.Frame — safe to retain past this call, unlike the
// image.Image mediadevices itself hands back (which its release
// callback may cause to be reused for a subsequent frame).
func (c *Camera) Read() (buffer.Frame, error) {
	img, release, err := c.reader.Read()
	if err != nil {
		return buffer.Frame{}, fmt.Errorf("capture: read frame: %w", err)
	}
	owned := cloneImage(img)
	release()

	return buffer.Frame{Image: owned, CapturedAt: time.Now()}, nil
}

// Close releases the camera device.
func (c *Camera) Close() error {
	return c.track.Close()
}

// cloneImage copies src's pixel data into a new, independently-owned
// image.RGBA.
func cloneImage(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}

// Microphone is an open handle to one audio source.
type Microphone struct {
	track  *mediadevices.AudioTrack
	reader audio.Reader

	// streamStart and samplesRead reconstruct each chunk's true capture
	// time from its position in the audio timeline, rather than trusting
	// time.Now() at the moment Read() happens to return — see Read's doc
	// comment for why that distinction turned out to matter in practice.
	streamStart time.Time
	samplesRead int64
}

// OpenMicrophone starts capturing mono audio from a microphone, at
// sampleRate as an ideal (not exact) constraint — same reasoning as
// Open's resolution/frame-rate constraints (docs/adr/0007's Windows
// sustained-capture lesson: exact constraints on this driver fail more
// often than they should). Unlike Open, no sample format (bit depth,
// float vs int, interleaving) is constrained at all — Read handles
// whatever format the driver actually returns via
// wave.Int16SampleFormat.Convert, so there's nothing to get wrong by
// asking for a format the device doesn't have.
func OpenMicrophone(sampleRate int) (*Microphone, error) {
	stream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Audio: func(c *mediadevices.MediaTrackConstraints) {
			c.ChannelCount = prop.Int(1)
			c.SampleRate = prop.Int(sampleRate)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("capture: open microphone: %w", err)
	}

	tracks := stream.GetAudioTracks()
	if len(tracks) == 0 {
		return nil, fmt.Errorf("capture: microphone opened but produced no audio track")
	}
	track, ok := tracks[0].(*mediadevices.AudioTrack)
	if !ok {
		return nil, fmt.Errorf("capture: unexpected audio track type %T", tracks[0])
	}

	reader := track.NewReader(true)

	return &Microphone{track: track, reader: reader, streamStart: time.Now()}, nil
}

// Read blocks until the next audio chunk is available and returns it as
// an owned, mono buffer.AudioChunk, converting whatever sample format
// the driver actually returned to signed 16-bit PCM via the wave
// package's own conversion (wave.Int16SampleFormat.Convert) — the same
// conversion its Int16Interleaved type uses internally, rather than a
// hand-rolled one that could get the scaling wrong.
//
// Channels are averaged down to mono regardless of how many the device
// actually delivers. OpenMicrophone requests ChannelCount=1 as an ideal,
// not exact, constraint (same reasoning as Open's resolution/frame-rate
// constraints), and found by testing against the real microphone: this
// device delivers 2 channels regardless. The rest of this package
// (AudioChunk, AudioRingBuffer, the WAV export in cmd/main.go) is all
// designed around mono audio — matching find_offset's expected 1-D
// input — so downmixing here, once, keeps that assumption true no matter
// what the hardware actually provides, rather than threading a channel
// count through every consumer.
//
// CapturedAt is derived from streamStart plus this chunk's position in
// the audio timeline (cumulative frames read / sample rate) — not
// time.Now() at the moment Read() returns, so a driver that delivers a
// backlog of already-captured audio in a rapid burst doesn't stamp
// genuinely old audio with near-identical recent timestamps and defeat
// buffer.AudioRingBuffer's time-window pruning.
func (m *Microphone) Read() (buffer.AudioChunk, error) {
	chunk, release, err := m.reader.Read()
	if err != nil {
		return buffer.AudioChunk{}, fmt.Errorf("capture: read audio chunk: %w", err)
	}

	info := chunk.ChunkInfo()
	samples := make([]int16, info.Len)
	for i := 0; i < info.Len; i++ {
		var sum int32
		for ch := 0; ch < info.Channels; ch++ {
			s := wave.Int16SampleFormat.Convert(chunk.At(i, ch)).(wave.Int16Sample)
			sum += int32(s)
		}
		samples[i] = int16(sum / int32(info.Channels))
	}
	release()

	capturedAt := m.streamStart.Add(time.Duration(float64(m.samplesRead) / float64(info.SamplingRate) * float64(time.Second)))
	m.samplesRead += int64(info.Len)

	return buffer.AudioChunk{Samples: samples, SampleRate: info.SamplingRate, CapturedAt: capturedAt}, nil
}

// Close releases the microphone device.
func (m *Microphone) Close() error {
	return m.track.Close()
}

// DeviceInfo is one enumerated video input device.
type DeviceInfo struct {
	DeviceID string
	Label    string
}

// ListDevices returns currently available video input devices — used for
// startup diagnostics (cmd/main.go logs this so a misconfigured or
// disconnected camera is obvious immediately) and, via Label, for
// pinning Open to a specific device (see Open's doc comment) when a
// machine has more than one candidate (e.g. a real UVC camera alongside a
// virtual/software camera).
func ListDevices() []DeviceInfo {
	var devices []DeviceInfo
	for _, d := range mediadevices.EnumerateDevices() {
		if d.Kind == mediadevices.VideoInput {
			devices = append(devices, DeviceInfo{DeviceID: d.DeviceID, Label: d.Label})
		}
	}
	return devices
}

// resolveDeviceID finds the current DeviceID of the video input device
// whose Label contains labelSubstring (case-insensitive). Returns ("",
// nil) if labelSubstring is empty, leaving device selection automatic.
func resolveDeviceID(labelSubstring string) (string, error) {
	if labelSubstring == "" {
		return "", nil
	}
	needle := strings.ToLower(labelSubstring)
	for _, d := range ListDevices() {
		if strings.Contains(strings.ToLower(d.Label), needle) {
			return d.DeviceID, nil
		}
	}
	return "", fmt.Errorf("capture: no video device found with label containing %q", labelSubstring)
}
