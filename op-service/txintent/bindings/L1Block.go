package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type L1Block struct {
	// Read-only functions
	Number              func() TypedCall[uint64]      `sol:"number"`
	Timestamp           func() TypedCall[uint64]      `sol:"timestamp"`
	Basefee             func() TypedCall[*big.Int]    `sol:"basefee"`
	Hash                func() TypedCall[common.Hash] `sol:"hash"`
	SequenceNumber      func() TypedCall[uint64]      `sol:"sequenceNumber"`
	BatcherHash         func() TypedCall[common.Hash] `sol:"batcherHash"`
	L1FeeOverhead       func() TypedCall[*big.Int]    `sol:"l1FeeOverhead"`
	L1FeeScalar         func() TypedCall[*big.Int]    `sol:"l1FeeScalar"`
	BasefeeScalar       func() TypedCall[uint32]      `sol:"baseFeeScalar"`
	BlobbasefeeScalar   func() TypedCall[uint32]      `sol:"blobBaseFeeScalar"`
	OperatorFeeScalar   func() TypedCall[uint32]      `sol:"operatorFeeScalar"`
	OperatorFeeConstant func() TypedCall[uint64]      `sol:"operatorFeeConstant"`
	BlobBaseFee         func() TypedCall[*big.Int]    `sol:"blobBaseFee"`
	BlobBaseFeeScalar   func() TypedCall[uint32]      `sol:"blobBaseFeeScalar"`
}

func NewL1Block(opts ...CallFactoryOption) *L1Block {
	l1b := NewBindings[L1Block](opts...)
	return &l1b
}
