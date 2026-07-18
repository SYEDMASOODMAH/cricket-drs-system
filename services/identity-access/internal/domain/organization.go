package domain

// OrganizationID identifies a tenant (club, league, or board). Every piece
// of data in this service is scoped to exactly one OrganizationID — that
// scoping is the tenant-isolation boundary rules.md Section 6.5 requires.
type OrganizationID string

type Organization struct {
	ID   OrganizationID
	Name string
}

// NewOrganization validates and constructs an Organization. Callers are
// expected to generate a fresh ID (e.g. via a UUID) before calling this.
func NewOrganization(id OrganizationID, name string) (Organization, error) {
	if name == "" {
		return Organization{}, ErrOrganizationNameEmpty
	}
	return Organization{ID: id, Name: name}, nil
}
