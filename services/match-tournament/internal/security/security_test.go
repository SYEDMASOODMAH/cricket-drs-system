package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

// signToken builds a token exactly as identity-access's issuer would,
// without depending on that service — used to confirm JWTVerifier's
// adapter correctly converts platformauth.Claims into this service's
// typed domain IDs. The verify logic itself (expired/wrong-key/garbage
// rejection) is tested once in services/platformauth, not re-tested here —
// see docs/adr/0008-platformauth-shared-package.md.
func signToken(t *testing.T, signingKey []byte, userID string, orgID domain.OrganizationID, role domain.Role, expiresAt time.Time) string {
	t.Helper()
	c := struct {
		OrganizationID string `json:"org"`
		Role           string `json:"role"`
		jwt.RegisteredClaims
	}{
		OrganizationID: string(orgID),
		Role:           string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(signingKey)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return token
}

func TestJWTVerifier_ConvertsClaimsToDomainTypes(t *testing.T) {
	key := []byte("shared-signing-key")
	token := signToken(t, key, "user-1", "org-1", domain.RoleOrganizerAdmin, time.Now().Add(time.Hour))

	v := NewJWTVerifier(key)
	claims, err := v.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-1" || claims.OrganizationID != "org-1" || claims.Role != domain.RoleOrganizerAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTVerifier_PropagatesVerificationErrors(t *testing.T) {
	v := NewJWTVerifier([]byte("shared-signing-key"))
	if _, err := v.Verify("not-a-jwt"); err == nil {
		t.Fatal("expected garbage input to be rejected")
	}
}
