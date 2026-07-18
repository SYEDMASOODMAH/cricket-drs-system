// Package memstore is a thread-safe, in-memory implementation of
// match-tournament's repository ports. Same rationale as identity-access's
// internal/memstore: no Docker/Postgres is available in this environment,
// and rules.md Section 1 requires domain logic to be unit-testable without
// a database, so persistence stays behind a port.
package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type tournamentKey struct {
	orgID domain.OrganizationID
	id    domain.TournamentID
}

type TournamentStore struct {
	mu   sync.RWMutex
	data map[tournamentKey]domain.Tournament
}

func NewTournamentStore() *TournamentStore {
	return &TournamentStore{data: make(map[tournamentKey]domain.Tournament)}
}

func (s *TournamentStore) Create(_ context.Context, t domain.Tournament) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[tournamentKey{t.OrganizationID, t.ID}] = t
	return nil
}

func (s *TournamentStore) Get(_ context.Context, orgID domain.OrganizationID, id domain.TournamentID) (domain.Tournament, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data[tournamentKey{orgID, id}]
	if !ok {
		return domain.Tournament{}, domain.ErrTournamentNotFound
	}
	return t, nil
}

func (s *TournamentStore) ListByOrganization(_ context.Context, orgID domain.OrganizationID) ([]domain.Tournament, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Tournament
	for k, t := range s.data {
		if k.orgID == orgID {
			out = append(out, t)
		}
	}
	return out, nil
}
