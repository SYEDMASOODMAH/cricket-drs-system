package identityaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsEligibleForDRS_BothGrantsPresent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected forwarded bearer token, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"grants":{"video_capture":true,"ai_analysis":true}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	eligible, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !eligible {
		t.Fatal("expected eligible=true when both grants are present")
	}
}

func TestIsEligibleForDRS_OnlyOneGrantPresent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"grants":{"video_capture":true}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	eligible, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eligible {
		t.Fatal("expected eligible=false when only one of the two required grants is present")
	}
}

func TestIsEligibleForDRS_NotFoundIsFalseNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	eligible, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err != nil {
		t.Fatalf("expected no error for a not-yet-created consent record, got %v", err)
	}
	if eligible {
		t.Fatal("expected eligible=false")
	}
}

func TestIsEligibleForDRS_ServerErrorIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}

func TestIsEligibleForDRS_MalformedJSONIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err == nil {
		t.Fatal("expected an error for a malformed response body")
	}
}

func TestIsEligibleForDRS_ForbiddenIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, nil)
	_, err := c.IsEligibleForDRS(context.Background(), "test-token", "user-1")
	if err == nil {
		t.Fatal("expected an error for a 403 response (distinct from 404's not-eligible-but-ok)")
	}
}
