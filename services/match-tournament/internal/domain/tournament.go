package domain

type Tournament struct {
	ID                TournamentID
	OrganizationID    OrganizationID
	Name              string
	DefaultConditions PlayingConditions
}

func NewTournament(id TournamentID, orgID OrganizationID, name string, defaults PlayingConditions) (Tournament, error) {
	if name == "" {
		return Tournament{}, ErrNameEmpty
	}
	return Tournament{
		ID:                id,
		OrganizationID:    orgID,
		Name:              name,
		DefaultConditions: defaults,
	}, nil
}
