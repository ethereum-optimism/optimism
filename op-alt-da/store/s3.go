// Package store provides altda.KVStore implementations backed by a local directory or an S3 bucket.
package store

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
)

type S3Config struct {
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
}

type S3Store struct {
	bucket string
	client *minio.Client
}

func NewS3Store(cfg S3Config) (*S3Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.AccessKeySecret, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}
	return NewS3StoreWithClient(client, cfg.Bucket), nil
}

// NewS3StoreWithClient returns a store that keeps inputs in bucket, reaching S3 through client.
func NewS3StoreWithClient(client *minio.Client, bucket string) *S3Store {
	return &S3Store{
		bucket: bucket,
		client: client,
	}
}

func (s *S3Store) Get(ctx context.Context, key []byte) ([]byte, error) {
	result, err := s.client.GetObject(ctx, s.bucket, hex.EncodeToString(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer result.Close()
	// GetObject sends no request; S3 errors such as NoSuchKey surface on the first read.
	data, err := io.ReadAll(result)
	if err != nil {
		if isMissingKey(err) {
			return nil, altda.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// isMissingKey reports whether err is S3's answer that the object doesn't exist. minio-go also
// reports NoSuchKey for any 404 without an S3 error body, such as a proxy's error page; only S3
// sets a request ID.
func isMissingKey(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" && resp.RequestID != ""
}

func (s *S3Store) Put(ctx context.Context, key []byte, value []byte) error {
	_, err := s.client.PutObject(ctx, s.bucket, hex.EncodeToString(key), bytes.NewReader(value), int64(len(value)), minio.PutObjectOptions{})

	return err
}
