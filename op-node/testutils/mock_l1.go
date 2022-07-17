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

func (m *MockL1Source) InfoByHash(ctx context.Context, hash common.Hash) (eth.L1Info, error) {
	out := m.Mock.MethodCalled("InfoByHash", hash)
	return *out[0].(*eth.L1Info), *out[1].(*error)
}

func (m *MockL1Source) ExpectInfoByHash(hash common.Hash, info eth.L1Info, err error) {
	m.Mock.On("InfoByHash", hash).Once().Return(&info, &err)
}

func (m *MockL1Source) L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("L1HeadBlockRef")
	return out[0].(eth.L1BlockRef), *out[1].(*error)
}

func (m *MockL1Source) L1BlockRefByNumber(ctx context.Context, u uint64) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("L1BlockRefByNumber", u)
	return out[0].(eth.L1BlockRef), *out[1].(*error)
}

func (m *MockL1Source) ExpectL1BlockRefByNumber(u uint64, ref eth.L1BlockRef, err error) {
	m.Mock.On("L1BlockRefByNumber", u).Once().Return(ref, &err)
}

func (m *MockL1Source) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	out := m.Mock.MethodCalled("L1BlockRefByHash", hash)
	return out[0].(eth.L1BlockRef), *out[1].(*error)
}

func (m *MockL1Source) ExpectL1BlockRefByHash(hash common.Hash, ref eth.L1BlockRef, err error) {
	m.Mock.On("L1BlockRefByHash", hash).Once().Return(ref, &err)
}

func (m *MockL1Source) Fetch(ctx context.Context, blockHash common.Hash) (eth.L1Info, types.Transactions, types.Receipts, error) {
	out := m.Mock.MethodCalled("Fetch", blockHash)
	return *out[0].(*eth.L1Info), out[1].(types.Transactions), out[2].(types.Receipts), *out[3].(*error)
}

func (m *MockL1Source) ExpectFetch(hash common.Hash, info eth.L1Info, transactions types.Transactions, receipts types.Receipts, err error) {
	m.Mock.On("Fetch", hash).Once().Return(&info, transactions, receipts, &err)
}

func (m *MockL1Source) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.L1Info, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByHash", hash)
	return out[0].(eth.L1Info), out[1].(types.Transactions), *out[2].(*error)
}

func (m *MockL1Source) ExpectInfoAndTxsByHash(hash common.Hash, info eth.L1Info, transactions types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByHash", hash).Once().Return(info, transactions, &err)
}
