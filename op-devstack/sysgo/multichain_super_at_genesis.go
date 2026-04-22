package sysgo

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// withSuperRootGamesAtGenesisDeployerFeatures enables both the OptimismPortal
// interop flag and the SuperRootGamesMigration dev flag in the deploy intent.
// With both bits set, DeployImplementations.s.sol deploys the super dispute
// game impls and DeployOPChain.s.sol wires SUPER_PERMISSIONED_CANNON into the
// permissioned slot and sets it as the respected game type, instead of
// PERMISSIONED_CANNON. No post-deploy OPCMv2 migration is needed.
func withSuperRootGamesAtGenesisDeployerFeatures(cfg PresetConfig) PresetConfig {
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
		WithDevFeatureEnabled(devfeatures.SuperRootGamesMigrationFlag),
	}, cfg.DeployerOptions...)
	return cfg
}

// NewSingleChainSuperRootAtGenesisRuntimeWithConfig builds a single-chain
// supernode runtime that has SuperPermissionedDisputeGame installed in the
// permissioned slot as part of the initial op-deployer apply. Only
// SUPER_PERMISSIONED_CANNON (game type 5) is active on-chain, so the runtime
// skips the standard permissioned-cannon proposer (which would have no
// implementation to target) and runs a super proposer against game type 5
// directly.
func NewSingleChainSuperRootAtGenesisRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperRootGamesAtGenesisDeployerFeatures(cfg)
	runtime := newSingleChainSupernodeRuntimeWithConfig(t, true, false, cfg)
	attachTestSequencerToRuntime(t, runtime, "dev")
	attachSuperChallengerAndProposer(t, runtime, cfg, gameTypes.SuperPermissionedGameType)
	return runtime
}
