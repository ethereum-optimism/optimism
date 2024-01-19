package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/mock"
)

type MockBlobsFetcher struct {
	mock.Mock
}

func (cl *MockBlobsFetcher) GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error) {
	out := cl.Mock.MethodCalled("GetBlobs", ref, hashes)
	return out.Get(0).([]*eth.Blob), out.Error(1)
}

func (cl *MockBlobsFetcher) GetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.BlobSidecar, error) {
	out := cl.Mock.MethodCalled("GetBlobSidecars", ref, hashes)
	return out.Get(0).([]*eth.BlobSidecar), out.Error(1)
}
