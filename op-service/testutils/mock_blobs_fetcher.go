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

func (cl *MockBlobsFetcher) ExpectOnGetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash, blobs []*eth.Blob, err error) {
	cl.Mock.On("GetBlobs", ref, hashes).Once().Return(blobs, err)
}

func (cl *MockBlobsFetcher) GetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.BlobSidecar, error) {
	out := cl.Mock.MethodCalled("GetBlobSidecars", ref, hashes)
	return out.Get(0).([]*eth.BlobSidecar), out.Error(1)
}

func (cl *MockBlobsFetcher) ExpectOnGetBlobSidecars(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash, commitment eth.Bytes48, blobs []*eth.Blob, err error) {
	cl.Mock.On("GetBlobSidecars", ref, hashes).Once().Return([]*eth.BlobSidecar{{
		Blob:          *blobs[0],
		Index:         eth.Uint64String(hashes[0].Index),
		KZGCommitment: commitment,
	}}, err)
}
