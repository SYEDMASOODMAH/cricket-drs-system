package domain

import "time"

// ConsentType is one of the separately-grantable consent categories per
// prd.md Section 13.3: video capture and AI analysis are the baseline for
// appearing in a DRS-enabled match; footage reuse (highlights/scouting/
// marketing) is a distinct, separate opt-in.
type ConsentType string

const (
	ConsentVideoCapture ConsentType = "video_capture"
	ConsentAIAnalysis   ConsentType = "ai_analysis"
	ConsentFootageReuse ConsentType = "footage_reuse"
)

// ConsentRecord is the per-player consent ground truth described in
// prd.md Section 5.6/13.3. It is keyed by (OrganizationID, UserID) — consent
// is captured per club/league, not globally, since a player's participation
// decision is made in the context of a specific organization's matches.
type ConsentRecord struct {
	UserID         UserID
	OrganizationID OrganizationID
	Grants         map[ConsentType]bool
	// IsMinor and GuardianUserID implement the distinct guardian-consent
	// flow prd.md Section 13.3 requires — never inferred from general club
	// registration.
	IsMinor        bool
	GuardianUserID *UserID
	UpdatedAt      time.Time
	// UpdatedBy is who captured/changed this record: the player themself,
	// a guardian, or an organizer/club admin during onboarding (Section
	// 5.6.1). It is distinct from UserID (whose consent this is).
	UpdatedBy UserID
}

// NewConsentRecord constructs an empty (all-grants-false) record, enforcing
// that a minor must have a guardian on file before any record exists at all
// — per prd.md Section 13.3, guardian consent is a distinct explicit flow,
// not an assumption folded into general registration.
func NewConsentRecord(userID UserID, orgID OrganizationID, isMinor bool, guardianUserID *UserID, updatedBy UserID, now time.Time) (ConsentRecord, error) {
	if isMinor && guardianUserID == nil {
		return ConsentRecord{}, ErrGuardianConsentRequired
	}
	return ConsentRecord{
		UserID:         userID,
		OrganizationID: orgID,
		Grants:         make(map[ConsentType]bool),
		IsMinor:        isMinor,
		GuardianUserID: guardianUserID,
		UpdatedAt:      now,
		UpdatedBy:      updatedBy,
	}, nil
}

// Grant records that consentType has been granted.
func (c *ConsentRecord) Grant(consentType ConsentType, updatedBy UserID, now time.Time) {
	if c.Grants == nil {
		c.Grants = make(map[ConsentType]bool)
	}
	c.Grants[consentType] = true
	c.UpdatedBy = updatedBy
	c.UpdatedAt = now
}

// Revoke records that consentType has been withdrawn. Per prd.md Section
// 5.6.3, revocation must be honored for all future matches — this service
// only maintains the ground truth; enforcing it at match time is the
// Match & Tournament Service's responsibility.
func (c *ConsentRecord) Revoke(consentType ConsentType, updatedBy UserID, now time.Time) {
	if c.Grants == nil {
		c.Grants = make(map[ConsentType]bool)
	}
	c.Grants[consentType] = false
	c.UpdatedBy = updatedBy
	c.UpdatedAt = now
}

// EligibleForDRS reports whether the baseline consent (video capture + AI
// analysis) has been granted — the precondition prd.md Section 5.6.2 sets
// for a player appearing in any DRS-enabled match roster. Footage reuse is
// intentionally excluded: it is a separate opt-in, not a DRS precondition.
func (c ConsentRecord) EligibleForDRS() bool {
	return c.Grants[ConsentVideoCapture] && c.Grants[ConsentAIAnalysis]
}
