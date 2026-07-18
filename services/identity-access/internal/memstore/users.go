package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

// userKey composes OrganizationID into every lookup key, mechanically
// enforcing that a user can never be fetched without stating which tenant
// it's expected to belong to — mirrors the port contract in
// service.UserRepository.
type userKey struct {
	orgID  domain.OrganizationID
	userID domain.UserID
}

type UserStore struct {
	mu    sync.RWMutex
	byKey map[userKey]domain.User
}

func NewUserStore() *UserStore {
	return &UserStore{byKey: make(map[userKey]domain.User)}
}

func (s *UserStore) Create(_ context.Context, u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := userKey{u.OrganizationID, u.ID}
	if _, exists := s.byKey[k]; exists {
		return domain.ErrUserExists
	}
	s.byKey[k] = u
	return nil
}

func (s *UserStore) Get(_ context.Context, orgID domain.OrganizationID, userID domain.UserID) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byKey[userKey{orgID, userID}]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (s *UserStore) GetByEmail(_ context.Context, orgID domain.OrganizationID, email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, u := range s.byKey {
		if k.orgID == orgID && u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (s *UserStore) Update(_ context.Context, u domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := userKey{u.OrganizationID, u.ID}
	if _, ok := s.byKey[k]; !ok {
		return domain.ErrUserNotFound
	}
	s.byKey[k] = u
	return nil
}

func (s *UserStore) ListByOrganization(_ context.Context, orgID domain.OrganizationID) ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.User
	for k, u := range s.byKey {
		if k.orgID == orgID {
			out = append(out, u)
		}
	}
	return out, nil
}
