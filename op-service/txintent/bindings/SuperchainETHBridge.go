package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// PendingETHSendResult is the tuple returned by the SuperchainETHBridge's `pendingETHSends`
// mapping getter. A zero From means no pending send is recorded for that message hash.
type PendingETHSendResult struct {
	From   common.Address
	Amount *big.Int
}

// SuperchainETHBridge binds the SuperchainETHBridge predeploy
// (predeploys.SuperchainETHBridgeAddr, 0x42...24).
type SuperchainETHBridge struct {
	// Read-only functions
	PendingETHSends func(msgHash [32]byte) TypedCall[PendingETHSendResult] `sol:"pendingETHSends"`
	Version         func() TypedCall[string]                               `sol:"version"`

	// Write functions. SendETH is payable: pass the amount with txplan.WithValue.
	SendETH func(to common.Address, chainID eth.ChainID) TypedCall[eth.Bytes32] `sol:"sendETH"`
}
