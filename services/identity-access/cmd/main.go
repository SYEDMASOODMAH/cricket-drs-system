// Command identity-access is the entrypoint for the identity-access
// service: multi-tenant org/user/RBAC and consent capture (docs/phases.md
// Phase 1). Per rules.md Section 2.2, this is the only place wiring
// happens — building concrete adapters (memstore, security) and injecting
// them into internal/service, which never imports them itself.
package main

import (
	"crypto/rand"
	"log/slog"
	"net/http"
	"os"

	"github.com/cricketdrs/services/identity-access/internal/httpapi"
	"github.com/cricketdrs/services/identity-access/internal/memstore"
	"github.com/cricketdrs/services/identity-access/internal/security"
	"github.com/cricketdrs/services/identity-access/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	signingKey := jwtSigningKey()

	svc := service.New(
		memstore.NewOrganizationStore(),
		memstore.NewUserStore(),
		memstore.NewConsentStore(),
		security.NewBcryptHasher(),
		security.NewJWTIssuer(signingKey),
	)

	api := httpapi.New(svc)

	slog.Info("identity-access starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.Router()); err != nil {
		slog.Error("identity-access server exited", "error", err)
		os.Exit(1)
	}
}

// jwtSigningKey reads JWT_SIGNING_KEY from the environment — never
// committed, and in a deployed environment injected by a secrets manager
// (architecture.md Section 15; wiring that injection is deferred until a
// cloud provider is chosen, see docs/adr/). If unset, a random ephemeral
// key is generated so local development works out of the box per Phase
// 1's completion criteria — this is explicitly dev-only: tokens won't
// survive a restart, and it must never be relied on outside local dev.
func jwtSigningKey() []byte {
	if key := os.Getenv("JWT_SIGNING_KEY"); key != "" {
		return []byte(key)
	}

	slog.Warn("JWT_SIGNING_KEY not set; generating an ephemeral signing key for local development only — tokens will not survive a restart, and this must never be relied on in a deployed environment")
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		slog.Error("identity-access: crypto/rand unavailable, cannot start", "error", err)
		os.Exit(1)
	}
	return key
}
