package bindings

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type L1StandardBridge struct {
	// Read-only functions
	MESSENGER     func() TypedCall[common.Address]                                      `sol:"MESSENGER"`
	OTHERBRIDGE   func() TypedCall[common.Address]                                      `sol:"OTHER_BRIDGE"`
	Deposits      func(account common.Address, token common.Address) TypedCall[eth.ETH] `sol:"deposits"`
	L2TokenBridge func() TypedCall[common.Address]                                      `sol:"l2TokenBridge"`
	Version       func() TypedCall[string]                                              `sol:"version"`

	// Write functions
	DepositERC20            func(l1Token common.Address, l2Token common.Address, amount eth.ETH, minGasLimit uint32, extraData []byte) TypedCall[any]                     `sol:"depositERC20"`
	DepositERC20To          func(l1Token common.Address, l2Token common.Address, to common.Address, amount eth.ETH, minGasLimit uint32, extraData []byte) TypedCall[any]  `sol:"depositERC20To"`
	DepositETH              func(minGasLimit uint32, extraData []byte) TypedCall[any]                                                                                     `sol:"depositETH"`
	DepositETHTo            func(to common.Address, minGasLimit uint32, extraData []byte) TypedCall[any]                                                                  `sol:"depositETHTo"`
	FinalizeERC20Withdrawal func(l1Token common.Address, l2Token common.Address, from common.Address, to common.Address, amount eth.ETH, extraData []byte) TypedCall[any] `sol:"finalizeERC20Withdrawal"`
	FinalizeETHWithdrawal   func(from common.Address, to common.Address, amount eth.ETH, extraData []byte) TypedCall[any]                                                 `sol:"finalizeETHWithdrawal"`
	Initialize              func(messenger common.Address, otherBridge common.Address) TypedCall[any]                                                                     `sol:"initialize"`
}
