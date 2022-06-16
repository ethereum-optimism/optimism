package testutils

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
)

type MockL1Source struct {
	mock.Mock
}

func (m *MockL1Source) L1BlockRefByNumber(ctx context.Context, u uint64) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("L1BlockRefByNumber", u)
	return out[0].(eth.L1BlockRef), out[1].(error)
}

func (m *MockL1Source) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("L1BlockRefByNumber", hash)
	return out[0].(eth.L1BlockRef), out[1].(error)
}

func (m *MockL1Source) Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error) {
	out := m.Mock.MethodCalled("Fetch", blockHash)
	return out[0].(eth.L1Info), out[0].(types.Transactions), out[0].(types.Receipts), out[1].(error)
}

func (m *MockL1Source) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.L1Info, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByHash", hash)
	return out[0].(eth.L1Info), out[0].(types.Transactions), out[1].(error)
}
