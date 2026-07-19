// Command identity-access is the entrypoint for the identity-access
// service: multi-tenant org/user/RBAC and consent capture (docs/phases.md
// Phase 1). Per rules.md Section 2.2, this is the only place wiring
// happens — building concrete adapters (memstore, security) and injecting
// them into internal/service, which never imports them itself.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cricketdrs/services/identity-access/internal/httpapi"
	"github.com/cricketdrs/services/identity-access/internal/memstore"
	"github.com/cricketdrs/services/identity-access/internal/security"
	"github.com/cricketdrs/services/identity-access/internal/service"
	"github.com/cricketdrs/services/observability"
)

// insecureDevSigningKey is the fallback JWT_SIGNING_KEY used by both
// identity-access and match-tournament when the env var is unset. It must
// be identical in both services' source — see the matching constant and
// comment in match-tournament/cmd/main.go. This exists purely so a
// zero-config single-command dev/preview run (e.g. this repo's
// .claude/launch.json, whose format has no way to inject an env var) has
// the two services able to talk to each other out of the box. It is
// obviously not a secret: it's committed, identical across services, and
// logged loudly on every use. Anything beyond solo local dev/preview
// (shared dev environments, staging, production) must set a real
// JWT_SIGNING_KEY explicitly — see architecture.md Section 15 and this
// service's README.
const insecureDevSigningKey = "INSECURE-DEV-ONLY-SHARED-SIGNING-KEY-DO-NOT-USE-BEYOND-LOCAL-PREVIEW"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	signingKey := jwtSigningKey()

	obs, err := observability.New("identity-access")
	if err != nil {
		slog.Error("identity-access: failed to set up observability", "error", err)
		os.Exit(1)
	}
	// This defer does not currently run in practice: http.ListenAndServe
	// blocks until it returns an error, and the error path below calls
	// os.Exit(1), which skips deferred calls. It's left in the shape a
	// graceful-shutdown implementation (signal.NotifyContext +
	// srv.Shutdown(ctx), no such mechanism exists anywhere in this
	// codebase yet) would need, rather than omitted and re-added later —
	// but that mechanism itself is out of scope here; flushing on exit is
	// a nice-to-have, not a correctness requirement, since a batched
	// stdout exporter losing its last few spans on process exit doesn't
	// lose anything that matters yet (no real trace backend to consume
	// them either).
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(ctx); err != nil {
			slog.Error("identity-access: observability shutdown failed", "error", err)
		}
	}()

	svc := service.New(
		memstore.NewOrganizationStore(),
		memstore.NewUserStore(),
		memstore.NewConsentStore(),
		security.NewBcryptHasher(),
		security.NewJWTIssuer(signingKey),
	)

	api := httpapi.New(svc, obs)

	slog.Info("identity-access starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.Router()); err != nil {
		slog.Error("identity-access server exited", "error", err)
		os.Exit(1)
	}
}

// jwtSigningKey reads JWT_SIGNING_KEY from the environment — never
// committed, and in a deployed environment injected by a secrets manager
// (architecture.md Section 15; wiring that injection is deferred until a
// cloud provider is chosen, see docs/adr/). If unset, falls back to
// insecureDevSigningKey so local development works out of the box per
// Phase 1's completion criteria, and so it matches match-tournament's own
// fallback (see that constant's doc comment) without requiring the env
// var to be set by hand for casual solo preview use.
func jwtSigningKey() []byte {
	if key := os.Getenv("JWT_SIGNING_KEY"); key != "" {
		return []byte(key)
	}
	slog.Warn("JWT_SIGNING_KEY not set; falling back to the shared insecure dev-only signing key — never rely on this outside local solo preview")
	return []byte(insecureDevSigningKey)
}
