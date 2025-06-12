package bindings

import (
	"math/big"
)

type AnchorRoot struct {
	Hash     [32]byte
	L2SeqNum *big.Int
}

type AnchorStateRegistry struct {
	BaseCallFactory

	RespectedGameType func() TypedCall[uint32]     `sol:"respectedGameType"`
	GetAnchorRoot     func() TypedCall[AnchorRoot] `sol:"getAnchorRoot"`
}

func NewAnchorStateRegistry(opts ...CallFactoryOption) *AnchorStateRegistry {
	impl := AnchorStateRegistry{BaseCallFactory: *NewBaseCallFactory(opts...)}
	InitImpl(&impl)
	return &impl
}
