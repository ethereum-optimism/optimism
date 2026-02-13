package stack

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Challenger interface {
	Common
	ChainID() eth.ChainID
	Config() *config.Config
}
