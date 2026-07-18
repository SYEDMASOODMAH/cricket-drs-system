package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cricketdrs/services/identity-access/internal/domain"
	"github.com/cricketdrs/services/identity-access/internal/service"
)

func TestGetUser_Success(t *testing.T) {
	svc := newTestService()
	org, admin, caller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	got, err := svc.GetUser(context.Background(), caller, org.ID, admin.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != admin.ID {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestAssignRole_PermissionDenied(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	playerCaller := service.Caller{OrganizationID: org.ID, UserID: player.ID, Role: player.Role}

	_, err = svc.AssignRole(context.Background(), playerCaller, org.ID, player.ID, domain.RoleCoach)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestAssignRole_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, adminA, callerA := bootstrapOrg(t, svc, "Org A", "admin-a@example.com", "password123")
	orgB, _, _ := bootstrapOrg(t, svc, "Org B", "admin-b@example.com", "password123")

	_, err := svc.AssignRole(context.Background(), callerA, orgB.ID, adminA.ID, domain.RoleCoach)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestGetConsent_SelfCanReadOwnRecordEvenBeforeAnyGrant(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	playerCaller := service.Caller{OrganizationID: org.ID, UserID: player.ID, Role: player.Role}

	_, err = svc.GetConsent(context.Background(), playerCaller, org.ID, player.ID)
	if !errors.Is(err, domain.ErrConsentNotFound) {
		t.Fatalf("expected ErrConsentNotFound (not permission-denied) for own not-yet-created record, got %v", err)
	}
}

func TestGetConsent_UnrelatedUserCannotProbeNotFoundRecord(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	playerA, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "a@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	playerB, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "b@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	callerB := service.Caller{OrganizationID: org.ID, UserID: playerB.ID, Role: playerB.Role}

	_, err = svc.GetConsent(context.Background(), callerB, org.ID, playerA.ID)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied (must not leak whether a record exists), got %v", err)
	}
}

func TestGetConsent_BoardAdminCanRead(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.CaptureConsent(context.Background(), adminCaller, org.ID, player.ID, false, nil, domain.ConsentVideoCapture, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	board, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "board@mca.example", "password123", domain.RoleBoardAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	boardCaller := service.Caller{OrganizationID: org.ID, UserID: board.ID, Role: board.Role}

	rec, err := svc.GetConsent(context.Background(), boardCaller, org.ID, player.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Grants[domain.ConsentVideoCapture] {
		t.Fatal("expected board admin to see the player's granted consent")
	}
}

func TestCaptureConsent_RevokeIsPersisted(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	playerCaller := service.Caller{OrganizationID: org.ID, UserID: player.ID, Role: player.Role}

	if _, err := svc.CaptureConsent(context.Background(), playerCaller, org.ID, player.ID, false, nil, domain.ConsentAIAnalysis, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec, err := svc.CaptureConsent(context.Background(), playerCaller, org.ID, player.ID, false, nil, domain.ConsentAIAnalysis, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Grants[domain.ConsentAIAnalysis] {
		t.Fatal("expected ai-analysis consent to be revoked")
	}
}

func TestCaptureConsent_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, adminA, callerA := bootstrapOrg(t, svc, "Org A", "admin-a@example.com", "password123")
	orgB, _, _ := bootstrapOrg(t, svc, "Org B", "admin-b@example.com", "password123")

	_, err := svc.CaptureConsent(context.Background(), callerA, orgB.ID, adminA.ID, false, nil, domain.ConsentVideoCapture, true)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}
