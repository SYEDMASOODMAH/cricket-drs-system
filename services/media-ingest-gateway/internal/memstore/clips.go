// Package memstore is a thread-safe, in-memory implementation of
// media-ingest-gateway's ClipRepository port. Same rationale as
// identity-access's and match-tournament's internal/memstore: no
// Docker/Postgres available in this environment, and rules.md Section 1
// requires domain logic to be unit-testable without a database.
package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

type clipKey struct {
	orgID   domain.OrganizationID
	matchID domain.MatchID
	clipID  domain.ClipID
}

type ClipStore struct {
	mu   sync.RWMutex
	data map[clipKey]domain.Clip
}

func NewClipStore() *ClipStore {
	return &ClipStore{data: make(map[clipKey]domain.Clip)}
}

func (s *ClipStore) Create(_ context.Context, clip domain.Clip) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[clipKey{clip.OrganizationID, clip.MatchID, clip.ID}] = clip
	return nil
}

func (s *ClipStore) Get(_ context.Context, orgID domain.OrganizationID, matchID domain.MatchID, clipID domain.ClipID) (domain.Clip, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clip, ok := s.data[clipKey{orgID, matchID, clipID}]
	if !ok {
		return domain.Clip{}, domain.ErrClipNotFound
	}
	return clip, nil
}

func (s *ClipStore) ListByMatch(_ context.Context, orgID domain.OrganizationID, matchID domain.MatchID) ([]domain.Clip, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Clip
	for k, clip := range s.data {
		if k.orgID == orgID && k.matchID == matchID {
			out = append(out, clip)
		}
	}
	return out, nil
}
