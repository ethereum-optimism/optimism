package sysext

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

func (o *Orchestrator) hydrateSuperchain(sys stack.ExtensibleSystem) {
	env := o.env
	sys.AddSuperchain(shim.NewSuperchain(shim.SuperchainConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           stack.SuperchainID(env.Env.Name),
		Deployment:   newL1AddressBook(sys.T(), env.Env.L1.Addresses),
	}))
}

func (o *Orchestrator) hydrateClustersMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping clusters")
		return
	}

	require := sys.T().Require()
	env := o.env

	depsets := o.env.Env.DepSets

	for _, d := range depsets {
		var depSet depset.StaticConfigDependencySet
		require.NoError(json.Unmarshal(d, &depSet))

		sys.AddCluster(shim.NewCluster(shim.ClusterConfig{
			CommonConfig:  shim.NewCommonConfig(sys.T()),
			ID:            stack.ClusterID(env.Env.Name),
			DependencySet: &depSet,
		}))
	}
}

func (o *Orchestrator) hydrateSupervisorsMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping supervisors")
		return
	}

	supervisors := make(map[stack.SupervisorID]bool)
	for _, l2 := range o.env.Env.L2 {
		if supervisorService, ok := l2.Services["supervisor"]; ok {
			for _, instance := range supervisorService {
				id := stack.SupervisorID(instance.Name)
				if supervisors[id] {
					// each supervisor appears in multiple L2s (covering the dependency set),
					// so we need to deduplicate
					continue
				}
				supervisors[id] = true
				sys.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
					CommonConfig: shim.NewCommonConfig(sys.T()),
					ID:           id,
					Client:       o.rpcClient(sys.T(), instance, RPCProtocol, "/"),
				}))
			}
		}
	}
}

func (o *Orchestrator) hydrateTestSequencersMaybe(sys stack.ExtensibleSystem) {
	// TODO(#15265) op-test-sequencer: add to devnet env and kurtosis
	sys.AddTestSequencer(shim.NewTestSequencer(shim.TestSequencerConfig{
		CommonConfig:       shim.NewCommonConfig(sys.T()),
		ID:                 stack.TestSequencerID("dummy"),
		Client:             nil,
		L2SequencerClients: nil,
	}))
}
