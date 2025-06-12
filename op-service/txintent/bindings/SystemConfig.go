package bindings

import (
	"github.com/ethereum/go-ethereum/common"
)

type SystemConfig struct {
	OptimismPortal     func() TypedCall[common.Address] `sol:"optimismPortal"`
	DisputeGameFactory func() TypedCall[common.Address] `sol:"disputeGameFactory"`
}

func NewSystemConfig(opts ...CallFactoryOption) SystemConfig {
	return NewBindings[SystemConfig](opts...)
}
