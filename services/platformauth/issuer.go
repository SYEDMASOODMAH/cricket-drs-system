package platformauth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer mints and verifies tokens — only identity-access constructs one of
// these; every other service only ever verifies tokens identity-access
// minted, via Verifier. Issue and Verify share the same tokenClaims wire
// struct on purpose: keeping them in one package/file is what prevents an
// issuer and a verifier from silently drifting apart on the wire format.
type Issuer struct {
	signingKey []byte
	verifier   *Verifier
}

// NewIssuer builds an issuer from a signing key. The key is read from the
// environment at process start by identity-access's cmd/main.go — never
// committed — with the intent that a deployed environment injects it from
// a secrets manager (architecture.md Section 15).
func NewIssuer(signingKey []byte) *Issuer {
	return &Issuer{signingKey: signingKey, verifier: NewVerifier(signingKey)}
}

// Issue mints a token for userID/orgID/role, expiring after ttl.
// identity-access supplies its own TokenTTL policy constant here rather
// than this package hardcoding one — token lifetime is a business
// decision, not shared auth plumbing.
func (i *Issuer) Issue(userID, orgID string, role Role, ttl time.Duration) (string, error) {
	now := time.Now()
	c := tokenClaims{
		OrganizationID: orgID,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(i.signingKey)
}

// Verify delegates to an internally-held Verifier — identity-access needs
// this too, to authenticate incoming requests to itself, not just to mint
// tokens for others.
func (i *Issuer) Verify(tokenString string) (Claims, error) {
	return i.verifier.Verify(tokenString)
}
