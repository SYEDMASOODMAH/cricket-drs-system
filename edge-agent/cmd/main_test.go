package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/cricketdrs/edge-agent/internal/config"
	"github.com/cricketdrs/edge-agent/internal/transport"
)

// fakeUploader lets these tests script Upload's behavior across
// successive calls without a real network — this package has no test
// coverage yet at all before this file.
type fakeUploader struct {
	// results[i] is returned on the (i+1)th call; the last entry repeats
	// for any further calls beyond len(results).
	results  []fakeUploadResult
	calls    int
	gotBytes [][]byte
}

type fakeUploadResult struct {
	clipID string
	err    error
}

func (f *fakeUploader) Upload(_ context.Context, _, _, _, _ string, clipBytes []byte) (string, error) {
	f.calls++
	f.gotBytes = append(f.gotBytes, append([]byte(nil), clipBytes...))
	idx := f.calls - 1
	if idx >= len(f.results) {
		idx = len(f.results) - 1
	}
	r := f.results[idx]
	return r.clipID, r.err
}

func testConfig() config.Config {
	return config.Config{
		BearerToken: "test-token",
		OrgID:       "org-1",
		MatchID:     "match-1",
		CameraID:    "cam-1",
	}
}

func TestUploadWithRetry_SucceedsFirstAttempt(t *testing.T) {
	u := &fakeUploader{results: []fakeUploadResult{{clipID: "clip_1"}}}

	clipID, err := uploadWithRetry(context.Background(), u, testConfig(), []byte("clip-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clipID != "clip_1" {
		t.Fatalf("unexpected clip id: %q", clipID)
	}
	if u.calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", u.calls)
	}
}

func TestUploadWithRetry_RejectionShortCircuits(t *testing.T) {
	u := &fakeUploader{results: []fakeUploadResult{
		{err: &transport.RejectedError{StatusCode: http.StatusBadRequest, Message: "camera is not registered"}},
	}}

	_, err := uploadWithRetry(context.Background(), u, testConfig(), []byte("clip-bytes"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if u.calls != 1 {
		t.Fatalf("expected exactly 1 call (a rejection must not be retried), got %d", u.calls)
	}
	var rejected *transport.RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a *transport.RejectedError, got %T", err)
	}
}

func TestUploadWithRetry_TransientFailureThenSuccess(t *testing.T) {
	origBackoff := uploadRetryBackoffForTest()
	defer origBackoff()

	u := &fakeUploader{results: []fakeUploadResult{
		{err: errors.New("network blip")},
		{clipID: "clip_2"},
	}}

	clipID, err := uploadWithRetry(context.Background(), u, testConfig(), []byte("clip-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clipID != "clip_2" {
		t.Fatalf("unexpected clip id: %q", clipID)
	}
	if u.calls != 2 {
		t.Fatalf("expected exactly 2 calls, got %d", u.calls)
	}
	if len(u.gotBytes) != 2 || !bytes.Equal(u.gotBytes[0], u.gotBytes[1]) {
		t.Fatal("expected the retry to resend the exact same clip bytes")
	}
}

func TestUploadWithRetry_TransientFailureExhaustsAttempts(t *testing.T) {
	origBackoff := uploadRetryBackoffForTest()
	defer origBackoff()

	u := &fakeUploader{results: []fakeUploadResult{
		{err: errors.New("network blip")},
	}}

	_, err := uploadWithRetry(context.Background(), u, testConfig(), []byte("clip-bytes"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if u.calls != maxUploadAttempts {
		t.Fatalf("expected exactly %d calls, got %d", maxUploadAttempts, u.calls)
	}
	var rejected *transport.RejectedError
	if errors.As(err, &rejected) {
		t.Fatal("a transient failure must not surface as a RejectedError")
	}
}

// uploadRetryBackoffForTest shortens the package-level retry backoff for
// the duration of a test and returns a func restoring it — these tests
// exercise the real retry loop timing-wise, just fast, rather than
// mocking time.Sleep away entirely.
func uploadRetryBackoffForTest() func() {
	orig := uploadRetryBackoff
	uploadRetryBackoff = 10 * time.Millisecond
	return func() { uploadRetryBackoff = orig }
}
