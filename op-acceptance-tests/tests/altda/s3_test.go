package altda

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ethereum-optimism/optimism/op-alt-da/store"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestAltDAChainDerivesSafeBlocksFromS3(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node does not support alt-DA")
	bucket := newFakeS3Bucket(t)
	sys := presets.NewMinimalNoFaultProofs(t, presets.WithAltDA(bucket.store))

	// The batcher posts only commitments to L1, so safe blocks can only be derived from the
	// inputs the DA server reads back from S3.
	sys.L2CL.Advanced(safety.LocalSafe, 3, 30)
	t.Require().NotZero(bucket.objectCount(t), "expected the batcher's inputs in the S3 bucket")
}

type fakeS3Bucket struct {
	client *minio.Client
	name   string
	store  *store.S3Store
}

// newFakeS3Bucket serves an empty bucket from an in-process S3 fake.
func newFakeS3Bucket(t devtest.T) *fakeS3Bucket {
	const name = "altda"
	backend := s3mem.New()
	t.Require().NoError(backend.CreateBucket(name))
	srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	t.Require().NoError(err)
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4("id", "secret", ""),
		Secure: false,
	})
	t.Require().NoError(err)
	return &fakeS3Bucket{
		client: client,
		name:   name,
		store:  store.NewS3StoreWithClient(client, name),
	}
}

func (b *fakeS3Bucket) objectCount(t devtest.T) int {
	n := 0
	for obj := range b.client.ListObjects(t.Ctx(), b.name, minio.ListObjectsOptions{}) {
		t.Require().NoError(obj.Err, "failed to list the S3 bucket")
		n++
	}
	return n
}
