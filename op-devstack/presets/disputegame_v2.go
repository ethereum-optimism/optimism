package presets

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func WithDisputeGameV2() stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithDevFeatureEnabled(deployer.DeployV2DisputeGamesDevFlag)))
}

func WithCannonKona() stack.CommonOption {
	return stack.Combine(
		// Enable dev features required
		WithCannonKonaFeatureEnabled(),
		// Add cannon-kona game type
		stack.MakeCommon(sysgo.WithCannonKonaGameTypeAdded()),
	)
}

func WithCannonKonaFeatureEnabled() stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		sysgo.WithDevFeatureEnabled(deployer.DeployV2DisputeGamesDevFlag), // Required for cannon kona
		sysgo.WithDevFeatureEnabled(deployer.CannonKonaDevFlag),
	))
}
