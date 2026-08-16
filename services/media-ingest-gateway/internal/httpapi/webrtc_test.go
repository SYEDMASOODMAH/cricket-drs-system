package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

func TestUploadClip_ViaWebRTC_RoundTrips(t *testing.T) {
	h := newTestAPI(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	clipBytes := make([]byte, 40*1024) // spans multiple 16KB chunks
	for i := range clipBytes {
		clipBytes[i] = byte(i % 199)
	}

	clipID, ackErr := uploadViaWebRTC(t, srv.URL, orgAAdminToken, "org-a", "match-1", "cam-1", clipBytes)
	if ackErr != "" {
		t.Fatalf("unexpected ack error: %s", ackErr)
	}
	if clipID == "" {
		t.Fatal("expected a non-empty clip id")
	}

	rec := doRaw(t, h, http.MethodGet, "/v1/organizations/org-a/matches/match-1/clips/"+clipID+"/download", orgAAdminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 downloading the clip stored via webrtc, got %d, body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), clipBytes) {
		t.Fatalf("downloaded content does not match what was sent over webrtc (got %d bytes, want %d)", rec.Body.Len(), len(clipBytes))
	}
}

func TestUploadClip_ViaWebRTC_MissingSizeRejected(t *testing.T) {
	h := newTestAPI(t)
	rec := doRaw(t, h, http.MethodPost, "/v1/organizations/org-a/matches/match-1/clips/webrtc-offer?camera_id=cam-1", orgAAdminToken, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a missing size param, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestUploadClip_ViaWebRTC_InvalidOfferRejected(t *testing.T) {
	h := newTestAPI(t)
	rec := doRaw(t, h, http.MethodPost, "/v1/organizations/org-a/matches/match-1/clips/webrtc-offer?camera_id=cam-1&size=10", orgAAdminToken, []byte("not an sdp offer"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a malformed SDP offer, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestUploadClip_ViaWebRTC_PermissionDeniedAckedAsError(t *testing.T) {
	h := newTestAPI(t)
	srv := httptest.NewServer(h)
	defer srv.Close()

	clipBytes := []byte("small clip")
	clipID, ackErr := uploadViaWebRTC(t, srv.URL, orgAPlayerToken, "org-a", "match-1", "cam-1", clipBytes)
	if clipID != "" {
		t.Fatalf("expected no clip id on a rejected upload, got %q", clipID)
	}
	if ackErr == "" {
		t.Fatal("expected the ack to carry an error message for a permission-denied upload")
	}
}

// uploadViaWebRTC drives the real client-side signaling+data-channel flow
// against a running server, exercising the actual handleWebRTCOffer route
// end to end. It mirrors what edge-agent's internal/webrtcupload.Client
// does — that package lives in a separate Go module and can't be imported
// here, so this reimplements the same minimal client logic against the
// real handler rather than mocking anything WebRTC-related. Returns
// (clipID, "") on success or ("", ackErrorMessage) if the server's ack
// carried an error (e.g. a permission-denied UploadClip call) — callers
// decide which of those two outcomes they expect.
func uploadViaWebRTC(t *testing.T, baseURL, token, orgID, matchID, cameraID string, clipBytes []byte) (string, string) {
	t.Helper()

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("client: new peer connection: %v", err)
	}
	defer func() { _ = pc.Close() }()

	dc, err := pc.CreateDataChannel("clip", nil)
	if err != nil {
		t.Fatalf("client: create data channel: %v", err)
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatalf("client: create offer: %v", err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatalf("client: set local description: %v", err)
	}
	<-gatherComplete

	body, err := json.Marshal(pc.LocalDescription())
	if err != nil {
		t.Fatalf("client: encode offer: %v", err)
	}

	endpoint := fmt.Sprintf("%s/v1/organizations/%s/matches/%s/clips/webrtc-offer?camera_id=%s&size=%d",
		baseURL, orgID, matchID, cameraID, len(clipBytes))
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("client: build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("client: signaling request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("client: signaling rejected: %d", resp.StatusCode)
	}

	var answer webrtc.SessionDescription
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		t.Fatalf("client: decode answer: %v", err)
	}
	if err := pc.SetRemoteDescription(answer); err != nil {
		t.Fatalf("client: set remote description: %v", err)
	}

	opened := make(chan struct{})
	dc.OnOpen(func() { close(opened) })
	select {
	case <-opened:
	case <-time.After(10 * time.Second):
		t.Fatal("client: timed out waiting for data channel to open")
	}

	type ackMessage struct {
		ClipID string `json:"clip_id,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	ackCh := make(chan ackMessage, 1)
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var ack ackMessage
		if err := json.Unmarshal(msg.Data, &ack); err == nil {
			ackCh <- ack
		}
	})

	const chunkSize = 16 * 1024
	for start := 0; start < len(clipBytes); start += chunkSize {
		end := min(start+chunkSize, len(clipBytes))
		if err := dc.Send(clipBytes[start:end]); err != nil {
			t.Fatalf("client: send chunk: %v", err)
		}
	}

	select {
	case ack := <-ackCh:
		return ack.ClipID, ack.Error
	case <-time.After(10 * time.Second):
		t.Fatal("client: timed out waiting for ack")
		return "", ""
	}
}
