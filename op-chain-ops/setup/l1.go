package setup

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/setup/deploy"
	"github.com/ethereum-optimism/optimism/op-chain-ops/setup/script"
)

type L1 interface {
	Chain

	InitialL1(cfg genesis.DevL1DeployConfig)

	DeploySuperchain(name string, cfg genesis.SuperchainL1DeployConfig) // TODO
}

var _ L1 = (*chain)(nil)

type l1Props struct {
}

func (ch *chain) InitialL1(cfg genesis.DevL1DeployConfig) {
	// assert empty prestate
	// TODO script for empty L1 genesis state
}

func (ch *chain) DeploySuperchain(name string, cfg genesis.SuperchainL1DeployConfig) {
	_, ok := ch.w.superchains[name]
	ch.w.req.False(ok, "must be new superchain")

	superChainTarget := &superchain{name: name}

	// TODO deploy a proxyadmin to L1 for superchain
	var proxyAdmin common.Address

	deployer := &deploy.Superchain{
		Args: struct {
			genesis.SuperchainL1DeployConfig
		}{
			SuperchainL1DeployConfig: cfg,
		},
		Addresses: struct {
			ProxyAdmin common.Address `json:"ProxyAdmin"`
		}{
			ProxyAdmin: proxyAdmin,
		},
	}

	res, err := script.Run(ch.w.ctx, ch.log, ch.w.cachePath, ch.state, deployer)
	ch.w.req.NoError(err)
	ch.state = res.Post
	ch.labels.AddLabels(res.LabelsDiff)
	ch.labels.LabelDeployments(res.DeploymentsDiff)

	superChainTarget.SuperchainConfig = res.DeploymentsDiff["SuperchainConfig"]
	superChainTarget.ProtocolVersions = res.DeploymentsDiff["ProtocolVersions"]

	ch.w.superchains[name] = superChainTarget
}
