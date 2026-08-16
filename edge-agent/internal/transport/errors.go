// Package transport holds error classification shared by edge-agent's
// upload transports (internal/uploader, internal/webrtcupload) so
// cmd/main.go can tell a definitive rejection apart from a transient
// failure worth retrying, regardless of which transport is active.
package transport

import "fmt"

// RejectedError means the gateway understood and definitively rejected
// the request — retrying with the same bytes will not succeed.
// StatusCode is the HTTP status if the rejection came with one (the plain
// HTTP path, or the WebRTC path's signaling round-trip); 0 for the WebRTC
// path's ack-carried rejections, which have no HTTP status of their own.
type RejectedError struct {
	StatusCode int
	Message    string
}

func (e *RejectedError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("upload rejected (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("upload rejected: %s", e.Message)
}
