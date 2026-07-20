// Package objectstore implements media-ingest-gateway's ObjectStore port
// (service.ObjectStore) two ways: an in-memory adapter (tests/local dev,
// same rationale as internal/memstore — no Docker/Postgres/real AWS
// access available in this environment) and an S3 adapter (s3.go) using
// the real AWS SDK, unit-tested against a fake implementing just the
// three S3 client methods actually used — no real AWS call happens
// anywhere in this codebase.
package objectstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
)

var ErrNotFound = errors.New("object not found")

// MemoryStore is a thread-safe, in-memory ObjectStore.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

func (m *MemoryStore) Put(_ context.Context, key string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return nil
}

func (m *MemoryStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	// Copy out from behind the lock so a caller reading the returned
	// io.ReadCloser at leisure can't race a concurrent Put/Delete on the
	// same key.
	out := make([]byte, len(data))
	copy(out, data)
	return io.NopCloser(bytes.NewReader(out)), nil
}

func (m *MemoryStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}
