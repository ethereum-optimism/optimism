package sync_tester

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimalWithSyncTester(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
