package main

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
	cfg    S3Config
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
	return &S3Store{
		cfg:    cfg,
		client: client,
	}, nil
}

func (s *S3Store) Get(ctx context.Context, key []byte) ([]byte, error) {
	result, err := s.client.GetObject(ctx, s.cfg.Bucket, hex.EncodeToString(key), minio.GetObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return nil, altda.ErrNotFound
		}
		return nil, err
	}
	defer result.Close()
	data, err := io.ReadAll(result)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3Store) Put(ctx context.Context, key []byte, value []byte) error {
	_, err := s.client.PutObject(ctx, s.cfg.Bucket, hex.EncodeToString(key), bytes.NewReader(value), int64(len(value)), minio.PutObjectOptions{})

	return err
}
