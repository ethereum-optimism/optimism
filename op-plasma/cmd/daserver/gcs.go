package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googleapis/gax-go/v2"
)

type GCSConfig struct {
	Bucket       string
	ObjectPrefix string
}

type GCSStore struct {
	bucket       string
	objectPrefix string
}

func NewGCSStore(cfg GCSConfig) (*GCSStore, error) {
	return &GCSStore{
		bucket:       cfg.Bucket,
		objectPrefix: cfg.ObjectPrefix,
	}, nil
}

func (s *GCSStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	rc, err := client.Bucket(s.bucket).Object(s.objectPrefix+hex.EncodeToString(key)).Retryer(
		storage.WithBackoff(gax.Backoff{
			Initial:    2 * time.Second,
			Max:        60 * time.Second,
			Multiplier: 3,
		}), storage.WithPolicy(storage.RetryAlways)).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %w", s.bucket, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return data, nil
}

func (s *GCSStore) Put(ctx context.Context, key []byte, value []byte) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	buf := bytes.NewBuffer(value)
	object := s.objectPrefix + hex.EncodeToString(key)
	wc := client.Bucket(s.bucket).Object(object).Retryer(storage.WithBackoff(gax.Backoff{
		Initial:    2 * time.Second,
		Max:        60 * time.Second,
		Multiplier: 3,
	}), storage.WithPolicy(storage.RetryAlways)).NewWriter(ctx)

	if _, err = io.Copy(wc, buf); err != nil {
		return fmt.Errorf("io.Copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %w", err)
	}

	return err
}
