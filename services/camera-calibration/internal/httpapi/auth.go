package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/cricketdrs/services/camera-calibration/internal/service"
)

type contextKey int

const callerContextKey contextKey = iota

const bearerPrefix = "Bearer "

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return ""
	}
	return strings.TrimPrefix(header, bearerPrefix)
}

// requireAuth verifies the bearer token via service.Authenticate and
// injects the resulting service.Caller into the request context — same
// pattern as every other service's internal/httpapi/auth.go. Tokens are
// verified locally (internal/security), but were minted by Identity &
// Access; this service and the other three must all share
// JWT_SIGNING_KEY (see this service's README).
func requireAuth(svc *service.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

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

func callerFromContext(ctx context.Context) service.Caller {
	caller, ok := ctx.Value(callerContextKey).(service.Caller)
	if !ok {
		panic("httpapi: callerFromContext called outside requireAuth middleware")
	}
	return caller
}
