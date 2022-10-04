package testutils

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	out := m.Mock.MethodCalled("InfoByHash", hash)
	return *out[0].(*eth.BlockInfo), *out[1].(*error)
}

func (m *MockEthClient) ExpectInfoByHash(hash common.Hash, info eth.BlockInfo, err error) {
	m.Mock.On("InfoByHash", hash).Once().Return(&info, &err)
}

func (m *MockEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	out := m.Mock.MethodCalled("InfoByNumber", number)
	return *out[0].(*eth.BlockInfo), *out[1].(*error)
}

func (m *MockEthClient) ExpectInfoByNumber(number uint64, info eth.BlockInfo, err error) {
	m.Mock.On("InfoByNumber", number).Once().Return(&info, &err)
}

func (m *MockEthClient) InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error) {
	out := m.Mock.MethodCalled("InfoByLabel", label)
	return *out[0].(*eth.BlockInfo), *out[1].(*error)
}

func (m *MockEthClient) ExpectInfoByLabel(label eth.BlockLabel, info eth.BlockInfo, err error) {
	m.Mock.On("InfoByLabel", label).Once().Return(&info, &err)
}

func (m *MockEthClient) InfoByRpcNumber(ctx context.Context, num rpc.BlockNumber) (eth.BlockInfo, error) {
	out := m.Mock.MethodCalled("InfoByRpcNumber", num)
	return *out[0].(*eth.BlockInfo), *out[1].(*error)
}

func (m *MockEthClient) ExpectInfoByRpcNumber(num rpc.BlockNumber, info eth.BlockInfo, err error) {
	m.Mock.On("InfoByRpcNumber", num).Once().Return(&info, &err)
}

func (m *MockEthClient) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByHash", hash)
	return out[0].(eth.BlockInfo), out[1].(types.Transactions), *out[2].(*error)
}

func (m *MockEthClient) ExpectInfoAndTxsByHash(hash common.Hash, info eth.BlockInfo, transactions types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByHash", hash).Once().Return(info, transactions, &err)
}

func (m *MockEthClient) InfoAndTxsByNumber(ctx context.Context, number uint64) (eth.BlockInfo, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByNumber", number)
	return out[0].(eth.BlockInfo), out[1].(types.Transactions), *out[2].(*error)
}

func (m *MockEthClient) ExpectInfoAndTxsByNumber(number uint64, info eth.BlockInfo, transactions types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByNumber", number).Once().Return(info, transactions, &err)
}

func (m *MockEthClient) InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error) {
	out := m.Mock.MethodCalled("InfoAndTxsByLabel", label)
	return out[0].(eth.BlockInfo), out[1].(types.Transactions), *out[2].(*error)
}

func (m *MockEthClient) ExpectInfoAndTxsByLabel(label eth.BlockLabel, info eth.BlockInfo, transactions types.Transactions, err error) {
	m.Mock.On("InfoAndTxsByLabel", label).Once().Return(info, transactions, &err)
}

func (m *MockEthClient) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByHash", hash)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEthClient) ExpectPayloadByHash(hash common.Hash, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("PayloadByHash", hash).Once().Return(payload, &err)
}

func (m *MockEthClient) PayloadByNumber(ctx context.Context, n uint64) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByNumber", n)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEthClient) ExpectPayloadByNumber(hash common.Hash, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("PayloadByNumber", hash).Once().Return(payload, &err)
}

func (m *MockEthClient) PayloadByLabel(ctx context.Context, label eth.BlockLabel) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByLabel", label)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEthClient) ExpectPayloadByLabel(label eth.BlockLabel, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("PayloadByLabel", label).Once().Return(payload, &err)
}

func (m *MockEthClient) Fetch(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	out := m.Mock.MethodCalled("Fetch", blockHash)
	return *out[0].(*eth.BlockInfo), out[1].(types.Receipts), *out[2].(*error)
}

func (m *MockEthClient) ExpectFetch(hash common.Hash, info eth.BlockInfo, receipts types.Receipts, err error) {
	m.Mock.On("Fetch", hash).Once().Return(&info, receipts, &err)
}

func (m *MockEthClient) GetProof(ctx context.Context, address common.Address, blockTag string) (*eth.AccountResult, error) {
	return m.Mock.MethodCalled("GetProof", address, blockTag).Get(0).(*eth.AccountResult), nil
}

func (m *MockEthClient) ExpectGetProof(address common.Address, blockTag string, result *eth.AccountResult, err error) {
	m.Mock.On("GetProof", address, blockTag).Once().Return(result, &err)
}
