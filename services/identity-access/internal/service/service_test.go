// Package service_test is an external test package (not "service") so it
// can import internal/security's concrete PasswordHasher/TokenIssuer
// adapters without creating an import cycle (security imports service for
// the Claims/port types).
package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cricketdrs/services/identity-access/internal/domain"
	"github.com/cricketdrs/services/identity-access/internal/memstore"
	"github.com/cricketdrs/services/identity-access/internal/security"
	"github.com/cricketdrs/services/identity-access/internal/service"
)

func newTestService() *service.Service {
	return service.New(
		memstore.NewOrganizationStore(),
		memstore.NewUserStore(),
		memstore.NewConsentStore(),
		security.NewBcryptHasher(),
		security.NewJWTIssuer([]byte("test-signing-key-do-not-use-in-prod")),
	)
}

// bootstrapOrg creates an org and returns it, its admin user, and a Caller
// for that admin — the starting point for most tests below.
func bootstrapOrg(t *testing.T, svc *service.Service, name, adminEmail, adminPassword string) (domain.Organization, domain.User, service.Caller) {
	t.Helper()
	org, admin, err := svc.CreateOrganization(context.Background(), name, adminEmail, adminPassword)
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}
	caller := service.Caller{OrganizationID: org.ID, UserID: admin.ID, Role: admin.Role}
	return org, admin, caller
}

func TestCreateOrganization_CreatesAdminUser(t *testing.T) {
	svc := newTestService()
	org, admin, err := svc.CreateOrganization(context.Background(), "MCA", "admin@mca.example", "hunter2hunter2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.Name != "MCA" {
		t.Fatalf("unexpected org: %+v", org)
	}
	if admin.Role != domain.RoleOrganizerAdmin || admin.OrganizationID != org.ID {
		t.Fatalf("unexpected admin user: %+v", admin)
	}
	if admin.PasswordHash == "hunter2hunter2" {
		t.Fatal("password must be hashed, not stored in plaintext")
	}
}

