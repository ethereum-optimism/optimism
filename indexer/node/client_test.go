package node

import (
	"math/big"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) FinalizedBlockHeight() (*big.Int, error) {
	args := m.Called()
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockEthClient) BlockHeadersByRange(from, to *big.Int) ([]*types.Header, error) {
	args := m.Called(from, to)
	return args.Get(0).([]*types.Header), args.Error(1)
}

func (m *MockEthClient) RawRpcClient() *rpc.Client {
	args := m.Called()
	return args.Get(0).(*rpc.Client)
}
