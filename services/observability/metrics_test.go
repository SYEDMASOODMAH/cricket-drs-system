package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/cricketdrs/services/observability"
)

func newTestObservability(t *testing.T) *observability.Observability {
	t.Helper()
	o, err := observability.New("test-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return o
}

func scrapeMetrics(t *testing.T, o *observability.Observability) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	o.MetricsHandler().ServeHTTP(rec, req)
	return rec.Body.String()
}

func TestMetricsMiddleware_UsesRoutePatternNotResolvedPath(t *testing.T) {
	o := newTestObservability(t)

	r := chi.NewRouter()
	r.Use(o.MetricsMiddleware)
	r.Get("/v1/organizations/{orgID}/matches/{matchID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/organizations/org-abc/matches/match-123", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := scrapeMetrics(t, o)

	if !strings.Contains(body, `route="/v1/organizations/{orgID}/matches/{matchID}"`) {
		t.Fatalf("expected metrics to be labeled with the route pattern; got:\n%s", body)
	}
	if strings.Contains(body, "org-abc") || strings.Contains(body, "match-123") {
		t.Fatal("metrics must never contain resolved path segments (unbounded cardinality risk)")
	}
}

func TestMetricsMiddleware_RecordsStatusAndMethod(t *testing.T) {
	o := newTestObservability(t)

	r := chi.NewRouter()
	r.Use(o.MetricsMiddleware)
	r.Post("/v1/teams", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/teams", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := scrapeMetrics(t, o)

	if !strings.Contains(body, `method="POST"`) || !strings.Contains(body, `status="201"`) {
		t.Fatalf("expected method=POST and status=201 labels; got:\n%s", body)
	}
}

func TestMetricsMiddleware_UnmatchedRouteDoesNotPanic(t *testing.T) {
	o := newTestObservability(t)

	r := chi.NewRouter()
	r.Use(o.MetricsMiddleware)
	r.Get("/known", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := scrapeMetrics(t, o)
	if !strings.Contains(body, `route="unmatched"`) {
		t.Fatalf("expected an unmatched route to be labeled \"unmatched\"; got:\n%s", body)
	}
}

func TestMetricsHandler_IncludesGoRuntimeCollectors(t *testing.T) {
	o := newTestObservability(t)
	body := scrapeMetrics(t, o)
	if !strings.Contains(body, "go_goroutines") {
		t.Fatalf("expected Go runtime collector metrics (saturation signal) to be present; got:\n%s", body)
	}
}
