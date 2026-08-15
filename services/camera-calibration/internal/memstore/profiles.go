package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

type profileKey struct {
	orgID    domain.OrganizationID
	cameraID domain.CameraID
}

// ProfileStore holds one CalibrationProfile per camera — Store is an
// upsert, matching the ProfileRepository port's documented "a new
// submission replaces the previous profile" semantics.
type ProfileStore struct {
	mu   sync.RWMutex
	data map[profileKey]domain.CalibrationProfile
}

func NewProfileStore() *ProfileStore {
	return &ProfileStore{data: make(map[profileKey]domain.CalibrationProfile)}
}

func (s *ProfileStore) Store(_ context.Context, profile domain.CalibrationProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[profileKey{profile.OrganizationID, profile.CameraID}] = profile
	return nil
}

func (s *ProfileStore) Get(_ context.Context, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.CalibrationProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.data[profileKey{orgID, cameraID}]
	if !ok {
		return domain.CalibrationProfile{}, domain.ErrProfileNotFound
	}
	return profile, nil
}
