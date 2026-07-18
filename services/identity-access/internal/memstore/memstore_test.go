package memstore

import (
	"context"
	"errors"
	"testing"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

func TestOrganizationStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewOrganizationStore()
	org := domain.Organization{ID: "org-1", Name: "MCA"}

	if err := s.Create(ctx, org); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := s.Get(ctx, "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != org {
		t.Fatalf("got %+v, want %+v", got, org)
	}
}

func TestOrganizationStore_GetNotFound(t *testing.T) {
	s := NewOrganizationStore()
	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, domain.ErrOrganizationNotFound) {
		t.Fatalf("expected ErrOrganizationNotFound, got %v", err)
	}
}

func TestUserStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewUserStore()

	userA := domain.User{ID: "u1", OrganizationID: "org-a", Email: "a@example.com", Role: domain.RolePlayer}
	userB := domain.User{ID: "u1", OrganizationID: "org-b", Email: "b@example.com", Role: domain.RolePlayer}

	if err := s.Create(ctx, userA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, userB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same UserID, different OrganizationID, must not collide or leak.
	got, err := s.Get(ctx, "org-a", "u1")
	if err != nil || got.Email != "a@example.com" {
		t.Fatalf("expected org-a's user, got %+v, err %v", got, err)
	}

	if _, err := s.Get(ctx, "org-b", "does-not-exist"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	// GetByEmail must not cross tenants.
	if _, err := s.GetByEmail(ctx, "org-b", "a@example.com"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected org-b lookup of org-a's email to fail, got %v", err)
	}

	listA, err := s.ListByOrganization(ctx, "org-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(listA) != 1 || listA[0].Email != "a@example.com" {
		t.Fatalf("expected only org-a's user in list, got %+v", listA)
	}
}

func TestUserStore_CreateDuplicateFails(t *testing.T) {
	ctx := context.Background()
	s := NewUserStore()
	u := domain.User{ID: "u1", OrganizationID: "org-a", Email: "a@example.com", Role: domain.RolePlayer}
	if err := s.Create(ctx, u); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(ctx, u); !errors.Is(err, domain.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestUserStore_UpdateNotFound(t *testing.T) {
	s := NewUserStore()
	u := domain.User{ID: "ghost", OrganizationID: "org-a", Email: "x@example.com", Role: domain.RolePlayer}
	if err := s.Update(context.Background(), u); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestConsentStore_GetNotFound(t *testing.T) {
	s := NewConsentStore()
	_, err := s.Get(context.Background(), "org-a", "u1")
	if !errors.Is(err, domain.ErrConsentNotFound) {
		t.Fatalf("expected ErrConsentNotFound, got %v", err)
	}
}

func TestConsentStore_UpsertAndGetDoesNotAlias(t *testing.T) {
	ctx := context.Background()
	s := NewConsentStore()
	rec := domain.ConsentRecord{
		UserID:         "u1",
		OrganizationID: "org-a",
		Grants:         map[domain.ConsentType]bool{domain.ConsentVideoCapture: true},
	}
	if err := s.Upsert(ctx, rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "org-a", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mutating the returned map must not affect the store's internal state.
	got.Grants[domain.ConsentAIAnalysis] = true

	got2, err := s.Get(ctx, "org-a", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2.Grants[domain.ConsentAIAnalysis] {
		t.Fatal("mutating a fetched record leaked into the store — Get must return an isolated copy")
	}
}

func TestConsentStore_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewConsentStore()
	recA := domain.ConsentRecord{UserID: "u1", OrganizationID: "org-a", Grants: map[domain.ConsentType]bool{domain.ConsentVideoCapture: true}}
	recB := domain.ConsentRecord{UserID: "u1", OrganizationID: "org-b", Grants: map[domain.ConsentType]bool{}}

	if err := s.Upsert(ctx, recA); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Upsert(ctx, recB); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotB, err := s.Get(ctx, "org-b", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotB.Grants[domain.ConsentVideoCapture] {
		t.Fatal("org-b's consent record leaked org-a's grant")
	}
}
