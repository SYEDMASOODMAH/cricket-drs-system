// Package memstore is a thread-safe, in-memory implementation of
// camera-calibration's CameraRepository and ProfileRepository ports. Same
// rationale as every other service's internal/memstore: no Docker/
// Postgres available in this environment, and rules.md Section 1 requires
// domain logic to be unit-testable without a database.
package memstore

import (
	"context"
	"sync"

	"github.com/cricketdrs/services/camera-calibration/internal/domain"
)

type cameraKey struct {
	orgID    domain.OrganizationID
	cameraID domain.CameraID
}

type CameraStore struct {
	mu   sync.RWMutex
	data map[cameraKey]domain.Camera
}

func NewCameraStore() *CameraStore {
	return &CameraStore{data: make(map[cameraKey]domain.Camera)}
}

func (s *CameraStore) Create(_ context.Context, camera domain.Camera) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cameraKey{camera.OrganizationID, camera.ID}] = camera
	return nil
}

func (s *CameraStore) Get(_ context.Context, orgID domain.OrganizationID, cameraID domain.CameraID) (domain.Camera, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	camera, ok := s.data[cameraKey{orgID, cameraID}]
	if !ok {
		return domain.Camera{}, domain.ErrCameraNotFound
	}
	return camera, nil
}

func (s *CameraStore) ListByOrganization(_ context.Context, orgID domain.OrganizationID) ([]domain.Camera, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Camera
	for k, camera := range s.data {
		if k.orgID == orgID {
			out = append(out, camera)
		}
	}
	return out, nil
}
