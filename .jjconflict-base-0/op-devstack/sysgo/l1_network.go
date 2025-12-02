package sysgo

import (
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type L1Network struct {
	id        stack.L1NetworkID
	genesis   *core.Genesis
	blockTime uint64
}

func (n *L1Network) hydrate(system stack.ExtensibleSystem) {
	sysL1Net := shim.NewL1Network(shim.L1NetworkConfig{
		NetworkConfig: shim.NetworkConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			ChainConfig:  n.genesis.Config,
		},
		ID: n.id,
	})
	system.AddL1Network(sysL1Net)
}
