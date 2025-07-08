package bindings

import (
	"math/big"
)

type SystemConfig struct {
	L2ChainID func() TypedCall[*big.Int] `sol:"l2ChainId"`
}

func NewSystemConfig(opts ...CallFactoryOption) *SystemConfig {
	sys := NewBindings[SystemConfig](opts...)
	return &sys
}
