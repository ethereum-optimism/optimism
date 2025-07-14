package sync_tester

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTester(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimalWithSyncTester(t)
	require := t.Require()

	dsl.CheckAll(t, sys.L2CL.AdvancedFn(types.LocalUnsafe, 5, 30))

	syncTester := sys.SyncTester.Escape()

	chainID, err := syncTester.API().ChainID(t.Ctx())
	require.NoError(err)

	require.Equal(chainID, sys.L2EL.ChainID())
}
