package sync_tester

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimalWithSyncTester(&stconf.TargetBlocks{
		Head:      1,
		Safe:      1,
		Finalized: 1,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
