package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
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

type WETH struct {
	WETHCallFactory

	BalanceOf func(addr common.Address) TypedCall[eth.ETH]              `sol:"balanceOf"`
	Transfer  func(dest common.Address, amount eth.ETH) TypedCall[bool] `sol:"transfer"`
}

func NewWETH(f *WETHCallFactory) *WETH {
	weth := WETH{WETHCallFactory: *f}
	InitImpl(&weth)
	return &weth
}
