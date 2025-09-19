package sync_tester_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithExecutionLayerSyncOnVerifiers(),
		presets.WithSimpleWithSyncTester(),
		presets.WithELSyncTarget(35),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
