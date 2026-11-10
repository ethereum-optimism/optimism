package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const testBucket = "altda"

// newFakeS3Client returns a client for an in-process S3 fake that holds an empty testBucket.
func newFakeS3Client(t *testing.T) *minio.Client {
	backend := s3mem.New()
	require.NoError(t, backend.CreateBucket(testBucket))
	srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4("id", "secret", ""),
		Secure: false,
	})
	require.NoError(t, err)
	return client
}

func newFakeS3Store(t *testing.T) *S3Store {
	return NewS3StoreWithClient(newFakeS3Client(t), testBucket)
}

func TestS3StoreReturnsStoredInput(t *testing.T) {
	s := newFakeS3Store(t)
	ctx := context.Background()
	key := []byte{0x00, 0xab, 0xcd}
	input := []byte("batch data")

	require.NoError(t, s.Put(ctx, key, input))

	got, err := s.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, input, got)
}

func TestS3StoreReportsMissingInputAsNotFound(t *testing.T) {
	s := newFakeS3Store(t)

	_, err := s.Get(context.Background(), []byte{0x00, 0x01})
	require.ErrorIs(t, err, altda.ErrNotFound)
}

func TestDAServerReportsMissingS3InputAsNotFound(t *testing.T) {
	server := altda.NewDAServer("127.0.0.1", 0, newFakeS3Store(t), testlog.Logger(t, log.LevelInfo), false)
	require.NoError(t, server.Start())
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	client := altda.NewDAClient(server.HttpEndpoint(), true, false)

	_, err := client.GetInput(context.Background(), altda.NewKeccak256Commitment([]byte("unknown input")))
	require.ErrorIs(t, err, altda.ErrNotFound)
}

// A 404 that doesn't come from S3, e.g. a proxy's error page, says nothing about the input.
func TestS3StoreReportsNonS3NotFoundAsError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4("id", "secret", ""),
		Secure: false,
		// A known region skips the bucket location lookup, so the 404 answers the object GET.
		Region: "us-east-1",
	})
	require.NoError(t, err)
	s := NewS3StoreWithClient(client, testBucket)

	_, err = s.Get(context.Background(), []byte{0x00, 0x01})
	require.Error(t, err)
	require.NotErrorIs(t, err, altda.ErrNotFound)
}

// A missing bucket is also an HTTP 404, but it is a misconfiguration, not a missing input.
func TestS3StoreReportsMissingBucketAsError(t *testing.T) {
	s := NewS3StoreWithClient(newFakeS3Client(t), "missing-bucket")

	_, err := s.Get(context.Background(), []byte{0x00, 0x01})
	require.Error(t, err)
	require.NotErrorIs(t, err, altda.ErrNotFound)
}
