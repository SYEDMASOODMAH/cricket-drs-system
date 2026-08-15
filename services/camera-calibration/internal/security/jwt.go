// Package security adapts the vetted golang-jwt/jwt/v5 library to
// camera-calibration's service.TokenVerifier port. This is a verify-only
// counterpart to identity-access's internal/security/jwt.go — this
// service never issues tokens, only validates ones Identity & Access
// minted. Duplicated rather than shared; this is the 4th instance of this
// exact duplication across the module (see docs/adr/0005 and this
// service's README for why extraction was explicitly deferred rather
// than done in this change).
//
// This service, identity-access, match-tournament, and
// media-ingest-gateway must all be started with the same
// JWT_SIGNING_KEY for tokens to verify here — see this service's README.
package security

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
	"github.com/cricketdrs/services/camera-calibration/internal/service"
)

// JWTVerifier implements service.TokenVerifier.
type JWTVerifier struct {
	signingKey []byte
}

func NewJWTVerifier(signingKey []byte) *JWTVerifier {
	return &JWTVerifier{signingKey: signingKey}
}

// tokenClaims mirrors every other service's internal/security tokenClaims
// shape exactly — all four sides of the same wire format.
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
