package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type rosterKey struct {
	orgID   domain.OrganizationID
	matchID domain.MatchID
	userID  domain.UserID
}

type RosterStore struct {
	mu   sync.RWMutex
	data map[rosterKey]domain.RosterEntry
}

func NewRosterStore() *RosterStore {
	return &RosterStore{data: make(map[rosterKey]domain.RosterEntry)}
}

func (s *RosterStore) Add(_ context.Context, entry domain.RosterEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := rosterKey{entry.OrganizationID, entry.MatchID, entry.UserID}
	if _, exists := s.data[k]; exists {
		return domain.ErrRosterEntryExists
	}
	s.data[k] = entry
	return nil
}

func (s *RosterStore) Remove(_ context.Context, orgID domain.OrganizationID, matchID domain.MatchID, userID domain.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := rosterKey{orgID, matchID, userID}
	if _, exists := s.data[k]; !exists {
		return domain.ErrRosterEntryNotFound
	}
	delete(s.data, k)
	return nil
}

func (s *RosterStore) ListByMatch(_ context.Context, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.RosterEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.RosterEntry
	for k, entry := range s.data {
		if k.orgID == orgID && k.matchID == matchID {
			out = append(out, entry)
		}
	}
	return out, nil
}
