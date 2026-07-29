package interopgen

import (
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1Output struct {
	Genesis *core.Genesis
}

type L2Output struct {
	Genesis   *core.Genesis
	RollupCfg *rollup.Config
}

type WorldOutput struct {
	L1  *L1Output
	L2s map[string]*L2Output
}

func (wo *WorldOutput) RollupConfigSet() config.StaticRollupConfigSet {
	rcfgs := make(map[eth.ChainID]*rollup.Config)
	for _, rcfg := range wo.L2s {
		rcfgs[eth.ChainIDFromBig(rcfg.RollupCfg.L2ChainID)] = rcfg.RollupCfg
	}
	return config.StaticRollupConfigSetFromRollupConfigMap(rcfgs, config.StaticTimestamp(wo.L1.Genesis.Timestamp))
}
