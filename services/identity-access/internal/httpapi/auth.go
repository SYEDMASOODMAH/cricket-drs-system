package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/cricketdrs/services/identity-access/internal/service"
)

type contextKey int

const callerContextKey contextKey = iota

// requireAuth verifies the bearer token via service.Authenticate and
// injects the resulting service.Caller into the request context. Every
// downstream handler behind this middleware reads the caller from context
// rather than trusting any client-supplied organization/user/role value —
// that's what makes the tenant-isolation checks in the service layer
// meaningful rather than bypassable by a forged request body.
func requireAuth(svc *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, prefix) {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(header, prefix)

			caller, err := svc.Authenticate(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), callerContextKey, caller)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// callerFromContext retrieves the Caller requireAuth injected. It is only
// ever called from handlers mounted behind requireAuth, so a missing value
// is a wiring bug, not a normal request-time condition.
func callerFromContext(ctx context.Context) service.Caller {
	caller, ok := ctx.Value(callerContextKey).(service.Caller)
	if !ok {
		panic("httpapi: callerFromContext called outside requireAuth middleware")
	}
	return caller
}
