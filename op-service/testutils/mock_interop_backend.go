package testutils

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type MockInteropBackend struct {
	Mock mock.Mock
}

func (m *MockInteropBackend) ExpectCheckBlock(chainID types.ChainID, blockNumber uint64, safety types.SafetyLevel, err error) {
	m.Mock.On("CheckBlock", chainID, blockNumber).Once().Return(safety, &err)
}

func (m *MockInteropBackend) CheckBlock(ctx context.Context, chainID types.ChainID, blockHash common.Hash, blockNumber uint64) (types.SafetyLevel, error) {
	result := m.Mock.MethodCalled("CheckBlock", chainID, blockNumber)
	return result.Get(0).(types.SafetyLevel), *result.Get(1).(*error)
}

func (m *MockInteropBackend) AssertExpectations(t mock.TestingT) {
	m.Mock.AssertExpectations(t)
}
