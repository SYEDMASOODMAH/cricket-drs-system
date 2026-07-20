package objectstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestMemoryStore_PutAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	if err := s.Put(ctx, "key1", bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rc, err := s.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	if err := s.Put(ctx, "key1", bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Delete(ctx, "key1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Get(ctx, "key1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_GetDoesNotAliasInternalState(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	if err := s.Put(ctx, "key1", bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rc, err := s.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(rc)
	data[0] = 'X' // mutate the caller's copy

	rc2, err := s.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data2, _ := io.ReadAll(rc2)
	if string(data2) != "hello" {
		t.Fatalf("mutating a fetched object leaked into the store: %q", data2)
	}
}
