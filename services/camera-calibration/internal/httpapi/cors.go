package httpapi

import "net/http"

// corsMiddleware allows any browser origin to call this API. This is
// dev-only and deliberately permissive — there's no production frontend
// deployment or CORS policy decided yet (see architecture.md Section 15's
// broader deferred security hardening); it exists purely so apps/web-app's
// Vite dev server (a different origin than this service's own) can call it
// during local development. Not extracted to a shared package: only two
// services need this so far — same "duplicate until a third real consumer"
// bias platformauth's own extraction validated for this codebase.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
