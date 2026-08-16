// Package security adapts the shared services/platformauth verifier to
// camera-calibration's service.TokenVerifier port. This service never
// issues tokens, only validates ones Identity & Access minted — see
// docs/adr/0008-platformauth-shared-package.md for why the actual
// verification logic now lives once in platformauth instead of being
// hand-copied per service.
//
// This service, identity-access, match-tournament, and
// media-ingest-gateway must all be started with the same
// JWT_SIGNING_KEY for tokens to verify here — see this service's README.
package security

import (
	"github.com/cricketdrs/services/camera-calibration/internal/domain"
	"github.com/cricketdrs/services/camera-calibration/internal/service"
	"github.com/cricketdrs/services/platformauth"
)

// JWTVerifier implements service.TokenVerifier by converting
// platformauth.Claims (plain strings) into this service's own typed
// domain IDs.
type JWTVerifier struct {
	v *platformauth.Verifier
}

func NewJWTVerifier(signingKey []byte) *JWTVerifier {
	return &JWTVerifier{v: platformauth.NewVerifier(signingKey)}
}

func (j *JWTVerifier) Verify(tokenString string) (service.Claims, error) {
	c, err := j.v.Verify(tokenString)
	if err != nil {
		return service.Claims{}, err
	}
	return service.Claims{
		UserID:         domain.UserID(c.UserID),
		OrganizationID: domain.OrganizationID(c.OrganizationID),
		Role:           c.Role,
	}, nil
}
