// Package security adapts the vetted golang-jwt/jwt/v5 library to
// match-tournament's service.TokenVerifier port. This is a verify-only
// counterpart to identity-access's internal/security/jwt.go — this service
// never issues tokens, only validates ones Identity & Access minted. It's
// duplicated rather than shared; see the implementation plan's "Decision
// flagged" note (Go's internal/ visibility rules block importing
// identity-access's package directly, and this is a small, low-risk
// duplication versus a new cross-service shared package on a single
// precedent).
//
// Both services must be started with the same JWT_SIGNING_KEY for tokens
// issued by Identity & Access to verify here — see this service's README.
package security

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
	"github.com/cricketdrs/services/match-tournament/internal/service"
)

// JWTVerifier implements service.TokenVerifier.
type JWTVerifier struct {
	signingKey []byte
}

// NewJWTVerifier builds a verifier from a signing key. Read from the
// environment at process start by cmd/main.go — never committed; see
// identity-access's internal/security/jwt.go for the same secrets-manager
// deferral note.
func NewJWTVerifier(signingKey []byte) *JWTVerifier {
	return &JWTVerifier{signingKey: signingKey}
}

// tokenClaims mirrors identity-access's internal/security tokenClaims
// shape exactly — both sides of the same wire format.
type tokenClaims struct {
	OrganizationID domain.OrganizationID `json:"org"`
	Role           domain.Role           `json:"role"`
	jwt.RegisteredClaims
}

func (v *JWTVerifier) Verify(tokenString string) (service.Claims, error) {
	var c tokenClaims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return v.signingKey, nil
	})
	if err != nil || !token.Valid {
		return service.Claims{}, errors.New("invalid or expired token")
	}
	return service.Claims{
		UserID:         domain.UserID(c.Subject),
		OrganizationID: c.OrganizationID,
		Role:           c.Role,
	}, nil
}
