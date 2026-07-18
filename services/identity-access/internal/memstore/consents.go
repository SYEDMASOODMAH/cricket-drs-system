package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/identity-access/internal/domain"
)

type consentKey struct {
	orgID  domain.OrganizationID
	userID domain.UserID
}

type ConsentStore struct {
	mu   sync.RWMutex
	data map[consentKey]domain.ConsentRecord
}

func NewConsentStore() *ConsentStore {
	return &ConsentStore{data: make(map[consentKey]domain.ConsentRecord)}
}

func (s *ConsentStore) Get(_ context.Context, orgID domain.OrganizationID, userID domain.UserID) (domain.ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data[consentKey{orgID, userID}]
	if !ok {
		return domain.ConsentRecord{}, domain.ErrConsentNotFound
	}
	return cloneConsentRecord(rec), nil
}

func (s *ConsentStore) Upsert(_ context.Context, rec domain.ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[consentKey{rec.OrganizationID, rec.UserID}] = cloneConsentRecord(rec)
	return nil
}

// cloneConsentRecord copies the Grants map so callers mutating a record
// they got from Get (or that we just stored) can never alias the store's
// internal state.
func cloneConsentRecord(rec domain.ConsentRecord) domain.ConsentRecord {
	grants := make(map[domain.ConsentType]bool, len(rec.Grants))
	for k, v := range rec.Grants {
		grants[k] = v
	}
	rec.Grants = grants
	return rec
}
