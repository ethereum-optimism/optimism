package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type OptimismMintableERC20 struct {
	// Read-only functions
	BRIDGE      func() TypedCall[common.Address]                                      `sol:"BRIDGE"`
	REMOTETOKEN func() TypedCall[common.Address]                                      `sol:"REMOTE_TOKEN"`
	BalanceOf   func(account common.Address) TypedCall[eth.ETH]                       `sol:"balanceOf"`
	Decimals    func() TypedCall[uint8]                                               `sol:"decimals"`
	Name        func() TypedCall[string]                                              `sol:"name"`
	Symbol      func() TypedCall[string]                                              `sol:"symbol"`
	TotalSupply func() TypedCall[eth.ETH]                                             `sol:"totalSupply"`
	Allowance   func(owner common.Address, spender common.Address) TypedCall[eth.ETH] `sol:"allowance"`
	Version     func() TypedCall[string]                                              `sol:"version"`

	// Write functions
	Approve           func(spender common.Address, amount eth.ETH) TypedCall[bool]                 `sol:"approve"`
	Transfer          func(to common.Address, amount eth.ETH) TypedCall[bool]                      `sol:"transfer"`
	TransferFrom      func(from common.Address, to common.Address, amount eth.ETH) TypedCall[bool] `sol:"transferFrom"`
	Mint              func(to common.Address, amount eth.ETH) TypedCall[any]                       `sol:"mint"`
	Burn              func(from common.Address, amount eth.ETH) TypedCall[any]                     `sol:"burn"`
	IncreaseAllowance func(spender common.Address, addedValue eth.ETH) TypedCall[bool]             `sol:"increaseAllowance"`
	DecreaseAllowance func(spender common.Address, subtractedValue eth.ETH) TypedCall[bool]        `sol:"decreaseAllowance"`
}
