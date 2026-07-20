package objectstore

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// fakeS3API implements s3API without any real AWS call — this environment
// has no AWS credentials to make one with, and s3.go's whole point is to
// keep that adapter's logic (key/bucket wiring, error mapping) testable
// without one. Same pattern as match-tournament's httptest-based
// identityaccess client tests.
type fakeS3API struct {
	putCalls    []*s3.PutObjectInput
	putErr      error
	objects     map[string][]byte
	getErr      error
	deleteCalls []*s3.DeleteObjectInput
	deleteErr   error
}

func (f *fakeS3API) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}
	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	if f.objects == nil {
		f.objects = make(map[string][]byte)
	}
	f.objects[*params.Key] = data
	f.putCalls = append(f.putCalls, params)
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3API) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	data, ok := f.objects[*params.Key]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (f *fakeS3API) DeleteObject(_ context.Context, params *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.deleteCalls = append(f.deleteCalls, params)
	delete(f.objects, *params.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func TestS3Store_Put(t *testing.T) {
	fake := &fakeS3API{}
	s := newS3Store(fake, "test-bucket")

	if err := s.Put(context.Background(), "org-1/match-1/clip-1", bytes.NewReader([]byte("video-bytes"))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(fake.putCalls))
	}
	if *fake.putCalls[0].Bucket != "test-bucket" || *fake.putCalls[0].Key != "org-1/match-1/clip-1" {
		t.Fatalf("unexpected put call: %+v", fake.putCalls[0])
	}
}

func TestS3Store_PutErrorWrapped(t *testing.T) {
	fake := &fakeS3API{putErr: errors.New("network error")}
	s := newS3Store(fake, "test-bucket")
	if err := s.Put(context.Background(), "key1", bytes.NewReader([]byte("x"))); err == nil {
		t.Fatal("expected an error")
	}
}

func TestS3Store_Get(t *testing.T) {
	fake := &fakeS3API{objects: map[string][]byte{"key1": []byte("hello")}}
	s := newS3Store(fake, "test-bucket")

	rc, err := s.Get(context.Background(), "key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "hello" {
		t.Fatalf("unexpected content: %q", data)
	}
}

func TestS3Store_GetNotFound(t *testing.T) {
	fake := &fakeS3API{}
	s := newS3Store(fake, "test-bucket")
	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestS3Store_GetOtherErrorWrapped(t *testing.T) {
	fake := &fakeS3API{getErr: errors.New("network error")}
	s := newS3Store(fake, "test-bucket")
	_, err := s.Get(context.Background(), "key1")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a generic wrapped error, not ErrNotFound, got %v", err)
	}
}

func TestS3Store_Delete(t *testing.T) {
	fake := &fakeS3API{objects: map[string][]byte{"key1": []byte("hello")}}
	s := newS3Store(fake, "test-bucket")
	if err := s.Delete(context.Background(), "key1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.deleteCalls) != 1 || *fake.deleteCalls[0].Key != "key1" {
		t.Fatalf("unexpected delete calls: %+v", fake.deleteCalls)
	}
}

func TestS3Store_DeleteErrorWrapped(t *testing.T) {
	fake := &fakeS3API{deleteErr: errors.New("network error")}
	s := newS3Store(fake, "test-bucket")
	if err := s.Delete(context.Background(), "key1"); err == nil {
		t.Fatal("expected an error")
	}
}
