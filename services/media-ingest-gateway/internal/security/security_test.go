package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

// signToken builds a token exactly as identity-access's JWTIssuer would,
// without depending on that service — this is the wire contract all three
// services agree on (see tokenClaims's doc comment).
func signToken(t *testing.T, signingKey []byte, userID string, orgID domain.OrganizationID, role domain.Role, expiresAt time.Time) string {
	t.Helper()
	c := tokenClaims{
		OrganizationID: orgID,
		Role:           role,
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

func TestJWTVerifier_ValidToken(t *testing.T) {
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

func TestJWTVerifier_ExpiredTokenRejected(t *testing.T) {
	key := []byte("shared-signing-key")
	token := signToken(t, key, "user-1", "org-1", domain.RoleOrganizerAdmin, time.Now().Add(-time.Hour))

	v := NewJWTVerifier(key)
	if _, err := v.Verify(token); err == nil {
		t.Fatal("expected an expired token to be rejected")
	}
}

func TestJWTVerifier_WrongKeyRejected(t *testing.T) {
	token := signToken(t, []byte("key-one"), "user-1", "org-1", domain.RoleOrganizerAdmin, time.Now().Add(time.Hour))

	v := NewJWTVerifier([]byte("key-two"))
	if _, err := v.Verify(token); err == nil {
		t.Fatal("expected verification to fail with a different signing key")
	}
}

func TestJWTVerifier_GarbageRejected(t *testing.T) {
	v := NewJWTVerifier([]byte("shared-signing-key"))
	if _, err := v.Verify("not-a-jwt"); err == nil {
		t.Fatal("expected garbage input to be rejected")
	}
}
