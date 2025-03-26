package systemext

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

type DefaultSystemExtIDs struct {
	L1    system2.L1NetworkID
	Nodes []DefaultSystemExtL1NodeIDs

	Superchain system2.SuperchainID
	Cluster    system2.ClusterID

	Supervisor system2.SupervisorID

	L2s []DefaultSystemExtL2IDs
}

type DefaultSystemExtL1NodeIDs struct {
	EL system2.L1ELNodeID
	CL system2.L1CLNodeID
}

type DefaultSystemExtL2NodeIDs struct {
	EL system2.L2ELNodeID
	CL system2.L2CLNodeID
}

type DefaultSystemExtL2IDs struct {
	L2 system2.L2NetworkID

	Nodes []DefaultSystemExtL2NodeIDs

	L2Batcher    system2.L2BatcherID
	L2Proposer   system2.L2ProposerID
	L2Challenger system2.L2ChallengerID
}

func DefaultSystemExt(env *descriptors.DevnetEnvironment, opts ...OrchestratorOption) (DefaultSystemExtIDs, system2.Option) {
	ids := collectSystemExtIDs(env)

	opt := system2.Option(func(setup *system2.Setup) {
		setup.Log.Info("Mapping descriptor")

		var orchestrator *Orchestrator
		if setup.Orchestrator == nil {
			orchestrator = &Orchestrator{}
			setup.Orchestrator = orchestrator
		} else {
			orchestrator = setup.Orchestrator.(*Orchestrator)
		}
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

func WithSuperchain(id system2.SuperchainID) system2.Option {
	return func(setup *system2.Setup) {
		commonConfig := setup.CommonConfig()
		env := getOrchestrator(setup).env

		setup.System.AddSuperchain(system2.NewSuperchain(system2.SuperchainConfig{
			CommonConfig: commonConfig,
			ID:           id,
			Deployment:   newL1AddressBook(setup, env.L1.Addresses),
		}))
	}
}

func WithSupervisor(id system2.SupervisorID) system2.Option {
	return func(setup *system2.Setup) {
		orchestrator := getOrchestrator(setup)
		if !orchestrator.isInterop() {
			return
		}

		// ideally we should check supervisor is consistent across all L2s
		// but that's what Kurtosis does.
		supervisorRPC, err := findProtocolService(setup, "supervisor", RPCProtocol, orchestrator.env.L2[0].Services)
		setup.Require.NoError(err)
		supervisorClient := rpcClient(setup, supervisorRPC)
		setup.System.AddSupervisor(system2.NewSupervisor(system2.SupervisorConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           id,
			Client:       supervisorClient,
		}))
	}
}

func WithCluster(id system2.ClusterID) system2.Option {
	return func(setup *system2.Setup) {
		orchestrator := getOrchestrator(setup)
		if !orchestrator.isInterop() {
			return
		}

		var depSet depset.StaticConfigDependencySet
		setup.Require.NoError(json.Unmarshal(orchestrator.env.DepSet, &depSet))

		setup.System.AddCluster(system2.NewCluster(system2.ClusterConfig{
			CommonConfig:  setup.CommonConfig(),
			ID:            id,
			DependencySet: &depSet,
		}))
	}
}
