package bindings

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type GasPriceOracle struct {
	// Read-only functions
	BaseFeeScalar      func() TypedCall[uint32]                         `sol:"baseFeeScalar"`
	BlobBaseFeeScalar  func() TypedCall[uint32]                         `sol:"blobBaseFeeScalar"`
	L1BaseFee          func() TypedCall[*eth.ETH]                       `sol:"l1BaseFee"`
	BlobBaseFee        func() TypedCall[*eth.ETH]                       `sol:"blobBaseFee"`
	IsFjord            func() TypedCall[bool]                           `sol:"isFjord"`
	GetL1Fee           func(data []byte) TypedCall[eth.ETH]             `sol:"getL1Fee"`
	GetL1GasUsed       func(data []byte) TypedCall[uint64]              `sol:"getL1GasUsed"`
	GetL1FeeUpperBound func(unsignedTxSize *big.Int) TypedCall[eth.ETH] `sol:"getL1FeeUpperBound"`
}

func NewGasPriceOracle(opts ...CallFactoryOption) *GasPriceOracle {
	gpo := NewBindings[GasPriceOracle](opts...)
	return &gpo
}
