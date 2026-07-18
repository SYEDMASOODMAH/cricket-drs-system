package domain

// UserID identifies a user. A UserID is only ever meaningful together with
// its OrganizationID — there is no cross-tenant user identity in Phase 1.
type UserID string

type User struct {
	ID             UserID
	OrganizationID OrganizationID
	Email          string
	// PasswordHash is opaque to the domain layer: it stores whatever the
	// service layer's chosen hashing algorithm (bcrypt) produced. Domain
	// code never hashes or compares passwords itself.
	PasswordHash string
	Role         Role
}

func NewUser(id UserID, orgID OrganizationID, email, passwordHash string, role Role) (User, error) {
	if email == "" {
		return User{}, ErrUserEmailEmpty
	}
	if !role.Valid() {
		return User{}, ErrInvalidRole
	}
	return User{
		ID:             id,
		OrganizationID: orgID,
		Email:          email,
		PasswordHash:   passwordHash,
		Role:           role,
	}, nil
}
