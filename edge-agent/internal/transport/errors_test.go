package transport

import "testing"

func TestRejectedError_ErrorWithStatusCode(t *testing.T) {
	err := &RejectedError{StatusCode: 403, Message: "permission denied"}
	got := err.Error()
	want := "upload rejected (403): permission denied"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRejectedError_ErrorWithoutStatusCode(t *testing.T) {
	err := &RejectedError{Message: "camera is not registered"}
	got := err.Error()
	want := "upload rejected: camera is not registered"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
