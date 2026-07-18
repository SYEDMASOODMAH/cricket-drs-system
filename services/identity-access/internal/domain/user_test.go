package domain

import (
	"errors"
	"testing"
)

func TestNewUser(t *testing.T) {
	u, err := NewUser("user-1", "org-1", "player@example.com", "hashed", RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "user-1" || u.OrganizationID != "org-1" || u.Email != "player@example.com" || u.Role != RolePlayer {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestNewUser_EmptyEmail(t *testing.T) {
	_, err := NewUser("user-1", "org-1", "", "hashed", RolePlayer)
	if !errors.Is(err, ErrUserEmailEmpty) {
		t.Fatalf("expected ErrUserEmailEmpty, got %v", err)
	}
}

func TestNewUser_InvalidRole(t *testing.T) {
	_, err := NewUser("user-1", "org-1", "player@example.com", "hashed", Role("astronaut"))
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}
