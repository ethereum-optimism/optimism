package sync_tester_e2e

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithSimpleWithSyncTester(eth.FCUState{
		Latest:    0,
		Safe:      0,
		Finalized: 0,
	}),
		presets.WithCompatibleTypes(compat.SysGo),
	)
}
