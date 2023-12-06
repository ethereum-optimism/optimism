package testutils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MockSystemConfigUpdateListener struct {
	mock.Mock
}

func (m *MockSystemConfigUpdateListener) OnP2PBlockSignerAddressUpdated(addr common.Address, l1Ref eth.L1BlockRef) {
	m.Mock.MethodCalled("OnP2PBlockSignerAddressUpdated", addr, l1Ref)
}

func (m *MockSystemConfigUpdateListener) ExpectOnP2PBlockSignerAddressUpdated(addr common.Address, l1Ref eth.L1BlockRef) {
	m.Mock.On("OnP2PBlockSignerAddressUpdated", addr, l1Ref).Once()
}
