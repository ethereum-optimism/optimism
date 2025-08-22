package sync_tester_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleWithSyncTester(sttypes.FCUState{
		Latest:    0,
		Safe:      0,
		Finalized: 0,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
