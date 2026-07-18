// Package memstore is a thread-safe, in-memory implementation of
// identity-access's repository ports (service.OrganizationRepository,
// service.UserRepository, service.ConsentRepository). It exists so
// cmd/main.go can run a real, working service today without a Postgres
// instance available in this environment — see the plan's "Persistence"
// note. A Postgres adapter implementing the same ports is a follow-up, not
// a replacement of this package (memstore stays useful for local dev/tests
// either way).
package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

type OrganizationStore struct {
	mu   sync.RWMutex
	data map[domain.OrganizationID]domain.Organization
}

func NewOrganizationStore() *OrganizationStore {
	return &OrganizationStore{data: make(map[domain.OrganizationID]domain.Organization)}
}

func (s *OrganizationStore) Create(_ context.Context, org domain.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[org.ID] = org
	return nil
}

func (s *OrganizationStore) Get(_ context.Context, id domain.OrganizationID) (domain.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, ok := s.data[id]
	if !ok {
		return domain.Organization{}, domain.ErrOrganizationNotFound
	}
	return org, nil
}
