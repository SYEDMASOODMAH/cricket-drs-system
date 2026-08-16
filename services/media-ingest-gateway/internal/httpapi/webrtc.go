// This file is the receiving side of edge-agent's internal/webrtcupload
// transport (docs/adr/0009). Signaling is a single HTTP round-trip
// (POST .../clips/webrtc-offer): decode the SDP offer, answer it, then
// accumulate the clip bytes edge-agent sends over the resulting data
// channel and hand them to the exact same service.UploadClip call
// handleUploadClip already makes — this adds a second way to deliver bytes
// to that one call, not a second storage path.
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pion/webrtc/v4"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/service"
)

// dataChannelTimeout bounds how long a signaled connection waits for the
// client to finish sending its declared byte count — generous for a
// LAN/localhost transfer, short enough that an abandoned connection
// doesn't hang around forever.
const dataChannelTimeout = 30 * time.Second

func (a *API) handleWebRTCOffer(w http.ResponseWriter, r *http.Request) {
	caller := callerFromContext(r.Context())
	token := bearerToken(r)
	orgID := domain.OrganizationID(chi.URLParam(r, "orgID"))
	matchID := domain.MatchID(chi.URLParam(r, "matchID"))
	cameraID := domain.CameraID(r.URL.Query().Get("camera_id"))

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil || size <= 0 {
		writeError(w, http.StatusBadRequest, "size query parameter must be a positive integer")
		return
	}

	var offer webrtc.SessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		writeError(w, http.StatusBadRequest, "invalid SDP offer")
		return
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		slog.Error("httpapi: webrtc: failed to create peer connection", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// The HTTP request (and its context) ends as soon as this handler
	// returns the SDP answer, but the actual clip transfer happens after
	// that, asynchronously, over the data channel — so this callback
	// cannot use r.Context() (net/http cancels it once ServeHTTP returns).
	// A fresh, independently-timed context is used instead.
	//
	// Must run in its own goroutine: pion invokes OnDataChannel on its
	// internal event-dispatch goroutine, and handleClipDataChannel blocks
	// (waiting on incoming messages) — running it inline would block that
	// same goroutine from ever delivering the OnMessage callbacks
	// handleClipDataChannel itself is waiting on, deadlocking the
	// transfer. Same pattern pion's own examples use.
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), dataChannelTimeout)
			defer cancel()
			a.handleClipDataChannel(ctx, pc, dc, caller, token, orgID, matchID, cameraID, size)
		}()
	})

	if err := pc.SetRemoteDescription(offer); err != nil {
		_ = pc.Close()
		slog.Error("httpapi: webrtc: failed to set remote description", "error", err)
		writeError(w, http.StatusBadRequest, "invalid SDP offer")
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		slog.Error("httpapi: webrtc: failed to create answer", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		slog.Error("httpapi: webrtc: failed to set local description", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	<-gatherComplete

	writeJSON(w, http.StatusOK, pc.LocalDescription())
}

// handleClipDataChannel accumulates size bytes from dc, stores them via
// the same UploadClip path handleUploadClip uses, and acks the result
// back over the data channel before closing the connection.
func (a *API) handleClipDataChannel(ctx context.Context, pc *webrtc.PeerConnection, dc *webrtc.DataChannel, caller service.Caller, token string, orgID domain.OrganizationID, matchID domain.MatchID, cameraID domain.CameraID, size int) {
	defer func() { _ = pc.Close() }()

	buf := make([]byte, 0, size)
	done := make(chan struct{})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		buf = append(buf, msg.Data...)
		if len(buf) >= size {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})

	select {
	case <-done:
	case <-ctx.Done():
		slog.Error("httpapi: webrtc: timed out waiting for clip bytes", "received", len(buf), "expected", size)
		return
	}

	clip, err := a.svc.UploadClip(ctx, caller, token, orgID, matchID, cameraID, bytes.NewReader(buf[:size]))
	ack := struct {
		ClipID string `json:"clip_id,omitempty"`
		Error  string `json:"error,omitempty"`
	}{}
	if err != nil {
		slog.Error("httpapi: webrtc: upload failed", "error", err)
		ack.Error = err.Error()
	} else {
		ack.ClipID = string(clip.ID)
	}

	ackBytes, err := json.Marshal(ack)
	if err != nil {
		slog.Error("httpapi: webrtc: failed to encode ack", "error", err)
		return
	}
	if err := dc.Send(ackBytes); err != nil {
		slog.Error("httpapi: webrtc: failed to send ack", "error", err)
		return
	}

	// Best-effort flush window before closing — pion doesn't expose a
	// "confirm the remote received this" primitive for a single small
	// message, and the client independently waits for this ack (with its
	// own timeout) before it closes on its end, which is the real
	// correctness guarantee. This delay just improves the odds the SCTP
	// stack has actually pushed the bytes out before the connection tears
	// down; a known, deliberate simplification for this slice.
	time.Sleep(500 * time.Millisecond)
}
