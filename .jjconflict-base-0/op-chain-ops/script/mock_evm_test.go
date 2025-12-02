package script

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
)

type mockEVM struct {
	mock.Mock
}

func (m *mockEVM) ChainConfig() *params.ChainConfig {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	return result.(*params.ChainConfig)
}

func (m *mockEVM) Context() *vm.BlockContext {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	return result.(*vm.BlockContext)
}

func (m *mockEVM) TxContext() *vm.TxContext {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	return result.(*vm.TxContext)
}

func (m *mockEVM) Call(from common.Address, to common.Address, input []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error) {
	args := m.Called(from, to, input, gas, value)
	return args.Get(0).([]byte), args.Get(1).(uint64), args.Error(2)
}

func (m *mockEVM) Create(from common.Address, code []byte, gas uint64, value *uint256.Int) ([]byte, common.Address, uint64, error) {
	args := m.Called(from, code, gas, value)
	return args.Get(0).([]byte), args.Get(1).(common.Address), args.Get(2).(uint64), args.Error(3)
}

func (m *mockEVM) Config() *vm.Config {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	return result.(*vm.Config)
}

func (m *mockEVM) SetTxContext(txContext vm.TxContext) {
	m.Called(txContext)
}

func (m *mockEVM) StateDB() vm.StateDB {
	args := m.Called()
	result := args.Get(0)
	if result == nil {
		return nil
	}
	return result.(vm.StateDB)
}
