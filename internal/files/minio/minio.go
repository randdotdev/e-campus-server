// Package minio is the byte-store adapter of the files context — the only
// code in the repository that talks to object storage. The bucket is
// private, always: bytes leave only through presigned URLs minted here for
// callers the HTTP edge already authorized.
package minio

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"github.com/randdotdev/e-campus-server/internal/files"
)

// tmpExpiryDays bounds how long an abandoned tmp/ upload survives.
const tmpExpiryDays = 1

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type Store struct {
	client *minio.Client
	bucket string
}

var _ files.ObjectStore = (*Store)(nil)

// New connects, ensures the bucket exists, and installs the tmp/
// lifecycle rule (non-fatal: without it only crash cleanup degrades).
func New(cfg Config, log *slog.Logger) (*Store, error) {
	endpoint := strings.TrimPrefix(cfg.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	lc := lifecycle.NewConfiguration()
	lc.Rules = []lifecycle.Rule{{
		ID:         "expire-tmp-uploads",
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: "tmp/"},
		Expiration: lifecycle.Expiration{Days: tmpExpiryDays},
	}}
	if err := client.SetBucketLifecycle(ctx, cfg.Bucket, lc); err != nil {
		log.Warn("files: tmp lifecycle rule not installed; crashed uploads will linger",
			"error", err)
	}

	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// Put streams r into key without buffering.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, opts)
	return err
}

// ServerCopy duplicates srcKey to dstKey inside MinIO — zero bytes
// through Go.
func (s *Store) ServerCopy(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: s.bucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: s.bucket, Object: srcKey})
	return err
}

// Presign mints a time-limited GET URL. Filename and content type ride
// as response-header overrides: the same deduplicated blob downloads
// under a different name per reference.
func (s *Store) Presign(ctx context.Context, key, filename, contentType string, ttl time.Duration) (string, error) {
	params := url.Values{}
	if filename != "" {
		params.Set("response-content-disposition",
			fmt.Sprintf("inline; filename*=UTF-8''%s", url.PathEscape(filename)))
	}
	if contentType != "" {
		params.Set("response-content-type", contentType)
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Remove deletes one object; removing a missing object succeeds.
func (s *Store) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// RemovePrefix deletes every object under prefix.
func (s *Store) RemovePrefix(ctx context.Context, prefix string) error {
	objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for result := range s.client.RemoveObjects(ctx, s.bucket, objects, minio.RemoveObjectsOptions{}) {
		if result.Err != nil {
			return fmt.Errorf("remove %s: %w", result.ObjectName, result.Err)
		}
	}
	return nil
}
