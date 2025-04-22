package sysext

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/shim"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
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

func (o *Orchestrator) hydrateClusterMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping cluster")
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

func (o *Orchestrator) hydrateSupervisorMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping supervisor")
		return
	}

	// hack, supervisor is part of the first L2
	supervisorService, ok := o.env.Env.L2[0].Services["supervisor"]
	if !ok {
		sys.T().Logger().Warn("Missing supervisor service")
		return
	}

	// ideally we should check supervisor is consistent across all L2s
	// but that's what Kurtosis does.
	sys.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           stack.SupervisorID(supervisorService.Name),
		Client:       o.rpcClient(sys.T(), supervisorService, RPCProtocol),
	}))
}
