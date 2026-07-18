package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type teamKey struct {
	orgID domain.OrganizationID
	id    domain.TeamID
}

type TeamStore struct {
	mu   sync.RWMutex
	data map[teamKey]domain.Team
}

func NewTeamStore() *TeamStore {
	return &TeamStore{data: make(map[teamKey]domain.Team)}
}

func (s *TeamStore) Create(_ context.Context, t domain.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[teamKey{t.OrganizationID, t.ID}] = t
	return nil
}

func (s *TeamStore) Get(_ context.Context, orgID domain.OrganizationID, id domain.TeamID) (domain.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data[teamKey{orgID, id}]
	if !ok {
		return domain.Team{}, domain.ErrTeamNotFound
	}
	return t, nil
}

func (s *TeamStore) ListByOrganization(_ context.Context, orgID domain.OrganizationID) ([]domain.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Team
	for k, t := range s.data {
		if k.orgID == orgID {
			out = append(out, t)
		}
	}
	return out, nil
}
