package domain

// OrganizationID and UserID reference Identity & Access's tenant and user
// identifiers. match-tournament does not own this data — it stores foreign
// references and, where correctness actually matters (the consent gate),
// verifies them via internal/identityaccess rather than trusting a copy.
type OrganizationID string
type UserID string

type TournamentID string
type TeamID string
type MatchID string
