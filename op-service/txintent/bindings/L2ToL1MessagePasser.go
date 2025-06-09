package bindings

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type L2ToL1MessagePasserCallFactory struct {
	BaseCallFactory
}

func NewL2ToL1MessagePasserFactory(opts ...CallFactoryOption) *L2ToL1MessagePasserCallFactory {
	return &L2ToL1MessagePasserCallFactory{BaseCallFactory: *NewBaseCallFactory(opts...)}
}

type L2ToL1MessagePasser struct {
	L2ToL1MessagePasserCallFactory

	// Read-only functions
	MESSAGEVERSION func() TypedCall[uint16]                   `sol:"MESSAGE_VERSION"`
	MessageNonce   func() TypedCall[*big.Int]                 `sol:"messageNonce"`
	SentMessages   func(messageHash [32]byte) TypedCall[bool] `sol:"sentMessages"`
	Version        func() TypedCall[string]                   `sol:"version"`

	// Write functions
	Burn               func() TypedCall[any]                                                      `sol:"burn"`
	InitiateWithdrawal func(target common.Address, gasLimit *big.Int, data []byte) TypedCall[any] `sol:"initiateWithdrawal"`
	Receive            func() TypedCall[any]                                                      `sol:"receive"`
}

func NewL2ToL1MessagePasser(f *L2ToL1MessagePasserCallFactory) *L2ToL1MessagePasser {
	l2tol1messagepasser := L2ToL1MessagePasser{L2ToL1MessagePasserCallFactory: *f}
	InitImpl(&l2tol1messagepasser)
	return &l2tol1messagepasser
}
