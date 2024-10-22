package interopgen

import (
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
