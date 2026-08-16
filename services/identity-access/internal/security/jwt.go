// JWT issuing/verification now adapts services/platformauth's Issuer
// instead of hand-rolling HS256 parsing here — see
// docs/adr/0008-platformauth-shared-package.md for why Issue/Verify moved
// there while this package's bcrypt hasher (see bcrypt.go) stayed local.
package security

import (
	"time"

	"github.com/cricketdrs/services/identity-access/internal/domain"
	"github.com/cricketdrs/services/identity-access/internal/service"
	"github.com/cricketdrs/services/platformauth"
)

// TokenTTL is the access-token lifetime. Short-lived per architecture.md
// Section 15's session-token guidance, applied service-wide since Phase 1
// has only one token type. Stays local: token lifetime is identity-access's
// own policy decision, not shared auth plumbing — platformauth.Issuer.Issue
// takes it as a parameter rather than hardcoding one.
const TokenTTL = 15 * time.Minute

// JWTIssuer implements service.TokenIssuer by converting between this
// service's typed domain IDs and platformauth's plain-string wire format.
type JWTIssuer struct {
	i *platformauth.Issuer
}

// NewJWTIssuer builds an issuer from a signing key. The key is read from
// the environment at process start by cmd/main.go — never committed —
// with the intent that a deployed environment injects it from a secrets
// manager (architecture.md Section 15); wiring that injection is deferred
// until a cloud provider is chosen.
func NewJWTIssuer(signingKey []byte) *JWTIssuer {
	return &JWTIssuer{i: platformauth.NewIssuer(signingKey)}
}

func (j *JWTIssuer) Issue(userID domain.UserID, orgID domain.OrganizationID, role domain.Role) (string, error) {
	return j.i.Issue(string(userID), string(orgID), role, TokenTTL)
}

func (j *JWTIssuer) Verify(tokenString string) (service.Claims, error) {
	c, err := j.i.Verify(tokenString)
	if err != nil {
		return service.Claims{}, err
	}
	return service.Claims{
		UserID:         domain.UserID(c.UserID),
		OrganizationID: domain.OrganizationID(c.OrganizationID),
		Role:           c.Role,
	}, nil
}
