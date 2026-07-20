// Command camera-calibration is the entrypoint for the Camera
// Calibration Service: camera/venue registration and calibration-profile
// storage (docs/phases.md Phase 2, second slice — see docs/adr/0005 for
// what's in and out of scope). Per rules.md Section 2.2, this is the only
// place wiring happens.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cricketdrs/services/camera-calibration/internal/httpapi"
	"github.com/cricketdrs/services/camera-calibration/internal/memstore"
	"github.com/cricketdrs/services/camera-calibration/internal/security"
	"github.com/cricketdrs/services/camera-calibration/internal/service"
	"github.com/cricketdrs/services/observability"
)

// insecureDevSigningKey must be byte-for-byte identical to the other
// three services' constants of the same name — see
// identity-access/cmd/main.go's doc comment for the full rationale.
const insecureDevSigningKey = "INSECURE-DEV-ONLY-SHARED-SIGNING-KEY-DO-NOT-USE-BEYOND-LOCAL-PREVIEW"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	obs, err := observability.New("camera-calibration")
	if err != nil {
		slog.Error("camera-calibration: failed to set up observability", "error", err)
		os.Exit(1)
	}
	// See identity-access/cmd/main.go's matching comment: this defer does
	// not currently run in practice (no graceful-shutdown mechanism exists
	// yet), left in the right shape for when one is added.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(ctx); err != nil {
			slog.Error("camera-calibration: observability shutdown failed", "error", err)
		}
	}()

	svc := service.New(
		memstore.NewCameraStore(),
		memstore.NewProfileStore(),
		security.NewJWTVerifier(jwtSigningKey()),
	)

	api := httpapi.New(svc, obs)

	slog.Info("camera-calibration starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.Router()); err != nil {
		slog.Error("camera-calibration server exited", "error", err)
		os.Exit(1)
	}
}

// jwtSigningKey reads JWT_SIGNING_KEY from the environment — never
// committed, injected by a secrets manager in a deployed environment
// (architecture.md Section 15; wiring deferred until real infra exists).
// If unset, falls back to insecureDevSigningKey, which is only useful
// because identity-access falls back to the exact same constant when *it*
// has no JWT_SIGNING_KEY set — an independently-generated random key here
// would never validate anything (this service only verifies, it doesn't
// issue). See this service's README.
func jwtSigningKey() []byte {
	if key := os.Getenv("JWT_SIGNING_KEY"); key != "" {
		return []byte(key)
	}
	slog.Warn("JWT_SIGNING_KEY not set; falling back to the shared insecure dev-only signing key — this only works if identity-access is also using its own fallback (i.e. also has no JWT_SIGNING_KEY set); never rely on this outside local solo preview")
	return []byte(insecureDevSigningKey)
}
