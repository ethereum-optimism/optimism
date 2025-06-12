package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

type Portal2 struct {
	AnchorStateRegistry func() TypedCall[common.Address] `sol:"anchorStateRegistry"`
}

func NewPortal2(opts ...CallFactoryOption) Portal2 {
	return NewBindings[Portal2](opts...)
}
