package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func newSingleChainInteropWorldNoSupervisor(t devtest.T, keys devkeys.Keys, cfg PresetConfig) singleChainRuntimeWorld {
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(deployer.OptimismPortalInteropDevFlag),
	}, cfg.DeployerOptions...)
	l1Net, l2Net, depSet, fullCfgSet := buildSingleChainWorldWithInterop(t, keys, true, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	return singleChainRuntimeWorld{
		L1Network: l1Net,
		L2Network: l2Net,
		Interop: &SingleChainInteropSupport{
			DependencySet: depSet,
			FullConfigSet: fullCfgSet,
		},
	}
}

func startSingleChainInteropPrimaryNoSupervisor(
	t devtest.T,
	keys devkeys.Keys,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
) singleChainPrimaryRuntime {
	t.Require().NotNil(world.Interop, "single-chain interop runtime requires interop support")

	sequencerIdentity := NewELNodeIdentity(0)
	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, sequencerIdentity)
	l2CL := startL2CLNode(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		UseReqResp:     true,
		DependencySet:  world.Interop.DependencySet,
		L2FollowSource: "",
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	return singleChainPrimaryRuntime{
		EL: l2EL,
		CL: l2CL,
	}
}

// NewMinimalInteropNoSupervisorRuntime constructs the single-chain interop world
// without supervisor wiring.
func NewMinimalInteropNoSupervisorRuntime(t devtest.T) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, PresetConfig{}, singleChainRuntimeSpec{
		BuildWorld:      newSingleChainInteropWorldNoSupervisor,
		StartPrimary:    startSingleChainInteropPrimaryNoSupervisor,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
}
