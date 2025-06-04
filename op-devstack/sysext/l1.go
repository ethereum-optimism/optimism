package sysext

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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

		// TODO(#16127): This is a hack to force direct connection to the L1 node.
		// Only enable this for kurtosis devnet.
		elService.Endpoints[RPCProtocol].ReverseProxyHeader = map[string][]string{}
		// TODO(#16127): Remove this once we have a proper reverse proxy.

		require.True(ok, "need L1 EL service %d", idx)

		l1.AddL1ELNode(shim.NewL1ELNode(shim.L1ELNodeConfig{
			ELNodeConfig: shim.ELNodeConfig{
				CommonConfig: commonConfig,
				Client:       o.rpcClient(t, elService, RPCProtocol, "/"),
				ChainID:      l1ID,
			},
			ID: stack.NewL1ELNodeID(elService.Name, l1ID),
		}))

		clService, ok := node.Services[CLServiceName]
		require.True(ok, "need L1 CL service %d", idx)

		l1.AddL1CLNode(shim.NewL1CLNode(shim.L1CLNodeConfig{
			ID:           stack.NewL1CLNodeID(clService.Name, l1ID),
			CommonConfig: commonConfig,
			Client:       o.httpClient(t, clService, HTTPProtocol, "/"),
		}))
	}

	if faucet, ok := env.Env.L1.Services["faucet"]; ok {
		for _, instance := range faucet {
			l1.AddFaucet(shim.NewFaucet(shim.FaucetConfig{
				CommonConfig: commonConfig,
				Client:       o.rpcClient(t, instance, RPCProtocol, fmt.Sprintf("/chain/%s", env.Env.L1.Config.ChainID.String())),
				ID:           stack.NewFaucetID(instance.Name, l1ID),
			}))
		}
	}

	system.AddL1Network(l1)
}
