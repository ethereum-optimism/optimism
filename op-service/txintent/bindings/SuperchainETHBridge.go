package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// SuperchainETHBridge binds the SuperchainETHBridge predeploy.
type SuperchainETHBridge struct {
	// Write functions. SendETH is payable: pass the amount with txplan.WithValue.
	SendETH   func(to common.Address, chainID eth.ChainID) TypedCall[eth.Bytes32]                                                   `sol:"sendETH"`
	RefundETH func(destination eth.ChainID, nonce *big.Int, from common.Address, to common.Address, amount *big.Int) TypedCall[any] `sol:"refundETH"`
}
