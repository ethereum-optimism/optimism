package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// SuperchainETHBridge binds the SuperchainETHBridge predeploy.
type SuperchainETHBridge struct {
	// Write functions. SendETH is payable: pass the amount with txplan.WithValue.
	SendETH func(to common.Address, chainID eth.ChainID) TypedCall[eth.Bytes32] `sol:"sendETH"`
}
