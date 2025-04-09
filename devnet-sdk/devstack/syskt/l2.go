package syskt

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func WithL2(idx int, id stack.L2NetworkID, nodeIDs []DefaultSystemExtL2NodeIDs, l1ID stack.L1NetworkID) stack.Option {
	return func(setup *stack.Setup) {
		commonConfig := shim.CommonConfigFromSetup(setup)
		orchestrator := getOrchestrator(setup)
		env := orchestrator.env
		net := env.L2[idx]

		l1 := setup.System.L1Network(l1ID)
		l1ChainID := l1.ChainID()
		l2ID := eth.ChainIDFromBig(net.Config.ChainID)

		cfg := shim.L2NetworkConfig{
			NetworkConfig: shim.NetworkConfig{
				CommonConfig: commonConfig,
				ChainConfig:  net.Config,
			},
			ID: id,
			RollupConfig: &rollup.Config{
				L1ChainID: l1ChainID.ToBig(),
				L2ChainID: l2ID.ToBig(),
			},
			Deployment: newL2AddressBook(setup, net.L1Addresses),
			Keys:       defineSystemKeys(setup),
			Superchain: setup.System.Superchain(stack.SuperchainID(env.Name)),
			L1:         l1,
		}
		if orchestrator.isInterop() {
			cfg.Cluster = setup.System.Cluster(stack.ClusterID(env.Name))
		}

		l2 := shim.NewL2Network(cfg)

		for idx, node := range net.Nodes {
			ids := nodeIDs[idx]

			elRPC, err := findProtocolService(setup, ELServiceName, RPCProtocol, node.Services)
			setup.Require.NoError(err)
			elClient := rpcClient(setup, elRPC)
			l2.AddL2ELNode(shim.NewL2ELNode(shim.L2ELNodeConfig{
				ELNodeConfig: shim.ELNodeConfig{
					CommonConfig: commonConfig,
					Client:       elClient,
					ChainID:      l2ID,
				},
				ID: ids.EL,
			}))

			clRPC, err := findProtocolService(setup, CLServiceName, HTTPProtocol, node.Services)
			setup.Require.NoError(err)
			clClient := rpcClient(setup, clRPC)
			l2.AddL2CLNode(shim.NewL2CLNode(shim.L2CLNodeConfig{
				ID:           ids.CL,
				CommonConfig: commonConfig,
				Client:       clClient,
			}))
		}

		for name, wallet := range net.Wallets {
			priv, err := decodePrivateKey(wallet.PrivateKey)
			setup.Require.NoError(err)
			l2.AddUser(shim.NewUser(shim.UserConfig{
				CommonConfig: commonConfig,
				ID:           stack.UserID{Key: name, ChainID: l2ID},
				Priv:         priv,
				EL:           l2.L2ELNode(l2.L2ELNodes()[0]),
			}))
		}

		setup.System.AddL2Network(l2)
	}
}

func WithBatcher(idx int, l2ID stack.L2NetworkID, id stack.L2BatcherID) stack.Option {
	return func(setup *stack.Setup) {
		commonConfig := shim.CommonConfigFromSetup(setup)
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		batcherRPC, err := findProtocolService(setup, "batcher", HTTPProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(stack.ExtensibleL2Network).AddL2Batcher(shim.NewL2Batcher(shim.L2BatcherConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Client:       rpcClient(setup, batcherRPC),
		}))
	}
}

func WithProposer(idx int, l2ID stack.L2NetworkID, id stack.L2ProposerID) stack.Option {
	return func(setup *stack.Setup) {
		commonConfig := shim.CommonConfigFromSetup(setup)
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		proposerRPC, err := findProtocolService(setup, "proposer", HTTPProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(stack.ExtensibleL2Network).AddL2Proposer(shim.NewL2Proposer(shim.L2ProposerConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Client:       rpcClient(setup, proposerRPC),
		}))
	}
}

func WithChallenger(idx int, l2ID stack.L2NetworkID, id stack.L2ChallengerID) stack.Option {
	return func(setup *stack.Setup) {
		commonConfig := shim.CommonConfigFromSetup(setup)
		env := getOrchestrator(setup).env
		net := env.L2[idx]

		l2 := setup.System.L2Network(l2ID)

		_, err := findProtocolService(setup, "challenger", MetricsProtocol, net.Services)
		setup.Require.NoError(err)
		l2.(stack.ExtensibleL2Network).AddL2Challenger(shim.NewL2Challenger(shim.L2ChallengerConfig{
			CommonConfig: commonConfig,
			ID:           id,
		}))
	}
}

func defineSystemKeys(setup *stack.Setup) stack.L2Keys {
	// TODO(#15040): get actual mnemonic from Kurtosis
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	setup.Require.NoError(err)

	return &keyring{
		keys:  keys,
		setup: setup,
	}
}
