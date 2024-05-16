package testutils

import (
	"context"
	"math/big"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/client"
)

var _ client.Client = &MockClient{}

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Close() {
	m.Mock.Called()
}

func (m *MockClient) ExpectClose() {
	m.Mock.On("Close").Once().Return()
}

func (m *MockClient) RPC() client.RPC {
	out := m.Mock.Called()
	return out.Get(0).(client.RPC)
}

func (m *MockClient) ExpectRPC(rpc client.RPC) {
	m.Mock.On("RPC").Once().Return(rpc)
}

func (m *MockClient) ChainID(ctx context.Context) (*big.Int, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectChainID(id *big.Int, err error) {
	m.Mock.On("ChainID", mock.Anything).Once().Return(id, err)
}

func (m *MockClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	out := m.Mock.Called(ctx, hash)
	return out.Get(0).(*types.Block), out.Error(1)
}

func (m *MockClient) ExpectBlockByHash(hash common.Hash, block *types.Block, err error) {
	m.Mock.On("BlockByHash", mock.Anything, hash).Once().Return(block, err)
}

func (m *MockClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	out := m.Mock.Called(ctx, number)
	return out.Get(0).(*types.Block), out.Error(1)
}

func (m *MockClient) ExpectBlockByNumber(number *big.Int, block *types.Block, err error) {
	m.Mock.On("BlockByNumber", mock.Anything, number).Once().Return(block, err)
}

func (m *MockClient) BlockNumber(ctx context.Context) (uint64, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(uint64), out.Error(1)
}

func (m *MockClient) ExpectBlockNumber(blockNum uint64, err error) {
	m.Mock.On("BlockNumber", mock.Anything).Once().Return(blockNum, err)
}

func (m *MockClient) PeerCount(ctx context.Context) (uint64, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(uint64), out.Error(1)
}

func (m *MockClient) ExpectPeerCount(count uint64, err error) {
	m.Mock.On("PeerCount", mock.Anything).Once().Return(count, err)
}

func (m *MockClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	out := m.Mock.Called(ctx, hash)
	return out.Get(0).(*types.Header), out.Error(1)
}

func (m *MockClient) ExpectHeaderByHash(hash common.Hash, header *types.Header, err error) {
	m.Mock.On("HeaderByHash", mock.Anything, hash).Once().Return(header, err)
}

func (m *MockClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	out := m.Mock.Called(ctx, number)
	return out.Get(0).(*types.Header), out.Error(1)
}

func (m *MockClient) ExpectHeaderByNumber(number *big.Int, header *types.Header, err error) {
	m.Mock.On("HeaderByNumber", mock.Anything, number).Once().Return(header, err)
}

func (m *MockClient) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	out := m.Mock.Called(ctx, hash)
	return out.Get(0).(*types.Transaction), out.Get(1).(bool), out.Error(2)
}

func (m *MockClient) ExpectTransactionByHash(hash common.Hash, tx *types.Transaction, isPending bool, err error) {
	m.Mock.On("TransactionByHash", mock.Anything, hash).Once().Return(tx, isPending, err)
}

func (m *MockClient) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	out := m.Mock.Called(ctx, tx, block, index)
	return out.Get(0).(common.Address), out.Error(1)
}

func (m *MockClient) ExpectTransactionSender(tx *types.Transaction, block common.Hash, index uint, sender common.Address, err error) {
	m.Mock.On("TransactionSender", mock.Anything, tx, block, index).Once().Return(sender, err)
}

func (m *MockClient) TransactionCount(ctx context.Context, hash common.Hash) (uint, error) {
	out := m.Mock.Called(ctx, hash)
	return out.Get(0).(uint), out.Error(1)
}

func (m *MockClient) ExpectTransactionCount(hash common.Hash, count uint, err error) {
	m.Mock.On("TransactionCount", mock.Anything, hash).Once().Return(count, err)
}

func (m *MockClient) TransactionInBlock(ctx context.Context, hash common.Hash, index uint) (*types.Transaction, error) {
	out := m.Mock.Called(ctx, hash, index)
	return out.Get(0).(*types.Transaction), out.Error(1)
}

func (m *MockClient) ExpectTransactionInBlock(hash common.Hash, index uint, tx *types.Transaction, err error) {
	m.Mock.On("TransactionInBlock", mock.Anything, hash, index).Once().Return(tx, err)
}

func (m *MockClient) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	out := m.Mock.Called(ctx, hash)
	return out.Get(0).(*types.Receipt), out.Error(1)
}

func (m *MockClient) ExpectTransactionReceipt(hash common.Hash, receipt *types.Receipt, err error) {
	m.Mock.On("TransactionReceipt", mock.Anything, hash).Once().Return(receipt, err)
}

func (m *MockClient) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(*ethereum.SyncProgress), out.Error(1)
}

func (m *MockClient) ExpectSyncProgress(prog *ethereum.SyncProgress, err error) {
	m.Mock.On("SyncProgress", mock.Anything).Once().Return(prog, err)
}

func (m *MockClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	out := m.Mock.Called(ctx, ch)
	return out.Get(0).(ethereum.Subscription), out.Error(1)
}

func (m *MockClient) ExpectSubscribeNewHead(ch chan<- *types.Header, sub ethereum.Subscription, err error) {
	m.Mock.On("SubscribeNewHead", mock.Anything, ch).Once().Return(sub, err)
}

