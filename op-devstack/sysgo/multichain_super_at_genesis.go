package sysgo

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func withSuperRootGamesAtGenesisDeployerFeatures(cfg PresetConfig) PresetConfig {
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(devfeatures.OptimismPortalInteropFlag),
		WithDevFeatureEnabled(devfeatures.SuperRootGamesMigrationFlag),
	}, cfg.DeployerOptions...)
	return cfg
}

// NewSingleChainSuperRootAtGenesisRuntimeWithConfig builds a single-chain
// supernode runtime with SUPER_PERMISSIONED_CANNON installed in the
// permissioned slot at initial deploy.
func NewSingleChainSuperRootAtGenesisRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperRootGamesAtGenesisDeployerFeatures(cfg)
	runtime := newSingleChainSupernodeRuntimeWithConfig(t, true, cfg)
	registerAddedSuperGameTypes(t, runtime, cfg)
	attachTestSequencerToRuntime(t, runtime, "dev")
	attachSuperChallengerAndProposer(t, runtime, cfg, gameTypes.SuperPermissionedGameType)
	return runtime
}

func registerAddedSuperGameTypes(t devtest.T, runtime *MultiChainRuntime, cfg PresetConfig) {
	if len(cfg.AddedSuperGameTypes) == 0 {
		return
	}
	for _, chain := range orderedRuntimeChains(runtime) {
		for _, add := range cfg.AddedSuperGameTypes {
			registerSuperDisputeGameForRuntime(t, runtime.Keys, runtime.L1Network.ChainID(), runtime.L1EL, chain.Network, add.GameType, add.Prestate)
		}
	}
}
