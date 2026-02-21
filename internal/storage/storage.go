// Package storage provides S3-compatible object storage using MinIO.
package storage

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type Client struct {
	client *minio.Client
	bucket string
}

func New(cfg Config) (*Client, error) {
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

	return &Client{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := c.client.PutObject(ctx, c.bucket, key, reader, size, opts)
	return err
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	_, err = obj.Stat()
	if err != nil {
		_ = obj.Close()
		if IsNotFoundErr(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	return obj, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func (c *Client) PresignedGetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, c.bucket, key, expires, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) PresignedPutURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	u, err := c.client.PresignedPutObject(ctx, c.bucket, key, expires)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if IsNotFoundErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
