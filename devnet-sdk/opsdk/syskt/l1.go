package syskt

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/opsdk/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func WithL1(id stack.L1NetworkID, nodes []DefaultSystemExtL1NodeIDs) stack.Option {
	return func(setup *stack.Setup) {
		env := getOrchestrator(setup).env

		commonConfig := setup.CommonConfig()
		l1ID := eth.ChainIDFromBig(env.L1.Config.ChainID)
		l1 := stack.NewL1Network(stack.L1NetworkConfig{
			NetworkConfig: stack.NetworkConfig{
				CommonConfig: commonConfig,
				ChainConfig:  env.L1.Config,
			},
			ID: stack.L1NetworkID{
				Key:     env.L1.Name,
				ChainID: l1ID,
			},
		})

		for idx, node := range env.L1.Nodes {
			ids := nodes[idx]

			elRPC, err := findProtocolService(setup, ELServiceName, RPCProtocol, node.Services)
			setup.Require.NoError(err)
			elClient := rpcClient(setup, elRPC)
			l1.AddL1ELNode(stack.NewL1ELNode(stack.L1ELNodeConfig{
				ELNodeConfig: stack.ELNodeConfig{
					CommonConfig: commonConfig,
					Client:       elClient,
					ChainID:      l1ID,
				},
				ID: ids.EL,
			}))

			clHTTP, err := findProtocolService(setup, CLServiceName, HTTPProtocol, node.Services)
			setup.Require.NoError(err)
			l1.AddL1CLNode(stack.NewL1CLNode(stack.L1CLNodeConfig{
				ID:           ids.CL,
				CommonConfig: commonConfig,
				Client:       client.NewBasicHTTPClient(clHTTP, setup.Log),
			}))
		}

		for name, wallet := range env.L1.Wallets {
			priv, err := decodePrivateKey(wallet.PrivateKey)
			setup.Require.NoError(err)
			l1.AddUser(stack.NewUser(stack.UserConfig{
				CommonConfig: commonConfig,
				ID:           stack.UserID{Key: name, ChainID: l1ID},
				Priv:         priv,
				EL:           l1.L1ELNode(l1.L1ELNodes()[0]),
			}))
		}

		setup.System.AddL1Network(l1)
	}
}
