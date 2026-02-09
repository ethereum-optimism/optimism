package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockBlobsFetcher struct {
	mock.Mock
}

func (cl *MockBlobsFetcher) GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	out := cl.Mock.MethodCalled("GetBlobsByHash", time, hashes)
	return out.Get(0).([]*eth.Blob), out.Error(1)
}

func (cl *MockBlobsFetcher) ExpectOnGetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash, blobs []*eth.Blob, err error) {
	cl.Mock.On("GetBlobsByHash", time, hashes).Once().Return(blobs, err)
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
