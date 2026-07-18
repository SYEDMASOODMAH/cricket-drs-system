// Package security adapts vetted crypto libraries (bcrypt, JWT) to the
// service.PasswordHasher and service.TokenIssuer ports. rules.md Section
// 3.4 prohibits hand-rolled auth crypto; this package is the one place
// those libraries are imported, so the service layer never touches them
// directly and stays testable against fakes.
package security

import "golang.org/x/crypto/bcrypt"

// BcryptHasher implements service.PasswordHasher.
type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

func (b *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (b *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
