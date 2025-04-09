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
		ID:           stack.SuperchainID(env.Name),
		Deployment:   newL1AddressBook(sys.T(), env.L1.Addresses),
	}))
}

func (o *Orchestrator) hydrateClusterMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping cluster")
		return
	}

	require := sys.T().Require()
	env := o.env

	var depSet depset.StaticConfigDependencySet
	require.NoError(json.Unmarshal(o.env.DepSet, &depSet))

	sys.AddCluster(shim.NewCluster(shim.ClusterConfig{
		CommonConfig:  shim.NewCommonConfig(sys.T()),
		ID:            stack.ClusterID(env.Name),
		DependencySet: &depSet,
	}))
}

func (o *Orchestrator) hydrateSupervisorMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping supervisor")
		return
	}

	// hack, supervisor is part of the first L2
	supervisorService, ok := o.env.L2[0].Services["supervisor"]
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
