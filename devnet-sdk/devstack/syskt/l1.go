package syskt

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func WithL1(id stack.L1NetworkID, nodes []DefaultSystemExtL1NodeIDs) stack.Option {
	return func(setup *stack.Setup) {
		env := getOrchestrator(setup).env

		commonConfig := shim.CommonConfigFromSetup(setup)
		l1ID := eth.ChainIDFromBig(env.L1.Config.ChainID)
		l1 := shim.NewL1Network(shim.L1NetworkConfig{
			NetworkConfig: shim.NetworkConfig{
				CommonConfig: commonConfig,
				ChainConfig:  env.L1.Config,
			},
			ID: id,
		})

		for idx, node := range env.L1.Nodes {
			ids := nodes[idx]

			elRPC, err := findProtocolService(setup, ELServiceName, RPCProtocol, node.Services)
			setup.Require.NoError(err)
			elClient := rpcClient(setup, elRPC)
			l1.AddL1ELNode(shim.NewL1ELNode(shim.L1ELNodeConfig{
				ELNodeConfig: shim.ELNodeConfig{
					CommonConfig: commonConfig,
					Client:       elClient,
					ChainID:      l1ID,
				},
				ID: ids.EL,
			}))

			clHTTP, err := findProtocolService(setup, CLServiceName, HTTPProtocol, node.Services)
			setup.Require.NoError(err)
			l1.AddL1CLNode(shim.NewL1CLNode(shim.L1CLNodeConfig{
				ID:           ids.CL,
				CommonConfig: commonConfig,
				Client:       client.NewBasicHTTPClient(clHTTP, setup.Log),
			}))
		}

		for name, wallet := range env.L1.Wallets {
			priv, err := decodePrivateKey(wallet.PrivateKey)
			setup.Require.NoError(err)
			l1.AddUser(shim.NewUser(shim.UserConfig{
				CommonConfig: commonConfig,
				ID:           stack.UserID{Key: name, ChainID: l1ID},
				Priv:         priv,
				EL:           l1.L1ELNode(l1.L1ELNodes()[0]),
			}))
		}

		setup.System.AddL1Network(l1)
	}
}
