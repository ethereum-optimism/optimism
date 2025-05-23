package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/lmittmann/w3"
)

type WETHCallFactory struct {
	BaseCallFactory
}

func NewWETHCallFactory(opts ...CallFactoryOption) *WETHCallFactory {
	return &WETHCallFactory{BaseCallFactory: *NewBaseCallFactory(opts...)}
}

func (f *WETHCallFactory) WithDefaultAddr() {
	f.ApplyFactoryOptions(WithTo(common.HexToAddress(predeploys.WETH)))
}

func (f *WETHCallFactory) BalanceOf(addr common.Address) txintent.CallView[eth.ETH] {
	return &BalanceOfCall{Addr: addr, WETHCallFactory: *f}
}

func (f *WETHCallFactory) Transfer(dest common.Address, amount eth.ETH) txintent.CallView[bool] {
	return &TransferCall{Dest: dest, Amount: amount, WETHCallFactory: *f}
}

type WETH struct {
	WETHCallFactory

	BalanceOf func(addr common.Address) txintent.CallView[eth.ETH]
	Transfer  func(dest common.Address, amount eth.ETH) txintent.CallView[bool]
}

func NewWETH(f *WETHCallFactory) *WETH {
	return &WETH{
		WETHCallFactory: *f,
		BalanceOf:       f.BalanceOf,
		Transfer:        f.Transfer,
	}
}

type BalanceOfCall struct {
	WETHCallFactory

	Addr common.Address
}

func (c *BalanceOfCall) EncodeInput() ([]byte, error) {
	abi := w3.MustNewFunc("balanceOf(address)", "uint256")
	calldata, err := abi.EncodeArgs(c.Addr)
	return calldata, err
}

func (c *BalanceOfCall) DecodeOutput(data []byte) (eth.ETH, error) {
	abi := w3.MustNewFunc("balanceOf(address)", "uint256")
	var result *big.Int // w3 does not like static types and panics
	err := abi.DecodeReturns(data, &result)
	var res eth.ETH
	if (*uint256.Int)(&res).SetFromBig(result) {
		panic("balanceOf result conversion failure: does not fit in uint256")
	}
	return res, err
}

type TransferCall struct {
	WETHCallFactory

	Dest   common.Address
	Amount eth.ETH
}

func (c *TransferCall) EncodeInput() ([]byte, error) {
	amount := c.Amount.ToBig()
	abi := w3.MustNewFunc("transfer(address, uint256)", "bool")
	calldata, err := abi.EncodeArgs(c.Dest, amount)
	return calldata, err
}

func (c *TransferCall) DecodeOutput(data []byte) (bool, error) {
	abi := w3.MustNewFunc("transfer(address, uint256)", "bool")
	var result bool
	err := abi.DecodeReturns(data, &result)
	return result, err
}

var _ txintent.CallView[eth.ETH] = (*BalanceOfCall)(nil)
var _ txintent.CallView[bool] = (*TransferCall)(nil)
