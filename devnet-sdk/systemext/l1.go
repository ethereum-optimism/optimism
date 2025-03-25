package systemext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func WithL1(id system2.L1NetworkID, nodes []DefaultSystemExtL1NodeIDs) system2.Option {
	return func(setup *system2.Setup) {
		env := getOrchestrator(setup).env

		commonConfig := setup.CommonConfig()
		l1ID := eth.ChainIDFromBig(env.L1.Config.ChainID)
		l1 := system2.NewL1Network(system2.L1NetworkConfig{
			NetworkConfig: system2.NetworkConfig{
				CommonConfig: commonConfig,
				ChainConfig:  env.L1.Config,
			},
			ID: system2.L1NetworkID{
				Key:     env.L1.Name,
				ChainID: l1ID,
			},
		})

		for idx, node := range env.L1.Nodes {
			ids := nodes[idx]

			elRPC, err := findProtocolService(setup, ELServiceName, RPCProtocol, node.Services)
			require.NoError(setup.T, err)
			elClient := rpcClient(setup, elRPC)
			l1.AddL1ELNode(system2.NewL1ELNode(system2.L1ELNodeConfig{
				ELNodeConfig: system2.ELNodeConfig{
					CommonConfig: commonConfig,
					Client:       elClient,
					ChainID:      l1ID,
				},
				ID: ids.EL,
			}))

			clHTTP, err := findProtocolService(setup, CLServiceName, HTTPProtocol, node.Services)
			require.NoError(setup.T, err)
			l1.AddL1CLNode(system2.NewL1CLNode(system2.L1CLNodeConfig{
				ID:           ids.CL,
				CommonConfig: commonConfig,
				Client:       client.NewBasicHTTPClient(clHTTP, setup.Log),
			}))
		}

		for name, wallet := range env.L1.Wallets {
			priv, err := crypto.HexToECDSA(wallet.PrivateKey)
			require.NoError(setup.T, err)
			l1.AddUser(system2.NewUser(system2.UserConfig{
				CommonConfig: commonConfig,
				ID:           system2.UserID{Key: name, ChainID: l1ID},
				Priv:         priv,
			}))
		}

		setup.System.AddL1Network(l1)
	}
}
