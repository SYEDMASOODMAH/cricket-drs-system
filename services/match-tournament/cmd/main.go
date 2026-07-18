// Command match-tournament is the entrypoint for the match-tournament
// service: tournaments, teams, matches, playing conditions, and
// consent-gated roster management (docs/phases.md Phase 1). Per rules.md
// Section 2.2, this is the only place wiring happens.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/cricketdrs/services/match-tournament/internal/httpapi"
	"github.com/cricketdrs/services/match-tournament/internal/identityaccess"
	"github.com/cricketdrs/services/match-tournament/internal/memstore"
	"github.com/cricketdrs/services/match-tournament/internal/security"
	"github.com/cricketdrs/services/match-tournament/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	identityAccessURL := os.Getenv("IDENTITY_ACCESS_URL")
	if identityAccessURL == "" {
		identityAccessURL = "http://localhost:8080"
	}

	svc := service.New(
		memstore.NewTournamentStore(),
		memstore.NewTeamStore(),
		memstore.NewMatchStore(),
		memstore.NewRosterStore(),
		security.NewJWTVerifier(jwtSigningKey()),
		identityaccess.NewClient(identityAccessURL),
	)

	api := httpapi.New(svc)

	slog.Info("match-tournament starting", "port", port, "identity_access_url", identityAccessURL)
	if err := http.ListenAndServe(":"+port, api.Router()); err != nil {
		slog.Error("match-tournament server exited", "error", err)
		os.Exit(1)
	}
}

// jwtSigningKey reads JWT_SIGNING_KEY from the environment — never
// committed, injected by a secrets manager in a deployed environment
// (architecture.md Section 15; wiring deferred until a cloud provider is
// chosen). Unlike identity-access, this service does NOT fall back to a
// random ephemeral key when unset: identity-access issues the tokens this
// service verifies, so an independently-generated random key here would
// never validate anything and would silently fail every request rather
// than being a harmless dev convenience. Both services must be started
// with the same value — see this service's README.
func jwtSigningKey() []byte {
	key := os.Getenv("JWT_SIGNING_KEY")
	if key == "" {
		slog.Error("JWT_SIGNING_KEY is required and must match the value identity-access was started with — this service only verifies tokens, it cannot generate a usable fallback key")
		os.Exit(1)
	}
	return []byte(key)
}
