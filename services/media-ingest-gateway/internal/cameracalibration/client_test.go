package cameracalibration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsRegistered_FoundReturnsTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected forwarded bearer token, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cam-1"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	registered, err := c.IsRegistered(context.Background(), "test-token", "org-a", "cam-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registered {
		t.Fatal("expected registered=true for a 200 response")
	}
}

func TestIsRegistered_NotFoundIsFalseNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	registered, err := c.IsRegistered(context.Background(), "test-token", "org-a", "cam-unregistered")
	if err != nil {
		t.Fatalf("expected no error for an unregistered camera, got %v", err)
	}
	if registered {
		t.Fatal("expected registered=false")
	}
}

func TestIsRegistered_ServerErrorIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.IsRegistered(context.Background(), "test-token", "org-a", "cam-1")
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

func TestIsRegistered_ForbiddenIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.IsRegistered(context.Background(), "test-token", "org-a", "cam-1")
	if err == nil {
		t.Fatal("expected an error for a 403 response (distinct from 404's not-registered-but-ok)")
	}
}
