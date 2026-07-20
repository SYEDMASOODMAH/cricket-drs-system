// Command media-ingest-gateway is the entrypoint for the Media Ingest
// Gateway service: accepts uploaded match video clips and stores them
// (docs/phases.md Phase 2, first slice — see the implementation plan for
// what's in and out of scope). Per rules.md Section 2.2, this is the only
// place wiring happens.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/httpapi"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/memstore"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/objectstore"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/security"
	"github.com/cricketdrs/services/media-ingest-gateway/internal/service"
	"github.com/cricketdrs/services/observability"
)

// insecureDevSigningKey must be byte-for-byte identical to
// identity-access's and match-tournament's constants of the same name —
// see identity-access/cmd/main.go's doc comment for the full rationale.
const insecureDevSigningKey = "INSECURE-DEV-ONLY-SHARED-SIGNING-KEY-DO-NOT-USE-BEYOND-LOCAL-PREVIEW"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	obs, err := observability.New("media-ingest-gateway")
	if err != nil {
		slog.Error("media-ingest-gateway: failed to set up observability", "error", err)
		os.Exit(1)
	}
	// See identity-access/cmd/main.go's matching comment: this defer does
	// not currently run in practice (no graceful-shutdown mechanism exists
	// yet), left in the right shape for when one is added.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(ctx); err != nil {
			slog.Error("media-ingest-gateway: observability shutdown failed", "error", err)
		}
	}()

	svc := service.New(
		memstore.NewClipStore(),
		buildObjectStore(),
		security.NewJWTVerifier(jwtSigningKey()),
	)

	api := httpapi.New(svc, obs)

	slog.Info("media-ingest-gateway starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.Router()); err != nil {
		slog.Error("media-ingest-gateway server exited", "error", err)
		os.Exit(1)
	}
}

// buildObjectStore uses a real S3 bucket if S3_BUCKET is set, otherwise
// falls back to an in-memory store — the same zero-config-local-default
// shape as jwtSigningKey below. infra/terraform/modules/storage
// provisions the real bucket this would point at, but that Terraform has
// never been applied (no AWS credentials in this environment) — see that
// module's README.
func buildObjectStore() service.ObjectStore {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		slog.Warn("S3_BUCKET not set; using an in-memory object store — uploaded clips will not survive a restart and this must never be relied on in a deployed environment")
		return objectstore.NewMemoryStore()
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("media-ingest-gateway: failed to load AWS config", "error", err)
		os.Exit(1)
	}
	return objectstore.NewS3Store(s3.NewFromConfig(cfg), bucket)
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
