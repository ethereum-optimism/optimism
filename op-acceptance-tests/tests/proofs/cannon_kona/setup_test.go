package cannon_kona

import (
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func cannonOpts() []presets.Option {
	return []presets.Option{
		presets.WithGameTypeAdded(gameTypes.CannonKonaGameType),
		presets.WithDeployerOptions(sysgo.WithJovianAtGenesis),
		presets.WithTimeTravelEnabled(),
	}
}

func newSystem(t devtest.T) *presets.Minimal {
	return presets.NewMinimal(t, cannonOpts()...)
}
