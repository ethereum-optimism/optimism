package sync_tester_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleWithSyncTester(&stconf.TargetBlocks{
		Head:      1,
		Safe:      1,
		Finalized: 1,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
