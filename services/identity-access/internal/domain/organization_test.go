package domain

import (
	"errors"
	"testing"
)

func TestNewOrganization(t *testing.T) {
	org, err := NewOrganization("org-1", "Montreal Cricket Association")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.ID != "org-1" || org.Name != "Montreal Cricket Association" {
		t.Fatalf("unexpected organization: %+v", org)
	}
}

func TestNewOrganization_EmptyName(t *testing.T) {
	_, err := NewOrganization("org-1", "")
	if !errors.Is(err, ErrOrganizationNameEmpty) {
		t.Fatalf("expected ErrOrganizationNameEmpty, got %v", err)
	}
}