func TestInviteUser_Success(t *testing.T) {
	svc := newTestService()
	_, _, caller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	u, err := svc.InviteUser(context.Background(), caller, caller.OrganizationID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != domain.RolePlayer || u.OrganizationID != caller.OrganizationID {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestInviteUser_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	_, _, callerA := bootstrapOrg(t, svc, "Org A", "admin-a@example.com", "password123")
	orgB, _, _ := bootstrapOrg(t, svc, "Org B", "admin-b@example.com", "password123")

	_, err := svc.InviteUser(context.Background(), callerA, orgB.ID, "intruder@example.com", "password123", domain.RolePlayer)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestInviteUser_PermissionDenied(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	playerCaller := service.Caller{OrganizationID: org.ID, UserID: player.ID, Role: player.Role}
	_, err = svc.InviteUser(context.Background(), playerCaller, org.ID, "another@mca.example", "password123", domain.RolePlayer)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestInviteUser_DuplicateEmailRejected(t *testing.T) {
	svc := newTestService()
	org, _, caller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	if _, err := svc.InviteUser(context.Background(), caller, org.ID, "dupe@mca.example", "password123", domain.RolePlayer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.InviteUser(context.Background(), caller, org.ID, "dupe@mca.example", "password123", domain.RolePlayer)
	if !errors.Is(err, domain.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestGetUser_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	orgA, adminA, callerA := bootstrapOrg(t, svc, "Org A", "admin-a@example.com", "password123")
	_ = orgA

	orgB, _, _ := bootstrapOrg(t, svc, "Org B", "admin-b@example.com", "password123")

	_, err := svc.GetUser(context.Background(), callerA, orgB.ID, adminA.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestAssignRole_Success(t *testing.T) {
	svc := newTestService()
	org, _, caller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), caller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := svc.AssignRole(context.Background(), caller, org.ID, player.ID, domain.RoleCoach)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Role != domain.RoleCoach {
		t.Fatalf("expected role to change to coach, got %v", updated.Role)
	}
}

func TestAssignRole_InvalidRoleRejected(t *testing.T) {
	svc := newTestService()
	org, admin, caller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	_, err := svc.AssignRole(context.Background(), caller, org.ID, admin.ID, domain.Role("astronaut"))
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestLogin_SuccessAndAuthenticateRoundTrip(t *testing.T) {
	svc := newTestService()
	org, admin, _ := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	token, err := svc.Login(context.Background(), org.ID, "admin@mca.example", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	caller, err := svc.Authenticate(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.UserID != admin.ID || caller.OrganizationID != org.ID || caller.Role != domain.RoleOrganizerAdmin {
		t.Fatalf("unexpected caller from token: %+v", caller)
	}
}

func TestLogin_WrongPasswordRejected(t *testing.T) {
	svc := newTestService()
	org, _, _ := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	_, err := svc.Login(context.Background(), org.ID, "admin@mca.example", "wrong-password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUserRejected(t *testing.T) {
	svc := newTestService()
	org, _, _ := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")

	_, err := svc.Login(context.Background(), org.ID, "nobody@mca.example", "password123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticate_InvalidTokenRejected(t *testing.T) {
	svc := newTestService()
	_, err := svc.Authenticate("not-a-real-token")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestCaptureConsent_SelfConsent(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	playerCaller := service.Caller{OrganizationID: org.ID, UserID: player.ID, Role: player.Role}

	rec, err := svc.CaptureConsent(context.Background(), playerCaller, org.ID, player.ID, false, nil, domain.ConsentVideoCapture, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Grants[domain.ConsentVideoCapture] {
		t.Fatal("expected video-capture consent to be granted")
	}
}

func TestCaptureConsent_UnrelatedUserRejected(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	otherPlayer, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "other@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	otherCaller := service.Caller{OrganizationID: org.ID, UserID: otherPlayer.ID, Role: otherPlayer.Role}

	_, err = svc.CaptureConsent(context.Background(), otherCaller, org.ID, player.ID, false, nil, domain.ConsentVideoCapture, true)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestCaptureConsent_OrganizerCanCaptureOnBehalfOfPlayer(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	player, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "player@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec, err := svc.CaptureConsent(context.Background(), adminCaller, org.ID, player.ID, false, nil, domain.ConsentVideoCapture, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Grants[domain.ConsentVideoCapture] {
		t.Fatal("expected video-capture consent to be granted by organizer on behalf of player")
	}
}

func TestCaptureConsent_MinorWithoutGuardianRejected(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	minor, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "minor@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.CaptureConsent(context.Background(), adminCaller, org.ID, minor.ID, true, nil, domain.ConsentVideoCapture, true)
	if !errors.Is(err, domain.ErrGuardianConsentRequired) {
		t.Fatalf("expected ErrGuardianConsentRequired, got %v", err)
	}
}

func TestCaptureConsent_GuardianCanConsentForMinor(t *testing.T) {
	svc := newTestService()
	org, _, adminCaller := bootstrapOrg(t, svc, "MCA", "admin@mca.example", "password123")
	guardian, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "guardian@mca.example", "password123", domain.RoleCoach)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	minor, err := svc.InviteUser(context.Background(), adminCaller, org.ID, "minor@mca.example", "password123", domain.RolePlayer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	guardianID := guardian.ID
	guardianCaller := service.Caller{OrganizationID: org.ID, UserID: guardian.ID, Role: guardian.Role}

	rec, err := svc.CaptureConsent(context.Background(), guardianCaller, org.ID, minor.ID, true, &guardianID, domain.ConsentVideoCapture, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.IsMinor || rec.GuardianUserID == nil || *rec.GuardianUserID != guardianID {
		t.Fatalf("unexpected consent record: %+v", rec)
	}
}

func TestGetConsent_CrossTenantRejected(t *testing.T) {
	svc := newTestService()
	orgA, adminA, callerA := bootstrapOrg(t, svc, "Org A", "admin-a@example.com", "password123")
	orgB, _, _ := bootstrapOrg(t, svc, "Org B", "admin-b@example.com", "password123")
	_ = orgA

	_, err := svc.GetConsent(context.Background(), callerA, orgB.ID, adminA.ID)
	if !errors.Is(err, domain.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestGetConsent_UnrelatedUserRejected(t *testing.T) {
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
	if _, err := svc.CaptureConsent(context.Background(), adminCaller, org.ID, playerA.ID, false, nil, domain.ConsentVideoCapture, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	callerB := service.Caller{OrganizationID: org.ID, UserID: playerB.ID, Role: playerB.Role}
	_, err = svc.GetConsent(context.Background(), callerB, org.ID, playerA.ID)
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}
