package stack

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
)

type L2Challenger interface {
	Common
	ID() ComponentID
	Config() *config.Config
}
