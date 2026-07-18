package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewConsentRecord_MinorRequiresGuardian(t *testing.T) {
	_, err := NewConsentRecord("user-1", "org-1", true, nil, "user-1", time.Now())
	if !errors.Is(err, ErrGuardianConsentRequired) {
		t.Fatalf("expected ErrGuardianConsentRequired, got %v", err)
	}
}

func TestNewConsentRecord_MinorWithGuardianSucceeds(t *testing.T) {
	guardian := UserID("guardian-1")
	rec, err := NewConsentRecord("user-1", "org-1", true, &guardian, "guardian-1", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.IsMinor || rec.GuardianUserID == nil || *rec.GuardianUserID != guardian {
		t.Fatalf("guardian consent not recorded correctly: %+v", rec)
	}
}

func TestNewConsentRecord_AdultNoGuardianNeeded(t *testing.T) {
	rec, err := NewConsentRecord("user-1", "org-1", false, nil, "user-1", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.EligibleForDRS() {
		t.Fatalf("a freshly created record should not be DRS-eligible before any grant")
	}
}

func TestConsentRecord_EligibleForDRS(t *testing.T) {
	now := time.Now()
	rec, err := NewConsentRecord("user-1", "org-1", false, nil, "user-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.EligibleForDRS() {
		t.Fatal("should not be eligible with no grants")
	}

	rec.Grant(ConsentVideoCapture, "user-1", now)
	if rec.EligibleForDRS() {
		t.Fatal("should not be eligible with only video-capture granted")
	}

	rec.Grant(ConsentAIAnalysis, "user-1", now)
	if !rec.EligibleForDRS() {
		t.Fatal("should be eligible once both video-capture and ai-analysis are granted")
	}

	// FootageReuse is a separate opt-in and must not affect DRS eligibility.
	if rec.Grants[ConsentFootageReuse] {
		t.Fatal("footage reuse should not be implicitly granted")
	}
}

func TestConsentRecord_RevokeIsHonored(t *testing.T) {
	now := time.Now()
	rec, err := NewConsentRecord("user-1", "org-1", false, nil, "user-1", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec.Grant(ConsentVideoCapture, "user-1", now)
	rec.Grant(ConsentAIAnalysis, "user-1", now)
	if !rec.EligibleForDRS() {
		t.Fatal("expected eligible after both grants")
	}

	later := now.Add(time.Hour)
	rec.Revoke(ConsentAIAnalysis, "user-1", later)

	if rec.EligibleForDRS() {
		t.Fatal("revoking ai-analysis consent must remove DRS eligibility")
	}
	if rec.UpdatedAt != later {
		t.Fatalf("UpdatedAt not refreshed on revoke: got %v, want %v", rec.UpdatedAt, later)
	}
}
