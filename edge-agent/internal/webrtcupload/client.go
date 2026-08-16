// Package webrtcupload pushes an encoded clip to Media Ingest Gateway over
// a WebRTC data channel instead of a plain HTTP POST — the transport
// docs/architecture.md Section 10 names as an alternative to SRT for this
// leg (see docs/adr/0009 for why WebRTC was built instead of SRT: pure Go,
// already a transitive dependency via pion/mediadevices, no new native
// toolchain risk). Selected via UPLOAD_TRANSPORT=webrtc (internal/config);
// the default HTTP path (internal/uploader) is untouched.
//
// Same "trigger -> push the buffered window" model as the HTTP path — this
// is not continuous streaming. Signaling (the SDP offer/answer exchange)
// is a single HTTP round-trip to media-ingest-gateway's
// POST .../clips/webrtc-offer endpoint; the actual clip bytes then flow
// directly between this process and that one over an ICE-negotiated data
// channel.
package webrtcupload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pion/webrtc/v4"
)

// chunkSize is the size of each DataChannel.Send call. The data channel
// itself guarantees ordering/reliability (default DataChannelInit), so the
// receiver can just concatenate chunks in arrival order — this is purely
// about not handing the SCTP stack one enormous message at once.
const chunkSize = 16 * 1024

// signalTimeout bounds ICE gathering, the signaling HTTP round-trip, and
// waiting for the data channel to open — generous for a same-venue/LAN
// connection, short enough to fail loudly rather than hang the trigger
// handler indefinitely. No STUN/TURN is configured (see the implementation
// plan's "Explicitly deferred"), so this is host-candidates-only.
const signalTimeout = 15 * time.Second

// ackTimeout bounds how long Upload waits for the server's ack after
// sending the last byte — this is what proves the server actually
// received and stored everything before Upload returns.
const ackTimeout = 30 * time.Second

type Client struct {
	httpClient       *http.Client
	signalingBaseURL string
}

func NewClient(signalingBaseURL string) *Client {
	return &Client{httpClient: http.DefaultClient, signalingBaseURL: signalingBaseURL}
}

type ackMessage struct {
	ClipID string `json:"clip_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Upload sends clipBytes to media-ingest-gateway over a WebRTC data
// channel and returns the clip ID it assigns — the same contract as
// internal/uploader.Client.Upload, so cmd/main.go can select either
// transport behind one interface.
func (c *Client) Upload(ctx context.Context, token, orgID, matchID, cameraID string, clipBytes []byte) (string, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return "", fmt.Errorf("webrtcupload: create peer connection: %w", err)
	}
	defer func() { _ = pc.Close() }()

	dc, err := pc.CreateDataChannel("clip", nil)
	if err != nil {
		return "", fmt.Errorf("webrtcupload: create data channel: %w", err)
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("webrtcupload: create offer: %w", err)
	}

	// No trickle ICE: signaling here is a single HTTP round-trip, so we
	// wait for gathering to finish and send a complete offer (all host
	// candidates already included) rather than exchanging candidates
	// incrementally — simpler, and fine for the LAN/localhost connectivity
	// this transport is verified against.
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("webrtcupload: set local description: %w", err)
	}

	signalCtx, cancel := context.WithTimeout(ctx, signalTimeout)
	defer cancel()

	select {
	case <-gatherComplete:
	case <-signalCtx.Done():
		return "", fmt.Errorf("webrtcupload: timed out gathering ICE candidates: %w", signalCtx.Err())
	}

	answer, err := c.signal(signalCtx, token, orgID, matchID, cameraID, len(clipBytes), *pc.LocalDescription())
	if err != nil {
		return "", err
	}

	if err := pc.SetRemoteDescription(answer); err != nil {
		return "", fmt.Errorf("webrtcupload: set remote description: %w", err)
	}

	opened := make(chan struct{})
	dc.OnOpen(func() { close(opened) })

	select {
	case <-opened:
	case <-signalCtx.Done():
		return "", fmt.Errorf("webrtcupload: timed out waiting for data channel to open: %w", signalCtx.Err())
	}

	ackCh := make(chan ackMessage, 1)
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var ack ackMessage
		if jsonErr := json.Unmarshal(msg.Data, &ack); jsonErr == nil {
			ackCh <- ack
		}
	})

	for start := 0; start < len(clipBytes); start += chunkSize {
		end := min(start+chunkSize, len(clipBytes))
		if err := dc.Send(clipBytes[start:end]); err != nil {
			return "", fmt.Errorf("webrtcupload: send chunk: %w", err)
		}
	}

	// Waiting for this ack (rather than closing right after the send loop)
	// is what guarantees the server actually received and stored
	// everything before we tear the connection down — a close-before-flush
	// race would otherwise be possible.
	ackCtx, ackCancel := context.WithTimeout(ctx, ackTimeout)
	defer ackCancel()

	select {
	case ack := <-ackCh:
		if ack.Error != "" {
			return "", fmt.Errorf("webrtcupload: gateway rejected upload: %s", ack.Error)
		}
		if ack.ClipID == "" {
			return "", fmt.Errorf("webrtcupload: gateway ack missing clip id")
		}
		return ack.ClipID, nil
	case <-ackCtx.Done():
		return "", fmt.Errorf("webrtcupload: timed out waiting for gateway ack: %w", ackCtx.Err())
	}
}

// signal POSTs the SDP offer to media-ingest-gateway's signaling endpoint
// and returns the SDP answer — the one HTTP round-trip this transport
// needs; everything after this is direct WebRTC traffic. size lets the
// receiver know exactly how many bytes to expect without inventing an
// in-band EOF sentinel, same role Content-Length plays for the HTTP path.
func (c *Client) signal(ctx context.Context, token, orgID, matchID, cameraID string, size int, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	body, err := json.Marshal(offer)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtcupload: encode offer: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/organizations/%s/matches/%s/clips/webrtc-offer?%s",
		c.signalingBaseURL,
		url.PathEscape(orgID),
		url.PathEscape(matchID),
		url.Values{"camera_id": {cameraID}, "size": {strconv.Itoa(size)}}.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtcupload: build signaling request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtcupload: signaling request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtcupload: gateway rejected signaling (%d)", resp.StatusCode)
	}

	var answer webrtc.SessionDescription
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("webrtcupload: decode answer: %w", err)
	}
	return answer, nil
}
