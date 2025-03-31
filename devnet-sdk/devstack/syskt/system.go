package syskt

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type DefaultSystemExtIDs struct {
	L1    stack.L1NetworkID
	Nodes []DefaultSystemExtL1NodeIDs

	Superchain stack.SuperchainID
	Cluster    stack.ClusterID

	Supervisor stack.SupervisorID

	L2s []DefaultSystemExtL2IDs
}

type DefaultSystemExtL1NodeIDs struct {
	EL stack.L1ELNodeID
	CL stack.L1CLNodeID
}

type DefaultSystemExtL2NodeIDs struct {
	EL stack.L2ELNodeID
	CL stack.L2CLNodeID
}

type DefaultSystemExtL2IDs struct {
	L2 stack.L2NetworkID

	Nodes []DefaultSystemExtL2NodeIDs

	L2Batcher    stack.L2BatcherID
	L2Proposer   stack.L2ProposerID
	L2Challenger stack.L2ChallengerID
}

func DefaultSystemExt(env *descriptors.DevnetEnvironment, opts ...OrchestratorOption) (DefaultSystemExtIDs, stack.Option) {
	ids := collectSystemExtIDs(env)

	opt := stack.Option(func(setup *stack.Setup) {
		setup.Log.Info("Mapping descriptor")

		setup.Require.NotNil(setup.Orchestrator, "need orchestrator")
		orchestrator, ok := setup.Orchestrator.(*Orchestrator)
		setup.Require.True(ok, "need orchestrator")
		setup.Require.Nil(orchestrator.env, "orchestrator env should be nil")
		setup.Require.NotNil(env, "env should not be nil")
		orchestrator.env = env
		for _, o := range opts {
			o(orchestrator)
		}
	})

	opt.Add(WithL1(ids.L1, ids.Nodes))
	opt.Add(WithSuperchain(ids.Superchain))
	opt.Add(WithSupervisor(ids.Supervisor))
	opt.Add(WithCluster(ids.Cluster))

	for idx := range env.L2 {
		l2IDs := ids.L2s[idx]
		opt.Add(WithL2(idx, l2IDs.L2, l2IDs.Nodes, ids.L1))

		opt.Add(WithBatcher(idx, l2IDs.L2, l2IDs.L2Batcher))
		opt.Add(WithProposer(idx, l2IDs.L2, l2IDs.L2Proposer))
		opt.Add(WithChallenger(idx, l2IDs.L2, l2IDs.L2Challenger))
	}

	return ids, opt
}

func WithSuperchain(id stack.SuperchainID) stack.Option {
	return func(setup *stack.Setup) {
		commonConfig := shim.CommonConfigFromSetup(setup)
		env := getOrchestrator(setup).env

		setup.System.AddSuperchain(shim.NewSuperchain(shim.SuperchainConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Deployment:   newL1AddressBook(setup, env.L1.Addresses),
		}))
	}
}

func WithSupervisor(id stack.SupervisorID) stack.Option {
	return func(setup *stack.Setup) {
		orchestrator := getOrchestrator(setup)
		if !orchestrator.isInterop() {
			return
		}

		// ideally we should check supervisor is consistent across all L2s
		// but that's what Kurtosis does.
		supervisorRPC, err := findProtocolService(setup, "supervisor", RPCProtocol, orchestrator.env.L2[0].Services)
		setup.Require.NoError(err)
		supervisorClient := rpcClient(setup, supervisorRPC)
		setup.System.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
			CommonConfig: shim.CommonConfigFromSetup(setup),
			ID:           id,
			Client:       supervisorClient,
		}))
	}
}

func WithCluster(id stack.ClusterID) stack.Option {
	return func(setup *stack.Setup) {
		orchestrator := getOrchestrator(setup)
		if !orchestrator.isInterop() {
			return
		}

		var depSet depset.StaticConfigDependencySet
		setup.Require.NoError(json.Unmarshal(orchestrator.env.DepSet, &depSet))

		setup.System.AddCluster(shim.NewCluster(shim.ClusterConfig{
			CommonConfig:  shim.CommonConfigFromSetup(setup),
			ID:            id,
			DependencySet: &depSet,
		}))
	}
}
