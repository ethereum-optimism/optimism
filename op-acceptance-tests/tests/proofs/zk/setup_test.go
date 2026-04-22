package zk

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func zkOpts() []presets.Option {
	return []presets.Option{
		presets.WithGameTypeAdded(gameTypes.ZKDisputeGameType),
		presets.WithDeployerOptions(sysgo.WithDevFeatureEnabled(devfeatures.ZKDisputeGameFlag)),
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
	}
}

func newSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(t, zkOpts()...)
}
