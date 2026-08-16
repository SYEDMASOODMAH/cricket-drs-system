package platformauth

import (
	"testing"
	"time"
)

func TestIssuer_IssueAndVerify(t *testing.T) {
	issuer := NewIssuer([]byte("test-signing-key"))

	token, err := issuer.Issue("user-1", "org-1", RolePlayer, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-1" || claims.OrganizationID != "org-1" || claims.Role != RolePlayer {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestIssuer_IssueRespectsTTL(t *testing.T) {
	issuer := NewIssuer([]byte("test-signing-key"))

	token, err := issuer.Issue("user-1", "org-1", RolePlayer, -time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := issuer.Verify(token); err == nil {
		t.Fatal("expected a token issued with a negative ttl to already be expired")
	}
}

func TestIssuer_VerifyRejectsWrongKey(t *testing.T) {
	issuer := NewIssuer([]byte("key-one"))
	token, err := issuer.Issue("user-1", "org-1", RolePlayer, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	otherIssuer := NewIssuer([]byte("key-two"))
	if _, err := otherIssuer.Verify(token); err == nil {
		t.Fatal("expected verification to fail with a different signing key")
	}
}

func TestIssuer_VerifyRejectsGarbage(t *testing.T) {
	issuer := NewIssuer([]byte("test-signing-key"))
	if _, err := issuer.Verify("not-a-jwt"); err == nil {
		t.Fatal("expected verification of garbage input to fail")
	}
}
