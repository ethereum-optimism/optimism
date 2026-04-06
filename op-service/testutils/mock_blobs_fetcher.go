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
