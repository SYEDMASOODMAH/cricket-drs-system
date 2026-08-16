package webrtcupload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/cricketdrs/edge-agent/internal/transport"
)

// newTestSignalingServer stands in for media-ingest-gateway's real
// webrtc-offer handler, using the identical algorithm (answer the offer,
// accumulate `size` bytes off the resulting data channel, ack) — this
// proves Client.Upload works against a real WebRTC peer connection and a
// real data channel, not a mock of the transport itself. That production
// handler lives in a separate Go module (services/media-ingest-gateway)
// and can't be imported here, so this is a from-scratch but behaviorally
// equivalent stand-in.
func newTestSignalingServer(t *testing.T, received *[]byte, wantClipID string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/organizations/org-1/matches/match-1/clips/webrtc-offer", func(w http.ResponseWriter, r *http.Request) {
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil {
			t.Errorf("server: invalid size param: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var offer webrtc.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			t.Errorf("server: decode offer: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			t.Fatalf("server: new peer connection: %v", err)
		}

		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			var buf []byte
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				buf = append(buf, msg.Data...)
				if len(buf) >= size {
					*received = buf
					ack, _ := json.Marshal(ackMessage{ClipID: wantClipID})
					_ = dc.Send(ack)
				}
			})
		})

		if err := pc.SetRemoteDescription(offer); err != nil {
			t.Fatalf("server: set remote description: %v", err)
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			t.Fatalf("server: create answer: %v", err)
		}
		gatherComplete := webrtc.GatheringCompletePromise(pc)
		if err := pc.SetLocalDescription(answer); err != nil {
			t.Fatalf("server: set local description: %v", err)
		}
		<-gatherComplete

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pc.LocalDescription())
	})
	return httptest.NewServer(mux)
}

func TestUpload_SendsClipAndReturnsAckedClipID(t *testing.T) {
	clipBytes := make([]byte, 40*1024) // spans multiple chunkSize (16KB) sends
	for i := range clipBytes {
		clipBytes[i] = byte(i % 251)
	}

	var received []byte
	srv := newTestSignalingServer(t, &received, "clip_test123")
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clipID, err := c.Upload(ctx, "test-token", "org-1", "match-1", "cam-1", clipBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clipID != "clip_test123" {
		t.Fatalf("unexpected clip id: %q", clipID)
	}
	if !bytes.Equal(received, clipBytes) {
		t.Fatalf("received bytes do not match sent bytes (got %d bytes, want %d)", len(received), len(clipBytes))
	}
}

func TestUpload_PropagatesSignalingNetworkError(t *testing.T) {
	// A closed test server's URL is a reliable way to get a real
	// connection-refused error, same pattern internal/uploader's own test
	// uses for its network-error case.
	dead := httptest.NewServer(http.NewServeMux())
	dead.Close()

	c := NewClient(dead.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := c.Upload(ctx, "test-token", "org-1", "match-1", "cam-1", []byte("x")); err == nil {
		t.Fatal("expected an error when the signaling server is unreachable")
	}
}

func TestUpload_PropagatesSignalingRejection(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/organizations/org-1/matches/match-1/clips/webrtc-offer", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Upload(ctx, "test-token", "org-1", "match-1", "cam-1", []byte("x"))
	if err == nil {
		t.Fatal("expected an error when signaling is rejected")
	}

	var rejected *transport.RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a *transport.RejectedError (a signaling rejection should never be retried), got %T", err)
	}
	if rejected.StatusCode != http.StatusForbidden {
		t.Fatalf("expected StatusCode 403, got %d", rejected.StatusCode)
	}
}

// TestUpload_PropagatesAckRejection confirms an application-level
// rejection reported back over the data channel (e.g. UploadClip itself
// returned an error, after a real transfer already completed) is also a
// RejectedError — just without a StatusCode, since there's no HTTP
// response to read one from at that point.
func TestUpload_PropagatesAckRejection(t *testing.T) {
	var received []byte
	srv := newRejectingSignalingServer(t, &received, "camera is not registered")
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Upload(ctx, "test-token", "org-1", "match-1", "cam-1", []byte("small clip"))
	if err == nil {
		t.Fatal("expected an error when the ack carries a rejection")
	}

	var rejected *transport.RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a *transport.RejectedError, got %T", err)
	}
	if rejected.StatusCode != 0 {
		t.Fatalf("expected no StatusCode for an ack-carried rejection, got %d", rejected.StatusCode)
	}
	if rejected.Message != "camera is not registered" {
		t.Fatalf("unexpected message: %q", rejected.Message)
	}
}

// newRejectingSignalingServer is newTestSignalingServer's counterpart for
// the ack-error path: it completes a real signaling+data-channel exchange
// but always acks back an error instead of a clip id, standing in for
// UploadClip itself failing server-side.
func newRejectingSignalingServer(t *testing.T, received *[]byte, ackErrMessage string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/organizations/org-1/matches/match-1/clips/webrtc-offer", func(w http.ResponseWriter, r *http.Request) {
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var offer webrtc.SessionDescription
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			t.Fatalf("server: new peer connection: %v", err)
		}

		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			var buf []byte
			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				buf = append(buf, msg.Data...)
				if len(buf) >= size {
					*received = buf
					ack, _ := json.Marshal(ackMessage{Error: ackErrMessage})
					_ = dc.Send(ack)
				}
			})
		})

		if err := pc.SetRemoteDescription(offer); err != nil {
			t.Fatalf("server: set remote description: %v", err)
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			t.Fatalf("server: create answer: %v", err)
		}
		gatherComplete := webrtc.GatheringCompletePromise(pc)
		if err := pc.SetLocalDescription(answer); err != nil {
			t.Fatalf("server: set local description: %v", err)
		}
		<-gatherComplete

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pc.LocalDescription())
	})
	return httptest.NewServer(mux)
}
