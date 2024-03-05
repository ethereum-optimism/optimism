package l2

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-test/components/superchain"
)

type StandardL2 struct {
	superchain    superchain.Superchain
	genesis       *core.Genesis
	rollupCfg     *rollup.Config
	l1Deployments *genesis.L1Deployments
}

func (g *StandardL2) ChainID() *big.Int {
	return g.genesis.Config.ChainID
}

func (g *StandardL2) ChainConfig() *params.ChainConfig {
	return g.genesis.Config
}

func (g *StandardL2) RollupConfig() *rollup.Config {
	return g.rollupCfg
}

func (g *StandardL2) Genesis() *core.Genesis {
	return g.genesis
}

func (g *StandardL2) Name() string {
	return fmt.Sprintf("l2_%d", g.ChainID())
}

func (g *StandardL2) L1Deployments() *genesis.L1Deployments {
	return g.l1Deployments
}

func (g *StandardL2) Superchain() superchain.Superchain {
	return g.superchain
}

var _ L2 = (*StandardL2)(nil)
