package platformauth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signToken builds a token exactly as Issuer.Issue would, without going
// through it — used to exercise Verifier's error paths independently of
// Issuer.
func signToken(t *testing.T, signingKey []byte, userID, orgID string, role Role, expiresAt time.Time) string {
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

func TestVerifier_ValidToken(t *testing.T) {
	key := []byte("shared-signing-key")
	token := signToken(t, key, "user-1", "org-1", RoleOrganizerAdmin, time.Now().Add(time.Hour))

	v := NewVerifier(key)
	claims, err := v.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-1" || claims.OrganizationID != "org-1" || claims.Role != RoleOrganizerAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifier_ExpiredTokenRejected(t *testing.T) {
	key := []byte("shared-signing-key")
	token := signToken(t, key, "user-1", "org-1", RoleOrganizerAdmin, time.Now().Add(-time.Hour))

	v := NewVerifier(key)
	if _, err := v.Verify(token); err == nil {
		t.Fatal("expected an expired token to be rejected")
	}
}

func TestVerifier_WrongKeyRejected(t *testing.T) {
	token := signToken(t, []byte("key-one"), "user-1", "org-1", RoleOrganizerAdmin, time.Now().Add(time.Hour))

	v := NewVerifier([]byte("key-two"))
	if _, err := v.Verify(token); err == nil {
		t.Fatal("expected verification to fail with a different signing key")
	}
}

func TestVerifier_GarbageRejected(t *testing.T) {
	v := NewVerifier([]byte("shared-signing-key"))
	if _, err := v.Verify("not-a-jwt"); err == nil {
		t.Fatal("expected garbage input to be rejected")
	}
}