func (m *MockClient) NetworkID(ctx context.Context) (*big.Int, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectNetworkID(id *big.Int, err error) {
	m.Mock.On("NetworkID", mock.Anything).Once().Return(id, err)
}

func (m *MockClient) BalanceAt(ctx context.Context, account common.Address, block *big.Int) (*big.Int, error) {
	out := m.Mock.Called(ctx, account, block)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectBalanceAt(account common.Address, block, amount *big.Int, err error) {
	m.Mock.On("BalanceAt", mock.Anything, account, block).Once().Return(amount, err)
}

func (m *MockClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, block *big.Int) ([]byte, error) {
	out := m.Mock.Called(ctx, account, key, block)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectStorageAt(account common.Address, key common.Hash, block *big.Int, data []byte, err error) {
	m.Mock.On("StorageAt", mock.Anything, account, key, block).Once().Return(data, err)
}

func (m *MockClient) CodeAt(ctx context.Context, account common.Address, block *big.Int) ([]byte, error) {
	out := m.Mock.Called(ctx, account, block)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectCodeAt(account common.Address, block *big.Int, code []byte, err error) {
	m.Mock.On("CodeAt", mock.Anything, account, block).Once().Return(code, err)
}

func (m *MockClient) NonceAt(ctx context.Context, account common.Address, block *big.Int) (uint64, error) {
	out := m.Mock.Called(ctx, account, block)
	return out.Get(0).(uint64), out.Error(1)
}

func (m *MockClient) ExpectNonceAt(account common.Address, block *big.Int, nonce uint64, err error) {
	m.Mock.On("NonceAt", mock.Anything, account, block).Once().Return(nonce, err)
}

func (m *MockClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	out := m.Mock.Called(ctx, q)
	return out.Get(0).([]types.Log), out.Error(1)
}

func (m *MockClient) ExpectFilterLogs(q ethereum.FilterQuery, logs []types.Log, err error) {
	m.Mock.On("FilterLogs", mock.Anything, q).Once().Return(logs, err)
}

func (m *MockClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	out := m.Mock.Called(ctx, q, ch)
	return out.Get(0).(ethereum.Subscription), out.Error(1)
}

func (m *MockClient) ExpectSubscribeFilterLogs(q ethereum.FilterQuery, ch chan<- types.Log, sub ethereum.Subscription, err error) {
	m.Mock.On("SubscribeFilterLogs", mock.Anything, q, ch).Once().Return(sub, err)
}

func (m *MockClient) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	out := m.Mock.Called(ctx, account)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectPendingBalanceAt(account common.Address, balance *big.Int, err error) {
	m.Mock.On("PendingBalanceAt", mock.Anything, account).Once().Return(balance, err)
}

func (m *MockClient) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	out := m.Mock.Called(ctx, account, key)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectPendingStorageAt(account common.Address, key common.Hash, data []byte, err error) {
	m.Mock.On("PendingStorageAt", mock.Anything, account, key).Once().Return(data, err)
}

func (m *MockClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	out := m.Mock.Called(ctx, account)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectPendingCodeAt(account common.Address, code []byte, err error) {
	m.Mock.On("PendingCodeAt", mock.Anything, account).Once().Return(code, err)
}

func (m *MockClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	out := m.Mock.Called(ctx, account)
	return out.Get(0).(uint64), out.Error(1)
}

func (m *MockClient) ExpectPendingNonceAt(account common.Address, nonce uint64, err error) {
	m.Mock.On("PendingNonceAt", mock.Anything, account).Once().Return(nonce, err)
}

func (m *MockClient) PendingTransactionCount(ctx context.Context) (uint, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(uint), out.Error(1)
}

func (m *MockClient) ExpectPendingTransactionCount(count uint, err error) {
	m.Mock.On("PendingTransactionCount", mock.Anything).Once().Return(count, err)
}

func (m *MockClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	out := m.Mock.Called(ctx, msg, blockNumber)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectCallContract(msg ethereum.CallMsg, blockNumber *big.Int, result []byte, err error) {
	m.Mock.On("CallContract", mock.Anything, msg, blockNumber).Once().Return(result, err)
}

func (m *MockClient) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
	out := m.Mock.Called(ctx, msg, blockHash)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectCallContractAtHash(msg ethereum.CallMsg, blockHash common.Hash, result []byte, err error) {
	m.Mock.On("CallContractAtHash", mock.Anything, msg, blockHash).Once().Return(result, err)
}

func (m *MockClient) PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	out := m.Mock.Called(ctx, msg)
	return out.Get(0).([]byte), out.Error(1)
}

func (m *MockClient) ExpectPendingCallContract(msg ethereum.CallMsg, result []byte, err error) {
	m.Mock.On("PendingCallContract", mock.Anything, msg).Once().Return(result, err)
}

func (m *MockClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectSuggestGasPrice(price *big.Int, err error) {
	m.Mock.On("SuggestGasPrice", mock.Anything).Once().Return(price, err)
}

func (m *MockClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	out := m.Mock.Called(ctx)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockClient) ExpectSuggestGasTipCap(tipCap *big.Int, err error) {
	m.Mock.On("SuggestGasTipCap", mock.Anything).Once().Return(tipCap, err)
}

func (m *MockClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	out := m.Mock.Called(ctx, msg)
	return out.Get(0).(uint64), out.Error(1)
}

func (m *MockClient) ExpectEstimateGas(msg ethereum.CallMsg, gas uint64, err error) {
	m.Mock.On("EstimateGas", mock.Anything, msg).Once().Return(gas, err)
}

func (m *MockClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	out := m.Mock.Called(ctx, tx)
	return out.Error(0)
}

func (m *MockClient) ExpectSendTransaction(tx *types.Transaction, err error) {
	m.Mock.On("SendTransaction", mock.Anything, tx).Once().Return(err)
}
