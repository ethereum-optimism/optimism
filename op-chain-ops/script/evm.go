package script

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type EVM interface {
	ChainConfig() *params.ChainConfig
	Context() *vm.BlockContext
	TxContext() *vm.TxContext
	Call(from common.Address, to common.Address, input []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error)
	Create(from common.Address, code []byte, gas uint64, value *uint256.Int) ([]byte, common.Address, uint64, error)
	Config() *vm.Config
	SetTxContext(txContext vm.TxContext)
	StateDB() vm.StateDB
}

type wrappedEVM struct {
	evm *vm.EVM
}

func WrapEVM(evm *vm.EVM) EVM {
	return &wrappedEVM{evm: evm}
}

func (w *wrappedEVM) ChainConfig() *params.ChainConfig {
	return w.evm.ChainConfig()
}

func (w *wrappedEVM) Context() *vm.BlockContext {
	return &w.evm.Context
}

func (w *wrappedEVM) TxContext() *vm.TxContext {
	return &w.evm.TxContext
}

func (w *wrappedEVM) Call(from common.Address, to common.Address, input []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error) {
	return w.evm.Call(from, to, input, gas, value)
}

func (w *wrappedEVM) Create(from common.Address, code []byte, gas uint64, value *uint256.Int) ([]byte, common.Address, uint64, error) {
	return w.evm.Create(from, code, gas, value)
}

func (w *wrappedEVM) Config() *vm.Config {
	return &w.evm.Config
}

func (w *wrappedEVM) SetTxContext(txContext vm.TxContext) {
	w.evm.SetTxContext(txContext)
}

func (w *wrappedEVM) StateDB() vm.StateDB {
	return w.evm.StateDB
}
