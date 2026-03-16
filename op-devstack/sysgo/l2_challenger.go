package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Challenger struct {
	name     string
	chainIDs []eth.ChainID
	service  cliapp.Lifecycle
	config   *config.Config
}

func (p *L2Challenger) Config() *config.Config {
	return p.config
}
