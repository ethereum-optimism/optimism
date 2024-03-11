package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	lvlerrors "github.com/syndtr/goleveldb/leveldb/errors"
)

var defaultTimeout = 5 * time.Second

type S3Store struct {
	timeout time.Duration
	bucket  string
	client  *s3.Client
}

func NewS3Store(bucket string) (*S3Store, error) {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return &S3Store{
		timeout: defaultTimeout,
		bucket:  bucket,
		client:  s3.NewFromConfig(sdkConfig),
	}, nil
}

func (s *S3Store) Get(key []byte) ([]byte, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), s.timeout)
	defer cancelFn()

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(hex.EncodeToString(key)),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return nil, lvlerrors.ErrNotFound
			}
		}
		return nil, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3Store) Put(key []byte, value []byte) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), s.timeout)
	defer cancelFn()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(hex.EncodeToString(key)),
		Body:   bytes.NewReader(value),
	})

	return err
}
