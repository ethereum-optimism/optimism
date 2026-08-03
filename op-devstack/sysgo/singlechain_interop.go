package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func newSingleChainInteropWorldNoSupernode(t devtest.T, keys devkeys.Keys, cfg PresetConfig) singleChainRuntimeWorld {
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
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

func startSingleChainInteropPrimaryNoSupernode(
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
	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, sequencerIdentity, ResolveMixedL2ELOpts(t)...)
	l2CL := startL2CLNode(t, keys, world.L1Network, world.L2Network, l1EL, l1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            "sequencer",
		IsSequencer:    true,
		NoDiscovery:    true,
		EnableReqResp:  true,
		DependencySet:  world.Interop.DependencySet,
		L2FollowSource: "",
		L2CLOptions:    cfg.GlobalL2CLOptions,
	})
	return singleChainPrimaryRuntime{
		EL: l2EL,
		CL: l2CL,
	}
}

// NewMinimalInteropNoSupernodeRuntime constructs the single-chain interop world
// without supernode wiring.
func NewMinimalInteropNoSupernodeRuntime(t devtest.T) *SingleChainRuntime {
	return newSingleChainRuntimeWithConfig(t, PresetConfig{}, singleChainRuntimeSpec{
		BuildWorld:      newSingleChainInteropWorldNoSupernode,
		StartPrimary:    startSingleChainInteropPrimaryNoSupernode,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
}

// NewSingleChainInteropNoSupernodeSuperRootRuntimeWithConfig constructs a single-chain
// interop world with no supernode, running an op-challenger that plays super-cannon-kona
// games sourcing super roots directly from the op-node's superroot_atTimestamp endpoint.
// The primary op-node enables its safe DB (required by superroot_atTimestamp).
func NewSingleChainInteropNoSupernodeSuperRootRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	cfg = withSuperRootGamesAtGenesisDeployerFeatures(cfg)
	cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameTypes.SuperCannonKonaGameType)
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newSingleChainInteropWorldNoSupernode,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   false,
		StartChallenger: true,
	})
}

// NewSingleChainInteropNoSupernodeZKDisputeRuntimeWithConfig constructs a single-chain interop
// world with no supernode, running an op-challenger that plays ZK dispute games sourcing super
// roots directly from the op-node's superroot_atTimestamp endpoint. The primary op-node enables
// its safe DB (required by superroot_atTimestamp).
func NewSingleChainInteropNoSupernodeZKDisputeRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	cfg = withSuperRootGamesAtGenesisDeployerFeatures(cfg)
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.ZKDisputeGameFlag),
	}, cfg.DeployerOptions...)
	// ZK games resolve only after the challenge/prove windows elapse, so time travel is required.
	cfg.EnableTimeTravel = true
	cfg.AddedGameTypes = append(cfg.AddedGameTypes, gameTypes.ZKDisputeGameType)
	return newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newSingleChainInteropWorldNoSupernode,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   false,
		StartChallenger: true,
	})
}
