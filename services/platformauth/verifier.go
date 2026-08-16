package platformauth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is what a verified token proves about its holder. OrganizationID
// and UserID are plain strings here (not a service-local named ID type)
// since this package can't import any service's internal/ package — each
// service's own thin security.JWTVerifier adapter converts these into its
// local domain.UserID/OrganizationID.
type Claims struct {
	UserID         string
	OrganizationID string
	Role           Role
}

// tokenClaims is the wire format every issuer and verifier in this module
// agrees on. Single definition now — previously duplicated verbatim across
// 4 services' internal/security/jwt.go files, which is exactly the kind of
// drift risk (issuer and verifier silently disagreeing on a JSON tag) this
// package exists to remove.
type tokenClaims struct {
	OrganizationID string `json:"org"`
	Role           Role   `json:"role"`
	jwt.RegisteredClaims
}

// Verifier implements token verification only — the shape every service
// except identity-access needs (identity-access uses Issuer, which also
// verifies).
type Verifier struct {
	signingKey []byte
}

func NewVerifier(signingKey []byte) *Verifier {
	return &Verifier{signingKey: signingKey}
}

func (v *Verifier) Verify(tokenString string) (Claims, error) {
	var c tokenClaims
	token, err := jwt.ParseWithClaims(tokenString, &c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return v.signingKey, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, errors.New("invalid or expired token")
	}
	return Claims{
		UserID:         c.Subject,
		OrganizationID: c.OrganizationID,
		Role:           c.Role,
	}, nil
}
