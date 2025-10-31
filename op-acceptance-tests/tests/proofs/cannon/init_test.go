package cannon

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithProofs(),
		presets.WithJovianAtGenesis(),
		presets.WithSafeDBEnabled(),
		// Requires access to a challenger config which only sysgo provides
		// These tests would also be exceptionally slow on real L1s
		presets.WithCompatibleTypes(compat.SysGo))
}
