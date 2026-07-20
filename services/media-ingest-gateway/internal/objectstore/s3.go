package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// s3API is the narrow slice of *s3.Client this package actually calls.
// Depending on this instead of the concrete client is what makes S3Store
// unit-testable with a fake — no real AWS call happens anywhere in this
// codebase (see s3_test.go); this environment has no AWS credentials to
// make one with anyway.
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3Store implements service.ObjectStore against a real S3 bucket (or any
// S3-compatible endpoint).
type S3Store struct {
	client s3API
	bucket string
}

// NewS3Store builds an S3Store against a real *s3.Client. The bucket
// itself (versioned, encrypted, lifecycle-policy'd) is provisioned by
// infra/terraform/modules/storage — written and validated, not applied,
// same status as the rest of infra/terraform/.
func NewS3Store(client *s3.Client, bucket string) *S3Store {
	return newS3Store(client, bucket)
}

// newS3Store is the test seam: takes the s3API interface directly so
// tests can inject a fake without a real *s3.Client.
func newS3Store(client s3API, bucket string) *S3Store {
	return &S3Store{client: client, bucket: bucket}
}

func (s *S3Store) Put(ctx context.Context, key string, content io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   content,
	})
	if err != nil {
		return fmt.Errorf("objectstore: s3 put %s: %w", key, err)
	}
	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("objectstore: s3 get %s: %w", key, err)
	}
	return out.Body, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("objectstore: s3 delete %s: %w", key, err)
	}
	return nil
}
