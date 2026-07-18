package domain

import "time"

// RosterEntry is kept separate from Match (its own repository, like
// identity-access keeps ConsentRecord separate from User) since "match
// configuration" and "roster management" are different use-cases with
// different write patterns. OrganizationID is carried directly (not just
// inferred via MatchID) so the repository can key and scope lookups by
// tenant the same way identity-access's memstore does.
type RosterEntry struct {
	OrganizationID OrganizationID
	MatchID        MatchID
	TeamID         TeamID
	UserID         UserID
	AddedAt        time.Time
	AddedBy        UserID
}

func NewRosterEntry(orgID OrganizationID, matchID MatchID, teamID TeamID, userID UserID, addedBy UserID, now time.Time) (RosterEntry, error) {
	if userID == "" {
		return RosterEntry{}, ErrUserIDEmpty
	}
	return RosterEntry{
		OrganizationID: orgID,
		MatchID:        matchID,
		TeamID:         teamID,
		UserID:         userID,
		AddedAt:        now,
		AddedBy:        addedBy,
	}, nil
}
