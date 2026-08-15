package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

// Service orchestrates media-ingest-gateway's use-cases. Constructed once
// in cmd/main.go (rules.md Section 2.2), holding only interfaces so it is
// unit-testable against fakes with no database, object store, HTTP
// server, or real JWT library involved.
type Service struct {
	clips   ClipRepository
	objects ObjectStore
	tokens  TokenVerifier
	now     Clock
}

func New(clips ClipRepository, objects ObjectStore, tokens TokenVerifier) *Service {
	return &Service{
		clips:   clips,
		objects: objects,
		tokens:  tokens,
		now:     time.Now,
	}
}

// Authenticate verifies a bearer token and returns the Caller it proves.
// httpapi's auth middleware calls this so it never needs to import the JWT
// library directly.
func (s *Service) Authenticate(token string) (Caller, error) {
	claims, err := s.tokens.Verify(token)
	if err != nil {
		return Caller{}, domain.ErrInvalidToken
	}
	return Caller{
		OrganizationID: claims.OrganizationID,
		UserID:         claims.UserID,
		Role:           claims.Role,
	}, nil
}

// newID generates a random, unique-enough opaque identifier — same
// approach as identity-access's and match-tournament's
// internal/service/service.go.
func newID(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("service: crypto/rand unavailable: " + err.Error())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
