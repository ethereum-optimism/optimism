package cannon

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
		// Requires access to a challenger config which only sysgo provides
		// These tests would also be exceptionally slow on real L1s
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
