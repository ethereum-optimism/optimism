package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// MessageExpiryHub binds the ownerless MessageExpiryHub L1 contract.
type MessageExpiryHub struct {
	// Read-only functions
	Facts   func(factID [32]byte) TypedCall[bool]                                                                        `sol:"facts"`
	FactId  func(cluster [32]byte, messageHash [32]byte, source eth.ChainID, undeliveredAt *big.Int) TypedCall[[32]byte] `sol:"factId"`
	Cluster func(systemConfig common.Address) TypedCall[[32]byte]                                                        `sol:"cluster"`
	Version func() TypedCall[string]                                                                                     `sol:"version"`

	// Write functions
	ForwardUndeliveredMessage func(source common.Address, messageHash [32]byte, undeliveredAt *big.Int, minGasLimit uint32) TypedCall[any] `sol:"forwardUndeliveredMessage"`
}
