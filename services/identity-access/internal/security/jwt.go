package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cricketdrs/services/identity-access/internal/domain"
	"github.com/cricketdrs/services/identity-access/internal/service"
)

// TokenTTL is the access-token lifetime. Short-lived per architecture.md
// Section 15's session-token guidance, applied service-wide since Phase 1
// has only one token type.
const TokenTTL = 15 * time.Minute

// JWTIssuer implements service.TokenIssuer.
type JWTIssuer struct {
	signingKey []byte
}

// NewJWTIssuer builds an issuer from a signing key. The key is read from
// the environment at process start by cmd/main.go — never committed —
// with the intent that a deployed environment injects it from a secrets
// manager (architecture.md Section 15); wiring that injection is deferred
// until a cloud provider is chosen.
func NewJWTIssuer(signingKey []byte) *JWTIssuer {
	return &JWTIssuer{signingKey: signingKey}
}

type tokenClaims struct {
	OrganizationID domain.OrganizationID `json:"org"`
	Role           domain.Role           `json:"role"`
	jwt.RegisteredClaims
}

func (j *JWTIssuer) Issue(userID domain.UserID, orgID domain.OrganizationID, role domain.Role) (string, error) {
	now := time.Now()
	c := tokenClaims{
		OrganizationID: orgID,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   string(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(j.signingKey)
}

func (j *JWTIssuer) Verify(tokenString string) (service.Claims, error) {
	var c tokenClaims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.signingKey, nil
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
