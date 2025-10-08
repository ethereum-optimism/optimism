package sync_tester_safesourcel2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m,
		presets.WithSimpleWithSyncTesterSafeSourceL2(),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
