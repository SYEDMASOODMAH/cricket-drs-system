package domain

type Team struct {
	ID             TeamID
	OrganizationID OrganizationID
	Name           string
}

func NewTeam(id TeamID, orgID OrganizationID, name string) (Team, error) {
	if name == "" {
		return Team{}, ErrNameEmpty
	}
	return Team{ID: id, OrganizationID: orgID, Name: name}, nil
}
