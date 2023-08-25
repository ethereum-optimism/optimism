package node

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
)

type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) BlockHeaderByNumber(number *big.Int) (*types.Header, error) {
	args := m.Called(number)
	return args.Get(0).(*types.Header), args.Error(1)
}

func (m *MockEthClient) FinalizedBlockHeight() (*big.Int, error) {
	args := m.Called()
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockEthClient) BlockHeadersByRange(from, to *big.Int) ([]types.Header, error) {
	args := m.Called(from, to)
	return args.Get(0).([]types.Header), args.Error(1)
}

func (m *MockEthClient) BlockHeaderByHash(hash common.Hash) (*types.Header, error) {
	args := m.Called(hash)
	return args.Get(0).(*types.Header), args.Error(1)
}

func (m *MockEthClient) StorageHash(address common.Address, blockNumber *big.Int) (common.Hash, error) {
	args := m.Called(address, blockNumber)
	return args.Get(0).(common.Hash), args.Error(1)
}

func (m *MockEthClient) GethRpcClient() *rpc.Client {
	args := m.Called()
	return args.Get(0).(*rpc.Client)
}

func (m *MockEthClient) GethEthClient() *ethclient.Client {
	args := m.Called()

	client, ok := args.Get(0).(*ethclient.Client)
	if !ok {
		return nil
	}
	return client
}
