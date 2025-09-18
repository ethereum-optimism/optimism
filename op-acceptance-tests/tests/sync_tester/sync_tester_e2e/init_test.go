package sync_tester_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleWithSyncTester(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
