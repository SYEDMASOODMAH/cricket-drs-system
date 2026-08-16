package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware_OptionsPreflightShortCircuits(t *testing.T) {
	called := false
	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/organizations/org-a/cameras", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if called {
		t.Fatal("expected the wrapped handler not to run for an OPTIONS preflight")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCorsMiddleware_NonOptionsPassesThroughWithHeaders(t *testing.T) {
	called := false
	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected the wrapped handler to run for a non-OPTIONS request")
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
