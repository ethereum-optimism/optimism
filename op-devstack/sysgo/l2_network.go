package sysgo

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Network struct {
	name       string
	chainID    eth.ChainID
	l1ChainID  eth.ChainID
	genesis    *opgenesis.Genesis
	rollupCfg  *rollup.Config
	deployment *L2Deployment
	opcmImpl   common.Address
	mipsImpl   common.Address
	keys       devkeys.Keys
	// depSet is the interop dependency set that op-deployer recorded for the deployment. A CL
	// node needs it to start once the rollup config schedules Lagoon, and a chain outside the
	// superchain-registry has no other source for it.
	depSet depset.DependencySet
}

func (c *L2Network) Name() string {
	return c.name
}

func (c *L2Network) ChainID() eth.ChainID {
	return c.chainID
}

func (c *L2Network) L1ChainID() eth.ChainID {
	return c.l1ChainID
}

func (c *L2Network) ChainConfig() *opparams.ChainConfig {
	return c.genesis.Config
}

func (c *L2Network) RollupConfig() *rollup.Config {
	return c.rollupCfg
}

func (c *L2Network) DependencySet() depset.DependencySet {
	return c.depSet
}

func (c *L2Network) Deployment() stack.L2Deployment {
	return c.deployment
}

func (c *L2Network) Keys() devkeys.Keys {
	return c.keys
}
