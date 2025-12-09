package outputs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithProofs(),
		stack.MakeCommon(sysgo.WithDeployerOptions(sysgo.WithJovianAtGenesis)),
		presets.WithSafeDBEnabled(),
		presets.WithCannonKona(),
		presets.WithCompatibleTypes(compat.SysGo, compat.Persistent),
	)
}
