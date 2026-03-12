package cannon

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func cannonOpts() []presets.Option {
	return []presets.Option{
		presets.WithGameTypeAdded(gameTypes.CannonGameType),
		presets.WithCannonKonaGameTypeAdded(),
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.SafeDBPath = p.TempDir()
		})),
	}
}

func newSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(t, cannonOpts()...)
}
