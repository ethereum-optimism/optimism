package systemext

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func WithL2(idx int, id system2.L2NetworkID, nodeIDs []DefaultSystemExtL2NodeIDs, l1ID system2.L1NetworkID) system2.Option {
	return func(setup *system2.Setup) {
		commonConfig := setup.CommonConfig()
		orchestrator := getOrchestrator(setup)
		env := orchestrator.env
		net := env.L2[idx]

		l1 := setup.System.L1Network(l1ID)
		l1ChainID := l1.ChainID()
		l2ID := eth.ChainIDFromBig(net.Config.ChainID)

		cfg := system2.L2NetworkConfig{
			NetworkConfig: system2.NetworkConfig{
				CommonConfig: commonConfig,
				ChainConfig:  net.Config,
			},
			ID: system2.L2NetworkID{
				Key:     net.Name,
				ChainID: l2ID,
			},
			RollupConfig: &rollup.Config{
				L1ChainID: l1ChainID.ToBig(),
				L2ChainID: l2ID.ToBig(),
			},
			Deployment: newL2AddressBook(setup, net.L1Addresses),
			Keys:       defineSystemKeys(setup),
			Superchain: setup.System.Superchain(system2.SuperchainID(env.Name)),
			L1:         l1,
		}
		if orchestrator.isInterop() {
			cfg.Cluster = setup.System.Cluster(system2.ClusterID(env.Name))
		}

		l2 := system2.NewL2Network(cfg)

		for idx, node := range net.Nodes {
			ids := nodeIDs[idx]

			elRPC, err := findProtocolService(setup, ELServiceName, RPCProtocol, node.Services)
			setup.Require.NoError(err)
			elClient := rpcClient(setup, elRPC)
			l2.AddL2ELNode(system2.NewL2ELNode(system2.L2ELNodeConfig{
				ELNodeConfig: system2.ELNodeConfig{
					CommonConfig: commonConfig,
					Client:       elClient,
					ChainID:      l2ID,
				},
				ID: ids.EL,
			}))

			clRPC, err := findProtocolService(setup, CLServiceName, HTTPProtocol, node.Services)
			setup.Require.NoError(err)
			clClient := rpcClient(setup, clRPC)
			l2.AddL2CLNode(system2.NewL2CLNode(system2.L2CLNodeConfig{
				ID:           ids.CL,
				CommonConfig: commonConfig,
				Client:       clClient,
			}))
		}

		for name, wallet := range net.Wallets {
			priv, err := decodePrivateKey(wallet.PrivateKey)
			setup.Require.NoError(err)
			l2.AddUser(system2.NewUser(system2.UserConfig{
				CommonConfig: commonConfig,
				ID:           system2.UserID{Key: name, ChainID: l2ID},
				Priv:         priv,
				EL:           l2.L2ELNode(l2.L2ELNodes()[0]),
			}))
		}

		setup.System.AddL2Network(l2)
	}
}

func WithBatcher(idx int, l2ID system2.L2NetworkID, id system2.L2BatcherID) system2.Option {
	return func(setup *system2.Setup) {
		commonConfig := setup.CommonConfig()
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		batcherRPC, err := findProtocolService(setup, "batcher", HTTPProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(system2.ExtensibleL2Network).AddL2Batcher(system2.NewL2Batcher(system2.L2BatcherConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Client:       rpcClient(setup, batcherRPC),
		}))
	}
}

func WithProposer(idx int, l2ID system2.L2NetworkID, id system2.L2ProposerID) system2.Option {
	return func(setup *system2.Setup) {
		commonConfig := setup.CommonConfig()
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		proposerRPC, err := findProtocolService(setup, "proposer", HTTPProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(system2.ExtensibleL2Network).AddL2Proposer(system2.NewL2Proposer(system2.L2ProposerConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Client:       rpcClient(setup, proposerRPC),
		}))
	}
}

func WithChallenger(idx int, l2ID system2.L2NetworkID, id system2.L2ChallengerID) system2.Option {
	return func(setup *system2.Setup) {
		commonConfig := setup.CommonConfig()
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		_, err := findProtocolService(setup, "challenger", MetricsProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(system2.ExtensibleL2Network).AddL2Challenger(system2.NewL2Challenger(system2.L2ChallengerConfig{
			CommonConfig: commonConfig,
			ID:           id,
		}))
	}
}

func defineSystemKeys(setup *system2.Setup) system2.L2Keys {
	// TODO(#15040): get actual mnemonic from Kurtosis
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	setup.Require.NoError(err)

	return &keyring{
		keys:  keys,
		setup: setup,
	}
}
