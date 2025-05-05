package sysext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func (o *Orchestrator) hydrateL1(system stack.ExtensibleSystem) {
	require := o.p.Require()
	t := system.T()

	env := o.env

	commonConfig := shim.NewCommonConfig(t)
	l1ID := eth.ChainIDFromBig(env.Env.L1.Config.ChainID)
	l1 := shim.NewL1Network(shim.L1NetworkConfig{
		NetworkConfig: shim.NetworkConfig{
			CommonConfig: commonConfig,
			ChainConfig:  env.Env.L1.Config,
		},
		ID: stack.L1NetworkID(l1ID),
	})

	for idx, node := range env.Env.L1.Nodes {
		elService, ok := node.Services[ELServiceName]
		require.True(ok, "need L1 EL service %d", idx)

		l1.AddL1ELNode(shim.NewL1ELNode(shim.L1ELNodeConfig{
			ELNodeConfig: shim.ELNodeConfig{
				CommonConfig: commonConfig,
				Client:       o.rpcClient(t, elService, RPCProtocol),
				ChainID:      l1ID,
			},
			ID: stack.L1ELNodeID{
				Key:     elService.Name,
				ChainID: l1ID,
			},
		}))

		clService, ok := node.Services[CLServiceName]
		require.True(ok, "need L1 CL service %d", idx)

		l1.AddL1CLNode(shim.NewL1CLNode(shim.L1CLNodeConfig{
			ID: stack.L1CLNodeID{
				Key:     clService.Name,
				ChainID: l1ID,
			},
			CommonConfig: commonConfig,
			Client:       o.httpClient(t, clService, HTTPProtocol),
		}))
	}

	if faucet, ok := env.Env.L1.Services["faucet"]; ok {
		l1.AddFaucet(shim.NewFaucet(shim.FaucetConfig{
			CommonConfig: commonConfig,
			Client:       o.rpcClient(t, faucet, RPCProtocol),
			ID:           stack.FaucetID{Key: faucet.Name, ChainID: l1ID},
		}))
	}

	system.AddL1Network(l1)
}
