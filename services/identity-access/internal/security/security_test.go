package security

import (
	"testing"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	h := NewBcryptHasher()
	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if err := h.Compare(hash, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("expected matching password to compare successfully: %v", err)
	}
	if err := h.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("expected mismatched password to fail comparison")
	}
}

func TestJWTIssuer_IssueAndVerify(t *testing.T) {
	issuer := NewJWTIssuer([]byte("test-signing-key"))

	token, err := issuer.Issue("user-1", "org-1", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-1" || claims.OrganizationID != "org-1" || claims.Role != domain.RolePlayer {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTIssuer_VerifyRejectsWrongKey(t *testing.T) {
	issuer := NewJWTIssuer([]byte("key-one"))
	token, err := issuer.Issue("user-1", "org-1", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	otherIssuer := NewJWTIssuer([]byte("key-two"))
	if _, err := otherIssuer.Verify(token); err == nil {
		t.Fatal("expected verification to fail with a different signing key")
	}
}

func TestJWTIssuer_VerifyRejectsGarbage(t *testing.T) {
	issuer := NewJWTIssuer([]byte("test-signing-key"))
	if _, err := issuer.Verify("not-a-jwt"); err == nil {
		t.Fatal("expected verification of garbage input to fail")
	}
}
