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

// insecureDevSigningKey must be byte-for-byte identical to
// identity-access/cmd/main.go's constant of the same name — see that
// file's doc comment for the full rationale (zero-config dev/preview
// interop; obviously not a real secret; anything beyond solo local
// preview must set a real JWT_SIGNING_KEY).
const insecureDevSigningKey = "INSECURE-DEV-ONLY-SHARED-SIGNING-KEY-DO-NOT-USE-BEYOND-LOCAL-PREVIEW"

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
// chosen). If unset, falls back to insecureDevSigningKey, which is only
// useful because identity-access falls back to the exact same constant
// when *it* has no JWT_SIGNING_KEY set — an independently-generated
// random key here would never validate anything (this service only
// verifies, it doesn't issue). This fallback exists for zero-config
// solo local preview only (e.g. this repo's .claude/launch.json, whose
// format can't inject an env var) — anything beyond that (shared dev
// environments, staging, production, or identity-access started with an
// explicit JWT_SIGNING_KEY) must set the real value here too, or every
// request will fail signature verification. See this service's README.
func jwtSigningKey() []byte {
	if key := os.Getenv("JWT_SIGNING_KEY"); key != "" {
		return []byte(key)
	}
	slog.Warn("JWT_SIGNING_KEY not set; falling back to the shared insecure dev-only signing key — this only works if identity-access is also using its own fallback (i.e. also has no JWT_SIGNING_KEY set); never rely on this outside local solo preview")
	return []byte(insecureDevSigningKey)
}
