// Package objstore stores file contents in an S3 compatible storage
// (Garage). Only PutObject, GetObject, StatObject, RemoveObject and
// BucketExists are used, which Garage supports.
package objstore

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is returned when the object does not exist.
var ErrNotFound = errors.New("object not found")

// partSize is the buffer size for uploads of unknown length.
const partSize = 16 << 20

// Options configure the client.
type Options struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// Store is an S3 bucket.
type Store struct {
	client *minio.Client
	bucket string
}

// New creates a client with path-style addressing and a fixed region, so
// no bucket location lookup is made.
func New(o Options) (*Store, error) {
	c, err := minio.New(o.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure:       o.UseSSL,
		Region:       o.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, err
	}
	return &Store{client: c, bucket: o.Bucket}, nil
}

// Ping checks that the storage is reachable and the bucket exists.
func (s *Store) Ping(ctx context.Context) error {
	ok, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("bucket %q does not exist", s.bucket)
	}
	return nil
}

// Put uploads r under key and returns the stored size. size may be -1.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
		PartSize:    partSize,
	})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

// Object is a readable and seekable object.
type Object interface {
	io.ReadSeekCloser
}

// Open returns the object; it fails with ErrNotFound if it is missing.
func (s *Store) Open(ctx context.Context, key string) (Object, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapErr(err)
	}
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, mapErr(err)
	}
	return obj, nil
}

// Remove deletes the object. A missing object is not an error.
func (s *Store) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func mapErr(err error) error {
	if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
		return ErrNotFound
	}
	return err
}
