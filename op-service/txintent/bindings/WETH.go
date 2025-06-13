package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type WETH struct {
	BalanceOf func(addr common.Address) TypedCall[eth.ETH]              `sol:"balanceOf"`
	Transfer  func(dest common.Address, amount eth.ETH) TypedCall[bool] `sol:"transfer"`
}
