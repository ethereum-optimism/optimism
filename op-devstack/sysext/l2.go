package sysext

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func getL2ID(net *descriptors.L2Chain) stack.L2NetworkID {
	return stack.L2NetworkID(eth.ChainIDFromBig(net.Config.ChainID))
}

func (o *Orchestrator) hydrateL2(net *descriptors.L2Chain, system stack.ExtensibleSystem) {
	t := system.T()
	commonConfig := shim.NewCommonConfig(t)

	env := o.env
	l2ID := getL2ID(net)

	l1 := system.L1Network(stack.L1NetworkID(eth.ChainIDFromBig(env.Env.L1.Config.ChainID)))

	cfg := shim.L2NetworkConfig{
		NetworkConfig: shim.NetworkConfig{
			CommonConfig: commonConfig,
			ChainConfig:  net.Config,
		},
		ID:           l2ID,
		RollupConfig: net.RollupConfig,
		Deployment:   newL2AddressBook(t, net.L1Addresses),
		Keys:         o.defineSystemKeys(t),
		Superchain:   system.Superchain(stack.SuperchainID(env.Env.Name)),
		L1:           l1,
	}
	if o.isInterop() {
		cfg.Cluster = system.Cluster(stack.ClusterID(env.Env.Name))
	}

	l2 := shim.NewL2Network(cfg)

	for _, node := range net.Nodes {
		o.hydrateL2ELCL(&node, l2)
	}
	o.hydrateBatcherMaybe(net, l2)
	o.hydrateProposerMaybe(net, l2)
	o.hydrateChallengerMaybe(net, l2)
	o.hydrateL2ProxydMaybe(net, l2)

	if faucet, ok := net.Services["faucet"]; ok {
		for _, instance := range faucet {
			l2.AddFaucet(shim.NewFaucet(shim.FaucetConfig{
				CommonConfig: commonConfig,
				Client:       o.rpcClient(t, instance, RPCProtocol, fmt.Sprintf("/chain/%s", l2.ChainID().String())),
				ID:           stack.FaucetID{Key: instance.Name, ChainID: l2.ChainID()},
			}))
		}
	}

	system.AddL2Network(l2)
}

func (o *Orchestrator) hydrateL2ELCL(node *descriptors.Node, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := l2Net.ID()

	elService, ok := node.Services[ELServiceName]
	require.True(ok, "need L2 EL service for chain", l2ID)
	elClient := o.rpcClient(l2Net.T(), elService, RPCProtocol, "/")
	l2EL := shim.NewL2ELNode(shim.L2ELNodeConfig{
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			Client:       elClient,
			ChainID:      l2ID.ChainID(),
		},
		ID: stack.L2ELNodeID{
			Key:     elService.Name,
			ChainID: l2ID.ChainID(),
		},
	})
	if strings.Contains(node.Name, "geth") {
		l2EL.SetLabel(match.LabelVendor, string(match.OpGeth))
	}
	if strings.Contains(node.Name, "reth") {
		l2EL.SetLabel(match.LabelVendor, string(match.OpReth))
	}
	l2Net.AddL2ELNode(l2EL)

	clService, ok := node.Services[CLServiceName]
	require.True(ok, "need L2 CL service for chain", l2ID)

	// it's an RPC, but 'http' in kurtosis descriptor
	clClient := o.rpcClient(l2Net.T(), clService, HTTPProtocol, "/")
	l2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		ID: stack.L2CLNodeID{
			Key:     clService.Name,
			ChainID: l2ID.ChainID(),
		},
		CommonConfig: shim.NewCommonConfig(l2Net.T()),
		Client:       clClient,
	})
	l2Net.AddL2CLNode(l2CL)
	l2CL.(stack.LinkableL2CLNode).LinkEL(l2EL)
}

func (o *Orchestrator) hydrateL2ProxydMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	proxydService, ok := net.Services["proxyd"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a proxyd service")
		return
	}

	for _, instance := range proxydService {
		l2Proxyd := shim.NewL2ELNode(shim.L2ELNodeConfig{
			ELNodeConfig: shim.ELNodeConfig{
				CommonConfig: shim.NewCommonConfig(l2Net.T()),
				Client:       o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
				ChainID:      l2ID.ChainID(),
			},
			ID: stack.L2ELNodeID{
				Key:     instance.Name,
				ChainID: l2ID.ChainID(),
			},
		})
		l2Proxyd.SetLabel(match.LabelVendor, string(match.Proxyd))
		l2Net.AddL2ELNode(l2Proxyd)
	}
}

func (o *Orchestrator) hydrateBatcherMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	batcherService, ok := net.Services["batcher"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a batcher service")
		return
	}

	for _, instance := range batcherService {
		l2Net.AddL2Batcher(shim.NewL2Batcher(shim.L2BatcherConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID: stack.L2BatcherID{
				Key:     instance.Name,
				ChainID: l2ID.ChainID(),
			},
			Client: o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
		}))
	}
}

func (o *Orchestrator) hydrateProposerMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	proposerService, ok := net.Services["proposer"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a proposer service")
		return
	}

	for _, instance := range proposerService {
		l2Net.AddL2Proposer(shim.NewL2Proposer(shim.L2ProposerConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID: stack.L2ProposerID{
				Key:     instance.Name,
				ChainID: l2ID.ChainID(),
			},
			Client: o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
		}))
	}
}

func (o *Orchestrator) hydrateChallengerMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	challengerService, ok := net.Services["challenger"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a challenger service")
		return
	}

	for _, instance := range challengerService {
		l2Net.AddL2Challenger(shim.NewL2Challenger(shim.L2ChallengerConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID:           stack.L2ChallengerID(instance.Name),
		}))
	}
}

func (o *Orchestrator) defineSystemKeys(t devtest.T) stack.Keys {
	// TODO(#15040): get actual mnemonic from Kurtosis
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	t.Require().NoError(err)

	return shim.NewKeyring(keys, t.Require())
}
