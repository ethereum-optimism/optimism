package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SystemConfig struct {
	// Read-only functions
	L2ChainID           func() TypedCall[*big.Int]       `sol:"l2ChainId"`
	OperatorFeeScalar   func() TypedCall[uint32]         `sol:"operatorFeeScalar"`
	OperatorFeeConstant func() TypedCall[uint64]         `sol:"operatorFeeConstant"`
	BasefeeScalar       func() TypedCall[uint32]         `sol:"baseFeeScalar"`
	BlobbasefeeScalar   func() TypedCall[uint32]         `sol:"blobBaseFeeScalar"`
	Owner               func() TypedCall[common.Address] `sol:"owner"`

	// Write functions
	SetOperatorFeeScalars func(operatorFeeScalar uint32, operatorFeeConstant uint64) TypedCall[any] `sol:"setOperatorFeeScalars"`
	SetGasConfig          func(overhead *big.Int, scalar *big.Int) TypedCall[any]                   `sol:"setGasConfig"`
	SetGasConfigEcotone   func(basefeeScalar uint32, blobbasefeeScalar uint32) TypedCall[any]       `sol:"setGasConfigEcotone"`
}

func NewSystemConfig(opts ...CallFactoryOption) *SystemConfig {
	sys := NewBindings[SystemConfig](opts...)
	return &sys
}
