package sysgo

import (
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Network struct {
	id         stack.L2NetworkID
	l1ChainID  eth.ChainID
	genesis    *core.Genesis
	rollupCfg  *rollup.Config
	deployment *L2Deployment
	keys       devkeys.Keys
}

func (c *L2Network) hydrate(system stack.ExtensibleSystem) {
	l1NetId := system.L1NetworkID(c.l1ChainID)
	l1Net := system.L1Network(l1NetId)
	sysL2Net := shim.NewL2Network(shim.L2NetworkConfig{
		NetworkConfig: shim.NetworkConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			ChainConfig:  c.genesis.Config,
		},
		ID:           c.id,
		RollupConfig: c.rollupCfg,
		Deployment:   c.deployment,
		Keys:         shim.NewKeyring(c.keys, system.T().Require()),
		Superchain:   nil,
		L1:           l1Net,
		Cluster:      nil,
	})
	system.AddL2Network(sysL2Net)
}
