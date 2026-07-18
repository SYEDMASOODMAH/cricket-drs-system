package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type matchKey struct {
	orgID domain.OrganizationID
	id    domain.MatchID
}

type MatchStore struct {
	mu   sync.RWMutex
	data map[matchKey]domain.Match
}

func NewMatchStore() *MatchStore {
	return &MatchStore{data: make(map[matchKey]domain.Match)}
}

func (s *MatchStore) Create(_ context.Context, m domain.Match) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[matchKey{m.OrganizationID, m.ID}] = m
	return nil
}

func (s *MatchStore) Get(_ context.Context, orgID domain.OrganizationID, id domain.MatchID) (domain.Match, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[matchKey{orgID, id}]
	if !ok {
		return domain.Match{}, domain.ErrMatchNotFound
	}
	return m, nil
}

func (s *MatchStore) ListByOrganization(_ context.Context, orgID domain.OrganizationID) ([]domain.Match, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Match
	for k, m := range s.data {
		if k.orgID == orgID {
			out = append(out, m)
		}
	}
	return out, nil
}
