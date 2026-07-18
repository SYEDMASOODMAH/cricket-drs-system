package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Service orchestrates identity-access's use-cases. It is constructed once
// in cmd/main.go (rules.md Section 2.2: "cmd/ is the only place wiring
// happens") and holds only interfaces, so it is unit-testable against fakes
// with no database, HTTP server, or real crypto library involved.
type Service struct {
	orgs     OrganizationRepository
	users    UserRepository
	consents ConsentRepository
	hasher   PasswordHasher
	tokens   TokenIssuer
	now      Clock
}

func New(orgs OrganizationRepository, users UserRepository, consents ConsentRepository, hasher PasswordHasher, tokens TokenIssuer) *Service {
	return &Service{
		orgs:     orgs,
		users:    users,
		consents: consents,
		hasher:   hasher,
		tokens:   tokens,
		now:      time.Now,
	}
}

// newID generates a random, unique-enough opaque identifier. A dedicated
// UUID dependency isn't justified for Phase 1's need (rules.md Section
// 3.3) — crypto/rand plus hex encoding is standard-library-only and
// sufficient collision resistance for an identifier space this small.
func newID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read failing means the OS entropy source is broken —
		// a genuinely unrecoverable startup-class condition, not normal
		// error-flow control (rules.md Section 4.2).
		panic("service: crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
